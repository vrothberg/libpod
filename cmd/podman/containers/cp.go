package containers

import (
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	buildahCopiah "github.com/containers/buildah/copier"
	"github.com/containers/podman/v3/cmd/podman/common"
	"github.com/containers/podman/v3/cmd/podman/registry"
	"github.com/containers/podman/v3/pkg/copy"
	"github.com/containers/podman/v3/pkg/domain/entities"
	"github.com/containers/podman/v3/pkg/errorhandling"
	"github.com/containers/storage/pkg/archive"
	"github.com/containers/storage/pkg/idtools"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	cpDescription = `Copy the contents of SRC_PATH to the DEST_PATH.

  You can copy from the container's file system to the local machine or the reverse, from the local filesystem to the container. If "-" is specified for either the SRC_PATH or DEST_PATH, you can also stream a tar archive from STDIN or to STDOUT. The CONTAINER can be a running or stopped container. The SRC_PATH or DEST_PATH can be a file or a directory.
`
	cpCommand = &cobra.Command{
		Use:               "cp [CONTAINER:]SRC_PATH [CONTAINER:]DEST_PATH",
		Short:             "Copy files/folders between a container and the local filesystem",
		Long:              cpDescription,
		Args:              cobra.ExactArgs(2),
		RunE:              cp,
		ValidArgsFunction: common.AutocompleteCpCommand,
		Example:           "podman cp [CONTAINER:]SRC_PATH [CONTAINER:]DEST_PATH",
	}

	containerCpCommand = &cobra.Command{
		Use:               cpCommand.Use,
		Short:             cpCommand.Short,
		Long:              cpCommand.Long,
		Args:              cpCommand.Args,
		RunE:              cpCommand.RunE,
		ValidArgsFunction: cpCommand.ValidArgsFunction,
		Example:           "podman container cp [CONTAINER:]SRC_PATH [CONTAINER:]DEST_PATH",
	}
)

var (
	cpOpts entities.ContainerCpOptions
)

func cpFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.BoolVar(&cpOpts.Extract, "extract", false, "Deprecated...")
	flags.BoolVar(&cpOpts.Pause, "pause", true, "Deprecated")
	_ = flags.MarkHidden("extract")
	_ = flags.MarkHidden("pause")
}

func init() {
	registry.Commands = append(registry.Commands, registry.CliCommand{
		Mode:    []entities.EngineMode{entities.ABIMode, entities.TunnelMode},
		Command: cpCommand,
	})
	cpFlags(cpCommand)

	registry.Commands = append(registry.Commands, registry.CliCommand{
		Mode:    []entities.EngineMode{entities.ABIMode, entities.TunnelMode},
		Command: containerCpCommand,
		Parent:  containerCmd,
	})
	cpFlags(containerCpCommand)
}

func cp(cmd *cobra.Command, args []string) error {
	// Parse user input.
	sourceContainerStr, sourcePath, destContainerStr, destPath, err := copy.ParseSourceAndDestination(args[0], args[1])
	if err != nil {
		return err
	}

	if len(sourceContainerStr) > 0 {
		return copyFromContainer(sourceContainerStr, sourcePath, destPath)
	}

	return copyToContainer(destContainerStr, destPath, sourcePath)
}

// containerMustExist returns an error if the specified container does not
// exist.
func containerMustExist(container string) error {
	exists, err := registry.ContainerEngine().ContainerExists(registry.GetContext(), container, entities.ContainerExistsOptions{})
	if err != nil {
		return err
	}
	if !exists.Value {
		return errors.Errorf("container %q does not exist", container)
	}
	return nil
}

// doCopy executes the two functions in parallel to copy data from A to B and
// joins the errors if any.
func doCopy(funcA func() error, funcB func() error) error {
	errChan := make(chan error)
	go func() {
		errChan <- funcA()
	}()
	var copyErrors []error
	copyErrors = append(copyErrors, funcB())
	copyErrors = append(copyErrors, <-errChan)
	return errorhandling.JoinErrors(copyErrors)
}

// copyFromContainer copies from the containerPath on the container to hostPath.
func copyFromContainer(container string, containerPath string, hostPath string) error {
	if err := containerMustExist(container); err != nil {
		return err
	}

	isStdout := false
	if hostPath == "-" {
		isStdout = true
		hostPath = os.Stdout.Name()
	}

	containerInfo, err := registry.ContainerEngine().ContainerStat(registry.GetContext(), container, containerPath)
	if err != nil {
		return errors.Wrapf(err, "%q could not be found on container %s", containerPath, container)
	}

	var hostBaseName string
	hostInfo, hostInfoErr := copy.ResolveHostPath(hostPath)
	if hostInfoErr != nil {
		if strings.HasSuffix(hostPath, "/") {
			return errors.Wrapf(hostInfoErr, "%q could not be found on the host", hostPath)
		}
		// If it doesn't exist, then let's have a look at the parent dir.
		parentDir := filepath.Dir(hostPath)
		hostInfo, err = copy.ResolveHostPath(parentDir)
		if err != nil {
			return errors.Wrapf(hostInfoErr, "%q could not be found on the host", hostPath)
		}
		// If the specified path does not exist, we need to assume that
		// it'll be created while copying.  Hence, we use it as the
		// base path.
		hostBaseName = filepath.Base(hostPath)
	} else {
		// If the specified path exists on the host, we must use its
		// base path as it may have changed due to symlink evaluations.
		hostBaseName = filepath.Base(hostInfo.LinkTarget)
	}

	if !isStdout {
		if err := validateFileInfo(hostInfo); err != nil {
			return errors.Wrap(err, "invalid destination")
		}
	}

	reader, writer := io.Pipe()
	hostCopy := func() error {
		defer reader.Close()
		if isStdout {
			_, err := io.Copy(os.Stdout, reader)
			return err
		}

		groot, err := user.Current()
		if err != nil {
			return err
		}

		// Set the {G,U}ID.  Let's be tolerant towards the different
		// operating systems and only log the errors, so we can debug
		// if necessary.
		idPair := idtools.IDPair{}
		if i, err := strconv.Atoi(groot.Uid); err == nil {
			idPair.UID = i
		} else {
			logrus.Debugf("Error converting UID %q to int: %v", groot.Uid, err)
		}
		if i, err := strconv.Atoi(groot.Gid); err == nil {
			idPair.GID = i
		} else {
			logrus.Debugf("Error converting GID %q to int: %v", groot.Gid, err)
		}

		putOptions := buildahCopiah.PutOptions{
			ChownDirs:     &idPair,
			ChownFiles:    &idPair,
			IgnoreDevices: true,
		}
		if (!containerInfo.IsDir && !hostInfo.IsDir) || hostInfoErr != nil {
			// If we're having a file-to-file copy, make sure to
			// rename accordingly.
			putOptions.Rename = map[string]string{filepath.Base(containerInfo.LinkTarget): hostBaseName}
		}
		dir := hostInfo.LinkTarget
		if !hostInfo.IsDir {
			dir = filepath.Dir(dir)
		}
		if err := buildahCopiah.Put(dir, "", putOptions, reader); err != nil {
			return errors.Wrap(err, "error copying to host")
		}
		return nil
	}

	containerCopy := func() error {
		defer writer.Close()
		copyFunc, err := registry.ContainerEngine().ContainerCopyToArchive(registry.GetContext(), container, containerInfo.LinkTarget, writer)
		if err != nil {
			return err
		}
		if err := copyFunc(); err != nil {
			return errors.Wrap(err, "error copying from container")
		}
		return nil
	}
	return doCopy(containerCopy, hostCopy)
}

// copyToContainer copies the hostPath to containerPath on the container.
func copyToContainer(container string, containerPath string, hostPath string) error {
	if err := containerMustExist(container); err != nil {
		return err
	}

	isStdin := false
	if hostPath == "-" {
		hostPath = os.Stdin.Name()
		isStdin = true
	}

	// Make sure that host path exists.
	hostInfo, err := copy.ResolveHostPath(hostPath)
	if err != nil {
		return errors.Wrapf(err, "%q could not be found on the host", hostPath)
	}

	// If the path on the container does not exist.  We need to make sure
	// that it's parent directory exists.  The destination may be created
	// while copying.
	var containerBaseName string
	containerInfo, containerInfoErr := registry.ContainerEngine().ContainerStat(registry.GetContext(), container, containerPath)
	if containerInfoErr != nil {
		if strings.HasSuffix(containerPath, "/") {
			return errors.Wrapf(containerInfoErr, "%q could not be found on container %s", containerPath, container)
		}
		if isStdin {
			return errors.New("destination must be a directory when copying from stdin")
		}
		// NOTE: containerInfo may actually be set.  That happens when
		// the container path is a symlink into nirvana.  In that case,
		// we must use the symlinked path instead.
		path := containerPath
		if containerInfo != nil {
			containerBaseName = filepath.Base(containerInfo.LinkTarget)
			path = containerInfo.LinkTarget
		} else {
			containerBaseName = filepath.Base(containerPath)
		}

		parentDir, err := containerParentDir(container, path)
		if err != nil {
			return errors.Wrapf(err, "could not determine parent dir of %q on container %s", path, container)
		}
		containerInfo, err = registry.ContainerEngine().ContainerStat(registry.GetContext(), container, parentDir)
		if err != nil {
			return errors.Wrapf(err, "%q could not be found on container %s", containerPath, container)
		}
	} else {
		// If the specified path exists on the container, we must use
		// its base path as it may have changed due to symlink
		// evaluations.
		containerBaseName = filepath.Base(containerInfo.LinkTarget)
	}

	var stdinFile string
	if isStdin {
		if !containerInfo.IsDir {
			return errors.New("destination must be a directory when copying from stdin")
		}

		// Copy from stdin to a temporary file *before* throwing it
		// over the wire.  This allows for proper client-side error
		// reporting.
		tmpFile, err := ioutil.TempFile("", "")
		if err != nil {
			return err
		}
		_, err = io.Copy(tmpFile, os.Stdin)
		if err != nil {
			return err
		}
		if err = tmpFile.Close(); err != nil {
			return err
		}
		if !archive.IsArchivePath(tmpFile.Name()) {
			return errors.New("source must be a (compressed) tar archive when copying from stdin")
		}
		stdinFile = tmpFile.Name()
	}

	reader, writer := io.Pipe()
	hostCopy := func() error {
		defer writer.Close()
		if isStdin {
			stream, err := os.Open(stdinFile)
			if err != nil {
				return err
			}
			defer stream.Close()
			_, err = io.Copy(writer, stream)
			return err
		}

		getOptions := buildahCopiah.GetOptions{
			// Unless the specified path points to ".", we want to copy the base directory.
			KeepDirectoryNames: hostInfo.IsDir && filepath.Base(hostPath) != ".",
		}
		if (!hostInfo.IsDir && !containerInfo.IsDir) || containerInfoErr != nil {
			// If we're having a file-to-file copy, make sure to
			// rename accordingly.
			getOptions.Rename = map[string]string{filepath.Base(hostInfo.LinkTarget): containerBaseName}
		}
		if err := buildahCopiah.Get("/", "", getOptions, []string{hostInfo.LinkTarget}, writer); err != nil {
			return errors.Wrap(err, "error copying from host")
		}
		return nil
	}

	containerCopy := func() error {
		defer reader.Close()
		target := containerInfo.FileInfo.LinkTarget
		if !containerInfo.IsDir {
			target = filepath.Dir(target)
		}

		copyFunc, err := registry.ContainerEngine().ContainerCopyFromArchive(registry.GetContext(), container, target, reader)
		if err != nil {
			return err
		}
		if err := copyFunc(); err != nil {
			return errors.Wrap(err, "error copying to container")
		}
		return nil
	}

	return doCopy(hostCopy, containerCopy)
}

// containerParentDir returns the parent directory of the specified path on the
// container.  If the path is relative, it will be resolved relative to the
// container's working directory (or "/" if the work dir isn't set).
func containerParentDir(container string, containerPath string) (string, error) {
	if filepath.IsAbs(containerPath) {
		return filepath.Dir(containerPath), nil
	}
	inspectData, _, err := registry.ContainerEngine().ContainerInspect(registry.GetContext(), []string{container}, entities.InspectOptions{})
	if err != nil {
		return "", err
	}
	if len(inspectData) != 1 {
		return "", errors.Errorf("inspecting container %q: expected 1 data item but got %d", container, len(inspectData))
	}
	workDir := filepath.Join("/", inspectData[0].Config.WorkingDir)
	workDir = filepath.Join(workDir, containerPath)
	return filepath.Dir(workDir), nil
}

// validateFileInfo returns an error if the specified FileInfo doesn't point to
// a directory or a regular file.
func validateFileInfo(info *copy.FileInfo) error {
	if info.Mode.IsDir() || info.Mode.IsRegular() {
		return nil
	}
	return errors.Errorf("%q must be a directory or a regular file", info.LinkTarget)
}
