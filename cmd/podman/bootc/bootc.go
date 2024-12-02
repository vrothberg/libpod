package bootc

import (
	"github.com/containers/podman/v5/cmd/podman/registry"
	"github.com/containers/podman/v5/cmd/podman/validate"
	"github.com/spf13/cobra"
)

var (
	// Pull in configured json library
	json = registry.JSONLibrary()

	// Command: podman _bootc_
	bootcCmd = &cobra.Command{
		Use:   "bootc",
		Short: "Manage and work with bootc images",
		Long:  "Manage and work with bootc images",
		RunE:  validate.SubCommandExists,
	}
)

func init() {
	registry.Commands = append(registry.Commands, registry.CliCommand{
		Command: bootcCmd,
	})
}
