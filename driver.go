package switchboard

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"syscall"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/builder/dockerfile/shell"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

type Driver interface {
	Execute(command *Command, env []string, streams *Streams) (int64, error)
}

type Streams struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

type LocalDriver struct{}
type DockerDriver struct{}

func LookupDriver(name string) (Driver, error) {
	switch name {
	case "local":
		return LocalDriver{}, nil
	case "docker":
		return DockerDriver{}, nil
	default:
		return nil, fmt.Errorf("driver \"%s\" not found", name)
	}
}

func (driver LocalDriver) Execute(command *Command, env []string, streams *Streams) (int64, error) {
	path := command.Command
	if command.Inline != "" {
		tmpfile, err := ioutil.TempFile("", fmt.Sprintf("switchboard-inline-command-%s-", command.Name))
		if err != nil {
			return -1, err
		}
		defer os.Remove(tmpfile.Name())

		if _, err := tmpfile.Write([]byte(command.Inline)); err != nil {
			return -1, err
		}

		if err := tmpfile.Close(); err != nil {
			return -1, err
		}

		if err := os.Chmod(tmpfile.Name(), 0777); err != nil {
			return -1, err
		}

		path = tmpfile.Name()
	}

	cmd := exec.Command("/bin/bash", "-c", path)
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdin = streams.Stdin
	cmd.Stdout = streams.Stdout
	cmd.Stderr = streams.Stderr

	err := cmd.Run()
	if err != nil {
		exiterr, ok := err.(*exec.ExitError)
		if !ok {
			return -1, err
		}

		status, ok := exiterr.Sys().(syscall.WaitStatus)
		if !ok {
			return -1, err
		}

		return int64(status.ExitStatus()), nil
	}

	return 0, nil
}

func (driver DockerDriver) Execute(command *Command, env []string, streams *Streams) (int64, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return -1, err
	}

	cli.NegotiateAPIVersion(context.Background())

	config := &container.Config{Image: command.Image}
	if command.Command != "" {
		s := shell.NewLex('\\')
		words, err := s.ProcessWords(command.Command, []string{})
		if err != nil {
			return -1, err
		}
		config.Cmd = words
	}

	log.Printf("creating container from image %s", command.Image)
	container, err := cli.ContainerCreate(
		context.Background(),
		config,
		nil,
		nil,
		"",
	)
	if err != nil {
		return -1, err
	}

	okc, errc := cli.ContainerWait(context.Background(), container.ID, "next-exit")
	select {
	case err = <-errc:
		return -1, err
	default:
	}

	log.Printf("starting container %s", container.ID)
	err = cli.ContainerStart(
		context.Background(),
		container.ID,
		types.ContainerStartOptions{},
	)
	if err != nil {
		return -1, err
	}

	log.Printf("waiting for container to exit %s", container.ID)
	var status int64
	select {
	case err = <-errc:
		return -1, err
	case ok := <-okc:
		status = ok.StatusCode
	}

	logs, err := cli.ContainerLogs(
		context.Background(),
		container.ID,
		types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true},
	)
	if err != nil {
		return -1, err
	}

	stdcopy.StdCopy(streams.Stdout, streams.Stderr, logs)

	return status, nil
}
