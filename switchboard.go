package switchboard

import (
	"fmt"
	"net/http"

	"github.com/urfave/cli"

	"github.com/julienschmidt/httprouter"
)

func Run(c *cli.Context) error {
	path := c.String("config")
	if path == "" {
		return cli.NewExitError("config is required", 1)
	}

	config, err := ParseConfig(path)
	if err != nil {
		message := fmt.Sprintf("error parsing config file: %s", err)
		return cli.NewExitError(message, 1)
	}

	router := httprouter.New()
	for _, route := range config.Routes {
		router.Handle(route.Method, route.Path, route.Execute)
	}

	return http.ListenAndServe(":8080", router)
}
