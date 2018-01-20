package switchboard

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/julienschmidt/httprouter"
)

type Route struct {
	Path    string
	Command *Command
	Method  string
	Routes  map[string]*Route
}

func BuildRouter(config *Config) (*httprouter.Router, error) {
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
	if len(route.Routes) == 0 {
		log.Printf("routing to %s %s", route.Method, route.Path)
		pipeline = append(pipeline, route)
		router.Handle(route.Method, route.Path, ExecutePipeline(pipeline))
	} else {
		log.Printf("inserting route in pipeline %s %s", route.Method, route.Path)
		pipeline = append(pipeline, route)
		for _, child := range route.Routes {
			childPipeline := make([]*Route, len(pipeline))
			copy(childPipeline, pipeline)
			err := BuildRoute(router, child, childPipeline)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func ExecutePipeline(pipeline []*Route) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		log.Printf("handling route %s", r.URL.Path)
		env := RequestToEnv(r)
		tags := make(map[string][]string)
		stdin := io.Reader(r.Body)

		for _, route := range pipeline {
			log.Printf("executing command for route %s", route.Path)
			status, routeTags, stdout, err := route.Command.Execute(env, stdin)
			if err != nil {
				log.Print("failed to execute command")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			body, err := ioutil.ReadAll(stdout)
			if err != nil {
				log.Print("command failed to execute correctly")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if status != 0 {
				log.Printf("command completed with a nonzero exit status %s", status)
				http.Error(w, string(body), http.StatusInternalServerError)
				return
			}

			halt, err := ApplyBetweenTags(routeTags, tags, &env)
			if err != nil {
				log.Print("failed to apply tags")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			stdin = bytes.NewReader(body)

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

func ApplyBetweenTags(routeTags map[string][]string, tags map[string][]string, env *[]string) (bool, error) {
	halt := false

	for key, values := range routeTags {
		value := values[len(values)-1]

		switch key {
		case "ENV_SET":
			envPair := strings.Split(value, "=")
			if len(envPair) != 2 {
				return false, errors.New("Unrecognized ENV_SET value")
			}
			*env = append(*env, value)
		case "DEBUG":
			for _, v := range values {
				log.Printf("DEBUG: %s", v)
			}
		case "HALT":
			switch value {
			case "true":
				halt = true
			case "false":
				halt = false
			default:
				return false, errors.New("Unrecognized HALT value")
			}
		default:
			tags[key] = values
		}
	}

	return halt, nil
}

func ApplyEndTags(tags map[string][]string, w http.ResponseWriter) error {
	for key, values := range tags {
		value := values[len(values)-1]
		switch key {
		case "HTTP_CONTENT_TYPE":
			log.Print("setting content-type header")
			w.Header().Set("Content-Type", value)
		case "HTTP_STATUS_CODE":
			log.Print("setting status code")
			status, err := strconv.ParseInt(value, 10, 32)
			if err != nil {
				return err
			}
			w.WriteHeader(int(status))
		case "HTTP_REDIRECT":
			log.Print("redirecting to %s", value)
			w.Header().Set("Location", value)
			w.WriteHeader(303)
		}
	}

	return nil
}

func RequestToEnv(r *http.Request) []string {
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

	return env
}
