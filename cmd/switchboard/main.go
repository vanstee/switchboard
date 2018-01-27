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
		cli.BoolFlag{
			Name:   "reload, r",
			EnvVar: "RELOAD",
		},
	}

	app.Commands = []cli.Command{
		cli.Command{
			Name:   "serve",
			Usage:  "",
			Action: switchboard.Serve,
		},
		cli.Command{
			Name:   "routes",
			Usage:  "",
			Action: switchboard.Routes,
		},
	}

	app.Action = switchboard.Serve
	app.Run(os.Args)
}
