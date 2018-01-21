package switchboard

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"regexp"
)

var (
	isTag = regexp.MustCompile(`^([A-Z_]+)\:\ (.*)$`)
)

type Command struct {
	Name    string
	Command string
	Driver  Driver
	Image   string
	Inline  string
}

func (command *Command) Execute(env []string, stdin io.Reader) (int64, Tags, io.Reader, error) {
	var stdout, stderr bytes.Buffer

	status, err := command.Driver.Execute(command, env, &Streams{stdin, &stdout, &stderr})
	if err != nil {
		return -1, nil, nil, err
	}

	err = LogStderr(&stderr)
	if err != nil {
		return -1, nil, nil, err
	}

	tags, rest, err := ParseTags(&stdout)
	if err != nil {
		return -1, nil, nil, err
	}

	return status, tags, rest, nil
}

func LogStderr(stderr io.Reader) error {
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		log.Print(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}
