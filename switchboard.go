package switchboard

import (
	"fmt"
	"log"

	"github.com/urfave/cli"
)

func Serve(c *cli.Context) error {
	config := c.GlobalString("config")
	port := c.Int("port")
	reload := c.Bool("reload")

	server, err := NewServer(config, port, reload)
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	log.Printf("starting http server on port %d", port)
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
