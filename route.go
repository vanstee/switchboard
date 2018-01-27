package switchboard

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
)

type Route struct {
	Path    string
	Command *Command
	Methods []string
	Type    string
	Routes  map[string]*Route
}

func BuildRouter(config *Config) (http.Handler, error) {
	router := httprouter.New()
	for _, route := range config.Routes {
		err := BuildRoute(router, route, []*Route{})
		if err != nil {
			return nil, err
		}
	}
	return router, nil
}

func BuildRoute(router *httprouter.Router, route *Route, pipeline []*Route) error {
	for _, method := range route.Methods {
		pl := make([]*Route, len(pipeline))
		copy(pl, pipeline)

		if route.Path != "*" {
			log.Printf("routing to %s %s", method, route.Path)
			pl = append(pl, route)
			router.Handle(method, route.Path, ExecutePipeline(pl))
		} else {
			log.Printf("inserting route in pipeline %s %s", method, route.Path)
			pl = append(pl, route)
			for _, child := range route.Routes {
				err := BuildRoute(router, child, pl)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func ExecutePipeline(pipeline []*Route) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		log.Printf("handling route %s", r.URL.Path)
		env := RequestToEnv(r, params)
		tags := make(Tags)
		stdin := io.Reader(r.Body)

		for _, route := range pipeline {
			routeTags, body, err := ExecuteRoute(route, env, stdin)
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
}

func ExecuteRoute(route *Route, env []string, stdin io.Reader) (Tags, string, error) {
	log.Printf("executing command %s for route %s", route.Command.Name, route.Path)
	status, routeTags, stdout, err := route.Command.Execute(env, stdin)
	if err != nil {
		log.Print("failed to execute command")
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

func RequestToEnv(r *http.Request, params httprouter.Params) []string {
	env := []string{
		fmt.Sprintf("HTTP_METHOD=%s", r.Method),
		fmt.Sprintf("HTTP_URL=%s", r.URL.String()),
	}

	for k, v := range r.Header {
		k = strings.Replace(k, "-", "_", -1)
		k = strings.ToUpper(k)
		k = fmt.Sprintf("HTTP_HEADER_%s", k)
		env = append(env, fmt.Sprintf("%s=%s", k, strings.Join(v, ", ")))
	}

	for _, param := range params {
		k := param.Key
		k = strings.Replace(k, "-", "_", -1)
		k = strings.ToUpper(k)
		k = fmt.Sprintf("HTTP_PARAM_%s", k)
		env = append(env, fmt.Sprintf("%s=%s", k, param.Value))
	}

	return env
}
