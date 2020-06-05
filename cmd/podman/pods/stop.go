package pods

import (
	"context"
	"fmt"

	"github.com/containers/libpod/cmd/podman/common"
	"github.com/containers/libpod/cmd/podman/parse"
	"github.com/containers/libpod/cmd/podman/registry"
	"github.com/containers/libpod/cmd/podman/utils"
	"github.com/containers/libpod/pkg/domain/entities"
	"github.com/spf13/cobra"
)

// allows for splitting API and CLI-only options
type podStopOptionsWrapper struct {
	entities.PodStopOptions

	PodIDFiles []string
	TimeoutCLI uint
}

var (
	stopOptions = podStopOptionsWrapper{
		PodStopOptions: entities.PodStopOptions{Timeout: -1},
	}
	podStopDescription = `The pod name or ID can be used.

  This command will stop all running containers in each of the specified pods.`

	stopCommand = &cobra.Command{
		Use:   "stop [flags] POD [POD...]",
		Short: "Stop one or more pods",
		Long:  podStopDescription,
		RunE:  stop,
		Args: func(cmd *cobra.Command, args []string) error {
			return parse.CheckAllLatestAndPodIDFile(cmd, args, false, true)
		},
		Example: `podman pod stop mywebserverpod
  podman pod stop --latest
  podman pod stop --time 0 490eb 3557fb`,
	}
)

func init() {
	registry.Commands = append(registry.Commands, registry.CliCommand{
		Mode:    []entities.EngineMode{entities.ABIMode, entities.TunnelMode},
		Command: stopCommand,
		Parent:  podCmd,
	})
	flags := stopCommand.Flags()
	flags.BoolVarP(&stopOptions.All, "all", "a", false, "Stop all running pods")
	flags.BoolVarP(&stopOptions.Ignore, "ignore", "i", false, "Ignore errors when a specified pod is missing")
	flags.BoolVarP(&stopOptions.Latest, "latest", "l", false, "Stop the latest pod podman is aware of")
	flags.UintVarP(&stopOptions.TimeoutCLI, "time", "t", containerConfig.Engine.StopTimeout, "Seconds to wait for pod stop before killing the container")
	flags.StringArrayVarP(&stopOptions.PodIDFiles, "pod-id-file", "", nil, "Read the pod ID from the file")
	if registry.IsRemote() {
		_ = flags.MarkHidden("latest")
		_ = flags.MarkHidden("ignore")
	}
	flags.SetNormalizeFunc(utils.AliasFlags)
}

func stop(cmd *cobra.Command, args []string) error {
	var (
		errs utils.OutputErrors
	)
	if cmd.Flag("time").Changed {
		stopOptions.Timeout = int(stopOptions.TimeoutCLI)
	}

	ids, err := common.ReadPodIDFiles(stopOptions.PodIDFiles)
	if err != nil {
		return err
	}
	args = append(args, ids...)

	responses, err := registry.ContainerEngine().PodStop(context.Background(), args, stopOptions.PodStopOptions)
	if err != nil {
		return err
	}
	// in the cli, first we print out all the successful attempts
	for _, r := range responses {
		if len(r.Errs) == 0 {
			fmt.Println(r.Id)
		} else {
			errs = append(errs, r.Errs...)
		}
	}
	return errs.PrintErrors()
}
