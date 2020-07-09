package tunnel

import (
	"context"

	"github.com/containers/podman/v2/pkg/bindings/network"
	"github.com/containers/podman/v2/pkg/domain/entities"
)

func (ic *ContainerEngine) NetworkList(ctx context.Context, options entities.NetworkListOptions) ([]*entities.NetworkListReport, error) {
	return network.List(ic.ClientCxt)
}

func (ic *ContainerEngine) NetworkInspect(ctx context.Context, namesOrIds []string, options entities.NetworkInspectOptions) ([]entities.NetworkInspectReport, error) {
	reports := make([]entities.NetworkInspectReport, 0, len(namesOrIds))
	for _, name := range namesOrIds {
		report, err := network.Inspect(ic.ClientCxt, name)
		if err != nil {
			return nil, err
		}
		reports = append(reports, report...)
	}
	return reports, nil
}

func (ic *ContainerEngine) NetworkRm(ctx context.Context, namesOrIds []string, options entities.NetworkRmOptions) ([]*entities.NetworkRmReport, error) {
	reports := make([]*entities.NetworkRmReport, 0, len(namesOrIds))
	for _, name := range namesOrIds {
		report, err := network.Remove(ic.ClientCxt, name, &options.Force)
		if err != nil {
			report[0].Err = err
		}
		reports = append(reports, report...)
	}
	return reports, nil
}

func (ic *ContainerEngine) NetworkCreate(ctx context.Context, name string, options entities.NetworkCreateOptions) (*entities.NetworkCreateReport, error) {
	return network.Create(ic.ClientCxt, options, &name)
}
