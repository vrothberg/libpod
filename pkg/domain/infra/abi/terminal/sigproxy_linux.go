package terminal

import (
	"os"
	"syscall"

	"github.com/containers/podman/v2/libpod"
	"github.com/containers/podman/v2/pkg/signal"
	"github.com/sirupsen/logrus"
)

// ProxySignals ...
func ProxySignals(ctr *libpod.Container) {
	sigBuffer := make(chan os.Signal, 128)
	signal.CatchAll(sigBuffer)

	logrus.Debugf("Enabling signal proxying")

	go func() {
		for s := range sigBuffer {
			// Ignore SIGCHLD and SIGPIPE - these are mostly likely
			// intended for the podman command itself.
			// SIGURG was added because of golang 1.14 and its preemptive changes
			// causing more signals to "show up".
			// https://github.com/containers/podman/issues/5483
			if s == syscall.SIGCHLD || s == syscall.SIGPIPE || s == syscall.SIGURG {
				continue
			}

			if err := ctr.Kill(uint(s.(syscall.Signal))); err != nil {
				// If the container dies, and we find out here,
				// we need to forward that one signal to
				// ourselves so that it is not lost, and then
				// we terminate the proxy and let the defaults
				// play out.
				logrus.Errorf("Error forwarding signal %d to container %s: %v", s, ctr.ID(), err)
				signal.StopCatch(sigBuffer)
				if err := syscall.Kill(syscall.Getpid(), s.(syscall.Signal)); err != nil {
					logrus.Errorf("failed to kill pid %d", syscall.Getpid())
				}
				return
			}
		}
	}()
}
