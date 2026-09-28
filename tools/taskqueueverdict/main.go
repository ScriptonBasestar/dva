// taskqueueverdict classifies the read-only TaskChain queue without selecting work.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ScriptonBasestar/dva/internal/taskqueue"
)

func main() {
	repoRoot, err := os.Getwd()
	if err != nil {
		fail(fmt.Errorf("get repository root: %w", err))
	}
	if err := run(context.Background(), os.Args[1:], repoRoot, os.Stdout); err != nil {
		fail(err)
	}
}

func run(ctx context.Context, args []string, repoRoot string, stdout io.Writer) error {
	flags := flag.NewFlagSet("taskqueueverdict", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	dir := flags.String("dir", "tasks", "TaskChain board directory")
	startType := flags.String("start-type", "", "unsupported; use compiled dva task-queue-start")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(flags.Args(), " "))
	}
	if *startType != "" {
		return fmt.Errorf("--start-type is unavailable in taskqueueverdict; use compiled dva task-queue-start")
	}
	return taskqueue.Verdict(ctx, repoRoot, *dir, stdout)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "task-queue-verdict:", err)
	os.Exit(1)
}
