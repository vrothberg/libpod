package main

import (
	"os"
	"strings"
	"testing"
)

func TestMain(t *testing.T) {
	var (
		args []string
	)

	for _, arg := range os.Args {
		switch {
		case strings.HasPrefix(arg, "-test"):
			// Make sure we don't pass `go test` specific flags to
			// Podman.
		default:
			args = append(args, arg)
		}
	}

	main()
}
