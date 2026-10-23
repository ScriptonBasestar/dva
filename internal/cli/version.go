package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ScriptonBasestar/dva/internal/config"
	"github.com/ScriptonBasestar/dva/internal/output"
)

var versionCmd = &cobra.Command{
	Use:     "version",
	Aliases: []string{"-v", "--version"},
	Short:   "Show DVA version",
	Long: `Print the dva binary's version, commit, and build date, injected at build time
via -ldflags; this reflects the binary, not the dva.yml in the current directory.
Without an ldflags commit, a build inside a checkout reports its VCS revision
marked "(vcs)" and a module install reports "unknown (module <version>)";
--json gives the source as commit_source.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		commit := config.ResolvedCommit()
		if jsonOutput {
			fields := map[string]string{
				"version":       config.Version,
				"commit":        commit.Value,
				"commit_source": commit.Source,
				"build_date":    config.BuildDate,
			}
			if commit.Module != "" {
				fields["module_version"] = commit.Module
			}
			return output.PrintJSON(fields)
		}
		fmt.Printf("dva version %s\n", config.Version)
		fmt.Printf("commit: %s\n", commit)
		fmt.Printf("build date: %s\n", config.BuildDate)
		return nil
	},
}
