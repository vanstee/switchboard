package switchboard

import (
	"bufio"
	"bytes"
	"errors"
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

func (command *Command) Execute(env []string, stdin io.Reader) (int64, map[string][]string, io.Reader, error) {
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

// Tags are header-like key value pairs that are included at the beginning of
// command output are used to control the HTTP response.
//
// Example:
//
//   HTTP_CONTENT_TYPE: application/json
//   HTTP_STATUS_CODE: 201
//
//   { "user": { "name": "Patrick" } }
//
// Supported Tags:
//
//   HTTP_CONTENT_TYPE
//   Sets the Content-Type header
//
//   HTTP_STATUS_CODE
//   Sets the status code
//
//   HTTP_REDIRECT
//   Sets the status code to 303 and the Location header
//
//   DEBUG
//   Logs to STDOUT
//
func ParseTags(stdout io.Reader) (map[string][]string, io.Reader, error) {
	tags := make(map[string][]string)
	scanner := bufio.NewScanner(stdout)
	isHeader := true
	var rest bytes.Buffer

	for scanner.Scan() {
		line := scanner.Text()
		matches := isTag.FindStringSubmatch(line)

		if len(matches) > 0 {
			log.Printf("tag found %s=%s", matches[1], matches[2])
			tags[matches[1]] = append(tags[matches[1]], matches[2])
		} else if line == "" {
			isHeader = false
		} else if isHeader && len(tags) > 0 {
			return nil, nil, errors.New("tags and output must be separated with a blank line")
		} else if !isHeader {
			rest.Write(append(scanner.Bytes(), byte('\n')))
		} else {
			rest.Write(append(scanner.Bytes(), byte('\n')))
			isHeader = false
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}

	return tags, &rest, nil
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
