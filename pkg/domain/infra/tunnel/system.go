package tunnel

import (
	"context"
	"errors"

	"github.com/containers/podman/v2/libpod/define"
	"github.com/containers/podman/v2/pkg/bindings/system"
	"github.com/containers/podman/v2/pkg/domain/entities"
	"github.com/spf13/cobra"
)

func (ic *ContainerEngine) Info(ctx context.Context) (*define.Info, error) {
	return system.Info(ic.ClientCxt)
}

func (ic *ContainerEngine) VarlinkService(_ context.Context, _ entities.ServiceOptions) error {
	panic(errors.New("varlink service is not supported when tunneling"))
}

func (ic *ContainerEngine) SetupRootless(_ context.Context, cmd *cobra.Command) error {
	panic(errors.New("rootless engine mode is not supported when tunneling"))
}

// SystemPrune prunes unused data from the system.
func (ic *ContainerEngine) SystemPrune(ctx context.Context, options entities.SystemPruneOptions) (*entities.SystemPruneReport, error) {
	return system.Prune(ic.ClientCxt, &options.All, &options.Volume)
}

func (ic *ContainerEngine) SystemDf(ctx context.Context, options entities.SystemDfOptions) (*entities.SystemDfReport, error) {
	return system.DiskUsage(ic.ClientCxt)
}

func (ic *ContainerEngine) Unshare(ctx context.Context, args []string) error {
	return errors.New("unshare is not supported on remote clients")
}

func (ic ContainerEngine) Version(ctx context.Context) (*entities.SystemVersionReport, error) {
	return system.Version(ic.ClientCxt)
}
