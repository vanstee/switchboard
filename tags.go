package switchboard

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type Tags map[string][]string

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
func ParseTags(stdout io.Reader) (Tags, io.Reader, error) {
	tags := make(Tags)
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

func ApplyBetweenTags(routeTags Tags, tags Tags, env *[]string) (bool, error) {
	halt := false

	for key, values := range routeTags {
		value := values[len(values)-1]

		switch key {
		case "ENV_SET":
			envPair := strings.Split(value, "=")
			if len(envPair) != 2 {
				return false, errors.New("Unrecognized ENV_SET value")
			}
			*env = append(*env, value)
		case "DEBUG":
			for _, v := range values {
				log.Printf("DEBUG: %s", v)
			}
		case "HALT":
			switch value {
			case "true":
				halt = true
			case "false":
				halt = false
			default:
				return false, errors.New("Unrecognized HALT value")
			}
		default:
			tags[key] = values
		}
	}

	return halt, nil
}

func ApplyEndTags(tags Tags, w http.ResponseWriter) error {
	for key, values := range tags {
		value := values[len(values)-1]
		switch key {
		case "HTTP_CONTENT_TYPE":
			log.Print("setting content-type header")
			w.Header().Set("Content-Type", value)
		case "HTTP_STATUS_CODE":
			log.Print("setting status code")
			status, err := strconv.ParseInt(value, 10, 32)
			if err != nil {
				return err
			}
			w.WriteHeader(int(status))
		case "HTTP_REDIRECT":
			log.Print("redirecting to %s", value)
			w.Header().Set("Location", value)
			w.WriteHeader(303)
		}
	}

	return nil
}
