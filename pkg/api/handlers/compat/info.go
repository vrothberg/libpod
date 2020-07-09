package compat

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	goRuntime "runtime"
	"strings"
	"time"

	"github.com/containers/common/pkg/config"
	"github.com/containers/common/pkg/sysinfo"
	"github.com/containers/podman/v2/libpod"
	"github.com/containers/podman/v2/libpod/define"
	"github.com/containers/podman/v2/pkg/api/handlers"
	"github.com/containers/podman/v2/pkg/api/handlers/utils"
	"github.com/containers/podman/v2/pkg/rootless"
	docker "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func GetInfo(w http.ResponseWriter, r *http.Request) {
	// 200 ok
	// 500 internal
	runtime := r.Context().Value("runtime").(*libpod.Runtime)

	infoData, err := runtime.Info()
	if err != nil {
		utils.Error(w, "Something went wrong.", http.StatusInternalServerError, errors.Wrapf(err, "Failed to obtain system memory info"))
		return
	}

	configInfo, err := runtime.GetConfig()
	if err != nil {
		utils.Error(w, "Something went wrong.", http.StatusInternalServerError, errors.Wrapf(err, "Failed to obtain runtime config"))
		return
	}
	versionInfo, err := define.GetVersion()
	if err != nil {
		utils.Error(w, "Something went wrong.", http.StatusInternalServerError, errors.Wrapf(err, "Failed to obtain podman versions"))
		return
	}
	stateInfo := getContainersState(runtime)
	sysInfo := sysinfo.New(true)

	// FIXME: Need to expose if runtime supports Checkpoint'ing
	// liveRestoreEnabled := criu.CheckForCriu() && configInfo.RuntimeSupportsCheckpoint()

	info := &handlers.Info{Info: docker.Info{
		Architecture:       goRuntime.GOARCH,
		BridgeNfIP6tables:  !sysInfo.BridgeNFCallIP6TablesDisabled,
		BridgeNfIptables:   !sysInfo.BridgeNFCallIPTablesDisabled,
		CPUCfsPeriod:       sysInfo.CPUCfsPeriod,
		CPUCfsQuota:        sysInfo.CPUCfsQuota,
		CPUSet:             sysInfo.Cpuset,
		CPUShares:          sysInfo.CPUShares,
		CgroupDriver:       configInfo.Engine.CgroupManager,
		ClusterAdvertise:   "",
		ClusterStore:       "",
		ContainerdCommit:   docker.Commit{},
		Containers:         infoData.Store.ContainerStore.Number,
		ContainersPaused:   stateInfo[define.ContainerStatePaused],
		ContainersRunning:  stateInfo[define.ContainerStateRunning],
		ContainersStopped:  stateInfo[define.ContainerStateStopped] + stateInfo[define.ContainerStateExited],
		Debug:              log.IsLevelEnabled(log.DebugLevel),
		DefaultRuntime:     configInfo.Engine.OCIRuntime,
		DockerRootDir:      infoData.Store.GraphRoot,
		Driver:             infoData.Store.GraphDriverName,
		DriverStatus:       getGraphStatus(infoData.Store.GraphStatus),
		ExperimentalBuild:  true,
		GenericResources:   nil,
		HTTPProxy:          getEnv("http_proxy"),
		HTTPSProxy:         getEnv("https_proxy"),
		ID:                 uuid.New().String(),
		IPv4Forwarding:     !sysInfo.IPv4ForwardingDisabled,
		Images:             infoData.Store.ImageStore.Number,
		IndexServerAddress: "",
		InitBinary:         "",
		InitCommit:         docker.Commit{},
		Isolation:          "",
		KernelMemory:       sysInfo.KernelMemory,
		KernelMemoryTCP:    false,
		KernelVersion:      infoData.Host.Kernel,
		Labels:             nil,
		LiveRestoreEnabled: false,
		LoggingDriver:      "",
		MemTotal:           infoData.Host.MemTotal,
		MemoryLimit:        sysInfo.MemoryLimit,
		NCPU:               goRuntime.NumCPU(),
		NEventsListener:    0,
		NFd:                getFdCount(),
		NGoroutines:        goRuntime.NumGoroutine(),
		Name:               infoData.Host.Hostname,
		NoProxy:            getEnv("no_proxy"),
		OSType:             goRuntime.GOOS,
		OSVersion:          infoData.Host.Distribution.Version,
		OomKillDisable:     sysInfo.OomKillDisable,
		OperatingSystem:    infoData.Host.Distribution.Distribution,
		PidsLimit:          sysInfo.PidsLimit,
		Plugins:            docker.PluginsInfo{},
		ProductLicense:     "Apache-2.0",
		RegistryConfig:     nil,
		RuncCommit:         docker.Commit{},
		Runtimes:           getRuntimes(configInfo),
		SecurityOptions:    getSecOpts(sysInfo),
		ServerVersion:      versionInfo.Version,
		SwapLimit:          sysInfo.SwapLimit,
		Swarm: swarm.Info{
			LocalNodeState: swarm.LocalNodeStateInactive,
		},
		SystemStatus: nil,
		SystemTime:   time.Now().Format(time.RFC3339Nano),
		Warnings:     []string{},
	},
		BuildahVersion:     infoData.Host.BuildahVersion,
		CPURealtimePeriod:  sysInfo.CPURealtimePeriod,
		CPURealtimeRuntime: sysInfo.CPURealtimeRuntime,
		CgroupVersion:      infoData.Host.CGroupsVersion,
		Rootless:           rootless.IsRootless(),
		SwapFree:           infoData.Host.SwapFree,
		SwapTotal:          infoData.Host.SwapTotal,
		Uptime:             infoData.Host.Uptime,
	}
	utils.WriteResponse(w, http.StatusOK, info)
}

func getGraphStatus(storeInfo map[string]string) [][2]string {
	graphStatus := make([][2]string, 0, len(storeInfo))
	for k, v := range storeInfo {
		graphStatus = append(graphStatus, [2]string{k, v})
	}
	return graphStatus
}

func getSecOpts(sysInfo *sysinfo.SysInfo) []string {
	var secOpts []string
	if sysInfo.AppArmor {
		secOpts = append(secOpts, "name=apparmor")
	}
	if sysInfo.Seccomp {
		// FIXME: get profile name...
		secOpts = append(secOpts, fmt.Sprintf("name=seccomp,profile=%s", "default"))
	}
	return secOpts
}

func getRuntimes(configInfo *config.Config) map[string]docker.Runtime {
	var runtimes = map[string]docker.Runtime{}
	for name, paths := range configInfo.Engine.OCIRuntimes {
		runtimes[name] = docker.Runtime{
			Path: paths[0],
			Args: nil,
		}
	}
	return runtimes
}

func getFdCount() (count int) {
	count = -1
	if entries, err := ioutil.ReadDir("/proc/self/fd"); err == nil {
		count = len(entries)
	}
	return
}

// Just ignoring Container errors here...
func getContainersState(r *libpod.Runtime) map[define.ContainerStatus]int {
	var states = map[define.ContainerStatus]int{}
	ctnrs, err := r.GetAllContainers()
	if err == nil {
		for _, ctnr := range ctnrs {
			state, err := ctnr.State()
			if err != nil {
				continue
			}
			states[state]++
		}
	}
	return states
}

func getEnv(value string) string {
	if v, exists := os.LookupEnv(strings.ToUpper(value)); exists {
		return v
	}
	if v, exists := os.LookupEnv(strings.ToLower(value)); exists {
		return v
	}
	return ""
}
