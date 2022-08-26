package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/containerd/containerd/mount"
	"github.com/urfave/cli"
)

// IO holds process IO information
type IO struct {
	Stdin    string
	Stdout   string
	Stderr   string
	Terminal bool
}

type Any struct {
	TypeURL string `json:"type_url"`
	Value   []byte `json:"value"`
}

type CreateOpts struct {
	// Spec is the OCI runtime spec
	Spec Any `json:"Spec"`
	// Rootfs mounts to perform to gain access to the container's filesystem
	Rootfs []mount.Mount `json:"Rootfs"`
	// IO for the container's main process
	IO IO `json:"IO"`
	// Checkpoint digest to restore container state
	Checkpoint string `json:"Checkpoint"`
	// RuntimeOptions for the runtime
	RuntimeOptions Any `json:"RuntimeOptions"`
	// TaskOptions received for the task
	TaskOptions Any `json:"TaskOptions"`
	// Runtime name to use (e.g. `io.containerd.NAME.VERSION`).
	// As an alternative full abs path to binary may be specified instead.
	Runtime string `json:"Runtime"`
	// SandboxID is an optional ID of sandbox this container belongs to
	SandboxID string `json:"SandboxID"`
}

type containerState struct {
	// // Version is the OCI version for the container
	// Version string `json:"ociVersion"`
	// ID is the container ID
	ID string `json:"id"`

	// Runtime
	Runtime string `json:"id"`

	// // InitProcessPid is the init process id in the parent namespace
	// InitProcessPid int `json:"pid"`
	// // Status is the current status of the container, running, paused, ...
	// Status string `json:"status"`
	// Bundle is the path on the filesystem to the bundle
	// Bundle string `json:"bundle"`
	// // Rootfs is a path to a directory containing the container's root filesystem.
	// Rootfs string `json:"rootfs"`
	// // Created is the unix timestamp for the creation time of the container in UTC
	// Created time.Time `json:"created"`
	// // Annotations is the user defined annotations added to the config.
	// Annotations map[string]string `json:"annotations,omitempty"`
	// // The owner of the state directory (the owner of the container).
	// Owner string `json:"owner"`
}

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
		root := "/run/runs"
		list, err := os.ReadDir(root)
		if err != nil {
			fmt.Fprintln(os.Stderr, "the path is wrong")
			return err
		}

		var s []containerState
		for _, item := range list {
			if !item.IsDir() {
				continue
			}
			state, err := loadState(root, item)
			if err != nil {
				fmt.Fprintln(os.Stderr, "the container has no info")
				return err
			}
			s = append(s, *state)
			// id, state := loadState(item)
			// fmt.Printf("state is %s\n", item.Name())
		}

		w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
		fmt.Fprint(w, "ID\tSTATUS\n")
		for _, item := range s {
			// fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%s\t%s\n",
			fmt.Fprintf(w, "%s\t%s\n",
				item.ID,
				// 		item.InitProcessPid,
				item.Runtime,
				// 		item.Bundle,
				// 		item.Created.Format(time.RFC3339Nano),
				// 		item.Owner)
			)
			// }
			if err := w.Flush(); err != nil {
				return err
			}
		}

		return nil
	},
}

func loadState(root string, path fs.DirEntry) (*containerState, error) {
	id := path.Name()
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

	var state *CreateOpts
	if err := json.NewDecoder(f).Decode(&state); err != nil {
		fmt.Fprintln(os.Stderr, "the path is wrong!!!")
		return nil, err
	}

	res := containerState{
		ID:      id,
		Runtime: state.Runtime,
	}

	return &res, nil
}
