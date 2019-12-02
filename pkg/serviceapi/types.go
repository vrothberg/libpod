package serviceapi

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/containers/image/v5/manifest"
	"github.com/containers/libpod/libpod"
	"github.com/containers/libpod/libpod/define"
	libpodImage "github.com/containers/libpod/libpod/image"
	docker "github.com/docker/docker/api/types"
	dockerContainer "github.com/docker/docker/api/types/container"
	dockerNetwork "github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/pkg/errors"
)

type ImageInspect struct {
	docker.ImageInspect
}

type ContainerConfig struct {
	dockerContainer.Config
}

type ImageSummary struct {
	docker.ImageSummary
}

type ContainersPruneReport struct {
	docker.ContainersPruneReport
}

type Info struct {
	docker.Info
	BuildahVersion     string
	CPURealtimePeriod  bool
	CPURealtimeRuntime bool
	CgroupVersion      string
	Rootless           bool
	SwapFree           int64
	SwapTotal          int64
	Uptime             string
}

type Container struct {
	docker.Container
	docker.ContainerCreateConfig
}

type ContainerStats struct {
	docker.ContainerStats
}

type Ping struct {
	docker.Ping
}

type Version struct {
	docker.Version
}

type DiskUsage struct {
	docker.DiskUsage
}

type VolumesPruneReport struct {
	docker.VolumesPruneReport
}

type ImagesPruneReport struct {
	docker.ImagesPruneReport
}

type BuildCachePruneReport struct {
	docker.BuildCachePruneReport
}

type NetworkPruneReport struct {
	docker.NetworksPruneReport
}

type ConfigCreateResponse struct {
	docker.ConfigCreateResponse
}

type PushResult struct {
	docker.PushResult
}

type BuildResult struct {
	docker.BuildResult
}

type ContainerWaitOKBody struct {
	StatusCode int
	Error      struct {
		Message string
	}
}

type CreateContainer struct {
	Name string
	dockerContainer.Config
	HostConfig       dockerContainer.HostConfig
	NetworkingConfig dockerNetwork.NetworkingConfig
}

type IDResponse struct {
	ID string `json:"id"`
}

type Stats struct {
	docker.StatsJSON
}

type ContainerTopOKBody struct {
	dockerContainer.ContainerTopOKBody
	ID string `json:"Id"`
}

type PodCreate struct {
	Name         string   `json:"name"`
	CGroupParent string   `json:"cgroup-parent"`
	Hostname     string   `json:"hostname"`
	Infra        bool     `json:"infra"`
	InfraCommand string   `json:"infra-command"`
	InfraImage   string   `json:"infra-image"`
	Labels       []string `json:"labels"`
	Publish      []string `json:"publish"`
	Share        string   `json:"share"`
}

func ImageToImageSummary(l *libpodImage.Image) (*ImageSummary, error) {
	containers, err := l.Containers()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to obtain Containers for image %s", l.ID())
	}
	containerCount := len(containers)

	var digests []string
	for _, d := range l.Digests() {
		digests = append(digests, string(d))
	}

	tags, err := l.RepoTags()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to obtain RepoTags for image %s", l.ID())
	}

	// FIXME: GetParent() panics
	// parent, err := l.GetParent(context.TODO())
	// if err != nil {
	// 	return nil, errors.Wrapf(err, "Failed to obtain ParentID for image %s", l.ID())
	// }

	labels, err := l.Labels(context.TODO())
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to obtain Labels for image %s", l.ID())
	}

	size, err := l.Size(context.TODO())
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to obtain Size for image %s", l.ID())
	}
	return &ImageSummary{docker.ImageSummary{
		Containers:  int64(containerCount),
		Created:     l.Created().Unix(),
		ID:          l.ID(),
		Labels:      labels,
		ParentID:    "parent.ID()",
		RepoDigests: digests,
		RepoTags:    tags,
		SharedSize:  0,
		Size:        int64(*size),
		VirtualSize: int64(*size),
	}}, nil
}

func ImageDataToImageInspect(ctx context.Context, l *libpodImage.Image) (*ImageInspect, error) {
	info, err := l.Inspect(context.Background())
	if err != nil {
		return nil, err
	}
	ports, err := portsToPortSet(info.Config.ExposedPorts)
	if err != nil {
		return nil, err
	}
	// TODO the rest of these still need wiring!
	config := dockerContainer.Config{
		//	Hostname:        "",
		//	Domainname:      "",
		User: info.User,
		//	AttachStdin:     false,
		//	AttachStdout:    false,
		//	AttachStderr:    false,
		ExposedPorts: ports,
		//	Tty:             false,
		//	OpenStdin:       false,
		//	StdinOnce:       false,
		Env: info.Config.Env,
		Cmd: info.Config.Cmd,
		//	Healthcheck:     nil,
		//	ArgsEscaped:     false,
		//	Image:           "",
		//	Volumes:         nil,
		//	WorkingDir:      "",
		//	Entrypoint:      nil,
		//	NetworkDisabled: false,
		//	MacAddress:      "",
		//	OnBuild:         nil,
		//	Labels:          nil,
		//	StopSignal:      "",
		//	StopTimeout:     nil,
		//	Shell:           nil,
	}
	ic, err := l.ToImageRef(ctx)
	if err != nil {
		return nil, err
	}
	dockerImageInspect := docker.ImageInspect{
		Architecture:  l.Architecture,
		Author:        l.Author,
		Comment:       info.Comment,
		Config:        &config,
		Created:       l.Created().Format(time.RFC3339Nano),
		DockerVersion: "",
		GraphDriver:   docker.GraphDriverData{},
		ID:            fmt.Sprintf("sha256:%s", l.ID()),
		Metadata:      docker.ImageMetadata{},
		Os:            l.Os,
		OsVersion:     l.Version,
		Parent:        l.Parent,
		RepoDigests:   info.RepoDigests,
		RepoTags:      info.RepoTags,
		RootFS:        docker.RootFS{},
		Size:          info.Size,
		Variant:       "",
		VirtualSize:   info.VirtualSize,
	}
	bi := ic.ConfigInfo()
	// For docker images, we need to get the container id and config
	// and populate the image with it.
	if bi.MediaType == manifest.DockerV2Schema2ConfigMediaType {
		d := manifest.Schema2Image{}
		b, err := ic.ConfigBlob(ctx)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(b, &d); err != nil {
			return nil, err
		}
		// populate the container id into the image
		dockerImageInspect.Container = d.Container
		containerConfig := dockerContainer.Config{}
		configBytes, err := json.Marshal(d.ContainerConfig)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(configBytes, &containerConfig); err != nil {
			return nil, err
		}
		// populate the container config in the image
		dockerImageInspect.ContainerConfig = &containerConfig
		// populate parent
		dockerImageInspect.Parent = d.Parent.String()
	}
	return &ImageInspect{dockerImageInspect}, nil

}

func LibpodToContainer(l *libpod.Container, infoData []define.InfoData) (*Container, error) {
	imageId, imageName := l.Image()
	sizeRW, err := l.RWSize()
	if err != nil {
		return nil, err
	}

	SizeRootFs, err := l.RootFsSize()
	if err != nil {
		return nil, err
	}

	state, err := l.State()
	if err != nil {
		return nil, err
	}

	return &Container{docker.Container{
		ID:         l.ID(),
		Names:      []string{l.Name()},
		Image:      imageName,
		ImageID:    imageId,
		Command:    strings.Join(l.Command(), " "),
		Created:    l.CreatedTime().Unix(),
		Ports:      nil,
		SizeRw:     sizeRW,
		SizeRootFs: SizeRootFs,
		Labels:     l.Labels(),
		State:      string(state),
		Status:     "",
		HostConfig: struct {
			NetworkMode string `json:",omitempty"`
		}{
			"host"},
		NetworkSettings: nil,
		Mounts:          nil,
	},
		docker.ContainerCreateConfig{},
	}, nil
}

func libpodToContainerJSON(l *libpod.Container) (*docker.ContainerJSON, error) {
	_, imageName := l.Image()
	inspect, err := l.Inspect(true)
	if err != nil {
		return nil, err
	}
	i, err := json.Marshal(inspect.State)
	if err != nil {
		return nil, err
	}
	state := docker.ContainerState{}
	if err := json.Unmarshal(i, &state); err != nil {
		return nil, err
	}

	// docker considers paused to be running
	if state.Paused {
		state.Running = true
	}

	h, err := json.Marshal(inspect.HostConfig)
	if err != nil {
		return nil, err
	}
	hc := dockerContainer.HostConfig{}
	if err := json.Unmarshal(h, &hc); err != nil {
		return nil, err
	}
	g, err := json.Marshal(inspect.GraphDriver)
	if err != nil {
		return nil, err
	}
	graphDriver := docker.GraphDriverData{}
	if err := json.Unmarshal(g, &graphDriver); err != nil {
		return nil, err
	}

	cb := docker.ContainerJSONBase{
		ID:              l.ID(),
		Created:         l.CreatedTime().String(),
		Path:            "",
		Args:            nil,
		State:           &state,
		Image:           imageName,
		ResolvConfPath:  inspect.ResolvConfPath,
		HostnamePath:    inspect.HostnamePath,
		HostsPath:       inspect.HostsPath,
		LogPath:         l.LogPath(),
		Node:            nil,
		Name:            l.Name(),
		RestartCount:    0,
		Driver:          inspect.Driver,
		Platform:        "linux",
		MountLabel:      inspect.MountLabel,
		ProcessLabel:    inspect.ProcessLabel,
		AppArmorProfile: inspect.AppArmorProfile,
		ExecIDs:         inspect.ExecIDs,
		HostConfig:      &hc,
		GraphDriver:     graphDriver,
		SizeRw:          &inspect.SizeRw,
		SizeRootFs:      &inspect.SizeRootFs,
	}

	stopTimeout := int(l.StopTimeout())

	ports := make(nat.PortSet)
	for p := range inspect.HostConfig.PortBindings {
		splitp := strings.Split(p, "/")
		port, err := nat.NewPort(splitp[0], splitp[1])
		if err != nil {
			return nil, err
		}
		ports[port] = struct{}{}
	}

	config := dockerContainer.Config{
		Hostname:        l.Hostname(),
		Domainname:      inspect.Config.DomainName,
		User:            l.User(),
		AttachStdin:     inspect.Config.AttachStdin,
		AttachStdout:    inspect.Config.AttachStdout,
		AttachStderr:    inspect.Config.AttachStderr,
		ExposedPorts:    ports,
		Tty:             inspect.Config.Tty,
		OpenStdin:       inspect.Config.OpenStdin,
		StdinOnce:       inspect.Config.StdinOnce,
		Env:             inspect.Config.Env,
		Cmd:             inspect.Config.Cmd,
		Healthcheck:     nil,
		ArgsEscaped:     false,
		Image:           imageName,
		Volumes:         nil,
		WorkingDir:      l.WorkingDir(),
		Entrypoint:      l.Entrypoint(),
		NetworkDisabled: false,
		MacAddress:      "",
		OnBuild:         nil,
		Labels:          l.Labels(),
		StopSignal:      string(l.StopSignal()),
		StopTimeout:     &stopTimeout,
		Shell:           nil,
	}

	m, err := json.Marshal(inspect.Mounts)
	if err != nil {
		return nil, err
	}
	mounts := []docker.MountPoint{}
	if err := json.Unmarshal(m, &mounts); err != nil {
		return nil, err
	}
	c := docker.ContainerJSON{
		ContainerJSONBase: &cb,
		Mounts:            mounts,
		Config:            &config,
		NetworkSettings:   nil, // TODO i dont have the guts for this right now
	}
	return &c, nil
}

// portsToPortSet converts libpods exposed ports to dockers structs
func portsToPortSet(input map[string]struct{}) (nat.PortSet, error) {
	ports := make(nat.PortSet)
	for k := range input {
		npTCP, err := nat.NewPort("tcp", k)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to create tcp port from %s", k)
		}
		npUDP, err := nat.NewPort("udp", k)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to create udp port from %s", k)
		}
		ports[npTCP] = struct{}{}
		ports[npUDP] = struct{}{}
	}
	return ports, nil
}
