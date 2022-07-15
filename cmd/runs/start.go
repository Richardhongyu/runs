package main

import (
	sctx "context"
	"errors"
	"fmt"

	"github.com/opencontainers/runc/libcontainer"

	"github.com/urfave/cli"

	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/protobuf"
	"github.com/containerd/containerd/runtime"

	"github.com/kata-contrib/runs/pkg/shim"
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
			// task, err := findTask()
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

// func findTask() (runtime.Task, error) {

// }