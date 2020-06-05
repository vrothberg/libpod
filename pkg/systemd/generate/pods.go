package generate

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/containers/libpod/libpod"
	"github.com/containers/libpod/pkg/domain/entities"
	"github.com/containers/libpod/version"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// podInfo contains data required for generating a pod's systemd
// unit file.
type podInfo struct {
	// ServiceName of the systemd service.
	ServiceName string
	// Name or ID of the infra container.
	InfraNameOrID string
	// StopTimeout sets the timeout Podman waits before killing the container
	// during service stop.
	StopTimeout uint
	// RestartPolicy of the systemd unit (e.g., no, on-failure, always).
	RestartPolicy string
	// PIDFile of the service. Required for forking services. Must point to the
	// PID of the associated conmon process.
	PIDFile string
	// GenerateTimestamp, if set the generated unit file has a time stamp.
	GenerateTimestamp bool
	// RequiredServices are services this service requires. Note that this
	// service runs before them.
	RequiredServices []string
	// PodmanVersion for the header. Will be set internally. Will be auto-filled
	// if left empty.
	PodmanVersion string
	// Executable is the path to the podman executable. Will be auto-filled if
	// left empty.
	Executable string
	// TimeStamp at the time of creating the unit file. Will be set internally.
	TimeStamp string
	// CreateCommand is the full command plus arguments of the process the
	// container has been created with.
	CreateCommand []string
	// EnvVariable is generate.EnvVariable and must not be set.
	EnvVariable string
}

const podTemplate = headerTemplate + `Requires={{- range $index, $value := .RequiredServices -}}{{if $index}} {{end}}{{ $value }}.service{{end}}
Before={{- range $index, $value := .RequiredServices -}}{{if $index}} {{end}}{{ $value }}.service{{end}}

[Service]
Environment={{.EnvVariable}}=%n
Restart={{.RestartPolicy}}
ExecStart={{.Executable}} start {{.InfraNameOrID}}
ExecStop={{.Executable}} stop {{if (ge .StopTimeout 0)}}-t {{.StopTimeout}}{{end}} {{.InfraNameOrID}}
PIDFile={{.PIDFile}}
KillMode=none
Type=forking

[Install]
WantedBy=multi-user.target default.target`

// PodUnits generates systemd units for the specified pod and its containers.
// Based on the options, the return value might be the content of all units or
// the files they been written to.
func PodUnits(pod *libpod.Pod, options entities.GenerateSystemdOptions) (string, error) {
	if options.New {
		return "", errors.New("--new is not supported for pods")
	}

	// Error out if the pod has no infra container, which we require to be the
	// main service.
	if !pod.HasInfraContainer() {
		return "", fmt.Errorf("error generating systemd unit files: Pod %q has no infra container", pod.Name())
	}

	podInfo, err := generatePodInfo(pod, options)
	if err != nil {
		return "", err
	}

	infraID, err := pod.InfraContainerID()
	if err != nil {
		return "", err
	}

	// Compute the container-dependency graph for the Pod.
	containers, err := pod.AllContainers()
	if err != nil {
		return "", err
	}
	if len(containers) == 0 {
		return "", fmt.Errorf("error generating systemd unit files: Pod %q has no containers", pod.Name())
	}
	graph, err := libpod.BuildContainerGraph(containers)
	if err != nil {
		return "", err
	}

	// Traverse the dependency graph and create systemdgen.containerInfo's for
	// each container.
	containerInfos := []*containerInfo{}
	for ctr, dependencies := range graph.DependencyMap() {
		// Skip the infra container as we already generated it.
		if ctr.ID() == infraID {
			continue
		}
		ctrInfo, err := generateContainerInfo(ctr, options)
		if err != nil {
			return "", err
		}
		// Now add the container's dependencies and at the container as a
		// required service of the infra container.
		for _, dep := range dependencies {
			if dep.ID() == infraID {
				ctrInfo.BoundToServices = append(ctrInfo.BoundToServices, podInfo.ServiceName)
			} else {
				_, serviceName := containerServiceName(dep, options)
				ctrInfo.BoundToServices = append(ctrInfo.BoundToServices, serviceName)
			}
		}
		podInfo.RequiredServices = append(podInfo.RequiredServices, ctrInfo.ServiceName)
		containerInfos = append(containerInfos, ctrInfo)
	}

	// Now generate the systemd service for all containers.
	builder := strings.Builder{}
	out, err := executePodTemplate(podInfo, options)
	if err != nil {
		return "", err
	}
	builder.WriteString(out)
	for _, info := range containerInfos {
		builder.WriteByte('\n')
		out, err := executeContainerTemplate(info, options)
		if err != nil {
			return "", err
		}
		builder.WriteString(out)
	}

	return builder.String(), nil
}

func generatePodInfo(pod *libpod.Pod, options entities.GenerateSystemdOptions) (*podInfo, error) {
	// Generate a systemdgen.containerInfo for the infra container. This
	// containerInfo acts as the main service of the pod.
	infraCtr, err := pod.InfraContainer()
	if err != nil {
		return nil, errors.Wrap(err, "could not find infra container")
	}

	timeout := infraCtr.StopTimeout()
	if options.StopTimeout != nil {
		timeout = *options.StopTimeout
	}

	config := infraCtr.Config()
	conmonPidFile := config.ConmonPidFile
	if conmonPidFile == "" {
		return nil, errors.Errorf("conmon PID file path is empty, try to recreate the container with --conmon-pidfile flag")
	}

	createCommand := []string{}

	nameOrID := pod.ID()
	ctrNameOrID := infraCtr.ID()
	if options.Name {
		nameOrID = pod.Name()
		ctrNameOrID = infraCtr.Name()
	}
	serviceName := fmt.Sprintf("%s%s%s", options.PodPrefix, options.Separator, nameOrID)

	info := podInfo{
		ServiceName:       serviceName,
		InfraNameOrID:     ctrNameOrID,
		RestartPolicy:     options.RestartPolicy,
		PIDFile:           conmonPidFile,
		StopTimeout:       timeout,
		GenerateTimestamp: true,
		CreateCommand:     createCommand,
	}
	return &info, nil
}

// executePodTemplate executes the pod template on the specified podInfo.  Note
// that the podInfo is also post processed and completed, which allows for an
// easier unit testing.
func executePodTemplate(info *podInfo, options entities.GenerateSystemdOptions) (string, error) {
	if err := validateRestartPolicy(info.RestartPolicy); err != nil {
		return "", err
	}

	// Make sure the executable is set.
	if info.Executable == "" {
		executable, err := os.Executable()
		if err != nil {
			executable = "/usr/bin/podman"
			logrus.Warnf("Could not obtain podman executable location, using default %s", executable)
		}
		info.Executable = executable
	}

	info.EnvVariable = EnvVariable

	if info.PodmanVersion == "" {
		info.PodmanVersion = version.Version
	}
	if info.GenerateTimestamp {
		info.TimeStamp = fmt.Sprintf("%v", time.Now().Format(time.UnixDate))
	}

	// Sort the slices to assure a deterministic output.
	sort.Strings(info.RequiredServices)

	// Generate the template and compile it.
	templ, err := template.New("pod_template").Parse(podTemplate)
	if err != nil {
		return "", errors.Wrap(err, "error parsing systemd service template")
	}

	var buf bytes.Buffer
	if err := templ.Execute(&buf, info); err != nil {
		return "", err
	}

	if !options.Files {
		return buf.String(), nil
	}

	buf.WriteByte('\n')
	cwd, err := os.Getwd()
	if err != nil {
		return "", errors.Wrap(err, "error getting current working directory")
	}
	path := filepath.Join(cwd, fmt.Sprintf("%s.service", info.ServiceName))
	if err := ioutil.WriteFile(path, buf.Bytes(), 0644); err != nil {
		return "", errors.Wrap(err, "error generating systemd unit")
	}
	return path, nil
}
