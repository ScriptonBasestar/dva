package cli

import (
	"errors"
	"strings"

	"github.com/ScriptonBasestar/dva/internal/cirun"
	"github.com/ScriptonBasestar/dva/internal/output"
	"github.com/spf13/cobra"
)

func newCIHistoryCommand(parent *cobra.Command, readStatus func(string) ([]cirun.Report, error)) *cobra.Command {
	var root, profile string
	cmd := &cobra.Command{Use: "history", Short: "Run the history profile or inspect CI receipts", Long: "Without --root and --profile, run the declared history CI profile. With both flags, inspect receipts for the exact recorded root and profile without running checks. Missing host, configuration, and cache identities make timing comparisons unknown.", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cmd.Flags().Changed("root") && !cmd.Flags().Changed("profile") {
				return parent.RunE(cmd, []string{"history"})
			}
			if strings.TrimSpace(root) == "" || strings.TrimSpace(profile) == "" {
				return errors.New("ci history analysis requires nonempty --root and --profile")
			}
			r, err := readStatus("")
			if err != nil {
				return err
			}
			return output.PrintJSON(cirun.AnalyzeHistory(r, root, profile))
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "Exact recorded root")
	cmd.Flags().StringVar(&profile, "profile", "", "Exact recorded profile")
	// Parent profile flags are local, so share them with this dispatch command.
	cmd.Flags().AddFlagSet(parent.Flags())
	return cmd
}

func init() {
	ciCmd.AddCommand(newCIHistoryCommand(ciCmd, cirun.Status))
}
