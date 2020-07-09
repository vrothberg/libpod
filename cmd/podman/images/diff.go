package images

import (
	"github.com/containers/podman/v2/cmd/podman/registry"
	"github.com/containers/podman/v2/cmd/podman/report"
	"github.com/containers/podman/v2/pkg/domain/entities"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	// podman container _inspect_
	diffCmd = &cobra.Command{
		Use:   "diff [flags] IMAGE",
		Args:  cobra.ExactArgs(1),
		Short: "Inspect changes to the image's file systems",
		Long:  `Displays changes to the image's filesystem.  The image will be compared to its parent layer.`,
		RunE:  diff,
		Example: `podman image diff myImage
  podman image diff --format json redis:alpine`,
	}
	diffOpts *entities.DiffOptions
)

func init() {
	registry.Commands = append(registry.Commands, registry.CliCommand{
		Mode:    []entities.EngineMode{entities.ABIMode, entities.TunnelMode},
		Command: diffCmd,
		Parent:  imageCmd,
	})
	diffFlags(diffCmd.Flags())
}

func diffFlags(flags *pflag.FlagSet) {
	diffOpts = &entities.DiffOptions{}
	flags.BoolVar(&diffOpts.Archive, "archive", true, "Save the diff as a tar archive")
	_ = flags.MarkDeprecated("archive", "Provided for backwards compatibility, has no impact on output.")
	flags.StringVar(&diffOpts.Format, "format", "", "Change the output format")
}

func diff(cmd *cobra.Command, args []string) error {
	if diffOpts.Latest {
		return errors.New("image diff does not support --latest")
	}

	results, err := registry.ImageEngine().Diff(registry.GetContext(), args[0], *diffOpts)
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
