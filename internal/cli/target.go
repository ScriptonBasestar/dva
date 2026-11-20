package cli

import (
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/ScriptonBasestar/dva/internal/output"
	"github.com/ScriptonBasestar/dva/internal/worktarget"
)

var targetCmd = &cobra.Command{
	Use:   "target",
	Short: "List the worktrees of this repository that can be run as targets",
	Long: `List the worktrees of the repository the current directory belongs to, read from Git each
time. Nothing is stored: there is no target registry and no remembered selection.

KIND is default for the main checkout, task for a worktree with a branch checked out, and
snapshot for a detached one (no ID). A bare entry and a worktree whose directory is gone are
left out. DIRTY marks uncommitted or untracked changes; LOCKED marks a locked worktree.
--json prints the repository identity and the same rows for a program.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("current directory: %w", err)
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

func init() {
	targetCmd.GroupID = "lifecycle"
	rootCmd.AddCommand(targetCmd)
}
