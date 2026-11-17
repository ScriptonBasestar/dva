package cirun

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRepositoryLocalIDAcrossWorktreesAndSeparateClones(t *testing.T) {
	base := t.TempDir()
	repo := filepath.Join(base, "one")
	other := filepath.Join(base, "two")
	linked := filepath.Join(base, "linked")
	run := func(args ...string) {
		t.Helper()
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git: %s %v", out, err)
		}
	}
	run("init", "-q", repo)
	run("-C", repo, "-c", "user.name=test", "-c", "user.email=test@example.invalid", "commit", "--allow-empty", "-qm", "initial")
	run("-C", repo, "worktree", "add", "--detach", linked, "HEAD")
	run("init", "-q", other)
	a, err := RepositoryLocalID(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	b, err := RepositoryLocalID(context.Background(), linked)
	if err != nil {
		t.Fatal(err)
	}
	c, err := RepositoryLocalID(context.Background(), other)
	if err != nil {
		t.Fatal(err)
	}
	if a != b || a == c {
		t.Fatalf("identities: %q %q %q", a, b, c)
	}
	if _, err = RepositoryLocalID(context.Background(), base); err == nil {
		t.Fatal("non-repository accepted")
	}
}

func TestAnalyzeRepositoryHistoryDoesNotAttributeUnassigned(t *testing.T) {
	r := []Report{{ID: "one", Root: "/tmp/one", Profile: "full", Observability: &Observability{RepositoryLocalID: "a"}}, {ID: "two", Root: "/tmp/two", Profile: "full", Observability: &Observability{RepositoryLocalID: "a"}}, {ID: "legacy", Profile: "full"}, {ID: "other", Profile: "full", Observability: &Observability{RepositoryLocalID: "b"}}}
	a := AnalyzeRepositoryHistory(r, "a", "full")
	if len(a.Runs) != 2 || a.ProfileUnassigned != 1 {
		t.Fatalf("unexpected result: %+v", a)
	}
}
