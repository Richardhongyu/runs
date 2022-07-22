package main

import (
	"errors"
	"fmt"

	"github.com/opencontainers/runc/libcontainer"

	"github.com/urfave/cli"
)

var startCommand = cli.Command{
	Name:  "start",
	Usage: "executes the user defined process in a created container",
	ArgsUsage: `<container-id>

Where "<container-id>" is your name for the instance of the container that you
are starting. The name you provide for the container instance must be unique on
your host.`,
	Description: `The start command executes the user defined process in a created container.`,
	Action: func(context *cli.Context) error {
		if err := checkArgs(context, 1, exactArgs); err != nil {
			return err
		}
		// container, err := getContainer(context)
		// if err != nil {
		// 	return err
		// }
		// status, err := container.Status()
		// if err != nil {
		// 	return err
		// }
		status := libcontainer.Created
		switch status {
		case libcontainer.Created:
			task, err := findTask(context)
			if err != nil {
				return err
			}
			
			err = task.Start(ctx)
			if err != nil {
				return err
			}
			pid, err := task.PID(ctx)
			if err != nil {
				return err
			}
			fmt.Printf("pid %d\n", pid)

			return nil
		case libcontainer.Stopped:
			return errors.New("cannot start a container that has stopped")
		case libcontainer.Running:
			return errors.New("cannot start an already running container")
		default:
			return fmt.Errorf("cannot start a container in the %s state", status)
		}
	},
}

func findTask(context *cli.Context) (_ *shimTask, error) {
	ctx := namespaces.WithNamespace(sctx.Background(), "default")

	path, err := os.Getwd()
	if err != nil {
		return err
	}
	bundle := &shim.Bundle{
		ID:        "abc",
		Path:      path,
		Namespace: "default",
	}

	s, err := shim.LoadShim(ctx, bundle, func() {})
	if err != nil {
		return err
	}
	state, err := s.State(ctx)
	if err != nil {
		// return err
	}

	// FIXME check state.

	fmt.Printf("state error: %+v\n", err)
	fmt.Printf("state: %+v\n", state)

	return s, err
}
