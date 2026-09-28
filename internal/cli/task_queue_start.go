package cli

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/ScriptonBasestar/dva/internal/taskqueue"
	"github.com/spf13/cobra"
)

// The compiled command is the only DVA path allowed to call CE run-start.
// Its board and CE working directory are the owning DVA configuration root,
// even when the caller invokes DVA from elsewhere.
var taskQueueStartCmd = newTaskQueueStartCommand(taskqueue.Start)

type taskQueueStarter func(context.Context, string, string, io.Writer) error

func newTaskQueueStartCommand(start taskQueueStarter) *cobra.Command {
	return &cobra.Command{
		Use:   "task-queue-start <feat|fix|refactor|docs|test|chore|perf>",
		Short: "Start a TaskChain candidate when an approved binary pin exists",
		Long:  "This compiled entry point requires an approved TaskChain artifact pin before starting CE.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "feat", "fix", "refactor", "docs", "test", "chore", "perf":
			default:
				return fmt.Errorf("invalid task branch type %q", args[0])
			}
			cfg := mustLoadConfig()
			root, err := filepath.Abs(cfg.FileDir())
			if err != nil {
				return fmt.Errorf("resolve DVA configuration root: %w", err)
			}
			return start(cmd.Context(), root, args[0], cmd.OutOrStdout())
		},
	}
}
