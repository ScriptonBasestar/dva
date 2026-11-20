package worktarget

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func testRepo(t *testing.T) string {
	t.Helper()
	repo := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "init", "-q", "-b", "main")
	runGit(t, repo, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "-q", "--allow-empty", "-m", "init")
	return repo
}

// Every kind is classified from Git, a dirty task worktree stays selectable, a locked one is
// flagged, and a worktree whose directory is gone (prunable) is left out.
func TestCandidatesClassifiesAndFilters(t *testing.T) {
	repo := testRepo(t)
	base := t.TempDir()
	feat := filepath.Join(base, "wt feat") // a space in the path must survive
	runGit(t, repo, "worktree", "add", "-q", "-b", "feat", feat)
	snap := filepath.Join(base, "snap")
	runGit(t, repo, "worktree", "add", "-q", "--detach", snap)
	locked := filepath.Join(base, "locked")
	runGit(t, repo, "worktree", "add", "-q", "-b", "held", locked)
	runGit(t, repo, "worktree", "lock", locked)
	gone := filepath.Join(base, "gone")
	runGit(t, repo, "worktree", "add", "-q", "-b", "gone", gone)
	if err := os.RemoveAll(gone); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(feat, "edit.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	repoID, list, err := Candidates(context.Background(), feat)
	if err != nil {
		t.Fatalf("Candidates: %v", err)
	}
	if want := Canonical(filepath.Join(repo, ".git")); repoID != want {
		t.Errorf("repo = %q, want %q", repoID, want)
	}
	got := map[string]Candidate{}
	for _, c := range list {
		got[Canonical(c.Path)] = c
	}
	if len(list) != 4 {
		t.Fatalf("candidates = %+v, want main, feat, snap, locked", list)
	}
	check := func(path, kind, id string, dirty, isLocked bool) {
		t.Helper()
		c, ok := got[Canonical(path)]
		if !ok {
			t.Fatalf("%s missing from %+v", path, list)
		}
		if c.Kind != kind || c.ID != id || c.Dirty != dirty || c.Locked != isLocked {
			t.Errorf("%s = %+v, want kind=%s id=%q dirty=%v locked=%v", path, c, kind, id, dirty, isLocked)
		}
	}
	check(repo, KindDefault, "main", false, false)
	check(feat, KindTask, "feat", true, false)
	check(snap, KindSnapshot, "", false, false)
	check(locked, KindTask, "held", false, true)
	if _, ok := got[Canonical(gone)]; ok {
		t.Errorf("a worktree whose directory is gone must not be a candidate: %+v", list)
	}
}

// Outside a repository the lookup fails instead of returning an empty list.
func TestCandidatesFailsOutsideARepository(t *testing.T) {
	if _, _, err := Candidates(context.Background(), t.TempDir()); err == nil {
		t.Fatal("Candidates outside a repository must fail")
	}
}
