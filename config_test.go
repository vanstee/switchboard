package switchboard_test

import (
	"strings"
	"testing"

	"github.com/vanstee/switchboard"
)

func TestParseConfig(t *testing.T) {
	body := `
commands:
 	hello:
    command: "echo hello"
routes:
  "/hello":
    command: hello`

	body = strings.Replace(body, "\t", "  ", -1)

	config, err := switchboard.ParseConfig(strings.NewReader(body))
	if err != nil {
		t.Fatalf("ParseConfig returned an error: %s", err)
	}

	commands := config.Commands
	if len(commands) != 1 {
		t.Fatalf("expected %d commands, got %d commands", 1, len(commands))
	}

	command, ok := commands["hello"]
	if !ok {
		t.Fatalf("command not found")
	}
	if command.Name != "hello" {
		t.Errorf("expected command named %s, got command named %s", "hello", command.Name)
	}
	if command.Command != "echo hello" {
		t.Errorf("expected command %s, got command %s", "echo hello", command.Command)
	}

	routes := config.Routes
	if len(routes) != 1 {
		t.Fatalf("expected %d routes, got %d routes", 1, len(routes))
	}

	route := routes["/hello"]
	if !ok {
		t.Fatal("route not found")
	}
	if route.Path != "/hello" {
		t.Errorf("expected route %s, got route %s", "/hello", route.Path)
	}
	if route.Command.Name != "hello" {
		t.Errorf("expected route to have command %s, got command %s", "hello", route.Command.Name)
	}
}
