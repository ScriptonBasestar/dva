package cirun

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
	ref := filepath.Join(root, ".git", filepath.FromSlash(strings.TrimSpace(string(head))))
	if err := os.WriteFile(ref, []byte(strings.Repeat("1", 40)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, ".git", "packed-refs")); err == nil {
		t.Skip("branch ref is packed; the loose ref above would not be the only definition")
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
