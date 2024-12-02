package bootc

import (
	"fmt"

	"github.com/containers/podman/v5/cmd/podman/common"
	"github.com/containers/podman/v5/cmd/podman/registry"
	"github.com/spf13/cobra"
)

var (
	imageListCmd = &cobra.Command{
		Use:               "images [options]",
		Args:              cobra.MaximumNArgs(1),
		Short:             "List bootc images in local storage",
		Long:              "Lists bootc images previously pulled to the system or created on the system.",
		RunE:              listImages,
		ValidArgsFunction: common.AutocompleteImages,
		Example:           `podman bootc images --format json`,
	}
)

func init() {
	registry.Commands = append(registry.Commands, registry.CliCommand{
		Command: imageListCmd,
		Parent:  bootcCmd,
	})
}

func listImages(cmd *cobra.Command, args []string) error {
	fmt.Println("IMAGES GO HERE")
	return nil
}
