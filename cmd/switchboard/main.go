package main

import (
	"os"

	"github.com/urfave/cli"
	"github.com/vanstee/switchboard"
)

func main() {
	app := cli.NewApp()
	app.Name = "switchboard"
	app.Usage = "A board for switching things"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "config, c",
			Value:  "config.yaml",
			EnvVar: "CONFIG",
		},
		cli.IntFlag{
			Name:   "port, p",
			Value:  8080,
			EnvVar: "PORT",
		},
	}

	app.Action = switchboard.Run
	app.Run(os.Args)
}
