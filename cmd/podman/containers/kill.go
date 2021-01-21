package containers

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/containers/common/pkg/completion"
	"github.com/containers/podman/v2/cmd/podman/common"
	"github.com/containers/podman/v2/cmd/podman/registry"
	"github.com/containers/podman/v2/cmd/podman/utils"
	"github.com/containers/podman/v2/cmd/podman/validate"
	"github.com/containers/podman/v2/pkg/domain/entities"
	"github.com/containers/podman/v2/pkg/signal"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	killDescription = "The main process inside each container specified will be sent SIGKILL, or any signal specified with option --signal."
	killCommand     = &cobra.Command{
		Use:   "kill [options] CONTAINER [CONTAINER...]",
		Short: "Kill one or more running containers with a specific signal",
		Long:  killDescription,
		RunE:  kill,
		Args: func(cmd *cobra.Command, args []string) error {
			return validate.CheckAllLatestAndCIDFile(cmd, args, false, true)
		},
		ValidArgsFunction: common.AutocompleteContainersRunning,
		Example: `podman kill mywebserver
  podman kill 860a4b23
  podman kill --signal TERM ctrID`,
	}

	containerKillCommand = &cobra.Command{
		Args: func(cmd *cobra.Command, args []string) error {
			return validate.CheckAllLatestAndCIDFile(cmd, args, false, true)
		},
		Use:               killCommand.Use,
		Short:             killCommand.Short,
		Long:              killCommand.Long,
		RunE:              killCommand.RunE,
		ValidArgsFunction: killCommand.ValidArgsFunction,
		Example: `podman container kill mywebserver
  podman container kill 860a4b23
  podman container kill --signal TERM ctrID`,
	}
)

var (
	killOptions = entities.KillOptions{}
)

func killFlags(cmd *cobra.Command) {
	flags := cmd.Flags()

	flags.BoolVarP(&killOptions.All, "all", "a", false, "Signal all running containers")

	signalFlagName := "signal"
	flags.StringVarP(&killOptions.Signal, signalFlagName, "s", "KILL", "Signal to send to the container")
	_ = cmd.RegisterFlagCompletionFunc(signalFlagName, common.AutocompleteStopSignal)
	cidfileFlagName := "cidfile"
	flags.StringArrayVar(&cidFiles, cidfileFlagName, []string{}, "Read the container ID from the file")
	_ = cmd.RegisterFlagCompletionFunc(cidfileFlagName, completion.AutocompleteDefault)
}

func init() {
	registry.Commands = append(registry.Commands, registry.CliCommand{
		Mode:    []entities.EngineMode{entities.ABIMode, entities.TunnelMode},
		Command: killCommand,
	})
	killFlags(killCommand)
	validate.AddLatestFlag(killCommand, &killOptions.Latest)

	registry.Commands = append(registry.Commands, registry.CliCommand{
		Mode:    []entities.EngineMode{entities.ABIMode, entities.TunnelMode},
		Command: containerKillCommand,
		Parent:  containerCmd,
	})
	killFlags(containerKillCommand)
	validate.AddLatestFlag(containerKillCommand, &killOptions.Latest)
}

func kill(_ *cobra.Command, args []string) error {
	var (
		err  error
		errs utils.OutputErrors
	)
	// Check if the signalString provided by the user is valid
	// Invalid signals will return err
	sig, err := signal.ParseSignalNameOrNumber(killOptions.Signal)
	if err != nil {
		return err
	}
	if sig < 1 || sig > 64 {
		return errors.New("valid signals are 1 through 64")
	}
	for _, cidFile := range cidFiles {
		content, err := ioutil.ReadFile(string(cidFile))
		if err != nil {
			return errors.Wrap(err, "error reading CIDFile")
		}
		id := strings.Split(string(content), "\n")[0]
		args = append(args, id)
	}

	responses, err := registry.ContainerEngine().ContainerKill(context.Background(), args, killOptions)
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
