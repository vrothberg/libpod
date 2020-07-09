package containers

import (
	"github.com/containers/podman/v2/cmd/podman/registry"
	"github.com/containers/podman/v2/cmd/podman/report"
	"github.com/containers/podman/v2/cmd/podman/validate"
	"github.com/containers/podman/v2/pkg/domain/entities"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	// podman container _diff_
	diffCmd = &cobra.Command{
		Use:   "diff [flags] CONTAINER",
		Args:  validate.IDOrLatestArgs,
		Short: "Inspect changes to the container's file systems",
		Long:  `Displays changes to the container filesystem's'.  The container will be compared to its parent layer.`,
		RunE:  diff,
		Example: `podman container diff myCtr
  podman container diff -l --format json myCtr`,
	}
	diffOpts *entities.DiffOptions
)

func init() {
	registry.Commands = append(registry.Commands, registry.CliCommand{
		Mode:    []entities.EngineMode{entities.ABIMode, entities.TunnelMode},
		Command: diffCmd,
		Parent:  containerCmd,
	})

	diffOpts = &entities.DiffOptions{}
	flags := diffCmd.Flags()
	flags.BoolVar(&diffOpts.Archive, "archive", true, "Save the diff as a tar archive")
	_ = flags.MarkHidden("archive")
	flags.StringVar(&diffOpts.Format, "format", "", "Change the output format")
	validate.AddLatestFlag(diffCmd, &diffOpts.Latest)
}

func diff(cmd *cobra.Command, args []string) error {
	if len(args) == 0 && !diffOpts.Latest {
		return errors.New("container must be specified: podman container diff [options [...]] ID-NAME")
	}

	var id string
	if len(args) > 0 {
		id = args[0]
	}
	results, err := registry.ContainerEngine().ContainerDiff(registry.GetContext(), id, *diffOpts)
	if err != nil {
		return err
	}

	switch diffOpts.Format {
	case "":
		return report.ChangesToTable(results)
	case "json":
		return report.ChangesToJSON(results)
	default:
		return errors.New("only supported value for '--format' is 'json'")
	}
}

func Diff(cmd *cobra.Command, args []string, options entities.DiffOptions) error {
	diffOpts = &options
	return diff(cmd, args)
}
