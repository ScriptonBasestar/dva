package compare

import (
	"context"
	"path/filepath"
	"testing"
)

func TestTargetIDParsesOnlyTheWorktreeProvider(t *testing.T) {
	if id, err := TargetID("worktree:feat/x"); err != nil || id != "feat/x" {
		t.Errorf("TargetID = %q, %v", id, err)
	}
	for _, bad := range []string{"worktree:", "worktree:  ", "ce:TASK-1", "wt:x"} {
		if _, err := TargetID(bad); err == nil {
			t.Errorf("TargetID(%q) should fail", bad)
		}
	}
	if IsTargetSource("feat/x") || !IsTargetSource("worktree:x") {
		t.Error("IsTargetSource must tell a git ref from a provider target")
	}
}

// The git provider matches a branch checked out in a worktree and nothing else: a detached
// worktree has no ID, an unknown branch lists the candidates, and the main checkout counts as
// the worktree of its own branch.
func TestResolveTargetFindsBranchWorktrees(t *testing.T) {
	repo := testRepo(t, ".sb/*")
	ctx := context.Background()
	wt := filepath.Join(t.TempDir(), "wt-feat")
	runGit(t, repo, "worktree", "add", "-q", "-b", "feat", wt)
	head := runGit(t, wt, "rev-parse", "HEAD")
	detached := filepath.Join(t.TempDir(), "wt-detached")
	runGit(t, repo, "worktree", "add", "-q", "--detach", detached)

	v, err := ResolveTarget(ctx, repo, "worktree:feat")
	if err != nil {
		t.Fatal(err)
	}
	if v.Mode != ModeWorktree || v.Source != "worktree:feat" || v.Commit != head || canonical(v.Path) != canonical(wt) {
		t.Errorf("variant = %+v", v)
	}
	if canonical(v.ConfigDir) != canonical(wt) {
		t.Errorf("config dir = %s, want %s", v.ConfigDir, wt)
	}
	if main, err := ResolveTarget(ctx, repo, "worktree:main"); err != nil || canonical(main.Path) != canonical(repo) {
		t.Errorf("main = %+v, %v", main, err)
	}

	_, err = ResolveTarget(ctx, repo, "worktree:nope")
	mustContain(t, err, `no worktree has branch "nope"`)
	mustContain(t, err, "feat, main")
	if _, err := ResolveTarget(ctx, repo, "worktree:"+runGit(t, detached, "rev-parse", "HEAD")); err == nil {
		t.Error("a detached worktree must not be reachable by its commit")
	}
}

// The config directory of a repository subdirectory is found below the worktree root, so a
// monorepo's dva.yml resolves inside the work worktree and not in the calling checkout.
func TestResolveTargetKeepsConfigDirBelowRepoRoot(t *testing.T) {
	repo := testRepo(t, ".sb/*")
	writeFile(t, filepath.Join(repo, "app", "dva.yml"), "version: \"0.1.0\"\n")
	commitAll(t, repo, "app")
	wt := filepath.Join(t.TempDir(), "wt-feat")
	runGit(t, repo, "worktree", "add", "-q", "-b", "feat", wt)

	v, err := ResolveTarget(context.Background(), filepath.Join(repo, "app"), "worktree:feat")
	if err != nil {
		t.Fatal(err)
	}
	if canonical(v.ConfigDir) != canonical(filepath.Join(wt, "app")) {
		t.Errorf("config dir = %s, want %s", v.ConfigDir, filepath.Join(wt, "app"))
	}
}
