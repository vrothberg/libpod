package generate

import (
	"strings"

	"github.com/containers/common/pkg/capabilities"
	"github.com/containers/common/pkg/config"
	"github.com/containers/podman/v2/libpod"
	"github.com/containers/podman/v2/libpod/image"
	"github.com/containers/podman/v2/pkg/specgen"
	"github.com/containers/podman/v2/pkg/util"
	"github.com/opencontainers/runtime-tools/generate"
	"github.com/opencontainers/selinux/go-selinux/label"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// setLabelOpts sets the label options of the SecurityConfig according to the
// input.
func setLabelOpts(s *specgen.SpecGenerator, runtime *libpod.Runtime, pidConfig specgen.Namespace, ipcConfig specgen.Namespace) error {
	if !runtime.EnableLabeling() || s.Privileged {
		s.SelinuxOpts = label.DisableSecOpt()
		return nil
	}

	var labelOpts []string
	if pidConfig.IsHost() {
		labelOpts = append(labelOpts, label.DisableSecOpt()...)
	} else if pidConfig.IsContainer() {
		ctr, err := runtime.LookupContainer(pidConfig.Value)
		if err != nil {
			return errors.Wrapf(err, "container %q not found", pidConfig.Value)
		}
		secopts, err := label.DupSecOpt(ctr.ProcessLabel())
		if err != nil {
			return errors.Wrapf(err, "failed to duplicate label %q ", ctr.ProcessLabel())
		}
		labelOpts = append(labelOpts, secopts...)
	}

	if ipcConfig.IsHost() {
		labelOpts = append(labelOpts, label.DisableSecOpt()...)
	} else if ipcConfig.IsContainer() {
		ctr, err := runtime.LookupContainer(ipcConfig.Value)
		if err != nil {
			return errors.Wrapf(err, "container %q not found", ipcConfig.Value)
		}
		secopts, err := label.DupSecOpt(ctr.ProcessLabel())
		if err != nil {
			return errors.Wrapf(err, "failed to duplicate label %q ", ctr.ProcessLabel())
		}
		labelOpts = append(labelOpts, secopts...)
	}

	s.SelinuxOpts = append(s.SelinuxOpts, labelOpts...)
	return nil
}

func securityConfigureGenerator(s *specgen.SpecGenerator, g *generate.Generator, newImage *image.Image, rtc *config.Config) error {
	var (
		caplist []string
		err     error
	)
	// HANDLE CAPABILITIES
	// NOTE: Must happen before SECCOMP
	if s.Privileged {
		g.SetupPrivileged(true)
		caplist = capabilities.AllCapabilities()
	} else {
		caplist, err = capabilities.MergeCapabilities(rtc.Containers.DefaultCapabilities, s.CapAdd, s.CapDrop)
		if err != nil {
			return err
		}

		privCapsRequired := []string{}

		// If the container image specifies an label with a
		// capabilities.ContainerImageLabel then split the comma separated list
		// of capabilities and record them.  This list indicates the only
		// capabilities, required to run the container.
		var capsRequiredRequested []string
		for key, val := range s.Labels {
			if util.StringInSlice(key, capabilities.ContainerImageLabels) {
				capsRequiredRequested = strings.Split(val, ",")
			}
		}
		if !s.Privileged && len(capsRequiredRequested) > 0 {

			// Pass capRequiredRequested in CapAdd field to normalize capabilities names
			capsRequired, err := capabilities.MergeCapabilities(nil, capsRequiredRequested, nil)
			if err != nil {
				logrus.Errorf("capabilities requested by user or image are not valid: %q", strings.Join(capsRequired, ","))
			} else {
				// Verify all capRequiered are in the capList
				for _, cap := range capsRequired {
					if !util.StringInSlice(cap, caplist) {
						privCapsRequired = append(privCapsRequired, cap)
					}
				}
			}
			if len(privCapsRequired) == 0 {
				caplist = capsRequired
			} else {
				logrus.Errorf("capabilities requested by user or image are not allowed by default: %q", strings.Join(privCapsRequired, ","))
			}
		}
	}
	configSpec := g.Config
	configSpec.Process.Capabilities.Bounding = caplist

	if s.User == "" || s.User == "root" || s.User == "0" {
		configSpec.Process.Capabilities.Effective = caplist
		configSpec.Process.Capabilities.Permitted = caplist
		configSpec.Process.Capabilities.Inheritable = caplist
		configSpec.Process.Capabilities.Ambient = caplist
	} else {
		configSpec.Process.Capabilities.Effective = []string{}
		configSpec.Process.Capabilities.Permitted = []string{}
		configSpec.Process.Capabilities.Inheritable = []string{}
		configSpec.Process.Capabilities.Ambient = []string{}
	}
	// HANDLE SECCOMP
	if s.SeccompProfilePath != "unconfined" {
		seccompConfig, err := getSeccompConfig(s, configSpec, newImage)
		if err != nil {
			return err
		}
		configSpec.Linux.Seccomp = seccompConfig
	}

	// Clear default Seccomp profile from Generator for privileged containers
	if s.SeccompProfilePath == "unconfined" || s.Privileged {
		configSpec.Linux.Seccomp = nil
	}

	g.SetRootReadonly(s.ReadOnlyFilesystem)
	for sysctlKey, sysctlVal := range s.Sysctl {
		g.AddLinuxSysctl(sysctlKey, sysctlVal)
	}

	return nil
}
