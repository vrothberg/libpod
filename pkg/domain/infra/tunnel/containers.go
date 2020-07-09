package tunnel

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/containers/common/pkg/config"
	"github.com/containers/image/v5/docker/reference"
	"github.com/containers/podman/v2/libpod/define"
	"github.com/containers/podman/v2/pkg/api/handlers"
	"github.com/containers/podman/v2/pkg/bindings"
	"github.com/containers/podman/v2/pkg/bindings/containers"
	"github.com/containers/podman/v2/pkg/domain/entities"
	"github.com/containers/podman/v2/pkg/specgen"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (ic *ContainerEngine) ContainerRunlabel(ctx context.Context, label string, image string, args []string, options entities.ContainerRunlabelOptions) error {
	return errors.New("not implemented")
}

func (ic *ContainerEngine) ContainerExists(ctx context.Context, nameOrID string) (*entities.BoolReport, error) {
	exists, err := containers.Exists(ic.ClientCxt, nameOrID)
	return &entities.BoolReport{Value: exists}, err
}

func (ic *ContainerEngine) ContainerWait(ctx context.Context, namesOrIds []string, options entities.WaitOptions) ([]entities.WaitReport, error) {
	cons, err := getContainersByContext(ic.ClientCxt, false, namesOrIds)
	if err != nil {
		return nil, err
	}
	responses := make([]entities.WaitReport, 0, len(cons))
	for _, c := range cons {
		response := entities.WaitReport{Id: c.ID}
		exitCode, err := containers.Wait(ic.ClientCxt, c.ID, &options.Condition)
		if err != nil {
			response.Error = err
		} else {
			response.ExitCode = exitCode
		}
		responses = append(responses, response)
	}
	return responses, nil
}

func (ic *ContainerEngine) ContainerPause(ctx context.Context, namesOrIds []string, options entities.PauseUnPauseOptions) ([]*entities.PauseUnpauseReport, error) {
	ctrs, err := getContainersByContext(ic.ClientCxt, options.All, namesOrIds)
	if err != nil {
		return nil, err
	}
	reports := make([]*entities.PauseUnpauseReport, 0, len(ctrs))
	for _, c := range ctrs {
		err := containers.Pause(ic.ClientCxt, c.ID)
		reports = append(reports, &entities.PauseUnpauseReport{Id: c.ID, Err: err})
	}
	return reports, nil
}

func (ic *ContainerEngine) ContainerUnpause(ctx context.Context, namesOrIds []string, options entities.PauseUnPauseOptions) ([]*entities.PauseUnpauseReport, error) {
	ctrs, err := getContainersByContext(ic.ClientCxt, options.All, namesOrIds)
	if err != nil {
		return nil, err
	}
	reports := make([]*entities.PauseUnpauseReport, 0, len(ctrs))
	for _, c := range ctrs {
		err := containers.Unpause(ic.ClientCxt, c.ID)
		reports = append(reports, &entities.PauseUnpauseReport{Id: c.ID, Err: err})
	}
	return reports, nil
}

func (ic *ContainerEngine) ContainerStop(ctx context.Context, namesOrIds []string, options entities.StopOptions) ([]*entities.StopReport, error) {
	reports := []*entities.StopReport{}
	for _, cidFile := range options.CIDFiles {
		content, err := ioutil.ReadFile(cidFile)
		if err != nil {
			return nil, errors.Wrapf(err, "error reading CIDFile %s", cidFile)
		}
		id := strings.Split(string(content), "\n")[0]
		namesOrIds = append(namesOrIds, id)
	}
	ctrs, err := getContainersByContext(ic.ClientCxt, options.All, namesOrIds)
	if err != nil && !(options.Ignore && errors.Cause(err) == define.ErrNoSuchCtr) {
		return nil, err
	}
	for _, c := range ctrs {
		report := entities.StopReport{Id: c.ID}
		if err = containers.Stop(ic.ClientCxt, c.ID, options.Timeout); err != nil {
			// These first two are considered non-fatal under the right conditions
			if errors.Cause(err).Error() == define.ErrCtrStopped.Error() {
				logrus.Debugf("Container %s is already stopped", c.ID)
				reports = append(reports, &report)
				continue
			} else if options.All && errors.Cause(err).Error() == define.ErrCtrStateInvalid.Error() {
				logrus.Debugf("Container %s is not running, could not stop", c.ID)
				reports = append(reports, &report)
				continue
			}

			// TODO we need to associate errors returned by http with common
			// define.errors so that we can equity tests. this will allow output
			// to be the same as the native client
			report.Err = err
			reports = append(reports, &report)
			continue
		}
		reports = append(reports, &report)
	}
	return reports, nil
}

func (ic *ContainerEngine) ContainerKill(ctx context.Context, namesOrIds []string, options entities.KillOptions) ([]*entities.KillReport, error) {
	ctrs, err := getContainersByContext(ic.ClientCxt, options.All, namesOrIds)
	if err != nil {
		return nil, err
	}
	reports := make([]*entities.KillReport, 0, len(ctrs))
	for _, c := range ctrs {
		reports = append(reports, &entities.KillReport{
			Id:  c.ID,
			Err: containers.Kill(ic.ClientCxt, c.ID, options.Signal),
		})
	}
	return reports, nil
}

func (ic *ContainerEngine) ContainerRestart(ctx context.Context, namesOrIds []string, options entities.RestartOptions) ([]*entities.RestartReport, error) {
	var (
		reports = []*entities.RestartReport{}
		timeout *int
	)
	if options.Timeout != nil {
		t := int(*options.Timeout)
		timeout = &t
	}

	ctrs, err := getContainersByContext(ic.ClientCxt, options.All, namesOrIds)
	if err != nil {
		return nil, err
	}
	for _, c := range ctrs {
		if options.Running && c.State != define.ContainerStateRunning.String() {
			continue
		}
		reports = append(reports, &entities.RestartReport{
			Id:  c.ID,
			Err: containers.Restart(ic.ClientCxt, c.ID, timeout),
		})
	}
	return reports, nil
}

func (ic *ContainerEngine) ContainerRm(ctx context.Context, namesOrIds []string, options entities.RmOptions) ([]*entities.RmReport, error) {
	for _, cidFile := range options.CIDFiles {
		content, err := ioutil.ReadFile(cidFile)
		if err != nil {
			return nil, errors.Wrapf(err, "error reading CIDFile %s", cidFile)
		}
		id := strings.Split(string(content), "\n")[0]
		namesOrIds = append(namesOrIds, id)
	}
	ctrs, err := getContainersByContext(ic.ClientCxt, options.All, namesOrIds)
	if err != nil && !(options.Ignore && errors.Cause(err) == define.ErrNoSuchCtr) {
		return nil, err
	}
	// TODO there is no endpoint for container eviction.  Need to discuss
	reports := make([]*entities.RmReport, 0, len(ctrs))
	for _, c := range ctrs {
		reports = append(reports, &entities.RmReport{
			Id:  c.ID,
			Err: containers.Remove(ic.ClientCxt, c.ID, &options.Force, &options.Volumes),
		})
	}
	return reports, nil
}

func (ic *ContainerEngine) ContainerPrune(ctx context.Context, options entities.ContainerPruneOptions) (*entities.ContainerPruneReport, error) {
	return containers.Prune(ic.ClientCxt, options.Filters)
}

func (ic *ContainerEngine) ContainerInspect(ctx context.Context, namesOrIds []string, options entities.InspectOptions) ([]*entities.ContainerInspectReport, []error, error) {
	var (
		reports = make([]*entities.ContainerInspectReport, 0, len(namesOrIds))
		errs    = []error{}
	)
	for _, name := range namesOrIds {
		inspect, err := containers.Inspect(ic.ClientCxt, name, &options.Size)
		if err != nil {
			errModel, ok := err.(entities.ErrorModel)
			if !ok {
				return nil, nil, err
			}
			if errModel.ResponseCode == 404 {
				errs = append(errs, errors.Errorf("no such container %q", name))
				continue
			}
			return nil, nil, err
		}
		reports = append(reports, &entities.ContainerInspectReport{InspectContainerData: inspect})
	}
	return reports, errs, nil
}

func (ic *ContainerEngine) ContainerTop(ctx context.Context, options entities.TopOptions) (*entities.StringSliceReport, error) {
	switch {
	case options.Latest:
		return nil, errors.New("latest is not supported")
	case options.NameOrID == "":
		return nil, errors.New("NameOrID must be specified")
	}

	topOutput, err := containers.Top(ic.ClientCxt, options.NameOrID, options.Descriptors)
	if err != nil {
		return nil, err
	}
	return &entities.StringSliceReport{Value: topOutput}, nil
}

func (ic *ContainerEngine) ContainerCommit(ctx context.Context, nameOrID string, options entities.CommitOptions) (*entities.CommitReport, error) {
	var (
		repo string
		tag  = "latest"
	)
	if len(options.ImageName) > 0 {
		ref, err := reference.Parse(options.ImageName)
		if err != nil {
			return nil, err
		}
		if t, ok := ref.(reference.Tagged); ok {
			tag = t.Tag()
		}
		if r, ok := ref.(reference.Named); ok {
			repo = r.Name()
		}
		if len(repo) < 1 {
			return nil, errors.Errorf("invalid image name %q", options.ImageName)
		}
	}
	commitOpts := containers.CommitOptions{
		Author:  &options.Author,
		Changes: options.Changes,
		Comment: &options.Message,
		Format:  &options.Format,
		Pause:   &options.Pause,
		Repo:    &repo,
		Tag:     &tag,
	}
	response, err := containers.Commit(ic.ClientCxt, nameOrID, commitOpts)
	if err != nil {
		return nil, err
	}
	return &entities.CommitReport{Id: response.ID}, nil
}

func (ic *ContainerEngine) ContainerExport(ctx context.Context, nameOrID string, options entities.ContainerExportOptions) error {
	var (
		err error
		w   io.Writer
	)
	if len(options.Output) > 0 {
		w, err = os.Create(options.Output)
		if err != nil {
			return err
		}
	}
	return containers.Export(ic.ClientCxt, nameOrID, w)
}

func (ic *ContainerEngine) ContainerCheckpoint(ctx context.Context, namesOrIds []string, options entities.CheckpointOptions) ([]*entities.CheckpointReport, error) {
	var (
		err  error
		ctrs = []entities.ListContainer{}
	)

	if options.All {
		allCtrs, err := getContainersByContext(ic.ClientCxt, true, []string{})
		if err != nil {
			return nil, err
		}
		// narrow the list to running only
		for _, c := range allCtrs {
			if c.State == define.ContainerStateRunning.String() {
				ctrs = append(ctrs, c)
			}
		}

	} else {
		ctrs, err = getContainersByContext(ic.ClientCxt, false, namesOrIds)
		if err != nil {
			return nil, err
		}
	}
	reports := make([]*entities.CheckpointReport, 0, len(ctrs))
	for _, c := range ctrs {
		report, err := containers.Checkpoint(ic.ClientCxt, c.ID, &options.Keep, &options.LeaveRunning, &options.TCPEstablished, &options.IgnoreRootFS, &options.Export)
		if err != nil {
			reports = append(reports, &entities.CheckpointReport{Id: c.ID, Err: err})
		}
		reports = append(reports, report)
	}
	return reports, nil
}

func (ic *ContainerEngine) ContainerRestore(ctx context.Context, namesOrIds []string, options entities.RestoreOptions) ([]*entities.RestoreReport, error) {
	var (
		err  error
		ctrs = []entities.ListContainer{}
	)
	if options.All {
		allCtrs, err := getContainersByContext(ic.ClientCxt, true, []string{})
		if err != nil {
			return nil, err
		}
		// narrow the list to exited only
		for _, c := range allCtrs {
			if c.State == define.ContainerStateExited.String() {
				ctrs = append(ctrs, c)
			}
		}

	} else {
		ctrs, err = getContainersByContext(ic.ClientCxt, false, namesOrIds)
		if err != nil {
			return nil, err
		}
	}
	reports := make([]*entities.RestoreReport, 0, len(ctrs))
	for _, c := range ctrs {
		report, err := containers.Restore(ic.ClientCxt, c.ID, &options.Keep, &options.TCPEstablished, &options.IgnoreRootFS, &options.IgnoreStaticIP, &options.IgnoreStaticMAC, &options.Name, &options.Import)
		if err != nil {
			reports = append(reports, &entities.RestoreReport{Id: c.ID, Err: err})
		}
		reports = append(reports, report)
	}
	return reports, nil
}

func (ic *ContainerEngine) ContainerCreate(ctx context.Context, s *specgen.SpecGenerator) (*entities.ContainerCreateReport, error) {
	response, err := containers.CreateWithSpec(ic.ClientCxt, s)
	if err != nil {
		return nil, err
	}
	for _, w := range response.Warnings {
		fmt.Fprintf(os.Stderr, "%s\n", w)
	}
	return &entities.ContainerCreateReport{Id: response.ID}, nil
}

func (ic *ContainerEngine) ContainerLogs(_ context.Context, nameOrIDs []string, options entities.ContainerLogsOptions) error {
	since := options.Since.Format(time.RFC3339)
	tail := strconv.FormatInt(options.Tail, 10)
	stdout := options.Writer != nil
	opts := containers.LogOptions{
		Follow:     &options.Follow,
		Since:      &since,
		Stderr:     &stdout,
		Stdout:     &stdout,
		Tail:       &tail,
		Timestamps: &options.Timestamps,
		Until:      nil,
	}

	var err error
	outCh := make(chan string)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		err = containers.Logs(ic.ClientCxt, nameOrIDs[0], opts, outCh, outCh)
		cancel()
	}()

	for {
		select {
		case <-ctx.Done():
			return err
		case line := <-outCh:
			_, _ = io.WriteString(options.Writer, line+"\n")
		}
	}
}

func (ic *ContainerEngine) ContainerAttach(ctx context.Context, nameOrID string, options entities.AttachOptions) error {
	return containers.Attach(ic.ClientCxt, nameOrID, &options.DetachKeys, nil, bindings.PTrue, options.Stdin, options.Stdout, options.Stderr, nil)
}

func makeExecConfig(options entities.ExecOptions) *handlers.ExecCreateConfig {
	env := []string{}
	for k, v := range options.Envs {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	createConfig := new(handlers.ExecCreateConfig)
	createConfig.User = options.User
	createConfig.Privileged = options.Privileged
	createConfig.Tty = options.Tty
	createConfig.AttachStdin = options.Interactive
	createConfig.AttachStdout = true
	createConfig.AttachStderr = true
	createConfig.Detach = false
	createConfig.DetachKeys = options.DetachKeys
	createConfig.Env = env
	createConfig.WorkingDir = options.WorkDir
	createConfig.Cmd = options.Cmd

	return createConfig
}

func (ic *ContainerEngine) ContainerExec(ctx context.Context, nameOrID string, options entities.ExecOptions, streams define.AttachStreams) (int, error) {
	createConfig := makeExecConfig(options)

	sessionID, err := containers.ExecCreate(ic.ClientCxt, nameOrID, createConfig)
	if err != nil {
		return 125, err
	}

	if err := containers.ExecStartAndAttach(ic.ClientCxt, sessionID, &streams); err != nil {
		return 125, err
	}

	inspectOut, err := containers.ExecInspect(ic.ClientCxt, sessionID)
	if err != nil {
		return 125, err
	}

	return inspectOut.ExitCode, nil
}

func (ic *ContainerEngine) ContainerExecDetached(ctx context.Context, nameOrID string, options entities.ExecOptions) (string, error) {
	createConfig := makeExecConfig(options)

	sessionID, err := containers.ExecCreate(ic.ClientCxt, nameOrID, createConfig)
	if err != nil {
		return "", err
	}

	if err := containers.ExecStart(ic.ClientCxt, sessionID); err != nil {
		return "", err
	}

	return sessionID, nil
}

func startAndAttach(ic *ContainerEngine, name string, detachKeys *string, input, output, errput *os.File) error { //nolint
	attachErr := make(chan error)
	attachReady := make(chan bool)
	go func() {
		err := containers.Attach(ic.ClientCxt, name, detachKeys, bindings.PFalse, bindings.PTrue, input, output, errput, attachReady)
		attachErr <- err
	}()
	// Wait for the attach to actually happen before starting
	// the container.
	select {
	case <-attachReady:
		if err := containers.Start(ic.ClientCxt, name, detachKeys); err != nil {
			return err
		}
	case err := <-attachErr:
		return err
	}
	// If attachReady happens first, wait for containers.Attach to complete
	return <-attachErr
}

func (ic *ContainerEngine) ContainerStart(ctx context.Context, namesOrIds []string, options entities.ContainerStartOptions) ([]*entities.ContainerStartReport, error) {
	reports := []*entities.ContainerStartReport{}
	for _, name := range namesOrIds {
		report := entities.ContainerStartReport{
			Id:       name,
			RawInput: name,
			ExitCode: 125,
		}
		if options.Attach {
			report.Err = startAndAttach(ic, name, &options.DetachKeys, options.Stdin, options.Stdout, options.Stderr)
			if report.Err == nil {
				exitCode, err := containers.Wait(ic.ClientCxt, name, nil)
				if err == nil {
					report.ExitCode = int(exitCode)
				}
			} else {
				report.ExitCode = define.ExitCode(report.Err)
			}
			reports = append(reports, &report)
			return reports, nil
		}
		report.Err = containers.Start(ic.ClientCxt, name, &options.DetachKeys)
		report.ExitCode = define.ExitCode(report.Err)
		reports = append(reports, &report)
	}
	return reports, nil
}

func (ic *ContainerEngine) ContainerList(ctx context.Context, options entities.ContainerListOptions) ([]entities.ListContainer, error) {
	return containers.List(ic.ClientCxt, options.Filters, &options.All, &options.Last, &options.Pod, &options.Size, &options.Sync)
}

func (ic *ContainerEngine) ContainerRun(ctx context.Context, opts entities.ContainerRunOptions) (*entities.ContainerRunReport, error) {
	if opts.Rm {
		logrus.Info("the remote client does not support --rm yet")
	}
	con, err := containers.CreateWithSpec(ic.ClientCxt, opts.Spec)
	if err != nil {
		return nil, err
	}
	for _, w := range con.Warnings {
		fmt.Fprintf(os.Stderr, "%s\n", w)
	}
	report := entities.ContainerRunReport{Id: con.ID}
	// Attach
	if !opts.Detach {
		err = startAndAttach(ic, con.ID, &opts.DetachKeys, opts.InputStream, opts.OutputStream, opts.ErrorStream)
		if err == nil {
			exitCode, err := containers.Wait(ic.ClientCxt, con.ID, nil)
			if err == nil {
				report.ExitCode = int(exitCode)
			}
		}
	} else {
		err = containers.Start(ic.ClientCxt, con.ID, nil)
	}
	if err != nil {
		report.ExitCode = define.ExitCode(err)
	}
	return &report, err
}

func (ic *ContainerEngine) ContainerDiff(ctx context.Context, nameOrID string, _ entities.DiffOptions) (*entities.DiffReport, error) {
	changes, err := containers.Diff(ic.ClientCxt, nameOrID)
	return &entities.DiffReport{Changes: changes}, err
}

func (ic *ContainerEngine) ContainerCleanup(ctx context.Context, namesOrIds []string, options entities.ContainerCleanupOptions) ([]*entities.ContainerCleanupReport, error) {
	return nil, errors.New("not implemented")
}

func (ic *ContainerEngine) ContainerInit(ctx context.Context, namesOrIds []string, options entities.ContainerInitOptions) ([]*entities.ContainerInitReport, error) {
	ctrs, err := getContainersByContext(ic.ClientCxt, options.All, namesOrIds)
	if err != nil {
		return nil, err
	}
	reports := make([]*entities.ContainerInitReport, 0, len(ctrs))
	for _, ctr := range ctrs {
		err := containers.ContainerInit(ic.ClientCxt, ctr.ID)
		// When using all, it is NOT considered an error if a container
		// has already been init'd.
		if err != nil && options.All && strings.Contains(errors.Cause(err).Error(), define.ErrCtrStateInvalid.Error()) {
			err = nil
		}
		reports = append(reports, &entities.ContainerInitReport{
			Err: err,
			Id:  ctr.ID,
		})
	}
	return reports, nil
}

func (ic *ContainerEngine) ContainerMount(ctx context.Context, nameOrIDs []string, options entities.ContainerMountOptions) ([]*entities.ContainerMountReport, error) {
	return nil, errors.New("mounting containers is not supported for remote clients")
}

func (ic *ContainerEngine) ContainerUnmount(ctx context.Context, nameOrIDs []string, options entities.ContainerUnmountOptions) ([]*entities.ContainerUnmountReport, error) {
	return nil, errors.New("unmounting containers is not supported for remote clients")
}

func (ic *ContainerEngine) Config(_ context.Context) (*config.Config, error) {
	return config.Default()
}

func (ic *ContainerEngine) ContainerPort(ctx context.Context, nameOrID string, options entities.ContainerPortOptions) ([]*entities.ContainerPortReport, error) {
	var (
		reports    = []*entities.ContainerPortReport{}
		namesOrIds = []string{}
	)
	if len(nameOrID) > 0 {
		namesOrIds = append(namesOrIds, nameOrID)
	}
	ctrs, err := getContainersByContext(ic.ClientCxt, options.All, namesOrIds)
	if err != nil {
		return nil, err
	}
	for _, con := range ctrs {
		if con.State != define.ContainerStateRunning.String() {
			continue
		}
		if len(con.Ports) > 0 {
			reports = append(reports, &entities.ContainerPortReport{
				Id:    con.ID,
				Ports: con.Ports,
			})
		}
	}
	return reports, nil
}

func (ic *ContainerEngine) ContainerCp(ctx context.Context, source, dest string, options entities.ContainerCpOptions) (*entities.ContainerCpReport, error) {
	return nil, errors.New("not implemented")
}

// Shutdown Libpod engine
func (ic *ContainerEngine) Shutdown(_ context.Context) {
}

func (ic *ContainerEngine) ContainerStats(ctx context.Context, namesOrIds []string, options entities.ContainerStatsOptions) error {
	return errors.New("not implemented")
}
