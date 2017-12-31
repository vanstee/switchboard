package switchboard

import "fmt"

type Command struct {
	Driver  string `yaml:"driver"`
	Command string `yaml:"command"`
	Image   string `yaml: 'image"`
}

func (command *Command) Execute(env []string, streams *Streams) error {
	driver, err := command.LookupDriver()
	if err != nil {
		return err
	}
	return driver.Execute(command, env, streams)
}

func (command *Command) LookupDriver() (Driver, error) {
	switch command.Driver {
	case "local":
		return LocalDriver{}, nil
	case "bash":
		return BashDriver{}, nil
	case "docker":
		return DockerDriver{}, nil
	default:
		return nil, fmt.Errorf("driver \"%s\" not found", command.Driver)
	}
}
