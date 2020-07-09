package pods

import (
	"context"
	"fmt"

	"github.com/containers/podman/v2/cmd/podman/registry"
	"github.com/containers/podman/v2/cmd/podman/utils"
	"github.com/containers/podman/v2/cmd/podman/validate"
	"github.com/containers/podman/v2/pkg/domain/entities"
	"github.com/spf13/cobra"
)

var (
	podKillDescription = `Signals are sent to the main process of each container inside the specified pod.

  The default signal is SIGKILL, or any signal specified with option --signal.`
	killCommand = &cobra.Command{
		Use:   "kill [flags] POD [POD...]",
		Short: "Send the specified signal or SIGKILL to containers in pod",
		Long:  podKillDescription,
		RunE:  kill,
		Args: func(cmd *cobra.Command, args []string) error {
			return validate.CheckAllLatestAndCIDFile(cmd, args, false, false)
		},
		Example: `podman pod kill podID
  podman pod kill --signal TERM mywebserver
  podman pod kill --latest`,
	}
)

var (
	killOpts entities.PodKillOptions
)

func init() {
	registry.Commands = append(registry.Commands, registry.CliCommand{
		Mode:    []entities.EngineMode{entities.ABIMode, entities.TunnelMode},
		Command: killCommand,
		Parent:  podCmd,
	})
	flags := killCommand.Flags()
	flags.BoolVarP(&killOpts.All, "all", "a", false, "Kill all containers in all pods")
	flags.StringVarP(&killOpts.Signal, "signal", "s", "KILL", "Signal to send to the containers in the pod")
	validate.AddLatestFlag(killCommand, &killOpts.Latest)
}

func kill(_ *cobra.Command, args []string) error {
	var (
		errs utils.OutputErrors
	)
	responses, err := registry.ContainerEngine().PodKill(context.Background(), args, killOpts)
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
