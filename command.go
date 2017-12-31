package switchboard

type Command struct {
	Name    string
	Command string
	Driver  Driver
	Image   string
}

func (command *Command) Execute(env []string, streams *Streams) error {
	return command.Driver.Execute(command, env, streams)
}
