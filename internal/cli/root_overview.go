package cli

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// Explicit help bypasses RunE; only a bare invocation discovers the project.
func runRootOverview(cmd *cobra.Command, _ []string) error {
	_, err := loadConfig()
	if err == nil {
		return cmd.Help()
	}
	if !errors.Is(err, config.ErrConfigNotFound) {
		return err
	}
	if jsonOutput {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
			"status":     "not_configured",
			"message":    "No DVA configuration found in the current directory or its parents.",
			"next_steps": []string{"dva config init", "dva config init --help", "DVA_FILE=/path/to/dva.yml dva show", "dva --help"},
		})
	}
	_, err = fmt.Fprint(cmd.OutOrStdout(), `No DVA configuration found.
Searched the current directory and its parents for dva.yml (or dva.yaml).

Get started:
  dva config init         Create a project configuration
  dva config init --help  See templates and initialization options

Use an existing configuration:
  DVA_FILE=/path/to/dva.yml dva show

Optional AI assistance after initialization (requires Agent Mesh and DVA flows):
  am run dva-discover     Inspect the project
  am run dva-improve      Improve the configuration

All commands:
  dva --help
`)
	return err
}
