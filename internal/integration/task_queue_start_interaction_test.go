//go:build integration && (darwin || linux)

package integration

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestTaskQueueStartInteractionFromUnrelatedDirectory exercises the checked-in
// start bridge at its DVA boundary. A fake TaskChain binary must fail before
// either child stub can run: hash mismatch on the authorized darwin/arm64 pin,
// and no authorized pin on every other platform.
func TestTaskQueueStartInteractionFromUnrelatedDirectory(t *testing.T) {
	root, err := filepath.EvalSymlinks(repoRoot())
	if err != nil {
		t.Fatal(err)
	}
	board := filepath.Join(root, "tasks")
	bin := t.TempDir()
	queueArgv := filepath.Join(t.TempDir(), "queue-argv")
	ceCalls := filepath.Join(t.TempDir(), "ce-calls")
	goCalls := filepath.Join(t.TempDir(), "go-calls")
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
	writeTaskQueueStartStub(t, filepath.Join(bin, "go"), "#!/bin/sh\ntouch \"$TASK_QUEUE_GO_CALLS\"\nexit 87\n")

	unrelated := filepath.Join(t.TempDir(), "unrelated", "nested")
	if err := os.MkdirAll(unrelated, 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(root, "dva.yml")
	validTask := taskQueueStartJSON("TASK-2")
	t.Run("fake binary cannot run queue or CE", func(t *testing.T) {
		for _, path := range []string{queueArgv, ceCalls} {
			if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
				t.Fatal(err)
			}
		}
		before := taskQueueBoardDigest(t, board)
		code, stdout, stderr := runTaskQueueStartDVA(t, unrelated, configPath, bin, "feat", map[string]string{
			"TASK_QUEUE_ARGV":             queueArgv,
			"TASK_QUEUE_WORKDIR":          filepath.Join(t.TempDir(), "queue-workdir"),
			"TASK_QUEUE_STDOUT":           validTask,
			"TASK_QUEUE_STDERR":           "",
			"TASK_QUEUE_EXIT":             "0",
			"TASK_QUEUE_START_CE_ARGV":    filepath.Join(t.TempDir(), "ce-argv"),
			"TASK_QUEUE_START_CE_WORKDIR": filepath.Join(t.TempDir(), "ce-workdir"),
			"TASK_QUEUE_START_CE_CALLS":   ceCalls,
			"TASK_QUEUE_START_CE_STDOUT":  "started task\n",
			"TASK_QUEUE_START_CE_STDERR":  "",
			"TASK_QUEUE_START_CE_EXIT":    "0",
			"TASK_QUEUE_GO_CALLS":         goCalls,
		})
		if after := taskQueueBoardDigest(t, board); after != before {
			t.Fatal("task-queue-start interaction changed the fixture board")
		}
		assertTaskQueueNoLock(t, board)
		if code == 0 || stdout != "" {
			t.Fatalf("exit=%d stdout=%q, want nonzero exit and no child output", code, stdout)
		}
		want := "no authorized TaskChain binary pin"
		if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
			want = "SHA-256 mismatch"
		}
		if !strings.Contains(stderr, want) {
			t.Fatalf("stderr=%q, want %s", stderr, want)
		}
		if _, err := os.Stat(queueArgv); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("TaskChain queue was invoked for a fake binary: %v", err)
		}
		if _, err := os.Stat(ceCalls); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("CE was invoked for a fake binary: %v", err)
		}
		if _, err := os.Stat(goCalls); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("PATH Go was invoked before the compiled pin check: %v", err)
		}
	})
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
