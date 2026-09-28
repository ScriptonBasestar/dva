//go:build integration && (darwin || linux)

package integration

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestTaskQueueStartInteractionFromUnrelatedDirectory exercises the checked-in
// start bridge at its DVA boundary. The queue and CE stubs make the test
// independent of host installations while retaining the two child contracts.
func TestTaskQueueStartInteractionFromUnrelatedDirectory(t *testing.T) {
	root, err := filepath.EvalSymlinks(repoRoot())
	if err != nil {
		t.Fatal(err)
	}
	board := filepath.Join(root, "tasks")
	bin := t.TempDir()
	queueArgv := filepath.Join(t.TempDir(), "queue-argv")
	queueWorkdir := filepath.Join(t.TempDir(), "queue-workdir")
	ceArgv := filepath.Join(t.TempDir(), "ce-argv")
	ceWorkdir := filepath.Join(t.TempDir(), "ce-workdir")
	ceCalls := filepath.Join(t.TempDir(), "ce-calls")
	writeTaskQueueStartStub(t, filepath.Join(bin, "taskchain-task-manager"), `#!/bin/sh
printf '%s\000' "$@" > "$TASK_QUEUE_ARGV"
pwd -P > "$TASK_QUEUE_WORKDIR"
printf '%s' "$TASK_QUEUE_STDOUT"
printf '%s' "$TASK_QUEUE_STDERR" >&2
exit "$TASK_QUEUE_EXIT"
`)
	writeTaskQueueStartStub(t, filepath.Join(bin, "ce"), `#!/bin/sh
printf '%s\000' "$@" >> "$TASK_QUEUE_START_CE_ARGV"
pwd -P >> "$TASK_QUEUE_START_CE_WORKDIR"
printf x >> "$TASK_QUEUE_START_CE_CALLS"
printf '%s' "$TASK_QUEUE_START_CE_STDOUT"
printf '%s' "$TASK_QUEUE_START_CE_STDERR" >&2
exit "$TASK_QUEUE_START_CE_EXIT"
`)

	unrelated := filepath.Join(t.TempDir(), "unrelated", "nested")
	if err := os.MkdirAll(unrelated, 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(root, "dva.yml")
	validTask := taskQueueStartJSON("TASK-2")
	validIssue := taskQueueStartJSON("ISSUE-9")
	empty := `{"outputVersion":1,"runnableCount":0,"agentRunnableCount":0,"runnable":[],"agentRunnable":[]}`
	human := `{"outputVersion":1,"runnableCount":1,"agentRunnableCount":0,"runnable":[{"path":"todo/TASK-1.md","card":{"id":"TASK-1"},"needsHuman":true,"executionMode":"external","allowedPaths":[]}],"agentRunnable":[]}`
	multiple := `{"outputVersion":1,"runnableCount":2,"agentRunnableCount":2,"runnable":[{"path":"todo/TASK-2.md","card":{"id":"TASK-2"},"needsHuman":false,"executionMode":"implementation","allowedPaths":["internal/integration"]},{"path":"todo/TASK-3.md","card":{"id":"TASK-3"},"needsHuman":false,"executionMode":"implementation","allowedPaths":["internal/integration"]}],"agentRunnable":[{"path":"todo/TASK-2.md","card":{"id":"TASK-2"},"needsHuman":false,"executionMode":"implementation","allowedPaths":["internal/integration"]},{"path":"todo/TASK-3.md","card":{"id":"TASK-3"},"needsHuman":false,"executionMode":"implementation","allowedPaths":["internal/integration"]}]}`

	for _, tc := range []struct {
		name       string
		queue      string
		branchType string
		extraArgs  []string
		ceStdout   string
		ceExit     string
		wantCE     string
		wantQueue  bool
		wantOK     bool
	}{
		{name: "task candidate", queue: validTask, branchType: "feat", ceStdout: "started task\n", ceExit: "0", wantCE: "task\x00run-start\x00task-2\x00--type\x00feat\x00--json\x00", wantQueue: true, wantOK: true},
		{name: "issue candidate", queue: validIssue, branchType: "feat", ceStdout: "started issue\n", ceExit: "0", wantCE: "task\x00run-start\x00issue-9\x00--type\x00feat\x00--json\x00", wantQueue: true, wantOK: true},
		{name: "empty queue", queue: empty, branchType: "feat", ceExit: "0", wantQueue: true},
		{name: "human only", queue: human, branchType: "feat", ceExit: "0", wantQueue: true},
		{name: "multiple candidates", queue: multiple, branchType: "feat", ceExit: "0", wantQueue: true},
		{name: "invalid type", queue: validTask, branchType: "invalid", ceExit: "0"},
		{name: "alternate board", queue: validTask, branchType: "feat", extraArgs: []string{"--dir", "/tmp/another-board"}, ceExit: "0"},
		{name: "invalid upstream", queue: "{not JSON", branchType: "feat", ceExit: "0", wantQueue: true},
		{name: "ce failure after partial mutation", queue: validTask, branchType: "feat", ceStdout: `{"schemaVersion":1,"status":"BLOCKED","receipt":{"worktree":"/tmp/created"}}` + "\n", ceExit: "23", wantCE: "task\x00run-start\x00task-2\x00--type\x00feat\x00--json\x00", wantQueue: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, path := range []string{queueArgv, queueWorkdir, ceArgv, ceWorkdir, ceCalls} {
				if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
					t.Fatal(err)
				}
			}
			before := taskQueueBoardDigest(t, board)
			code, stdout, _ := runTaskQueueStartDVA(t, unrelated, configPath, bin, tc.branchType, map[string]string{
				"TASK_QUEUE_ARGV":             queueArgv,
				"TASK_QUEUE_WORKDIR":          queueWorkdir,
				"TASK_QUEUE_STDOUT":           tc.queue,
				"TASK_QUEUE_STDERR":           "",
				"TASK_QUEUE_EXIT":             "0",
				"TASK_QUEUE_START_CE_ARGV":    ceArgv,
				"TASK_QUEUE_START_CE_WORKDIR": ceWorkdir,
				"TASK_QUEUE_START_CE_CALLS":   ceCalls,
				"TASK_QUEUE_START_CE_STDOUT":  tc.ceStdout,
				"TASK_QUEUE_START_CE_STDERR":  "",
				"TASK_QUEUE_START_CE_EXIT":    tc.ceExit,
			}, tc.extraArgs...)
			if after := taskQueueBoardDigest(t, board); after != before {
				t.Fatal("task-queue-start interaction changed the fixture board")
			}
			assertTaskQueueNoLock(t, board)
			if tc.wantOK {
				if code != 0 || stdout != tc.ceStdout {
					t.Fatalf("exit=%d stdout=%q, want exit=0 stdout=%q", code, stdout, tc.ceStdout)
				}
			} else if code == 0 || stdout != tc.ceStdout {
				t.Fatalf("exit=%d stdout=%q, want nonzero exit and preserved CE response %q", code, stdout, tc.ceStdout)
			}

			if tc.wantQueue {
				assertTaskQueueStartQueueContract(t, queueArgv, queueWorkdir, root)
			} else if _, err := os.Stat(queueArgv); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("queue was invoked for invalid input: %v", err)
			}
			if tc.wantCE == "" {
				if _, err := os.Stat(ceCalls); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("CE was invoked unexpectedly: %v", err)
				}
				return
			}
			if got, err := os.ReadFile(ceCalls); err != nil || string(got) != "x" {
				t.Fatalf("CE call count = %q, err=%v, want exactly one", got, err)
			}
			if got, err := os.ReadFile(ceArgv); err != nil || !bytes.Equal(got, []byte(tc.wantCE)) {
				t.Fatalf("CE argv = %q, err=%v", got, err)
			}
			if got, err := os.ReadFile(ceWorkdir); err != nil || strings.TrimSpace(string(got)) != root {
				t.Fatalf("CE workdir = %q, want %q, err=%v", got, root, err)
			}
		})
	}
}

func taskQueueStartJSON(id string) string {
	return `{"outputVersion":1,"runnableCount":1,"agentRunnableCount":1,"runnable":[{"path":"todo/` + id + `.md","card":{"id":"` + id + `"},"needsHuman":false,"executionMode":"implementation","allowedPaths":["internal/integration"]}],"agentRunnable":[{"path":"todo/` + id + `.md","card":{"id":"` + id + `"},"needsHuman":false,"executionMode":"implementation","allowedPaths":["internal/integration"]}]}`
}

func writeTaskQueueStartStub(t *testing.T, path, script string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
}

func runTaskQueueStartDVA(t *testing.T, cwd, configPath, bin, branchType string, values map[string]string, extraArgs ...string) (int, string, string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), taskQueueInteractionTimeout)
	defer cancel()
	args := append([]string{"task-queue-start", branchType}, extraArgs...)
	cmd := exec.CommandContext(ctx, dvaBinary(t), args...)
	cmd.Dir = cwd
	values["DVA_FILE"] = configPath
	values["PATH"] = bin + string(os.PathListSeparator) + os.Getenv("PATH")
	cmd.Env = taskQueueEnv(values)
	var stdout, stderr strings.Builder
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Fatalf("dva task-queue-start timed out after %s", taskQueueInteractionTimeout)
	}
	if err == nil {
		return 0, stdout.String(), stderr.String()
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), stdout.String(), stderr.String()
	}
	t.Fatalf("dva task-queue-start: %v", err)
	return 0, "", ""
}

func assertTaskQueueStartQueueContract(t *testing.T, argvPath, workdirPath, root string) {
	t.Helper()
	if got, err := os.ReadFile(argvPath); err != nil || !bytes.Equal(got, []byte("queue\x00--dir\x00tasks\x00--json\x00")) {
		t.Fatalf("queue argv = %q, err=%v", got, err)
	}
	if got, err := os.ReadFile(workdirPath); err != nil || strings.TrimSpace(string(got)) != root {
		t.Fatalf("queue workdir = %q, want %q, err=%v", got, root, err)
	}
}
