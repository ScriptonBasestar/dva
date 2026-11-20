package worktarget

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func commitIn(t *testing.T, dir string) {
	t.Helper()
	runGit(t, dir, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "-q", "--allow-empty", "-m", "c")
}

// A newline inside a path and a lock reason survive the NUL-terminated porcelain.
func TestParsePorcelainZ(t *testing.T) {
	out := "worktree /r\x00HEAD aa\x00branch refs/heads/main\x00\x00" +
		"worktree /x\ny\x00HEAD bb\x00branch refs/heads/f\x00locked why\nnot\x00\x00" +
		"worktree /p\x00HEAD cc\x00detached\x00prunable gitdir gone\x00\x00"
	got := parsePorcelainZ(out)
	if len(got) != 3 {
		t.Fatalf("entries = %+v", got)
	}
	if got[1].Path != "/x\ny" || !got[1].Locked || got[1].Branch != "f" {
		t.Errorf("entry 1 = %+v", got[1])
	}
	if !got[2].Detached || !got[2].Prunable {
		t.Errorf("entry 2 = %+v", got[2])
	}
}

// A worktree whose .git file is gone is reported prunable by Git while its directory still
// exists; only the prunable flag can leave it out.
func TestCandidatesSkipsPrunableWithDirectory(t *testing.T) {
	repo := testRepo(t)
	wt := filepath.Join(t.TempDir(), "orphan")
	runGit(t, repo, "worktree", "add", "-q", "-b", "orphan", wt)
	if err := os.Remove(filepath.Join(wt, ".git")); err != nil {
		t.Fatal(err)
	}
	_, list, err := Candidates(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range list {
		if c.ID == "orphan" {
			t.Fatalf("prunable worktree listed: %+v", list)
		}
	}
}

func TestSelect(t *testing.T) {
	repo := testRepo(t)
	feat := filepath.Join(t.TempDir(), "feat")
	runGit(t, repo, "worktree", "add", "-q", "-b", "feat/x", feat)
	ctx := context.Background()

	for _, id := range []string{"feat/x", "git:feat/x"} {
		s, err := Select(ctx, repo, id)
		if err != nil || Canonical(s.Path) != Canonical(feat) || s.Kind != KindTask {
			t.Errorf("Select(%q) = %+v, %v", id, s, err)
		}
	}
	if s, err := Select(ctx, feat, "main"); err != nil || s.Kind != KindDefault {
		t.Errorf("the default checkout is selectable by name: %+v, %v", s, err)
	}
	for id, want := range map[string]string{
		"nope":       "no target",
		"ce:TASK-12": "unknown provider",
		"git:":       "empty target",
	} {
		if _, err := Select(ctx, repo, id); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("Select(%q) err = %v, want %q", id, err, want)
		}
	}
}

func TestVerify(t *testing.T) {
	repo := testRepo(t)
	feat := filepath.Join(t.TempDir(), "feat")
	runGit(t, repo, "worktree", "add", "-q", "-b", "feat", feat)
	ctx := context.Background()
	s, err := Select(ctx, repo, "feat")
	if err != nil {
		t.Fatal(err)
	}

	// Work in progress is not a failure.
	commitIn(t, feat)
	if err := os.WriteFile(filepath.Join(feat, "wip"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Verify(ctx, s); err != nil {
		t.Fatalf("new commit and dirty tree must verify: %v", err)
	}

	runGit(t, feat, "switch", "-q", "-c", "other")
	if err := Verify(ctx, s); err == nil || !strings.Contains(err.Error(), "now has other") {
		t.Errorf("switched branch: err = %v", err)
	}
	runGit(t, feat, "switch", "-q", "feat")

	runGit(t, repo, "worktree", "remove", "--force", feat)
	if err := Verify(ctx, s); err == nil || !strings.Contains(err.Error(), "gone") {
		t.Errorf("removed worktree: err = %v", err)
	}
}

// A replacement repository at the same path is not the selected target.
// Only a missing path is "gone". A file in its place, or a path that cannot be read, is
// reported as what it is, so a permission problem is not mistaken for a removed worktree.
func TestVerifyTellsGoneFromUnreadable(t *testing.T) {
	repo := testRepo(t)
	parent := t.TempDir()
	feat := filepath.Join(parent, "feat")
	runGit(t, repo, "worktree", "add", "-q", "-b", "feat", feat)
	ctx := context.Background()
	s, err := Select(ctx, repo, "feat")
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Chmod(parent, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(parent, 0o755) })
	err = Verify(ctx, s)
	if err == nil || strings.Contains(err.Error(), "gone") || !errors.Is(err, fs.ErrPermission) {
		t.Errorf("unreadable parent: err = %v, want a permission error that is not \"gone\"", err)
	}
	if err := os.Chmod(parent, 0o755); err != nil {
		t.Fatal(err)
	}

	runGit(t, repo, "worktree", "remove", "--force", feat)
	if err := os.WriteFile(feat, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Verify(ctx, s); err == nil || !strings.Contains(err.Error(), "no longer a directory") {
		t.Errorf("file in place of the worktree: err = %v", err)
	}
}

func TestVerifyRejectsAnotherRepository(t *testing.T) {
	repo := testRepo(t)
	feat := filepath.Join(t.TempDir(), "feat")
	runGit(t, repo, "worktree", "add", "-q", "-b", "feat", feat)
	ctx := context.Background()
	s, err := Select(ctx, repo, "feat")
	if err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "worktree", "remove", "--force", feat)
	if err := os.MkdirAll(feat, 0o755); err != nil {
		t.Fatal(err)
	}
	runGit(t, feat, "init", "-q", "-b", "feat")
	if err := Verify(ctx, s); err == nil || !strings.Contains(err.Error(), "another repository") {
		t.Errorf("replaced repository: err = %v", err)
	}
}

func TestConfigDirKeepsTheSubdirectory(t *testing.T) {
	repo := testRepo(t)
	if err := os.MkdirAll(filepath.Join(repo, "app"), 0o755); err != nil {
		t.Fatal(err)
	}
	feat := filepath.Join(t.TempDir(), "feat")
	runGit(t, repo, "worktree", "add", "-q", "-b", "feat", feat)
	ctx := context.Background()
	s, err := Select(ctx, repo, "feat")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ConfigDir(ctx, filepath.Join(repo, "app"), s); err == nil {
		t.Error("app/ is untracked in feat and must not be found there")
	}
	if err := os.MkdirAll(filepath.Join(feat, "app"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := ConfigDir(ctx, filepath.Join(repo, "app"), s)
	if err != nil || Canonical(got) != Canonical(filepath.Join(feat, "app")) {
		t.Errorf("ConfigDir = %q, %v", got, err)
	}
}
