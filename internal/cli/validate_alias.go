package cli

import "github.com/spf13/cobra"

func init() {
	rootValidateCmd := &cobra.Command{
		Use:     validateCmd.Use,
		Short:   validateCmd.Short,
		Long:    validateCmd.Long,
		GroupID: "advanced",
		RunE:    validateCmd.RunE,
	}
	addValidateFlags(rootValidateCmd)
	rootCmd.AddCommand(rootValidateCmd)
}

func addValidateFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("fix", false, "Auto-fix compose file project name mismatches")
	cmd.Flags().BoolVar(&validateStrict, "strict", false, "Fail validation when config drift warnings are detected")
	// Both are read with cmd.Flags().GetBool rather than bound to a package variable: the
	// two commands sharing this RunE would otherwise share one variable's value across a
	// process that registers both (validateStrict predates that lesson).
	cmd.Flags().Bool("show-ignored", false, "List every finding that dva.yml's ignore lists suppressed")
	cmd.Flags().Bool("suggest-ignore", false, "Print the current suggestions as a suggestion_ignore: block to paste into dva.yml")
}
