package cli

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/ScriptonBasestar/dva/internal/cirun"
	"github.com/ScriptonBasestar/dva/internal/config"
	"github.com/ScriptonBasestar/dva/internal/output"
)

var ciProject string

var ciCmd = &cobra.Command{
	Use:   "ci [profile]",
	Short: "Run budgeted CI checks (default: commit), with scoped exclusion",
	Long:  "Run a declared ci.profiles entry. Commit checks warn after five minutes and have a ten-minute maximum. Full checks require an explicit timeout. A concurrent CI that has the same owning config directory or declared shared lock returns busy instead of starting. Use --dry-run to inspect the resolved profile without executing checks.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := loadConfig()
		if err != nil {
			return err
		}
		c, err = ciOwner(c, ciProject)
		if err != nil {
			return err
		}
		name := "commit"
		if len(args) != 0 {
			name = args[0]
		}
		profile, err := c.ResolveCIProfile(name)
		if err != nil {
			return err
		}
		if dryRun {
			return output.PrintJSON(profile)
		}
		owned := newOwnedConfigEnvironment(c)
		if err := envIncompleteError(owned.report); err != nil {
			return err
		}
		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
		defer stop()
		writer := cmd.OutOrStdout()
		if jsonOutput {
			writer = cmd.ErrOrStderr()
		}
		report, runErr := cirun.Run(ctx, cirun.Options{Root: c.FileDir(), ProfileName: name, Profile: profile, Env: owned.env.EnvSlice(), Output: writer})
		if jsonOutput {
			if err := output.PrintJSON(report); err != nil {
				return err
			}
		} else if report.ID != "" {
			_, printErr := fmt.Fprintf(cmd.OutOrStdout(), "CI %s: %s (%s)\n", report.ID, report.Status, report.Duration)
			runErr = errors.Join(runErr, printErr)
		}
		return runErr
	},
}

func ciOwner(c *config.Config, project string) (*config.Config, error) {
	if project == "" {
		return c, nil
	}
	sub, ok := c.Subprojects[project]
	if !ok {
		return nil, fmt.Errorf("unknown CI project %q", project)
	}
	children, err := config.LoadSubprojects(c.FileDir(), map[string]config.SubprojectConfig{project: sub})
	if err != nil {
		return nil, err
	}
	return children[project], nil
}

func init() {
	ciCmd.GroupID = "core"
	ciCmd.Flags().StringVar(&ciProject, "project", "", "Run a declared subproject's own CI profile")
	ciCmd.AddCommand(&cobra.Command{
		Use: "status", Short: "List CI run receipts on this machine", Args: cobra.NoArgs,
		Long: "List local CI receipts, including the active execution. This read-only query does not load the current project's configuration or start checks. An interrupted running receipt is reported as stale when its owning lock is no longer held.",
		RunE: func(cmd *cobra.Command, args []string) error {
			reports, err := cirun.Status("")
			if err != nil {
				return err
			}
			return output.PrintJSON(reports)
		},
	}, &cobra.Command{
		Use: "logs <run-id>", Short: "Read an existing CI log without starting checks", Args: cobra.ExactArgs(1),
		Long: "Read the local log for an existing CI run ID. This read-only query does not load project configuration or execute checks. Use --json for a structured run_id and log document.",
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := cirun.ReadLog("", args[0])
			if err != nil {
				return err
			}
			if jsonOutput {
				return output.PrintJSON(map[string]string{"run_id": args[0], "log": string(data)})
			}
			_, err = cmd.OutOrStdout().Write(data)
			return err
		},
	})
	rootCmd.AddCommand(ciCmd)
}
