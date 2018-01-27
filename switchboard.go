package switchboard

import (
	"fmt"

	"github.com/urfave/cli"
)

func Serve(c *cli.Context) error {
	server, err := NewServer(c.GlobalString("config"), c.Int("port"), c.Bool("reload"))
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
