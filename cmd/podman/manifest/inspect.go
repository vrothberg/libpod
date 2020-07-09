package manifest

import (
	"context"
	"fmt"

	"github.com/containers/podman/v2/cmd/podman/registry"
	"github.com/containers/podman/v2/pkg/domain/entities"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	inspectCmd = &cobra.Command{
		Use:                   "inspect IMAGE",
		Short:                 "Display the contents of a manifest list or image index",
		Long:                  "Display the contents of a manifest list or image index.",
		RunE:                  inspect,
		Example:               "podman manifest inspect localhost/list",
		Args:                  cobra.ExactArgs(1),
		DisableFlagsInUseLine: true,
	}
)

func init() {
	registry.Commands = append(registry.Commands, registry.CliCommand{
		Mode:    []entities.EngineMode{entities.ABIMode, entities.TunnelMode},
		Command: inspectCmd,
		Parent:  manifestCmd,
	})
}

func inspect(cmd *cobra.Command, args []string) error {
	buf, err := registry.ImageEngine().ManifestInspect(context.Background(), args[0])
	if err != nil {
		return errors.Wrapf(err, "error inspect manifest %s", args[0])
	}
	fmt.Printf("%s\n", buf)
	return nil
}
