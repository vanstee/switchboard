package switchboard

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/urfave/cli"
)

func Run(c *cli.Context) error {
	path := c.String("config")
	log.Printf("reading config at path %s", path)
	file, err := os.Open(path)
	if err != nil {
		message := fmt.Sprintf("error opening config file: %s", err)
		return cli.NewExitError(message, 1)
	}

	config, err := ParseConfig(file)
	if err != nil {
		message := fmt.Sprintf("error parsing config file: %s", err)
		return cli.NewExitError(message, 1)
	}

	router, err := BuildRouter(config)
	if err != nil {
		message := fmt.Sprintf("error building routes: %s", err)
		return cli.NewExitError(message, 1)
	}

	port := c.Int("port")
	log.Printf("starting http server on port %d", port)
	return http.ListenAndServe(fmt.Sprintf(":%d", port), router)
}
