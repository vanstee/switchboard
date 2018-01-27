package switchboard

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/urfave/cli"
)

func Serve(c *cli.Context) error {
	path := c.GlobalString("config")
	config, err := ReadConfig(path)
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	server, err := NewServer(config, c.Int("port"))
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	err = server.ListenAndServe()
	return cli.NewExitError(err.Error(), 1)
}

func Routes(c *cli.Context) error {
	path := c.GlobalString("config")
	config, err := ReadConfig(path)
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}
	fmt.Printf("%#v\n", config)
	return nil
}

func ReadConfig(path string) (*Config, error) {
	log.Printf("reading config at path %s", path)

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error opening config file: %s", err)
	}

	config, err := ParseConfig(file)
	if err != nil {
		return nil, fmt.Errorf("error parsing config file: %s", err)
	}

	return config, nil
}

func NewServer(config *Config, port int) (*http.Server, error) {
	router, err := BuildRouter(config)
	if err != nil {
		return nil, fmt.Errorf("error building routes: %s", err)
	}

	return &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}, nil
}
