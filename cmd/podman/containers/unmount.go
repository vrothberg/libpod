package containers

import (
	"fmt"

	"github.com/containers/podman/v2/cmd/podman/registry"
	"github.com/containers/podman/v2/cmd/podman/utils"
	"github.com/containers/podman/v2/cmd/podman/validate"
	"github.com/containers/podman/v2/pkg/domain/entities"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	description = `Container storage increments a mount counter each time a container is mounted.

  When a container is unmounted, the mount counter is decremented. The container's root filesystem is physically unmounted only when the mount counter reaches zero indicating no other processes are using the mount.

  An unmount can be forced with the --force flag.
`
	umountCommand = &cobra.Command{
		Use:     "umount [flags] CONTAINER [CONTAINER...]",
		Aliases: []string{"unmount"},
		Short:   "Unmounts working container's root filesystem",
		Long:    description,
		RunE:    unmount,
		Args: func(cmd *cobra.Command, args []string) error {
			return validate.CheckAllLatestAndCIDFile(cmd, args, false, false)
		},
		Example: `podman umount ctrID
  podman umount ctrID1 ctrID2 ctrID3
  podman umount --all`,
	}

	containerUnmountCommand = &cobra.Command{
		Use:   umountCommand.Use,
		Short: umountCommand.Short,
		Long:  umountCommand.Long,
		RunE:  umountCommand.RunE,
		Args: func(cmd *cobra.Command, args []string) error {
			return validate.CheckAllLatestAndCIDFile(cmd, args, false, false)
		},
		Example: `podman container umount ctrID
  podman container umount ctrID1 ctrID2 ctrID3
  podman container umount --all`,
	}
)

var (
	unmountOpts entities.ContainerUnmountOptions
)

func umountFlags(flags *pflag.FlagSet) {
	flags.BoolVarP(&unmountOpts.All, "all", "a", false, "Umount all of the currently mounted containers")
	flags.BoolVarP(&unmountOpts.Force, "force", "f", false, "Force the complete umount all of the currently mounted containers")
}

func init() {
	registry.Commands = append(registry.Commands, registry.CliCommand{
		Mode:    []entities.EngineMode{entities.ABIMode},
		Command: umountCommand,
	})
	umountFlags(umountCommand.Flags())
	validate.AddLatestFlag(umountCommand, &unmountOpts.Latest)

	registry.Commands = append(registry.Commands, registry.CliCommand{
		Mode:    []entities.EngineMode{entities.ABIMode},
		Command: containerUnmountCommand,
		Parent:  containerCmd,
	})
	umountFlags(containerUnmountCommand.Flags())
	validate.AddLatestFlag(containerUnmountCommand, &unmountOpts.Latest)
}

func unmount(cmd *cobra.Command, args []string) error {
	var errs utils.OutputErrors
	reports, err := registry.ContainerEngine().ContainerUnmount(registry.GetContext(), args, unmountOpts)
	if err != nil {
		return err
	}
	for _, r := range reports {
		if r.Err == nil {
			fmt.Println(r.Id)
		} else {
			errs = append(errs, r.Err)
		}
	}
	return errs.PrintErrors()
}
