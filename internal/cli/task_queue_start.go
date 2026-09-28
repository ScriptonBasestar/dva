package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// task-queue-start is compiled into DVA so its authorization boundary cannot
// be replaced by a PATH-selected Go toolchain. No artifact is approved yet;
// W07c2 must integrate the verified queue implementation before activation.
var taskQueueStartCmd = &cobra.Command{
	Use:   "task-queue-start <feat|fix|refactor|docs|test|chore|perf>",
	Short: "Start a TaskChain candidate when an approved binary pin exists",
	Long:  "This compiled entry point fails closed until an approved TaskChain artifact and its verified queue implementation are integrated into DVA.",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		switch args[0] {
		case "feat", "fix", "refactor", "docs", "test", "chore", "perf":
		default:
			return fmt.Errorf("invalid task branch type %q", args[0])
		}
		return fmt.Errorf("no authorized TaskChain binary pin for %s/%s; CE start is disabled", runtime.GOOS, runtime.GOARCH)
	},
}
