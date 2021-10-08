package registry

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"

	"github.com/containers/podman/v3/pkg/domain/entities"
	"github.com/containers/podman/v3/pkg/domain/infra"
	"github.com/containers/podman/v3/pkg/parallel"
	"github.com/containers/podman/v3/pkg/rootless"
	"github.com/containers/podman/v3/pkg/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// DefaultRootAPIPath is the default path of the REST socket
const DefaultRootAPIPath = "/run/podman/podman.sock"

// DefaultRootAPIAddress is the default address of the REST socket with unix: prefix
const DefaultRootAPIAddress = "unix:" + DefaultRootAPIPath

type CliCommand struct {
	Command *cobra.Command
	Parent  *cobra.Command
}

var (
	cliCtx          context.Context
	containerEngine entities.ContainerEngine
	exitCode        = 0
	imageEngine     entities.ImageEngine

	// Commands holds the cobra.Commands to present to the user, including
	// parent if not a child of "root"
	Commands []CliCommand
)

func SetExitCode(code int) {
	exitCode = code
}

func GetExitCode() int {
	return exitCode
}

func ImageEngine() entities.ImageEngine {
	return imageEngine
}

// NewImageEngine is a wrapper for building an ImageEngine to be used for PreRunE functions
func NewImageEngine(cmd *cobra.Command, args []string) (entities.ImageEngine, error) {
	if imageEngine == nil {
		podmanOptions.FlagSet = cmd.Flags()
		engine, err := infra.NewImageEngine(&podmanOptions)
		if err != nil {
			return nil, err
		}
		imageEngine = engine
	}
	return imageEngine, nil
}

func ContainerEngine() entities.ContainerEngine {
	return containerEngine
}

// NewContainerEngine is a wrapper for building an ContainerEngine to be used for PreRunE functions
func NewContainerEngine(cmd *cobra.Command, args []string) (entities.ContainerEngine, error) {
	if containerEngine == nil {
		podmanOptions.FlagSet = cmd.Flags()
		engine, err := infra.NewContainerEngine(&podmanOptions)
		if err != nil {
			return nil, err
		}
		containerEngine = engine
	}
	return containerEngine, nil
}

type PodmanOptionsKey struct{}

func Context() context.Context {
	if cliCtx == nil {
		cliCtx = ContextWithOptions(context.Background())
	}
	return cliCtx
}

func ContextWithOptions(ctx context.Context) context.Context {
	cliCtx = context.WithValue(ctx, PodmanOptionsKey{}, podmanOptions)
	return cliCtx
}

// GetContextWithOptions deprecated, use  NewContextWithOptions()
func GetContextWithOptions() context.Context {
	return ContextWithOptions(context.Background())
}

// GetContext deprecated, use  Context()
func GetContext() context.Context {
	return Context()
}

func DefaultAPIAddress() string {
	if rootless.IsRootless() {
		xdg, err := util.GetRuntimeDir()
		if err != nil {
			logrus.Warnf("Failed to get rootless runtime dir for DefaultAPIAddress: %s", err)
			return DefaultRootAPIAddress
		}
		return "unix:" + filepath.Join(xdg, "podman", "podman.sock")
	}
	return DefaultRootAPIAddress
}

var requireCleanup bool

func PersistentPreRunE(cmd *cobra.Command, args []string) error {
	logrus.Debugf("Called %s.PersistentPreRunE(%s)", cmd.Name(), strings.Join(os.Args, " "))

	// Help, completion and commands with subcommands are special cases, no need for more setup
	// Completion cmd is used to generate the shell scripts
	if cmd.Name() == "help" || cmd.Name() == "completion" || cmd.HasSubCommands() {
		requireCleanup = false
		return nil
	}

	cfg := PodmanConfig()

	// --connection is not as "special" as --remote so we can wait and process it here
	conn := cmd.Root().LocalFlags().Lookup("connection")
	if conn != nil && conn.Changed {
		cfg.Engine.ActiveService = conn.Value.String()

		var err error
		cfg.URI, cfg.Identity, err = cfg.ActiveDestination()
		if err != nil {
			return errors.Wrap(err, "failed to resolve active destination")
		}

		if err := cmd.Root().LocalFlags().Set("url", cfg.URI); err != nil {
			return errors.Wrap(err, "failed to override --url flag")
		}

		if err := cmd.Root().LocalFlags().Set("identity", cfg.Identity); err != nil {
			return errors.Wrap(err, "failed to override --identity flag")
		}
	}

	// Special case if command is hidden completion command ("__complete","__completeNoDesc")
	// Since __completeNoDesc is an alias the cm.Name is always __complete
	if cmd.Name() == cobra.ShellCompRequestCmd {
		// Parse the cli arguments after the the completion cmd (always called as second argument)
		// This ensures that the --url, --identity and --connection flags are properly set
		compCmd, _, err := cmd.Root().Traverse(os.Args[2:])
		if err != nil {
			return err
		}
		// If we don't complete the root cmd hide all root flags
		// so they won't show up in the completions on subcommands.
		if compCmd != compCmd.Root() {
			compCmd.Root().Flags().VisitAll(func(flag *pflag.Flag) {
				flag.Hidden = true
			})
		}
		// No need for further setup the completion logic setups the engines as needed.
		requireCleanup = false
		return nil
	}

	// Prep the engines
	if _, err := NewImageEngine(cmd, args); err != nil {
		return err
	}
	if _, err := NewContainerEngine(cmd, args); err != nil {
		return err
	}

	for _, env := range cfg.Engine.Env {
		splitEnv := strings.SplitN(env, "=", 2)
		if len(splitEnv) != 2 {
			return fmt.Errorf("invalid environment variable for engine %s, valid configuration is KEY=value pair", env)
		}
		// skip if the env is already defined
		if _, ok := os.LookupEnv(splitEnv[0]); ok {
			logrus.Debugf("environment variable %s is already defined, skip the settings from containers.conf", splitEnv[0])
			continue
		}
		if err := os.Setenv(splitEnv[0], splitEnv[1]); err != nil {
			return err
		}
	}
	// Hard code TMPDIR functions to use /var/tmp, if user did not override
	if _, ok := os.LookupEnv("TMPDIR"); !ok {
		if tmpdir, err := cfg.ImageCopyTmpDir(); err != nil {
			logrus.Warnf("Failed to retrieve default tmp dir: %s", err.Error())
		} else {
			os.Setenv("TMPDIR", tmpdir)
		}
	}

	context := cmd.Root().LocalFlags().Lookup("context")
	if context.Value.String() != "default" {
		return errors.New("Podman does not support swarm, the only --context value allowed is \"default\"")
	}
	if !IsRemote() {
		if cmd.Flag("cpu-profile").Changed {
			f, err := os.Create(cfg.CPUProfile)
			if err != nil {
				return err
			}
			if err := pprof.StartCPUProfile(f); err != nil {
				return err
			}
		}
		if cmd.Flag("memory-profile").Changed {
			// Same value as the default in github.com/pkg/profile.
			runtime.MemProfileRate = 4096
			if rate := os.Getenv("MemProfileRate"); rate != "" {
				r, err := strconv.Atoi(rate)
				if err != nil {
					return err
				}
				runtime.MemProfileRate = r
			}
		}

		if cfg.MaxWorks <= 0 {
			return errors.Errorf("maximum workers must be set to a positive number (got %d)", cfg.MaxWorks)
		}
		if err := parallel.SetMaxThreads(uint(cfg.MaxWorks)); err != nil {
			return err
		}
	}
	// Setup Rootless environment, IFF:
	// 1) in ABI mode
	// 2) running as non-root
	// 3) command doesn't require Parent Namespace
	_, found := cmd.Annotations[ParentNSRequired]
	if !IsRemote() && rootless.IsRootless() && !found {
		_, noMoveProcess := cmd.Annotations[NoMoveProcess]
		err := ContainerEngine().SetupRootless(Context(), noMoveProcess)
		if err != nil {
			return err
		}
	}
	return nil
}

func PersistentPostRunE(cmd *cobra.Command, args []string) error {
	logrus.Debugf("Called %s.PersistentPostRunE(%s)", cmd.Name(), strings.Join(os.Args, " "))

	if !requireCleanup {
		return nil
	}

	ImageEngine().Shutdown(Context())
	ContainerEngine().Shutdown(Context())

	if IsRemote() {
		return nil
	}

	// CPU and memory profiling.
	if cmd.Flag("cpu-profile").Changed {
		pprof.StopCPUProfile()
	}
	if cmd.Flag("memory-profile").Changed {
		f, err := os.Create(PodmanConfig().MemoryProfile)
		if err != nil {
			return errors.Wrap(err, "creating memory profile")
		}
		defer f.Close()
		runtime.GC() // get up-to-date GC statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			return errors.Wrap(err, "writing memory profile")
		}
	}

	return nil
}

