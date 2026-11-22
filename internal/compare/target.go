package compare

import (
	"context"

	"github.com/ScriptonBasestar/dva/internal/worktarget"
)

// compare is the one package that wires worktree-target lookup into DVA: the `worktree:<id>`
// variant source and `dva target` read the same worktrees through it, so the CLI never imports
// internal/worktarget directly (tools/archcheck holds that direction).

// TargetCandidate is one worktree `dva target` can show.
type TargetCandidate = worktarget.Candidate

// TargetSelection is a target chosen by ID from the caller's repository.
type TargetSelection = worktarget.Selection

// TargetCandidates lists the worktrees of the repository dir belongs to, with that repository's
// identity. See worktarget.Candidates for what is left out and why.
func TargetCandidates(ctx context.Context, dir string) (string, []TargetCandidate, error) {
	return worktarget.Candidates(ctx, dir)
}

// SelectTarget resolves id against the candidates of the repository dir belongs to. Only an
// exact ID selects; the caller's own checkout is never a fallback.
func SelectTarget(ctx context.Context, dir, id string) (TargetSelection, error) {
	return worktarget.Select(ctx, dir, id)
}

// TargetConfigDir is the directory inside the selected target that matches dir's position in
// the caller's checkout.
func TargetConfigDir(ctx context.Context, dir string, s TargetSelection) (string, error) {
	return worktarget.ConfigDir(ctx, dir, s)
}

// VerifyTarget re-checks, right before running, that the selection is still a worktree of the
// same repository with the same branch checked out.
func VerifyTarget(ctx context.Context, s TargetSelection) error {
	return worktarget.Verify(ctx, s)
}
