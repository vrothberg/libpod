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
	initDescription = `Initialize one or more containers, creating the OCI spec and mounts for inspection. Container names or IDs can be used.`

	initCommand = &cobra.Command{
		Use:   "init [flags] CONTAINER [CONTAINER...]",
		Short: "Initialize one or more containers",
		Long:  initDescription,
		RunE:  initContainer,
		Args: func(cmd *cobra.Command, args []string) error {
			return validate.CheckAllLatestAndCIDFile(cmd, args, false, false)
		},
		Example: `podman init --latest
  podman init 3c45ef19d893
  podman init test1`,
	}

	containerInitCommand = &cobra.Command{
		Use:   initCommand.Use,
		Short: initCommand.Short,
		Long:  initCommand.Long,
		RunE:  initCommand.RunE,
		Args:  initCommand.Args,
		Example: `podman container init --latest
  podman container init 3c45ef19d893
  podman container init test1`,
	}
)

var (
	initOptions entities.ContainerInitOptions
)

func initFlags(flags *pflag.FlagSet) {
	flags.BoolVarP(&initOptions.All, "all", "a", false, "Initialize all containers")
}

func init() {
	registry.Commands = append(registry.Commands, registry.CliCommand{
		Mode:    []entities.EngineMode{entities.ABIMode, entities.TunnelMode},
		Command: initCommand,
	})
	flags := initCommand.Flags()
	initFlags(flags)
	validate.AddLatestFlag(initCommand, &initOptions.Latest)

	registry.Commands = append(registry.Commands, registry.CliCommand{
		Mode:    []entities.EngineMode{entities.ABIMode, entities.TunnelMode},
		Parent:  containerCmd,
		Command: containerInitCommand,
	})
	containerInitFlags := containerInitCommand.Flags()
	initFlags(containerInitFlags)
	validate.AddLatestFlag(containerInitCommand, &initOptions.Latest)
}

func initContainer(cmd *cobra.Command, args []string) error {
	var errs utils.OutputErrors
	report, err := registry.ContainerEngine().ContainerInit(registry.GetContext(), args, initOptions)
	if err != nil {
		return err
	}
	for _, r := range report {
		if r.Err == nil {
			fmt.Println(r.Id)
		} else {
			errs = append(errs, r.Err)
		}
	}
	return errs.PrintErrors()
}
