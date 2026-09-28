//go:build integration && (darwin || linux)

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const validTaskQueueJSON = `{"outputVersion":1,"runnable":[{"path":"todo/TASK-1.md","card":{"id":"TASK-1","title":"Human handoff","status":"pending","priority":"P1","dependsOn":null},"needsHuman":true,"executionMode":"external","allowedPaths":[]},{"path":"todo/TASK-2.md","card":{"id":"TASK-2","title":"Implement queue verdict","status":"pending","priority":"P1","dependsOn":null},"needsHuman":false,"executionMode":"implementation","allowedPaths":["tools/taskqueueverdict"]}],"runnableCount":2,"agentRunnable":[{"path":"todo/TASK-2.md","card":{"id":"TASK-2","title":"Implement queue verdict","status":"pending","priority":"P1","dependsOn":null},"needsHuman":false,"executionMode":"implementation","allowedPaths":["tools/taskqueueverdict"]}],"agentRunnableCount":1}`

// TestTaskQueueVerdictInteractionFromUnrelatedDirectory exercises the checked-in
// verdict interaction through DVA. It uses the source Go toolchain and a stubbed
// upstream binary, so the test covers the process boundary without depending on
// a host taskchain-task-manager installation.
func TestTaskQueueVerdictInteractionFromUnrelatedDirectory(t *testing.T) {
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
	t.Run("success", func(t *testing.T) {
		code, stdout, stderr := runTaskQueueVerdictDVA(t, unrelated, configPath, bin, map[string]string{
			"TASK_QUEUE_ARGV":    argvPath,
			"TASK_QUEUE_WORKDIR": workdirPath,
			"TASK_QUEUE_STDOUT":  validTaskQueueJSON,
			"TASK_QUEUE_STDERR":  "",
			"TASK_QUEUE_EXIT":    "0",
		})
		if code != 0 || stderr != "" {
			t.Fatalf("exit=%d stderr=%q, want success with empty stderr", code, stderr)
		}
		assertTaskQueueVerdictCandidate(t, stdout)
		assertTaskQueueUpstreamContract(t, argvPath, workdirPath, root)
	})

	for _, tc := range []struct {
		name   string
		stdout string
		stderr string
		exit   string
	}{
		{name: "upstream failure", stdout: validTaskQueueJSON, stderr: "upstream diagnostic\n", exit: "23"},
		{name: "malformed upstream JSON", stdout: "{not JSON", exit: "0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			code, stdout, _ := runTaskQueueVerdictDVA(t, unrelated, configPath, bin, map[string]string{
				"TASK_QUEUE_ARGV":    argvPath,
				"TASK_QUEUE_WORKDIR": workdirPath,
				"TASK_QUEUE_STDOUT":  tc.stdout,
				"TASK_QUEUE_STDERR":  tc.stderr,
				"TASK_QUEUE_EXIT":    tc.exit,
			})
			if code == 0 || stdout != "" {
				t.Fatalf("exit=%d stdout=%q, want nonzero exit and no success JSON", code, stdout)
			}
			assertTaskQueueUpstreamContract(t, argvPath, workdirPath, root)
		})
	}

	if after := taskQueueBoardDigest(t, board); after != before {
		t.Fatal("task-queue-verdict interaction changed the fixture board")
	}
	assertTaskQueueNoLock(t, board)
}

func runTaskQueueVerdictDVA(t *testing.T, cwd, configPath, bin string, values map[string]string) (int, string, string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), taskQueueInteractionTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, dvaBinary(t), "task-queue-verdict")
	cmd.Dir = cwd
	values["DVA_FILE"] = configPath
	values["PATH"] = bin + string(os.PathListSeparator) + os.Getenv("PATH")
	cmd.Env = taskQueueEnv(values)
	var stdout, stderr strings.Builder
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Fatalf("dva task-queue-verdict timed out after %s", taskQueueInteractionTimeout)
	}
	if err == nil {
		return 0, stdout.String(), stderr.String()
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), stdout.String(), stderr.String()
	}
	t.Fatalf("dva task-queue-verdict: %v", err)
	return 0, "", ""
}

func assertTaskQueueUpstreamContract(t *testing.T, argvPath, workdirPath, root string) {
	t.Helper()
	if got, err := os.ReadFile(argvPath); err != nil || !bytes.Equal(got, []byte("queue\x00--dir\x00tasks\x00--json\x00")) {
		t.Fatalf("stub argv = %q, err=%v", got, err)
	}
	if got, err := os.ReadFile(workdirPath); err != nil || strings.TrimSpace(string(got)) != root {
		t.Fatalf("stub workdir = %q, want %q, err=%v", got, root, err)
	}
}

func assertTaskQueueVerdictCandidate(t *testing.T, stdout string) {
	t.Helper()
	var verdict struct {
		VerdictVersion     int             `json:"verdictVersion"`
		State              string          `json:"state"`
		RunnableCount      int             `json:"runnableCount"`
		AgentRunnableCount int             `json:"agentRunnableCount"`
		Candidate          json.RawMessage `json:"candidate"`
		Runnable           json.RawMessage `json:"runnable"`
	}
	if err := json.Unmarshal([]byte(stdout), &verdict); err != nil {
		t.Fatalf("verdict is not JSON: %v; stdout=%q", err, stdout)
	}
	if verdict.VerdictVersion != 1 || verdict.State != "candidate" {
		t.Fatalf("verdictVersion=%d state=%q, want 1 and candidate", verdict.VerdictVersion, verdict.State)
	}
	if verdict.RunnableCount != 2 || verdict.AgentRunnableCount != 1 {
		t.Fatalf("runnableCount=%d agentRunnableCount=%d, want 2 and 1", verdict.RunnableCount, verdict.AgentRunnableCount)
	}
	if !jsonEqual(t, verdict.Runnable, json.RawMessage(`[{"path":"todo/TASK-1.md","card":{"id":"TASK-1","title":"Human handoff","status":"pending","priority":"P1","dependsOn":null},"needsHuman":true,"executionMode":"external","allowedPaths":[]},{"path":"todo/TASK-2.md","card":{"id":"TASK-2","title":"Implement queue verdict","status":"pending","priority":"P1","dependsOn":null},"needsHuman":false,"executionMode":"implementation","allowedPaths":["tools/taskqueueverdict"]}]`)) {
		t.Fatalf("runnable = %s, want upstream runnable preserved", verdict.Runnable)
	}
	if !jsonEqual(t, verdict.Candidate, json.RawMessage(`{"path":"todo/TASK-2.md","card":{"id":"TASK-2","title":"Implement queue verdict","status":"pending","priority":"P1","dependsOn":null},"needsHuman":false,"executionMode":"implementation","allowedPaths":["tools/taskqueueverdict"]}`)) {
		t.Fatalf("candidate = %s, want the sole agentRunnable entry", verdict.Candidate)
	}
}

func jsonEqual(t *testing.T, got, want json.RawMessage) bool {
	t.Helper()
	var gotValue, wantValue any
	return json.Unmarshal(got, &gotValue) == nil && json.Unmarshal(want, &wantValue) == nil && reflect.DeepEqual(gotValue, wantValue)
}
