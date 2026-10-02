package cli

import (
	"fmt"
	"os"

	"github.com/ScriptonBasestar/dva/internal/config"
	dvaexec "github.com/ScriptonBasestar/dva/internal/exec"
)

// execComposeSubprocess runs a docker compose command as a subprocess,
// returning control to the caller after completion.
func execComposeSubprocess(e *config.Environment, c *config.Config, args []string) error {
	composeCmd, composeArgs, err := buildComposeArgs(e, c, args)
	if err != nil {
		return err
	}

	if dvaexec.Debug {
		fmt.Fprintf(os.Stderr, "[debug] compose subprocess: %s %v\n", composeCmd, composeArgs)
	}

	return dvaexec.ExecSubprocess(e, composeCmd, composeArgs, false)
}

// execComposePassthrough builds and execs a docker compose command using config.
// When forceSubprocess is true (set by hook wrapper for after-hooks), it delegates
// to execComposeSubprocess so the Go process survives for post-command hooks.
func execComposePassthrough(e *config.Environment, c *config.Config, args []string) error {
	if forceSubprocess {
		return execComposeSubprocess(e, c, args)
	}

	composeCmd, composeArgs, err := buildComposeArgs(e, c, args)
	if err != nil {
		return err
	}

	if dvaexec.Debug {
		fmt.Fprintf(os.Stderr, "[debug] compose: %s %v\n", composeCmd, composeArgs)
	}

	return dvaexec.ExecReplace(e, composeCmd, composeArgs, false)
}

// execComposePassthroughForEntry runs docker compose against a specific stack entry.
//
// profiles is nil on the stack path and the plan entry's profiles on the plan path; see
// buildComposeArgsForEntry for where they land in the argv.
func execComposePassthroughForEntry(e *config.Environment, c *config.Config, entry *config.LifecycleEntry, profiles, args []string) error {
	composeCmd, composeArgs, err := buildComposeArgsForEntry(e, c, entry, profiles, args)
	if err != nil {
		return err
	}

	if forceSubprocess {
		if dvaexec.Debug {
			fmt.Fprintf(os.Stderr, "[debug] compose subprocess [%s]: %s %v\n", entry.Name, composeCmd, composeArgs)
		}
		return dvaexec.ExecSubprocess(e, composeCmd, composeArgs, false)
	}

	if dvaexec.Debug {
		fmt.Fprintf(os.Stderr, "[debug] compose [%s]: %s %v\n", entry.Name, composeCmd, composeArgs)
	}
	return dvaexec.ExecReplace(e, composeCmd, composeArgs, false)
}

// buildComposeArgsForEntry builds docker compose arguments from a specific lifecycle entry.
//
// profiles are the plan entry's compose profiles and go on as `--profile` flags between the
// -f/--project-name prefix and the subcommand, which is the one position docker accepts:
// --profile is a top-level flag, so after the subcommand it belongs to the subcommand and
// is rejected. That is the same shape and the same reason as the mode-derived injection in
// internal/lifecycle/compose.go, deliberately mirrored rather than re-invented — the two
// paths address the same compose project and must agree on which services exist. nil for
// the stack path, which has no plan and therefore no plan profiles.
func buildComposeArgsForEntry(e *config.Environment, c *config.Config, entry *config.LifecycleEntry, profiles, args []string) (string, []string, error) {
	composeCmd, composeArgs, err := dvaexec.ComposeArgv(e, entry.ComposeConfig(), c.FileDir())
	if err != nil {
		return "", nil, fmt.Errorf("entry %q: %w", entry.Name, err)
	}
	for _, profile := range profiles {
		composeArgs = append(composeArgs, "--profile", profile)
	}
	return composeCmd, append(composeArgs, args...), nil
}

// buildComposeArgs builds docker compose arguments using config settings.
// Returns the command and args that can be used with exec or shell.
func buildComposeArgs(e *config.Environment, c *config.Config, args []string) (string, []string, error) {
	composeCmd, composeArgs, err := dvaexec.ComposeArgv(e, c.PrimaryComposeConfig(), c.FileDir())
	if err != nil {
		return "", nil, err
	}
	return composeCmd, append(composeArgs, args...), nil
}
