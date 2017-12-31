package switchboard

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type Driver interface {
	Execute(command *Command, env []string, streams *Streams) error
}

type Streams struct {
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

type LocalDriver struct{}
type BashDriver struct{}
type DockerDriver struct{}

func LookupDriver(name string) (Driver, error) {
	switch name {
	case "local":
		return LocalDriver{}, nil
	case "bash":
		return BashDriver{}, nil
	case "docker":
		return DockerDriver{}, nil
	default:
		return nil, fmt.Errorf("driver \"%s\" not found", name)
	}
}

func (driver LocalDriver) Execute(command *Command, env []string, streams *Streams) error {
	cmd := exec.Command(command.Command)
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdin = streams.stdin
	cmd.Stdout = streams.stdout
	cmd.Stderr = streams.stderr
	return cmd.Run()
}

func (driver BashDriver) Execute(command *Command, env []string, streams *Streams) error {
	cmd := exec.Command("/bin/bash", "-c", command.Command)
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdin = streams.stdin
	cmd.Stdout = streams.stdout
	cmd.Stderr = streams.stderr
	return cmd.Run()
}

func (driver DockerDriver) Execute(command *Command, env []string, streams *Streams) error {
	cli, err := client.NewEnvClient()
	if err != nil {
		return err
	}

	// cli.ImagePull(context.Background(), driver.Image)

	container, err := cli.ContainerCreate(
		context.Background(),
		&container.Config{Image: command.Image},
		nil,
		nil,
		"",
	)
	if err != nil {
		return err
	}

	okc, errc := cli.ContainerWait(context.Background(), container.ID, "next-exit")
	select {
	case err = <-errc:
		return err
	default:
	}

	err = cli.ContainerStart(
		context.Background(),
		container.ID,
		types.ContainerStartOptions{},
	)
	if err != nil {
		return err
	}

	select {
	case err = <-errc:
		return err
	case <-okc:
	}

	stdout, err := cli.ContainerLogs(
		context.Background(),
		container.ID,
		types.ContainerLogsOptions{ShowStdout: true},
	)
	if err != nil {
		return err
	}

	stderr, err := cli.ContainerLogs(
		context.Background(),
		container.ID,
		types.ContainerLogsOptions{ShowStderr: true},
	)
	if err != nil {
		return err
	}

	io.Copy(streams.stdout, stdout)
	io.Copy(streams.stderr, stderr)

	return nil
}
