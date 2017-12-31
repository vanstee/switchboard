package switchboard

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

const (
	DefaultCommandDriverName = "local"
	DefaultRouteMethod       = "GET"
	DefaultRouteType         = "endpoint"
)

type Config struct {
	Commands map[string]*Command
	Routes   map[string]*Route
}

type ConfigYAML struct {
	Commands map[string]*CommandYAML `yaml:"commands"`
	Routes   map[string]*RouteYAML   `yaml:"routes"`
}

type CommandYAML struct {
	Command string `yaml:"command"`
	Driver  string `yaml:"driver"`
	Image   string `yaml:"image"`
}

type RouteYAML struct {
	Command interface{}           `yaml:"command"`
	Method  string                `yaml:"method"`
	Type    string                `yaml:"type"`
	Routes  map[string]*RouteYAML `yaml:"routes"`
}

func ParseConfig(path string) (*Config, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	configYAML := ConfigYAML{}
	err = yaml.Unmarshal(b, &configYAML)
	if err != nil {
		return nil, err
	}

	config, err := configYAML.ToConfig()
	if err != nil {
		return nil, err
	}

	return config, nil
}

func (configYAML *ConfigYAML) ToConfig() (*Config, error) {
	config := &Config{
		Commands: make(map[string]*Command),
		Routes:   make(map[string]*Route),
	}

	for name, commandYAML := range configYAML.Commands {
		command, err := commandYAML.ToCommand(name)
		if err != nil {
			return nil, err
		}

		config.Commands[name] = command
	}

	for path, routeYAML := range configYAML.Routes {
		route, err := routeYAML.ToRoute(path, config.Commands)
		if err != nil {
			return nil, err
		}

		config.Routes[path] = route
	}

	return config, nil
}

func (commandYAML *CommandYAML) ToCommand(name string) (*Command, error) {
	command := &Command{Name: name}

	driverName := DefaultCommandDriverName
	if commandYAML.Driver != "" {
		driverName = commandYAML.Driver
	}

	driver, err := LookupDriver(driverName)
	if err != nil {
		return nil, err
	}

	command.Driver = driver
	command.Command = commandYAML.Command
	command.Image = commandYAML.Image

	return command, nil
}

func (routeYAML *RouteYAML) ToRoute(path string, commands map[string]*Command) (*Route, error) {
	route := &Route{Path: path}

	malformedErr := fmt.Errorf("command malformed for route \"%s\"", path)

	switch c := routeYAML.Command.(type) {
	case string:
		command, ok := commands[c]
		if !ok {
			return nil, fmt.Errorf("command \"%s\" not found", c)
		}

		route.Command = command
	case map[interface{}]interface{}:
		cs := make(map[string]string)
		for k, v := range c {
			ks, ok := k.(string)
			if !ok {
				return nil, malformedErr
			}

			vs, ok := v.(string)
			if !ok {
				return nil, malformedErr
			}

			cs[ks] = vs
		}

		commandYAML := CommandYAML{
			Command: cs["command"],
			Driver:  cs["driver"],
			Image:   cs["image"],
		}

		command, err := commandYAML.ToCommand(path)
		if err != nil {
			return nil, err
		}

		route.Command = command
	default:
		return nil, malformedErr
	}

	method := DefaultRouteMethod
	if routeYAML.Method != "" {
		method = routeYAML.Method
	}

	route.Method = method

	routeType := DefaultRouteType
	if routeYAML.Type != "" {
		routeType = routeYAML.Type
	}

	route.Type = routeType

	// TODO: Parse routes for middleware

	return route, nil
}
