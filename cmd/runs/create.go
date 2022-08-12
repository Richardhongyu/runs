package main

import (
	sctx "context"
	"fmt"
	"io"
	"os"

	"github.com/urfave/cli"

	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/log"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/protobuf"
	"github.com/containerd/containerd/runtime"
	"github.com/kata-contrib/runs/pkg/shim"
	"golang.org/x/sys/unix"

	//	"cio"
	"github.com/kata-contrib/runs/pkg/cio"

	securejoin "github.com/cyphar/filepath-securejoin"
)

const (
	stateFilename = "state.json"
)

type stdinCloser struct {
	stdin  *os.File
	closer func()
}

func (s *stdinCloser) Read(p []byte) (int, error) {
	n, err := s.stdin.Read(p)
	if err == io.EOF {
		if s.closer != nil {
			s.closer()
		}
	}
	return n, err
}

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
		var (
			id  string
			ref string
		//	config = context.IsSet("config")
		)

		if 1 == 1 {
			id = context.Args().First()
			if context.NArg() > 1 {
				return fmt.Errorf("with spec config file, only container id should be provided: %w", errdefs.ErrInvalidArgument)
			}
		} else {
			id = context.Args().Get(1)
			ref = context.Args().First()
			if ref == "" {
				return fmt.Errorf("image ref must be provided: %w", errdefs.ErrInvalidArgument)
			}
		}
		if id == "" {
			return fmt.Errorf("container id must be provided: %w", errdefs.ErrInvalidArgument)
		}

		containerRoot, err := securejoin.SecureJoin("/run/runs", id)
		if err != nil {
			return err
		}
		os.Stat(containerRoot)
		os.MkdirAll(containerRoot, 0711)
		os.Chown(containerRoot, unix.Geteuid(), unix.Getegid())

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

		// con = false
		// nullIO = context.Bool("null-io")
		// context.String("log-uri")
		stdinC := &stdinCloser{
			stdin: os.Stdin,
		}

		ioOpts := []cio.Opt{cio.WithFIFODir(context.String("fifo-dir"))}
		ioCreator := cio.NewCreator(append([]cio.Opt{cio.WithStreams(stdinC, os.Stdout, os.Stderr)}, ioOpts...)...)

		i, err := ioCreator(id)
		cfg := i.Config()

		// container, err := client.LoadContainer(ctx, id)

		opts := runtime.CreateOpts{
			Spec: specAny,
			IO: runtime.IO{
				Stdin:    cfg.Stdin,
				Stdout:   cfg.Stdout,
				Stderr:   cfg.Stderr,
				Terminal: cfg.Terminal,
			},
			// TaskOptions:    r.Options,
			// SandboxID:      container.SandboxID,
		}

		opts.Runtime = "io.containerd.runc.v2"

		// for _, m := range spec.Mounts {
		// 	//cm, err := createLibcontainerMount(cwd, m)
		// 	//if err != nil {
		// 	//	return nil, fmt.Errorf("invalid mount %+v: %w", m, err)
		// 	//}
		// 	opts.Rootfs = append(opts.Rootfs, mount.Mount{
		// 		Type:    m.Type,
		// 		// Destination:  m.Destination,
		// 		Source:  m.Source,
		// 		Options: m.Options,
		// 	})
		// }

		// opts.Rootfs = append(opts.Rootfs, mount.Mount{
		// 	Type:    m.Type,
		// 	Source:  "./rootfs",
		// 	Options: [],
		// })

		taskManager := shim.NewTaskManager(shimManager)
		taskManager.Create(ctx, id, opts)
		if err != nil {
			return err
		}

		if err := saveContainerState(ctx, id, opts); err != nil {
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

func saveContainerState(ctx sctx.Context, taskID string, opts runtime.CreateOpts) error {
	log.G(ctx).Errorf("AAAAA TaskManager save %+v", opts)
	containerRoot, err := securejoin.SecureJoin("/run/runs", taskID)
	// if err != nil {
	// 	return err
	// }
	// os.Stat(containerRoot)
	// os.MkdirAll(containerRoot, 0711)
	// os.Chown(containerRoot, unix.Geteuid(), unix.Getegid())
	tmpFile, err := os.CreateTemp(containerRoot, "state.json")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
		}
	}()

	WriteJSON(tmpFile, opts)
	// stateFilePath := filepath.Join("/run/runs/lhy/", stateFilename)
	// os.Rename(tmpFile.Name(), stateFilePath)
	return nil
}
