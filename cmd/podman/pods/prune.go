package pods

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/containers/podman/v2/cmd/podman/registry"
	"github.com/containers/podman/v2/cmd/podman/utils"
	"github.com/containers/podman/v2/pkg/domain/entities"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	pruneOptions = entities.PodPruneOptions{}
)

var (
	pruneDescription = fmt.Sprintf(`podman pod prune Removes all exited pods`)

	pruneCommand = &cobra.Command{
		Use:     "prune [flags]",
		Short:   "Remove all stopped pods and their containers",
		Long:    pruneDescription,
		RunE:    prune,
		Example: `podman pod prune`,
	}
)

func init() {
	registry.Commands = append(registry.Commands, registry.CliCommand{
		Mode:    []entities.EngineMode{entities.ABIMode, entities.TunnelMode},
		Command: pruneCommand,
		Parent:  podCmd,
	})
	flags := pruneCommand.Flags()
	flags.BoolVarP(&pruneOptions.Force, "force", "f", false, "Do not prompt for confirmation.  The default is false")
}

func prune(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return errors.Errorf("`%s` takes no arguments", cmd.CommandPath())
	}
	if !pruneOptions.Force {
		reader := bufio.NewReader(os.Stdin)
		fmt.Println("WARNING! This will remove all stopped/exited pods..")
		fmt.Print("Are you sure you want to continue? [y/N] ")
		answer, err := reader.ReadString('\n')
		if err != nil {
			return errors.Wrapf(err, "error reading input")
		}
		if strings.ToLower(answer)[0] != 'y' {
			return nil
		}
	}
	responses, err := registry.ContainerEngine().PodPrune(context.Background(), pruneOptions)
	if err != nil {
		return err
	}
	return utils.PrintPodPruneResults(responses)
}
