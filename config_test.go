package switchboard_test

import (
	"strings"
	"testing"

	"github.com/vanstee/switchboard"
)

var (
	helloCommand = &switchboard.Command{
		Name:    "hello",
		Command: "echo hello",
		Driver:  switchboard.LocalDriver{},
		Image:   "",
		Inline:  "",
	}

	configTests = []struct {
		body     string
		commands map[string]*switchboard.Command
		routes   map[string]*switchboard.Route
	}{
		{
			body: `
commands:
	hello:
		command: "echo hello"
routes:
	"/hello":
		command: hello`,
			commands: map[string]*switchboard.Command{
				"hello": helloCommand,
			},
			routes: map[string]*switchboard.Route{
				"/hello": &switchboard.Route{
					Path:    "/hello",
					Command: helloCommand,
					Method:  "GET",
					Routes:  nil,
				},
			},
		},
	}
)

func TestParseConfig(t *testing.T) {
	for _, tt := range configTests {
		body := strings.Replace(tt.body, "\t", "  ", -1)

		config, err := switchboard.ParseConfig(strings.NewReader(body))
		if err != nil {
			t.Fatalf("ParseConfig returned an error: %s", err)
		}

		commands := config.Commands
		if len(commands) != len(tt.commands) {
			t.Fatalf("expected %d commands, got %d commands", len(tt.commands), len(commands))
		}

		for name, ttcommand := range tt.commands {
			command, ok := commands[name]
			if !ok {
				t.Fatalf("command %s not found", name)
			}
			if command.Name != ttcommand.Name {
				t.Errorf("expected command named %s, got command named %s", ttcommand.Name, command.Name)
			}
			if command.Command != ttcommand.Command {
				t.Errorf("expected command %s, got command %s", ttcommand.Command, command.Command)
			}
		}

		routes := config.Routes
		if len(routes) != 1 {
			t.Fatalf("expected %d routes, got %d routes", 1, len(routes))
		}

		for path, ttroute := range tt.routes {
			route, ok := routes[path]
			if !ok {
				t.Fatal("route %s not found", path)
			}
			if route.Path != ttroute.Path {
				t.Errorf("expected route %s, got route %s", ttroute.Path, route.Path)
			}
			if route.Command.Name != ttroute.Command.Name {
				t.Errorf("expected route to have command %s, got command %s", ttroute.Command.Name, route.Command.Name)
			}
		}
	}
}
