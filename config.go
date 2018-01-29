package switchboard

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"regexp"

	"gopkg.in/yaml.v2"
)

const (
	DefaultCommandDriverName = "local"

	DefaultRouteMethod = "GET"

	BasicRouteType    = "basic"
	ResourceRouteType = "resource"
	DefaultRouteType  = BasicRouteType
)

var (
	onlyAlphanumericRegexp        = regexp.MustCompile("[^a-zA-Z0-9-]")
	removeSurroundingDashesRegexp = regexp.MustCompile("(^-*)|(-*$)")
	consolidateDashesRegexp       = regexp.MustCompile("-+")
)

type Config struct {
	Commands map[string]*Command
	Routes   map[string]Route
}

type ConfigYAML struct {
	Commands map[string]*CommandYAML `yaml:"commands"`
	Routes   map[string]*RouteYAML   `yaml:"routes"`
}

type CommandYAML struct {
	Command string `yaml:"command"`
	Driver  string `yaml:"driver"`
	Image   string `yaml:"image"`
	Inline  string `yaml:"inline"`
}

type RouteYAML struct {
	Command interface{}           `yaml:"command"`
	Method  interface{}           `yaml:"method"`
	Type    string                `yaml:"type"`
	Routes  map[string]*RouteYAML `yaml:"routes"`
}

func ParseConfig(r io.Reader) (*Config, error) {
	b, err := ioutil.ReadAll(r)
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
		Routes:   make(map[string]Route),
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
	command.Inline = commandYAML.Inline

	return command, nil
}

func (routeYAML *RouteYAML) ToRoute(path string, commands map[string]*Command) (Route, error) {
	var command *Command
	malformedErr := fmt.Errorf("command malformed for route \"%s\"", path)

	switch c := routeYAML.Command.(type) {
	case string:
		var ok bool
		command, ok = commands[c]
		if !ok {
			return nil, fmt.Errorf("command \"%s\" not found", c)
		}
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
			Inline:  cs["inline"],
		}

		name := PathToName(path)
		var err error
		command, err = commandYAML.ToCommand(name)
		if err != nil {
			return nil, err
		}
	default:
		return nil, malformedErr
	}

	routeType := routeYAML.Type
	if routeType == "" {
		routeType = DefaultRouteType
	}

	switch routeType {
	case BasicRouteType:
		route := &BasicRoute{
			Path:    path,
			Command: command,
		}

		switch method := routeYAML.Method.(type) {
		case string:
			route.Methods = []string{method}
		case []interface{}:
			methods := make([]string, len(method))
			for i, m := range method {
				methods[i] = m.(string)
			}
			route.Methods = methods
		default:
			route.Methods = []string{DefaultRouteMethod}
		}

		route.Routes = make(map[string]Route)
		for childPath, childRouteYAML := range routeYAML.Routes {
			path := JoinPaths("/", path, childPath)
			fmt.Printf("%s\n", path)
			r, err := childRouteYAML.ToRoute(path, commands)
			if err != nil {
				return nil, err
			}
			route.Routes[childPath] = r
		}

		return route, nil
	case ResourceRouteType:
		route := &ResourceRoute{
			Path:    path,
			Command: command,
		}

		route.Routes = make(map[string]Route)
		for childPath, childRouteYAML := range routeYAML.Routes {
			path := JoinPaths("/", path, ":id", childPath)
			fmt.Printf("%s\n", path)
			r, err := childRouteYAML.ToRoute(path, commands)
			if err != nil {
				return nil, err
			}
			route.Routes[childPath] = r
		}

		return route, nil
	default:
		unsupportedRouteType := fmt.Errorf("unsupported route type \"%s\"", routeType)
		return nil, unsupportedRouteType
	}
}

func ReadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	config, err := ParseConfig(file)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func JoinPaths(paths ...string) string {
	segments := make([]string, 0, len(paths))
	for _, path := range paths {
		switch path {
		case "*":
			continue
		default:
			segments = append(segments, path)
		}
	}
	return path.Join(segments...)
}

func PathToName(path string) string {
	name := path
	name = onlyAlphanumericRegexp.ReplaceAllString(name, "-")
	name = removeSurroundingDashesRegexp.ReplaceAllString(name, "")
	name = consolidateDashesRegexp.ReplaceAllString(name, "-")
	return name
}
