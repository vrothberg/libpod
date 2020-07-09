package images

import (
	"github.com/containers/podman/v2/cmd/podman/registry"
	"github.com/containers/podman/v2/pkg/domain/entities"
	"github.com/spf13/cobra"
)

var (
	existsCmd = &cobra.Command{
		Use:   "exists IMAGE",
		Short: "Check if an image exists in local storage",
		Long:  `If the named image exists in local storage, podman image exists exits with 0, otherwise the exit code will be 1.`,
		Args:  cobra.ExactArgs(1),
		RunE:  exists,
		Example: `podman image exists ID
  podman image exists IMAGE && podman pull IMAGE`,
		DisableFlagsInUseLine: true,
	}
)

func init() {
	registry.Commands = append(registry.Commands, registry.CliCommand{
		Mode:    []entities.EngineMode{entities.ABIMode, entities.TunnelMode},
		Command: existsCmd,
		Parent:  imageCmd,
	})
}

func exists(cmd *cobra.Command, args []string) error {
	found, err := registry.ImageEngine().Exists(registry.GetContext(), args[0])
	if err != nil {
		return err
	}
	if !found.Value {
		registry.SetExitCode(1)
	}
	return nil
}
