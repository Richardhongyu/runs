package main

import (
	// "fmt"
	sctx "context"

	"github.com/urfave/cli"

	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/protobuf"
	"github.com/containerd/containerd/runtime"
	"github.com/containerd/containerd/log"
	"github.com/kata-contrib/runs/pkg/shim"

)

const (
	stateFilename    = "state.json"
)

var createCommand = cli.Command{
	Name:  "create",
	Usage: "create a container",
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
		if err := checkArgs(context, 1, exactArgs); err != nil {
			return err
		}

		ctx := namespaces.WithNamespace(sctx.Background(), "default")

		shimManager, err := shim.NewShimManager(ctx, &shim.ManagerConfig{
			State:        "/var/run/runs",
			Address:      "/run/containerd/containerd.sock",
			TTRPCAddress: "/run/containerd/containerd.sock.ttrpc",
		})
		if err != nil {
			return err
		}
		spec, err := loadSpec(specConfig)
		if err != nil {
			return err
		}

		specAny, err := protobuf.MarshalAnyToProto(spec)
		if err != nil {
			return err
		}

		opts := runtime.CreateOpts{
			Spec: specAny,
			// IO: runtime.IO{
			// 	Stdin:    r.Stdin,
			// 	Stdout:   r.Stdout,
			// 	Stderr:   r.Stderr,
			// 	Terminal: r.Terminal,
			// },
			// TaskOptions:    r.Options,
			// SandboxID:      container.SandboxID,
		}

		opts.Runtime = "io.containerd.runc.v2"

		// for _, m := range r.Rootfs {
		// 	opts.Rootfs = append(opts.Rootfs, mount.Mount{
		// 		Type:    m.Type,
		// 		Source:  m.Source,
		// 		Options: m.Options,
		// 	})
		// }

		taskManager := shim.NewTaskManager(shimManager)
		taskManager.Create(ctx, "abc", opts)
		if err != nil {
			return err
		}

		if err := saveContainerState(ctx, opts); err != nil {
			return err
		}

		// status, err := startContainer(context, CT_ACT_CREATE, nil)
		// if err == nil {
		// 	// exit with the container's exit status so any external supervisor
		// 	// is notified of the exit with the correct exit status.
		// 	os.Exit(status)
		// }
		// return fmt.Errorf("runc create failed: %w", err)
		return nil
	},
}

func saveContainerState(ctx sctx.Context, opts runtime.CreateOpts) error {
	log.G(ctx).Errorf("AAAAA TaskManager Create %+v", opts)
	return nil
}
