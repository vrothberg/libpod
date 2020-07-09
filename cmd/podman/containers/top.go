package containers

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/containers/podman/v2/cmd/podman/registry"
	"github.com/containers/podman/v2/cmd/podman/validate"
	"github.com/containers/podman/v2/pkg/domain/entities"
	"github.com/containers/podman/v2/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	topDescription = `Similar to system "top" command.

  Specify format descriptors to alter the output.

  Running "podman top -l pid pcpu seccomp" will print the process ID, the CPU percentage and the seccomp mode of each process of the latest container.`

	topOptions = entities.TopOptions{}

	topCommand = &cobra.Command{
		Use:   "top [flags] CONTAINER [FORMAT-DESCRIPTORS|ARGS...]",
		Short: "Display the running processes of a container",
		Long:  topDescription,
		RunE:  top,
		Args:  cobra.ArbitraryArgs,
		Example: `podman top ctrID
podman top --latest
podman top ctrID pid seccomp args %C
podman top ctrID -eo user,pid,comm`,
	}

	containerTopCommand = &cobra.Command{
		Use:   topCommand.Use,
		Short: topCommand.Short,
		Long:  topCommand.Long,
		RunE:  topCommand.RunE,
		Example: `podman container top ctrID
podman container top --latest
podman container top ctrID pid seccomp args %C
podman container top ctrID -eo user,pid,comm`,
	}
)

func topFlags(flags *pflag.FlagSet) {
	flags.SetInterspersed(false)
	flags.BoolVar(&topOptions.ListDescriptors, "list-descriptors", false, "")
	_ = flags.MarkHidden("list-descriptors") // meant only for bash completion
}

func init() {
	registry.Commands = append(registry.Commands, registry.CliCommand{
		Mode:    []entities.EngineMode{entities.ABIMode, entities.TunnelMode},
		Command: topCommand,
	})
	topFlags(topCommand.Flags())
	validate.AddLatestFlag(topCommand, &topOptions.Latest)

	descriptors, err := util.GetContainerPidInformationDescriptors()
	if err == nil {
		topDescription = fmt.Sprintf("%s\n\n  Format Descriptors:\n    %s", topDescription, strings.Join(descriptors, ","))
		topCommand.Long = topDescription
	}

	registry.Commands = append(registry.Commands, registry.CliCommand{
		Mode:    []entities.EngineMode{entities.ABIMode, entities.TunnelMode},
		Command: containerTopCommand,
		Parent:  containerCmd,
	})
	topFlags(containerTopCommand.Flags())
	validate.AddLatestFlag(containerTopCommand, &topOptions.Latest)
}

func top(cmd *cobra.Command, args []string) error {
	if topOptions.ListDescriptors {
		descriptors, err := util.GetContainerPidInformationDescriptors()
		if err != nil {
			return err
		}
		fmt.Println(strings.Join(descriptors, "\n"))
		return nil
	}

	if len(args) < 1 && !topOptions.Latest {
		return errors.Errorf("you must provide the name or id of a running container")
	}

	if topOptions.Latest {
		topOptions.Descriptors = args
	} else {
		topOptions.NameOrID = args[0]
		topOptions.Descriptors = args[1:]
	}

	topResponse, err := registry.ContainerEngine().ContainerTop(context.Background(), topOptions)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 5, 1, 3, ' ', 0)
	for _, proc := range topResponse.Value {
		if _, err := fmt.Fprintln(w, proc); err != nil {
			return err
		}
	}
	return w.Flush()
}
