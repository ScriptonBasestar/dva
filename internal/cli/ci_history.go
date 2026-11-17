package cli

import (
	"github.com/ScriptonBasestar/dva/internal/cirun"
	"github.com/ScriptonBasestar/dva/internal/output"
	"github.com/spf13/cobra"
)

func init() {
	var root, profile string
	cmd := &cobra.Command{Use: "history", Short: "Inspect individual CI receipts without running checks", Long: "Read local CI receipts for one exact recorded root and profile. Missing host, configuration, and cache identities make timing comparisons unknown. This command does not run checks.", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := cirun.Status("")
			if err != nil {
				return err
			}
			return output.PrintJSON(cirun.AnalyzeHistory(r, root, profile))
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "Exact recorded root")
	cmd.Flags().StringVar(&profile, "profile", "", "Exact recorded profile")
	_ = cmd.MarkFlagRequired("root")
	_ = cmd.MarkFlagRequired("profile")
	ciCmd.AddCommand(cmd)
}
