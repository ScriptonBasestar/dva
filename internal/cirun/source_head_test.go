package cirun

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// requireFilesRefBackend skips tests that corrupt refs by writing files: only
// the files backend stores HEAD and branches as files, while reftable keeps
// them in tables and would make such a test fail for the wrong reason.
func requireFilesRefBackend(t *testing.T, root string) {
	t.Helper()
	if info, err := os.Stat(filepath.Join(root, ".git", "refs", "heads")); err != nil || !info.IsDir() {
		t.Skip("repository does not use the files ref backend")
	}
}

func TestSourceCommitRejectsUnresolvableHead(t *testing.T) {
	root := t.TempDir()
	if err := runGit(root, "init"); err != nil {
		t.Skipf("git unavailable: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "input"), []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	commitAll(t, root)
	requireFilesRefBackend(t, root)
	// A detached HEAD naming a missing object is broken, not unborn.
	if err := os.WriteFile(filepath.Join(root, ".git", "HEAD"), []byte(strings.Repeat("1", 40)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := sourceCommit(context.Background(), root); err == nil || !strings.Contains(err.Error(), "read git HEAD") {
		t.Fatalf("unresolvable HEAD err=%v", err)
	}
}

// A branch whose ref names a missing object is broken, not unborn: the ref
// exists, so the attestation must fail instead of omitting the commit.
func TestSourceCommitRejectsBranchRefToMissingObject(t *testing.T) {
	root := t.TempDir()
	if err := runGit(root, "init"); err != nil {
		t.Skipf("git unavailable: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "input"), []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	commitAll(t, root)
	head, err := gitStdout(context.Background(), root, "symbolic-ref", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	// A loose ref file overrides any packed entry.
	requireFilesRefBackend(t, root)
	ref := filepath.Join(root, ".git", filepath.FromSlash(strings.TrimSpace(string(head))))
	if err := os.WriteFile(ref, []byte(strings.Repeat("1", 40)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := sourceCommit(context.Background(), root); err == nil || !strings.Contains(err.Error(), "read git HEAD") {
		t.Fatalf("branch ref to missing object err=%v", err)
	}
}

// A detached HEAD on a real commit is a normal checkout: the commit is
// recorded and the clean tree is not dirty.
func TestSourceCommitRecordsDetachedHead(t *testing.T) {
	root := t.TempDir()
	if err := runGit(root, "init"); err != nil {
		t.Skipf("git unavailable: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "input"), []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	commitAll(t, root)
	want, err := gitStdout(context.Background(), root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if err := runGit(root, "checkout", "-q", "--detach"); err != nil {
		t.Fatal(err)
	}
	commit, dirty, err := sourceCommit(context.Background(), root)
	if err != nil || commit != strings.TrimSpace(string(want)) || dirty == nil || *dirty {
		t.Fatalf("detached HEAD commit=%q dirty=%v err=%v, want %s clean", commit, dirty, err, want)
	}
}
