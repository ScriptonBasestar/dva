package cli

import (
	"errors"
	"os"
	"regexp"
	"strings"

	"github.com/ScriptonBasestar/dva/internal/cirun"
	"github.com/ScriptonBasestar/dva/internal/output"
	"github.com/spf13/cobra"
)

func newCIHistoryCommand(parent *cobra.Command, readStatus func(string) ([]cirun.Report, error)) *cobra.Command {
	var root, profile, repositoryID string
	cmd := &cobra.Command{Use: "history", Short: "Run the history profile or inspect CI receipts", Long: "Without --root and --profile, run the declared history CI profile. With both flags, inspect receipts for the exact recorded root and profile without running checks. Missing host, configuration, and cache identities make timing comparisons unknown.", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cmd.Flags().Changed("root") && !cmd.Flags().Changed("profile") && !cmd.Flags().Changed("repository-local-id") {
				return parent.RunE(cmd, []string{"history"})
			}
			if strings.TrimSpace(profile) == "" || (strings.TrimSpace(root) == "") == (strings.TrimSpace(repositoryID) == "") || (cmd.Flags().Changed("root") && cmd.Flags().Changed("repository-local-id")) {
				return errors.New("ci history analysis requires --profile and exactly one of --root or --repository-local-id")
			}
			if repositoryID == "current" {
				cwd, err := os.Getwd()
				if err != nil {
					return err
				}
				repositoryID, err = cirun.RepositoryLocalID(cmd.Context(), cwd)
				if err != nil {
					return err
				}
			}
			if repositoryID != "" && !regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(repositoryID) {
				return errors.New("repository local ID must be current or 64 lowercase hexadecimal characters")
			}
			r, err := readStatus("")
			if err != nil {
				return err
			}
			if repositoryID != "" {
				return output.PrintJSON(cirun.AnalyzeRepositoryHistory(r, repositoryID, profile))
			}
			return output.PrintJSON(cirun.AnalyzeHistory(r, root, profile))
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "Exact recorded root")
	cmd.Flags().StringVar(&profile, "profile", "", "Exact recorded profile")
	cmd.Flags().StringVar(&repositoryID, "repository-local-id", "", "Local clone identity or current")
	// Parent profile flags are local, so share them with this dispatch command.
	cmd.Flags().AddFlagSet(parent.Flags())
	return cmd
}

func init() {
	ciCmd.AddCommand(newCIHistoryCommand(ciCmd, cirun.Status))
}
