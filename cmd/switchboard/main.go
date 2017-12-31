package main

import (
	"os"

	"github.com/urfave/cli"
	"github.com/vanstee/switchboard"
)

func main() {
	// Read in yaml file
	// Determine routes
	// Validate that route scenarios are supported
	// Start http server
	// Parse requests and turn them into env vars and stdin pipe
	// Execute scenario with env and stdin, reading from stdout and stderr
	// Format output as http response

	app := cli.NewApp()

	app.Name = "switchboard"
	app.Usage = "A board for switching things"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "config, c",
			EnvVar: "SWITCHBOARD_CONFIG",
		},
	}

	app.Action = switchboard.Run

	app.Run(os.Args)
}
