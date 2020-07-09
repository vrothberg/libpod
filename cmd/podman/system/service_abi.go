// +build linux,!remote

package system

import (
	"context"
	"net"
	"strings"

	api "github.com/containers/podman/v2/pkg/api/server"
	"github.com/containers/podman/v2/pkg/domain/entities"
	"github.com/containers/podman/v2/pkg/domain/infra"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
)

func restService(opts entities.ServiceOptions, flags *pflag.FlagSet, cfg *entities.PodmanConfig) error {
	var (
		listener *net.Listener
		err      error
	)

	if opts.URI != "" {
		fields := strings.Split(opts.URI, ":")
		if len(fields) == 1 {
			return errors.Errorf("%s is an invalid socket destination", opts.URI)
		}
		address := strings.Join(fields[1:], ":")
		l, err := net.Listen(fields[0], address)
		if err != nil {
			return errors.Wrapf(err, "unable to create socket %s", opts.URI)
		}
		listener = &l
	}

	rt, err := infra.GetRuntime(context.Background(), flags, cfg)
	if err != nil {
		return err
	}

	server, err := api.NewServerWithSettings(rt, opts.Timeout, listener)
	if err != nil {
		return err
	}
	defer func() {
		if err := server.Shutdown(); err != nil {
			logrus.Warnf("Error when stopping API service: %s", err)
		}
	}()

	err = server.Serve()
	if listener != nil {
		_ = (*listener).Close()
	}
	return err
}
