package generate

import (
	"context"

	"github.com/containers/image/v5/manifest"
	"github.com/containers/podman/v2/libpod"
	"github.com/containers/podman/v2/libpod/image"
	ann "github.com/containers/podman/v2/pkg/annotations"
	envLib "github.com/containers/podman/v2/pkg/env"
	"github.com/containers/podman/v2/pkg/signal"
	"github.com/containers/podman/v2/pkg/specgen"
	"github.com/pkg/errors"
	"golang.org/x/sys/unix"
)

// Fill any missing parts of the spec generator (e.g. from the image).
// Returns a set of warnings or any fatal error that occurred.
func CompleteSpec(ctx context.Context, r *libpod.Runtime, s *specgen.SpecGenerator) ([]string, error) {
	var (
		newImage *image.Image
		err      error
	)

	// Only add image configuration if we have an image
	if s.Image != "" {
		newImage, err = r.ImageRuntime().NewFromLocal(s.Image)
		if err != nil {
			return nil, err
		}

		_, mediaType, err := newImage.Manifest(ctx)
		if err != nil {
			return nil, err
		}

		if s.HealthConfig == nil && mediaType == manifest.DockerV2Schema2MediaType {
			s.HealthConfig, err = newImage.GetHealthCheck(ctx)
			if err != nil {
				return nil, err
			}
		}

		// Image stop signal
		if s.StopSignal == nil {
			stopSignal, err := newImage.StopSignal(ctx)
			if err != nil {
				return nil, err
			}
			if stopSignal != "" {
				sig, err := signal.ParseSignalNameOrNumber(stopSignal)
				if err != nil {
					return nil, err
				}
				s.StopSignal = &sig
			}
		}
	}

	rtc, err := r.GetConfig()
	if err != nil {
		return nil, err
	}
	// Get Default Environment
	defaultEnvs, err := envLib.ParseSlice(rtc.Containers.Env)
	if err != nil {
		return nil, errors.Wrap(err, "Env fields in containers.conf failed to parse")
	}

	var envs map[string]string

	if newImage != nil {
		// Image envs from the image if they don't exist
		// already, overriding the default environments
		imageEnvs, err := newImage.Env(ctx)
		if err != nil {
			return nil, err
		}

		envs, err = envLib.ParseSlice(imageEnvs)
		if err != nil {
			return nil, errors.Wrap(err, "Env fields from image failed to parse")
		}
	}

	s.Env = envLib.Join(envLib.Join(defaultEnvs, envs), s.Env)

	// Labels and Annotations
	annotations := make(map[string]string)
	if newImage != nil {
		labels, err := newImage.Labels(ctx)
		if err != nil {
			return nil, err
		}

		// labels from the image that dont exist already
		if len(labels) > 0 && s.Labels == nil {
			s.Labels = make(map[string]string)
		}
		for k, v := range labels {
			if _, exists := s.Labels[k]; !exists {
				s.Labels[k] = v
			}
		}

		// Add annotations from the image
		imgAnnotations, err := newImage.Annotations(ctx)
		if err != nil {
			return nil, err
		}
		for k, v := range imgAnnotations {
			annotations[k] = v
		}
	}

	// in the event this container is in a pod, and the pod has an infra container
	// we will want to configure it as a type "container" instead defaulting to
	// the behavior of a "sandbox" container
	// In Kata containers:
	// - "sandbox" is the annotation that denotes the container should use its own
	//   VM, which is the default behavior
	// - "container" denotes the container should join the VM of the SandboxID
	//   (the infra container)

	if len(s.Pod) > 0 {
		annotations[ann.SandboxID] = s.Pod
		annotations[ann.ContainerType] = ann.ContainerTypeContainer
	}

	// now pass in the values from client
	for k, v := range s.Annotations {
		annotations[k] = v
	}
	s.Annotations = annotations

	// workdir
	if newImage != nil {
		workingDir, err := newImage.WorkingDir(ctx)
		if err != nil {
			return nil, err
		}
		if len(s.WorkDir) < 1 && len(workingDir) > 1 {
			s.WorkDir = workingDir
		}
	}

	if len(s.SeccompProfilePath) < 1 {
		p, err := libpod.DefaultSeccompPath()
		if err != nil {
			return nil, err
		}
		s.SeccompProfilePath = p
	}

	if len(s.User) == 0 && newImage != nil {
		s.User, err = newImage.User(ctx)
		if err != nil {
			return nil, err
		}
	}
	if err := finishThrottleDevices(s); err != nil {
		return nil, err
	}
	// Unless already set via the CLI, check if we need to disable process
	// labels or set the defaults.
	if len(s.SelinuxOpts) == 0 {
		if err := setLabelOpts(s, r, s.PidNS, s.IpcNS); err != nil {
			return nil, err
		}
	}

	return verifyContainerResources(s)
}

// finishThrottleDevices takes the temporary representation of the throttle
// devices in the specgen and looks up the major and major minors. it then
// sets the throttle devices proper in the specgen
func finishThrottleDevices(s *specgen.SpecGenerator) error {
	if bps := s.ThrottleReadBpsDevice; len(bps) > 0 {
		for k, v := range bps {
			statT := unix.Stat_t{}
			if err := unix.Stat(k, &statT); err != nil {
				return err
			}
			v.Major = (int64(unix.Major(statT.Rdev)))
			v.Minor = (int64(unix.Minor(statT.Rdev)))
			s.ResourceLimits.BlockIO.ThrottleReadBpsDevice = append(s.ResourceLimits.BlockIO.ThrottleReadBpsDevice, v)
		}
	}
	if bps := s.ThrottleWriteBpsDevice; len(bps) > 0 {
		for k, v := range bps {
			statT := unix.Stat_t{}
			if err := unix.Stat(k, &statT); err != nil {
				return err
			}
			v.Major = (int64(unix.Major(statT.Rdev)))
			v.Minor = (int64(unix.Minor(statT.Rdev)))
			s.ResourceLimits.BlockIO.ThrottleWriteBpsDevice = append(s.ResourceLimits.BlockIO.ThrottleWriteBpsDevice, v)
		}
	}
	if iops := s.ThrottleReadIOPSDevice; len(iops) > 0 {
		for k, v := range iops {
			statT := unix.Stat_t{}
			if err := unix.Stat(k, &statT); err != nil {
				return err
			}
			v.Major = (int64(unix.Major(statT.Rdev)))
			v.Minor = (int64(unix.Minor(statT.Rdev)))
			s.ResourceLimits.BlockIO.ThrottleReadIOPSDevice = append(s.ResourceLimits.BlockIO.ThrottleReadIOPSDevice, v)
		}
	}
	if iops := s.ThrottleWriteIOPSDevice; len(iops) > 0 {
		for k, v := range iops {
			statT := unix.Stat_t{}
			if err := unix.Stat(k, &statT); err != nil {
				return err
			}
			v.Major = (int64(unix.Major(statT.Rdev)))
			v.Minor = (int64(unix.Minor(statT.Rdev)))
			s.ResourceLimits.BlockIO.ThrottleWriteIOPSDevice = append(s.ResourceLimits.BlockIO.ThrottleWriteIOPSDevice, v)
		}
	}
	return nil
}
