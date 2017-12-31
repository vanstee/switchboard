package switchboard

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
)

type Route struct {
	Method    string            `yaml:"method"`
	Command   interface{}       `yaml:"command"`
	RouteType string            `yaml:"type"`
	Routes    map[string]*Route `yaml:"routes"`
	Config    *Config
}

func (route *Route) Execute(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	command, err := route.LookupCommand(route.Config)
	if err != nil {
		log.Printf("error looking up command for route: %s\n", err)
		return
	}

	env := requestToEnv(r)
	var stdout, stderr bytes.Buffer
	err = command.Execute(env, &Streams{r.Body, &stdout, &stderr})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, stdout.String())
}

func (route *Route) LookupMethod() string {
	method := "GET"
	if route.Method != "" {
		method = route.Method
	}
	return method
}

func (route *Route) LookupCommand(config *Config) (*Command, error) {
	switch command := route.Command.(type) {
	case string:
		c, ok := config.Commands[command]
		if !ok {
			return nil, fmt.Errorf("command \"%s\" not found", c)
		}
		return c, nil
	case *Command:
		return command, nil
	default:
		return nil, errors.New("command for route malformed")
	}
}

func requestToEnv(r *http.Request) []string {
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
