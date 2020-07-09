package containers

import (
	"context"
	"fmt"

	"github.com/containers/podman/v2/cmd/podman/registry"
	"github.com/containers/podman/v2/cmd/podman/utils"
	"github.com/containers/podman/v2/pkg/domain/entities"
	"github.com/containers/podman/v2/pkg/rootless"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	pauseDescription = `Pauses one or more running containers.  The container name or ID can be used.`
	pauseCommand     = &cobra.Command{
		Use:   "pause [flags] CONTAINER [CONTAINER...]",
		Short: "Pause all the processes in one or more containers",
		Long:  pauseDescription,
		RunE:  pause,
		Example: `podman pause mywebserver
  podman pause 860a4b23
  podman pause -a`,
	}

	containerPauseCommand = &cobra.Command{
		Use:   pauseCommand.Use,
		Short: pauseCommand.Short,
		Long:  pauseCommand.Long,
		RunE:  pauseCommand.RunE,
		Example: `podman container pause mywebserver
  podman container pause 860a4b23
  podman container pause -a`,
	}

	pauseOpts = entities.PauseUnPauseOptions{}
)

func pauseFlags(flags *pflag.FlagSet) {
	flags.BoolVarP(&pauseOpts.All, "all", "a", false, "Pause all running containers")
}

func init() {
	registry.Commands = append(registry.Commands, registry.CliCommand{
		Mode:    []entities.EngineMode{entities.ABIMode, entities.TunnelMode},
		Command: pauseCommand,
	})
	flags := pauseCommand.Flags()
	pauseFlags(flags)

	registry.Commands = append(registry.Commands, registry.CliCommand{
		Mode:    []entities.EngineMode{entities.ABIMode, entities.TunnelMode},
		Command: containerPauseCommand,
		Parent:  containerCmd,
	})
	containerPauseFlags := containerPauseCommand.Flags()
	pauseFlags(containerPauseFlags)
}

func pause(cmd *cobra.Command, args []string) error {
	var (
		errs utils.OutputErrors
	)
	if rootless.IsRootless() && !registry.IsRemote() {
		return errors.New("pause is not supported for rootless containers")
	}

	if len(args) < 1 && !pauseOpts.All {
		return errors.Errorf("you must provide at least one container name or id")
	}
	responses, err := registry.ContainerEngine().ContainerPause(context.Background(), args, pauseOpts)
	if err != nil {
		return err
	}
	for _, r := range responses {
		if r.Err == nil {
			fmt.Println(r.Id)
		} else {
			errs = append(errs, r.Err)
		}
	}
	return errs.PrintErrors()
}
