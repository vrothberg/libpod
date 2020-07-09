package containers

import (
	"bufio"
	"fmt"
	"os"

	"github.com/containers/podman/v2/cmd/podman/registry"
	"github.com/containers/podman/v2/cmd/podman/validate"
	"github.com/containers/podman/v2/libpod/define"
	"github.com/containers/podman/v2/pkg/domain/entities"
	envLib "github.com/containers/podman/v2/pkg/env"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	execDescription = `Execute the specified command inside a running container.
`
	execCommand = &cobra.Command{
		Use:                   "exec [flags] CONTAINER [COMMAND [ARG...]]",
		Short:                 "Run a process in a running container",
		Long:                  execDescription,
		RunE:                  exec,
		DisableFlagsInUseLine: true,
		Example: `podman exec -it ctrID ls
  podman exec -it -w /tmp myCtr pwd
  podman exec --user root ctrID ls`,
	}

	containerExecCommand = &cobra.Command{
		Use:                   execCommand.Use,
		Short:                 execCommand.Short,
		Long:                  execCommand.Long,
		RunE:                  execCommand.RunE,
		DisableFlagsInUseLine: true,
		Example: `podman container exec -it ctrID ls
  podman container exec -it -w /tmp myCtr pwd
  podman container exec --user root ctrID ls`,
	}
)

var (
	envInput, envFile []string
	execOpts          entities.ExecOptions
	execDetach        bool
)

func execFlags(flags *pflag.FlagSet) {
	flags.SetInterspersed(false)
	flags.BoolVarP(&execDetach, "detach", "d", false, "Run the exec session in detached mode (backgrounded)")
	flags.StringVar(&execOpts.DetachKeys, "detach-keys", containerConfig.DetachKeys(), "Select the key sequence for detaching a container. Format is a single character [a-Z] or ctrl-<value> where <value> is one of: a-z, @, ^, [, , or _")
	flags.StringArrayVarP(&envInput, "env", "e", []string{}, "Set environment variables")
	flags.StringSliceVar(&envFile, "env-file", []string{}, "Read in a file of environment variables")
	flags.BoolVarP(&execOpts.Interactive, "interactive", "i", false, "Keep STDIN open even if not attached")
	flags.BoolVar(&execOpts.Privileged, "privileged", false, "Give the process extended Linux capabilities inside the container.  The default is false")
	flags.BoolVarP(&execOpts.Tty, "tty", "t", false, "Allocate a pseudo-TTY. The default is false")
	flags.StringVarP(&execOpts.User, "user", "u", "", "Sets the username or UID used and optionally the groupname or GID for the specified command")
	flags.UintVar(&execOpts.PreserveFDs, "preserve-fds", 0, "Pass N additional file descriptors to the container")
	flags.StringVarP(&execOpts.WorkDir, "workdir", "w", "", "Working directory inside the container")

	if registry.IsRemote() {
		_ = flags.MarkHidden("preserve-fds")
	}
}

func init() {
	registry.Commands = append(registry.Commands, registry.CliCommand{
		Mode:    []entities.EngineMode{entities.ABIMode, entities.TunnelMode},
		Command: execCommand,
	})
	execFlags(execCommand.Flags())
	validate.AddLatestFlag(execCommand, &execOpts.Latest)

	registry.Commands = append(registry.Commands, registry.CliCommand{
		Mode:    []entities.EngineMode{entities.ABIMode, entities.TunnelMode},
		Command: containerExecCommand,
		Parent:  containerCmd,
	})
	execFlags(containerExecCommand.Flags())
	validate.AddLatestFlag(containerExecCommand, &execOpts.Latest)
}

func exec(_ *cobra.Command, args []string) error {
	var nameOrID string

	if len(args) == 0 && !execOpts.Latest {
		return errors.New("exec requires the name or ID of a container or the --latest flag")
	}
	execOpts.Cmd = args
	if !execOpts.Latest {
		execOpts.Cmd = args[1:]
		nameOrID = args[0]
	}
	// Validate given environment variables
	execOpts.Envs = make(map[string]string)
	for _, f := range envFile {
		fileEnv, err := envLib.ParseFile(f)
		if err != nil {
			return err
		}
		execOpts.Envs = envLib.Join(execOpts.Envs, fileEnv)
	}

	cliEnv, err := envLib.ParseSlice(envInput)
	if err != nil {
		return errors.Wrap(err, "error parsing environment variables")
	}

	execOpts.Envs = envLib.Join(execOpts.Envs, cliEnv)

	if !execDetach {
		streams := define.AttachStreams{}
		streams.OutputStream = os.Stdout
		streams.ErrorStream = os.Stderr
		if execOpts.Interactive {
			streams.InputStream = bufio.NewReader(os.Stdin)
			streams.AttachInput = true
		}
		streams.AttachOutput = true
		streams.AttachError = true

		exitCode, err := registry.ContainerEngine().ContainerExec(registry.GetContext(), nameOrID, execOpts, streams)
		registry.SetExitCode(exitCode)
		return err
	}

	id, err := registry.ContainerEngine().ContainerExecDetached(registry.GetContext(), nameOrID, execOpts)
	if err != nil {
		return err
	}
	fmt.Println(id)
	return nil
}
