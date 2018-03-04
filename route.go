package switchboard

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

type Route interface {
	AttachHandlers(*mux.Router, Pipeline) error
	Handle([]string, io.Reader) (Tags, string, error)
}

type BasicRoute struct {
	Path    string
	Command *Command
	Methods []string
	Type    string
	Routes  map[string]Route
}

type ResourceRoute struct {
	Path    string
	Command *Command
	Routes  map[string]Route
}

type RootRoute struct {
	Routes map[string]Route
}

type Pipeline []Route

func (route *BasicRoute) AttachHandlers(router *mux.Router, pipeline Pipeline) error {
	for _, method := range route.Methods {
		if len(route.Routes) == 0 {
			log.Printf("routing to %s %s", method, route.Path)
			router.HandleFunc(route.Path, pipeline.Append(route).Handle).Methods(method)
		} else {
			log.Printf("inserting route in pipeline %s", route.Path)
			for _, child := range route.Routes {
				err := child.AttachHandlers(router, pipeline.Append(route))
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (route *BasicRoute) Handle(env []string, stdin io.Reader) (Tags, string, error) {
	log.Printf("executing command %s for route %s", route.Command.Name, route.Path)
	status, routeTags, stdout, err := route.Command.Execute(env, stdin)
	if err != nil {
		log.Print("failed to execute command: %s", err)
		return nil, "", err
	}

	body, err := ioutil.ReadAll(stdout)
	if err != nil {
		log.Print("command failed to execute correctly")
		return nil, "", err
	}

	if status != 0 {
		log.Printf("command completed with a nonzero exit status %d", status)
		return nil, string(body), err
	}

	return routeTags, string(body), nil
}

func (route *ResourceRoute) AttachHandlers(router *mux.Router, pipeline Pipeline) error {
	resourcesPath := route.Path
	resourcePath := fmt.Sprintf("%s/:id", resourcesPath)

	for _, method := range []string{"GET", "POST"} {
		log.Printf("routing to %s %s", method, resourcesPath)
		router.HandleFunc(resourcesPath, pipeline.Append(route).Handle).Methods(method)
	}

	for _, method := range []string{"GET", "PUT", "PATCH", "DELETE"} {
		log.Printf("routing to %s %s", method, resourcePath)
		router.HandleFunc(resourcePath, pipeline.Append(route).Handle).Methods(method)
	}

	if len(route.Routes) > 0 {
		log.Printf("inserting route in pipeline %s", route.Path)
		for _, child := range route.Routes {
			err := child.AttachHandlers(router, pipeline.Append(route))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (route *ResourceRoute) Handle(env []string, stdin io.Reader) (Tags, string, error) {
	log.Printf("executing command %s for route %s", route.Command.Name, route.Path)
	status, routeTags, stdout, err := route.Command.Execute(env, stdin)
	if err != nil {
		log.Print("failed to execute command: %s", err)
		return nil, "", err
	}

	body, err := ioutil.ReadAll(stdout)
	if err != nil {
		log.Print("command failed to execute correctly")
		return nil, "", err
	}

	if status != 0 {
		log.Printf("command completed with a nonzero exit status %d", status)
		return nil, string(body), err
	}

	return routeTags, string(body), nil
}

func (route *RootRoute) AttachHandlers(router *mux.Router, pipeline Pipeline) error {
	for _, child := range route.Routes {
		child.AttachHandlers(router, pipeline.Copy())
	}

	return nil
}

func (route *RootRoute) Handle([]string, io.Reader) (Tags, string, error) {
	return nil, "", errors.New("root route cannot be executed")
}

func (pipeline Pipeline) Handle(w http.ResponseWriter, r *http.Request) {
	log.Printf("handling route %s", r.URL.Path)
	env := RequestToEnv(r)
	tags := make(Tags)
	stdin := io.Reader(r.Body)

	for _, route := range pipeline {
		routeTags, body, err := route.Handle(env, stdin)
		if err != nil {
			if body == "" {
				body = err.Error()
			}
			http.Error(w, body, http.StatusInternalServerError)
			return
		}

		halt, err := ApplyBetweenTags(routeTags, tags, &env)
		if err != nil {
			log.Print("failed to apply tags")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		stdin = strings.NewReader(body)

		if halt {
			break
		}
	}

	err := ApplyEndTags(tags, w)
	if err != nil {
		log.Print("failed to apply tags")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	io.Copy(w, stdin)
}

func (pipeline Pipeline) Append(route Route) Pipeline {
	return append(pipeline.Copy(), route)
}

func (pipeline Pipeline) Copy() Pipeline {
	p := make(Pipeline, len(pipeline))
	copy(p, pipeline)
	return p
}

func RequestToEnv(r *http.Request) []string {
	r.URL.Host = r.Host
	r.URL.Scheme = "http"
	if r.TLS != nil {
		r.URL.Scheme = "https"
	}

	env := []string{
		fmt.Sprintf("HTTP_METHOD=%s", r.Method),
		fmt.Sprintf("HTTP_URL=%s", r.URL.String()),
		fmt.Sprintf("HTTP_URL_SCHEME=%s", r.URL.Scheme),
		fmt.Sprintf("HTTP_URL_HOST=%s", r.URL.Host),
		fmt.Sprintf("HTTP_URL_PORT=%s", r.URL.Port()),
		fmt.Sprintf("HTTP_URL_PATH=%s", r.URL.Path),
		fmt.Sprintf("HTTP_URL_QUERY=%s", r.URL.RawQuery),
		fmt.Sprintf("HTTP_URL_FRAGMENT=%s", r.URL.Fragment),
	}

	if r.URL.User != nil {
		username := r.URL.User.Username()
		env = append(env, fmt.Sprintf("HTTP_URL_USERNAME=%s", username))

		password, ok := r.URL.User.Password()
		if ok {
			env = append(env, fmt.Sprintf("HTTP_URL_PASSWORD=%s", password))
		}
	}

	for k, v := range r.Header {
		k = strings.Replace(k, "-", "_", -1)
		k = strings.ToUpper(k)
		k = fmt.Sprintf("HTTP_HEADER_%s", k)
		env = append(env, fmt.Sprintf("%s=%s", k, strings.Join(v, ", ")))
	}

	for key, value := range mux.Vars(r) {
		key = strings.Replace(key, "-", "_", -1)
		key = strings.ToUpper(key)
		key = fmt.Sprintf("HTTP_PARAM_%s", key)
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	return env
}
