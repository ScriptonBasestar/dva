//go:build integration && (darwin || linux)

package integration

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

const taskQueueInteractionTimeout = 15 * time.Second

// TestTaskQueueInteractionFromUnrelatedDirectory exercises the checked-in DVA
// interaction through the built binary. The stub keeps the test independent of
// a taskchain-task-manager installation while preserving the child contract.
func TestTaskQueueInteractionFromUnrelatedDirectory(t *testing.T) {
	root, err := filepath.EvalSymlinks(repoRoot())
	if err != nil {
		t.Fatal(err)
	}
	board := filepath.Join(root, "tasks")
	before := taskQueueBoardDigest(t, board)

	bin := t.TempDir()
	argvPath := filepath.Join(t.TempDir(), "argv")
	workdirPath := filepath.Join(t.TempDir(), "workdir")
	stub := filepath.Join(bin, "taskchain-task-manager")
	const script = `#!/bin/sh
printf '%s\000' "$@" > "$TASK_QUEUE_ARGV"
pwd -P > "$TASK_QUEUE_WORKDIR"
printf '%s' "$TASK_QUEUE_STDOUT"
printf '%s' "$TASK_QUEUE_STDERR" >&2
exit "$TASK_QUEUE_EXIT"
`
	if err := os.WriteFile(stub, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}

	unrelated := filepath.Join(t.TempDir(), "unrelated", "nested")
	if err := os.MkdirAll(unrelated, 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(root, "dva.yml")
	for _, tc := range []struct {
		name   string
		stdout string
		stderr string
		exit   string
		code   int
	}{
		{name: "success", stdout: "queue output\n", stderr: "queue diagnostic\n", exit: "0", code: 0},
		{name: "failure", stdout: "partial queue output\n", stderr: "queue failure\n", exit: "23", code: 23},
	} {
		t.Run(tc.name, func(t *testing.T) {
			code, stdout, stderr := runTaskQueueDVA(t, unrelated, configPath, bin, map[string]string{
				"TASK_QUEUE_ARGV":    argvPath,
				"TASK_QUEUE_WORKDIR": workdirPath,
				"TASK_QUEUE_STDOUT":  tc.stdout,
				"TASK_QUEUE_STDERR":  tc.stderr,
				"TASK_QUEUE_EXIT":    tc.exit,
			})
			if code != tc.code || stdout != tc.stdout || stderr != tc.stderr {
				t.Fatalf("exit=%d stdout=%q stderr=%q, want exit=%d stdout=%q stderr=%q", code, stdout, stderr, tc.code, tc.stdout, tc.stderr)
			}
			if got, err := os.ReadFile(argvPath); err != nil || !bytes.Equal(got, []byte("queue\x00--dir\x00tasks\x00--json\x00")) {
				t.Fatalf("stub argv = %q, err=%v", got, err)
			}
			if got, err := os.ReadFile(workdirPath); err != nil || strings.TrimSpace(string(got)) != root {
				t.Fatalf("stub workdir = %q, want %q, err=%v", got, root, err)
			}
		})
	}

	if after := taskQueueBoardDigest(t, board); after != before {
		t.Fatal("task-queue interaction changed the fixture board")
	}
	assertTaskQueueNoLock(t, board)
}

func runTaskQueueDVA(t *testing.T, cwd, configPath, bin string, values map[string]string) (int, string, string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), taskQueueInteractionTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, dvaBinary(t), "task-queue")
	cmd.Dir = cwd
	values["DVA_FILE"] = configPath
	values["PATH"] = bin + string(os.PathListSeparator) + os.Getenv("PATH")
	cmd.Env = taskQueueEnv(values)
	var stdout, stderr strings.Builder
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Fatalf("dva task-queue timed out after %s", taskQueueInteractionTimeout)
	}
	if err == nil {
		return 0, stdout.String(), stderr.String()
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), stdout.String(), stderr.String()
	}
	t.Fatalf("dva task-queue: %v", err)
	return 0, "", ""
}

func taskQueueEnv(values map[string]string) []string {
	env := os.Environ()
	for key, value := range values {
		prefix := key + "="
		filtered := env[:0]
		for _, entry := range env {
			if !strings.HasPrefix(entry, prefix) {
				filtered = append(filtered, entry)
			}
		}
		env = append(filtered, prefix+value)
	}
	return env
}

func taskQueueBoardDigest(t *testing.T, root string) [sha256.Size]byte {
	t.Helper()
	var paths []string
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("unexpected fixture board entry %s", path)
		}
		paths = append(paths, path)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	sort.Strings(paths)
	hash := sha256.New()
	for _, path := range paths {
		relative, err := filepath.Rel(root, path)
		if err != nil {
			t.Fatal(err)
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fmt.Fprintf(hash, "%s\x00%s\x00", filepath.ToSlash(relative), contents); err != nil {
			t.Fatal(err)
		}
	}
	var digest [sha256.Size]byte
	copy(digest[:], hash.Sum(nil))
	return digest
}

func assertTaskQueueNoLock(t *testing.T, root string) {
	t.Helper()
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Name() == ".task-manager.lock" {
			return fmt.Errorf("task-queue interaction left lock %s", path)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
