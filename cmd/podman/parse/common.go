package parse

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// TODO: the two functions here are almost identical. It may be worth looking
// into generalizing the two a bit more and share code but time is scarce and
// we only live once.

// CheckAllLatestAndCIDFile checks that --all and --latest are used correctly.
// If cidfile is set, also check for the --cidfile flag.
func CheckAllLatestAndCIDFile(c *cobra.Command, args []string, ignoreArgLen bool, cidfile bool) error {
	argLen := len(args)
	if c.Flags().Lookup("all") == nil || c.Flags().Lookup("latest") == nil {
		if !cidfile {
			return errors.New("unable to lookup values for 'latest' or 'all'")
		} else if c.Flags().Lookup("cidfile") == nil {
			return errors.New("unable to lookup values for 'latest', 'all' or 'cidfile'")
		}
	}

	specifiedAll, _ := c.Flags().GetBool("all")
	specifiedLatest, _ := c.Flags().GetBool("latest")
	specifiedCIDFile := false
	if cid, _ := c.Flags().GetStringArray("cidfile"); len(cid) > 0 {
		specifiedCIDFile = true
	}

	if specifiedCIDFile && (specifiedAll || specifiedLatest) {
		return errors.Errorf("--all, --latest and --cidfile cannot be used together")
	} else if specifiedAll && specifiedLatest {
		return errors.Errorf("--all and --latest cannot be used together")
	}

	if (argLen > 0) && specifiedAll {
		return errors.Errorf("no arguments are needed with --all")
	}

	if ignoreArgLen {
		return nil
	}

	if argLen > 0 {
		if specifiedLatest {
			return errors.Errorf("no arguments are needed with --latest")
		} else if cidfile && (specifiedLatest || specifiedCIDFile) {
			return errors.Errorf("no arguments are needed with --latest or --cidfile")
		}
	}

	if specifiedCIDFile {
		return nil
	}

	if argLen < 1 && !specifiedAll && !specifiedLatest && !specifiedCIDFile {
		return errors.Errorf("you must provide at least one name or id")
	}
	return nil
}

// CheckAllLatestAndPodIDFile checks that --all and --latest are used correctly.
// If withIDFile is set, also check for the --pod-id-file flag.
func CheckAllLatestAndPodIDFile(c *cobra.Command, args []string, ignoreArgLen bool, withIDFile bool) error {
	argLen := len(args)
	if c.Flags().Lookup("all") == nil || c.Flags().Lookup("latest") == nil {
		if !withIDFile {
			return errors.New("unable to lookup values for 'latest' or 'all'")
		} else if c.Flags().Lookup("cidfile") == nil {
			return errors.New("unable to lookup values for 'latest', 'all' or 'pod-id-file'")
		}
	}

	specifiedAll, _ := c.Flags().GetBool("all")
	specifiedLatest, _ := c.Flags().GetBool("latest")
	specifiedPodIDFile := false
	if pid, _ := c.Flags().GetStringArray("pod-id-file"); len(pid) > 0 {
		specifiedPodIDFile = true
	}

	if specifiedPodIDFile && (specifiedAll || specifiedLatest) {
		return errors.Errorf("--all, --latest and --cidfile cannot be used together")
	} else if specifiedAll && specifiedLatest {
		return errors.Errorf("--all and --latest cannot be used together")
	}

	if (argLen > 0) && specifiedAll {
		return errors.Errorf("no arguments are needed with --all")
	}

	if ignoreArgLen {
		return nil
	}

	if argLen > 0 {
		if specifiedLatest {
			return errors.Errorf("no arguments are needed with --latest")
		} else if withIDFile && (specifiedLatest || specifiedPodIDFile) {
			return errors.Errorf("no arguments are needed with --latest or --pod-id-file")
		}
	}

	if specifiedPodIDFile {
		return nil
	}

	if argLen < 1 && !specifiedAll && !specifiedLatest && !specifiedPodIDFile {
		return errors.Errorf("you must provide at least one name or id")
	}
	return nil
}
