# Podman CLI

The following is an example of how to add a new primary command (`manifest`) and a sub-command (`inspect`) to the Podman CLI.
This is example code, the production code has additional error checking and the business logic provided.

See items below for details on building, installing, contributing to Podman:
 - [Readme](README.md)
 - [Contributing](CONTRIBUTING.md)
 - [Podman Usage](transfer.md)
 - [Trouble Shooting](troubleshooting.md)
 - [Code Of Conduct](CODE-OF-CONDUCT.md)

## Adding a new command `podman manifest`
```shell script
$ mkdir -p $GOPATH/src/github.com/containers/podman/cmd/podmanV2/manifests
```
Create the file ```$GOPATH/src/github.com/containers/podman/cmd/podmanV2/manifests/manifest.go```
```go
package manifests

import (
    "github.com/containers/podman/cmd/podman/registry"
    "github.com/containers/podman/cmd/podman/validate"
    "github.com/containers/podman/pkg/domain/entities"
    "github.com/spf13/cobra"
)

var (
    // podman _manifests_
    manifestCmd = &cobra.Command{
        Use:               "manifest",
        Short:             "Manage manifests",
        Args:              cobra.ExactArgs(1),
        Long:              "Manage manifests",
        Example:           "podman manifest IMAGE",
        TraverseChildren:  true,
        RunE:              validate.SubCommandExists, // Report error if there is no sub command given
    }
)
func init() {
    // Subscribe command to podman
    registry.Commands = append(registry.Commands, registry.CliCommand{
        // _podman manifest_ will support both ABIMode and TunnelMode
        Mode:    []entities.EngineMode{entities.ABIMode, entities.TunnelMode},
        // The definition for this command
        Command: manifestCmd,
    })
}
```
To "wire" in the `manifest` command, edit the file ```$GOPATH/src/github.com/containers/podman/cmd/podmanV2/main.go``` to add:
```go
package main

import	_ "github.com/containers/podman/cmd/podman/manifests"
```

## Adding a new sub command `podman manifests list`
Create the file ```$GOPATH/src/github.com/containers/podman/cmd/podmanV2/manifests/inspect.go```
```go
package manifests

import (
    "github.com/containers/podman/cmd/podman/registry"
    "github.com/containers/podman/pkg/domain/entities"
    "github.com/spf13/cobra"
)

var (
    // podman manifests _inspect_
    inspectCmd = &cobra.Command{
        Use:     "inspect IMAGE",
        Short:   "Display manifest from image",
        Long:    "Displays the low-level information on a manifest identified by image name or ID",
        RunE:    inspect,
        Annotations: map[string]string{
            // Add this annotation if this command cannot be run rootless
            // registry.ParentNSRequired: "",
        },
        Example: "podman manifest inspect DEADBEEF",
    }
)

func init() {
    // Subscribe inspect sub command to manifest command
    registry.Commands = append(registry.Commands, registry.CliCommand{
        // _podman manifest inspect_ will support both ABIMode and TunnelMode
        Mode:    []entities.EngineMode{entities.ABIMode, entities.TunnelMode},
        // The definition for this command
        Command: inspectCmd,
        // The parent command to proceed this command on the CLI
        Parent:  manifestCmd,
    })

    // This is where you would configure the cobra flags using inspectCmd.Flags()
}

// Business logic: cmd is inspectCmd, args is the positional arguments from os.Args
func inspect(cmd *cobra.Command, args []string) error {
    // Business logic using registry.ImageEngine()
    // Do not pull from libpod directly use the domain objects and types
    return nil
}
```

## Helper functions

The complete set can be found in the `validate` package, here are some examples:

 - `cobra.Command{ Args: validate.NoArgs }` used when the command does not accept errors
 - `cobra.Command{ Args: validate.IdOrLatestArgs }` used to ensure either a list of ids given or the --latest flag
 - `cobra.Command{ RunE: validate.SubCommandExists }` used to validate a subcommand given to a command
 - `validate.ChoiceValue` used to create a `pflag.Value` that validate user input against a provided slice of values. For example:
    ```go
    flags := cobraCommand.Flags()
    created := validate.ChoiceValue(&opts.Sort, "command", "created", "id", "image", "names", "runningfor", "size", "status")
    flags.Var(created, "sort", "Sort output by: "+created.Choices())
    ```
