package switchboard

import (
	"fmt"
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
	Type    string
	Routes  map[string]*Route
}

func (route *Route) Execute(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	log.Printf("executing command %s", route.Command.Command)
	env := requestToEnv(r)
	status, tags, stdout, err := route.Command.Execute(env, r.Body)
	if err != nil {
		log.Print("command failed to execute correctly")
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
		log.Print("command completed with a nonzero exit status %s", status)
		http.Error(w, string(body), http.StatusInternalServerError)
		return
	}

	for key, values := range tags {
		last := values[len(values)-1]
		switch key {
		case "HTTP_CONTENT_TYPE":
			log.Print("setting content-type header")
			w.Header().Set("Content-Type", last)
		case "HTTP_STATUS_CODE":
			log.Print("setting status code")
			status, err := strconv.ParseInt(last, 10, 32)
			if err != nil {
				http.Error(w, string(body), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(int(status))
		case "HTTP_REDIRECT":
			log.Print("redirecting to %s", last)
			w.Header().Set("Location", last)
			w.WriteHeader(303)
		}
	}

	fmt.Fprintf(w, string(body))
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
