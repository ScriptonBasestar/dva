package cli

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/ScriptonBasestar/dva/internal/worktarget"
)

// A snapshot has no ID and prints "-"; the head is shortened and flags read yes/no.
func TestWriteTargetTable(t *testing.T) {
	var buf bytes.Buffer
	err := writeTargetTable(&buf, []worktarget.Candidate{
		{ID: "main", Kind: worktarget.KindDefault, Path: "/r", Head: "0123456789abcdef"},
		{Kind: worktarget.KindSnapshot, Path: "/s", Head: "fedcba", Dirty: true, Locked: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("table = %q", buf.String())
	}
	if f := strings.Fields(lines[1]); strings.Join(f, " ") != "main default 0123456789ab no no /r" {
		t.Errorf("row 1 = %q", lines[1])
	}
	if f := strings.Fields(lines[2]); strings.Join(f, " ") != "- snapshot fedcba yes yes /s" {
		t.Errorf("row 2 = %q", lines[2])
	}
}

func targetTestRepo(t *testing.T) (repo, feat string) {
	t.Helper()
	gitc := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.email=t@t", "-c", "user.name=t"}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	repo = filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(filepath.Join(repo, "app"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "app", "dva.yml"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitc(repo, "init", "-q", "-b", "main")
	gitc(repo, "add", "-A")
	gitc(repo, "commit", "-q", "-m", "init")
	feat = filepath.Join(t.TempDir(), "feat")
	gitc(repo, "worktree", "add", "-q", "-b", "feat", feat)
	return repo, feat
}

func newTargetTestCmd() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("json", false, "")
	cmd.Flags().Bool("dry-run", false, "")
	cmd.Flags().Bool("debug", false, "")
	cmd.SetErr(&bytes.Buffer{})
	return cmd
}

func captureTargetRun(t *testing.T) *[]string {
	t.Helper()
	var got []string
	old := runTargetIn
	runTargetIn = func(_ context.Context, dir string, args []string) error {
		got = append([]string{dir}, args...)
		return nil
	}
	t.Cleanup(func() { runTargetIn = old })
	return &got
}

// The verb runs in the same subdirectory of the target, with its arguments untouched.
func TestRunTargetHandsTheVerbToTheTarget(t *testing.T) {
	repo, feat := targetTestRepo(t)
	got := captureTargetRun(t)
	if err := runTarget(newTargetTestCmd(), filepath.Join(repo, "app"), []string{"git:feat", "up", "dev", "--dry-run"}); err != nil {
		t.Fatal(err)
	}
	if len(*got) != 4 || worktarget.Canonical((*got)[0]) != worktarget.Canonical(filepath.Join(feat, "app")) ||
		strings.Join((*got)[1:], " ") != "up dev --dry-run" {
		t.Errorf("ran %q", *got)
	}
}

// Nothing runs on a bad request, and the caller's checkout is never used instead.
func TestRunTargetRefusesWithoutRunning(t *testing.T) {
	repo, _ := targetTestRepo(t)
	got := captureTargetRun(t)
	cases := map[string][]string{
		"usage":            {"feat"},
		"usage: dva":       {"feat", "build"},
		"no target":        {"nope", "up"},
		"unknown provider": {"ce:TASK-1", "up"},
	}
	for want, args := range cases {
		err := runTarget(newTargetTestCmd(), repo, args)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%v: err = %v, want %q", args, err, want)
		}
	}
	cmd := newTargetTestCmd()
	if err := cmd.Flags().Set("json", "true"); err != nil {
		t.Fatal(err)
	}
	if err := runTarget(cmd, repo, []string{"feat", "status"}); err == nil || !strings.Contains(err.Error(), "after") {
		t.Errorf("--json before the ID: err = %v", err)
	}
	if len(*got) != 0 {
		t.Errorf("something ran: %q", *got)
	}
}
