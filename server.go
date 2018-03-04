package switchboard

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func NewServer(path string, port int, reload bool) (*http.Server, error) {
	log.Printf("reading config at path %s", path)

	config, err := ReadConfig(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config: %s", err)
	}

	var router http.Handler
	if !reload {
		router, err = BuildRouter(config)
		if err != nil {
			return nil, fmt.Errorf("error building routes: %s", err)
		}
	} else {
		router, err = BuildReloadRouter(path)
		if err != nil {
			return nil, fmt.Errorf("error building routes: %s", err)
		}
	}

	return &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}, nil
}

func BuildRouter(config *Config) (http.Handler, error) {
	router := mux.NewRouter()
	route := &RootRoute{Routes: config.Routes}
	err := route.AttachHandlers(router, Pipeline{})
	if err != nil {
		return nil, err
	}
	return router, nil
}

func BuildReloadRouter(path string) (http.Handler, error) {
	log.Printf("watching config at path %s", path)

	_, err := ReadConfig(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config: %s", err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("reloading config at path %s", path)

		config, err := ReadConfig(path)
		if err != nil {
			log.Printf("error rereading config: %s", err)
			return
		}

		router, err := BuildRouter(config)
		if err != nil {
			log.Printf("error rebuilding routes: %s", err)
			return
		}

		router.ServeHTTP(w, r)
	})

	return handler, nil
}
