package switchboard

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
)

type Route struct {
	Path    string
	Command *Command
	Method  string
	Type    string
	Routes  map[string]*Route
}

func (route *Route) Execute(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	env := requestToEnv(r)
	var stdout, stderr bytes.Buffer
	err := route.Command.Execute(env, &Streams{r.Body, &stdout, &stderr})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, stdout.String())
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
