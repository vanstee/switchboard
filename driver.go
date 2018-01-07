package switchboard

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/moby/moby/pkg/stdcopy"
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
	cmd := exec.Command("/bin/bash", "-c", command.Command)
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

	// cli.ImagePull(context.Background(), driver.Image)

	config := &container.Config{Image: command.Image}
	if command.Command != "" {
		config.Cmd = strings.Split(command.Command, " ")
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
