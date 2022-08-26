package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/containerd/containerd/runtime"
	"github.com/kata-contrib/runs/pkg/shim"
	"github.com/opencontainers/runc/libcontainer/user"
	"github.com/urfave/cli"
)

type containerState struct {
	// // Version is the OCI version for the container
	// Version string `json:"ociVersion"`
	// ID is the container ID
	ID string `json:"id"`
	// InitProcessPid is the init process id in the parent namespace
	InitProcessPid int `json:"pid"`
	// Status is the current status of the container, running, paused, ...
	Status string `json:"status"`
	// Bundle is the path on the filesystem to the bundle
	Bundle string `json:"bundle"`
	// // Rootfs is a path to a directory containing the container's root filesystem.
	// Rootfs string `json:"rootfs"`
	// Created is the unix timestamp for the creation time of the container in UTC
	Created time.Time `json:"created"`
	// // Annotations is the user defined annotations added to the config.
	// Annotations map[string]string `json:"annotations,omitempty"`
	// The owner of the state directory (the owner of the container).
	Owner string `json:"owner"`
}

// type containerState struct {
// 	// // Version is the OCI version for the container
// 	// Version string `json:"ociVersion"`
// 	// ID is the container ID
// 	ID string `json:"id"`

// 	// Runtime
// 	Runtime string `json:"id"`

// 	// // InitProcessPid is the init process id in the parent namespace
// 	// InitProcessPid int `json:"pid"`
// 	// // Status is the current status of the container, running, paused, ...
// 	// Status string `json:"status"`
// 	// Bundle is the path on the filesystem to the bundle
// 	// Bundle string `json:"bundle"`
// 	// // Rootfs is a path to a directory containing the container's root filesystem.
// 	// Rootfs string `json:"rootfs"`
// 	// // Created is the unix timestamp for the creation time of the container in UTC
// 	// Created time.Time `json:"created"`
// 	// // Annotations is the user defined annotations added to the config.
// 	// Annotations map[string]string `json:"annotations,omitempty"`
// 	// // The owner of the state directory (the owner of the container).
// 	// Owner string `json:"owner"`
// }

var listCommand = cli.Command{
	Name:  "list",
	Usage: "list the containers",
	ArgsUsage: `<container-id>

Where "<container-id>" is your name for the instance of the container that you
are starting. The name you provide for the container instance must be unique on
your host.`,
	Description: `The create command creates an instance of a container for a bundle. The bundle
is a directory with a specification file named "` + specConfig + `" and a root
filesystem.

The specification file includes an args parameter. The args parameter is used
to specify command(s) that get run when the container is started. To change the
command(s) that get executed on start, edit the args parameter of the spec. See
"runc spec --help" for more explanation.`,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "bundle, b",
			Value: "",
			Usage: `path to the root of the bundle directory, defaults to the current directory`,
		},
		cli.StringFlag{
			Name:  "console-socket",
			Value: "",
			Usage: "path to an AF_UNIX socket which will receive a file descriptor referencing the master end of the console's pseudoterminal",
		},
		cli.StringFlag{
			Name:  "pid-file",
			Value: "",
			Usage: "specify the file to write the process id to",
		},
		cli.BoolFlag{
			Name:  "no-pivot",
			Usage: "do not use pivot root to jail process inside rootfs.  This should be used whenever the rootfs is on top of a ramdisk",
		},
		cli.BoolFlag{
			Name:  "no-new-keyring",
			Usage: "do not create a new session keyring for the container.  This will cause the container to inherit the calling processes session key",
		},
		cli.IntFlag{
			Name:  "preserve-fds",
			Usage: "Pass N additional file descriptors to the container (stdio + $LISTEN_FDS + N in total)",
		},
	},
	Action: func(context *cli.Context) error {

		s, _ := loadStates(context)

		w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
		fmt.Fprint(w, "ID\tPID\tSTATUS\tBUNDLE\tCREATED\tOWNER\n")
		for _, item := range s {
			fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%s\t%s\n",
				item.ID,
				item.InitProcessPid,
				item.Status,
				item.Bundle,
				item.Created.Format(time.RFC3339Nano),
				item.Owner,
			)

			if err := w.Flush(); err != nil {
				return err
			}
		}
		return nil
	},
}

func loadStates(context *cli.Context) ([]containerState, error) {
	root := context.GlobalString("root")
	list, err := os.ReadDir(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && context.IsSet("root") {
			// Ignore non-existing default root directory
			// (no containers created yet).
			return nil, nil
		}
		// Report other errors, including non-existent custom --root.
		return nil, err
	}

	var s []containerState
	for _, item := range list {
		if !item.IsDir() {
			continue
		}
		st, err := item.Info()
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				// Possible race with runc delete.
				continue
			}
			return nil, err
		}
		// This cast is safe on Linux.
		uid := st.Sys().(*syscall.Stat_t).Uid
		owner, err := user.LookupUid(int(uid))
		if err != nil {
			owner.Name = fmt.Sprintf("#%d", uid)
		}

		id := item.Name()
		parent := filepath.Join(root, id)
		stateFilePath := filepath.Join(parent, "state.json")
		fmt.Println("the path is ", stateFilePath)

		f, err := os.Open(stateFilePath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "the path is wrong~~~~~~~")
			if os.IsNotExist(err) {
				fmt.Fprintln(os.Stderr, "the path is wrong1111111111")
				return nil, os.ErrNotExist
			}
			fmt.Fprintln(os.Stderr, "the path is wrong333333333333")
			return nil, err
		}
		defer f.Close()

		var state *shim.State
		if err := json.NewDecoder(f).Decode(&state); err != nil {
			fmt.Fprintln(os.Stderr, "the path is wrong!!!")
			return nil, err
		}

		status := ""
		switch state.Status {
		case runtime.CreatedStatus:
			status = "CreatedStatus"
		case runtime.RunningStatus:
			status = "RunningStatus"
		case runtime.StoppedStatus:
			status = "StoppedStatus"
		case runtime.DeletedStatus:
			status = "DeletedStatus"
		case runtime.PausedStatus:
			status = "PausedStatus"
		case runtime.PausingStatus:
			status = "PausingStatus"
		default:
			status = "wrong parameter"
		}

		s = append(s, containerState{
			ID:             id,
			InitProcessPid: state.InitProcessPid,
			Status:         status,
			Bundle:         state.Bundle,
			Created:        state.Created,
			Owner:          owner.Name,
		})
	}

	return s, nil
}
