package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/ScriptonBasestar/dva/internal/output"
	"github.com/ScriptonBasestar/dva/internal/worktarget"
)

// targetVerbs are the lifecycle verbs `dva target <id>` runs in a target (TASK-587 contract).
var targetVerbs = map[string]bool{"up": true, "down": true, "status": true, "stop": true}

var targetCmd = &cobra.Command{
	Use:   "target [<id> up|down|status|stop [args...]]",
	Short: "List worktree targets, or run a lifecycle verb in one",
	Long: `Without arguments, list the worktrees of the repository the current directory belongs to,
read from Git each time. Nothing is stored: there is no target registry and no remembered
selection.

KIND is default for the main checkout, task for a worktree with a branch checked out, and
snapshot for a detached one (no ID). A bare entry and a worktree whose directory is gone are
left out. DIRTY marks uncommitted or untracked changes; LOCKED marks a locked worktree.
--json prints the repository identity and the same rows for a program.

dva target <id> <verb> [args...] runs dva <verb> [args...] in that target, from the same
subdirectory this command was run in, exactly as running it there would. <id> is the branch,
optionally written git:<branch>; the default checkout runs only when named. An unknown ID, a
provider prefix other than git, or a failed lookup runs nothing, and the current checkout is
never used instead. Right before running, dva re-checks that the path is still a worktree of
this repository with the same branch checked out; new commits and uncommitted changes are fine.
The target is printed on stderr first. Flags such as --json and --dry-run go after the verb,
where they are passed through; the verb's exit status is dva target's exit status.`,
	Example: `  dva target                          # list candidates
  dva target feat/login up dev        # start plan dev in the feat/login worktree
  dva target feat/login status --json
  dva target main down dev --dry-run  # the default checkout, named explicitly`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("current directory: %w", err)
		}
		if len(args) > 0 {
			return runTarget(cmd, dir, args)
		}
		repo, list, err := worktarget.Candidates(compareContext(cmd), dir)
		if err != nil {
			return err
		}
		if jsonOutput {
			return output.PrintJSON(targetListing{Repository: repo, Candidates: list})
		}
		return writeTargetTable(cmd.OutOrStdout(), list)
	},
}

type targetListing struct {
	Repository string                 `json:"repository"`
	Candidates []worktarget.Candidate `json:"candidates"`
}

func writeTargetTable(w io.Writer, list []worktarget.Candidate) error {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "ID\tKIND\tHEAD\tDIRTY\tLOCKED\tPATH"); err != nil {
		return err
	}
	for _, c := range list {
		id := c.ID
		if id == "" {
			id = "-"
		}
		head := c.Head
		if len(head) > 12 {
			head = head[:12]
		}
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n", id, c.Kind, head, yesNo(c.Dirty), yesNo(c.Locked), c.Path); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// runTargetIn runs one dva invocation in dir with this process's stdio. A variable so tests can
// observe the hand-off without starting a second dva.
var runTargetIn = func(ctx context.Context, dir string, args []string) error {
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate dva executable: %w", err)
	}
	child := exec.CommandContext(ctx, self, args...)
	child.Dir = dir
	child.Stdin, child.Stdout, child.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := child.Run(); err != nil {
		if exit, ok := errors.AsType[*exec.ExitError](err); ok {
			// The verb already reported its own failure; only its status is passed on.
			return &resourceChildExit{code: resourceSignalExit(exit), cause: err}
		}
		return fmt.Errorf("run dva %s in %s: %w", strings.Join(args, " "), dir, err)
	}
	return nil
}

func runTarget(cmd *cobra.Command, dir string, args []string) error {
	if len(args) < 2 || !targetVerbs[args[1]] {
		return fmt.Errorf("usage: dva target <id> up|down|status|stop [args...]")
	}
	for _, name := range []string{"json", "dry-run", "debug"} {
		if cmd.Flags().Changed(name) {
			return fmt.Errorf("--%s before the target ID is not passed to the verb; put it after %q", name, args[1])
		}
	}
	ctx := compareContext(cmd)
	sel, err := worktarget.Select(ctx, dir, args[0])
	if err != nil {
		return err
	}
	runDir, err := worktarget.ConfigDir(ctx, dir, sel)
	if err != nil {
		return err
	}
	head := sel.Head
	if len(head) > 12 {
		head = head[:12]
	}
	if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "target %s (%s): %s branch %s head %s dirty %s\n",
		sel.ID, sel.Kind, runDir, sel.Branch, head, yesNo(sel.Dirty)); err != nil {
		return err
	}
	if err := worktarget.Verify(ctx, sel); err != nil {
		return err
	}
	return runTargetIn(ctx, runDir, args[1:])
}

func init() {
	targetCmd.Flags().SetInterspersed(false) // everything after <id> belongs to the verb
	targetCmd.GroupID = "lifecycle"
	rootCmd.AddCommand(targetCmd)
}
