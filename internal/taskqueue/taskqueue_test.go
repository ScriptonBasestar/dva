package taskqueue

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestVerdictUsesRepositoryRootAndPreservesDir(t *testing.T) {
	bin, repoRoot := t.TempDir(), t.TempDir()
	argv, cwd := filepath.Join(t.TempDir(), "argv"), filepath.Join(t.TempDir(), "cwd")
	writeExecutable(t, filepath.Join(bin, "taskchain-task-manager"), "#!/bin/sh\nprintf '%s\\000' \"$@\" > \"$QUEUE_ARGV\"\npwd > \"$QUEUE_CWD\"\nprintf '%s' \"$QUEUE_JSON\"\n")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("QUEUE_ARGV", argv)
	t.Setenv("QUEUE_CWD", cwd)
	item := queueItemJSON("TASK-1", "todo/TASK-1.md")
	t.Setenv("QUEUE_JSON", queueJSON(item))
	var output bytes.Buffer
	if err := Verdict(context.Background(), repoRoot, "alternate-board", &output); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(argv); err != nil || string(got) != "queue\x00--dir\x00alternate-board\x00--json\x00" {
		t.Fatalf("queue argv=%q err=%v", got, err)
	}
	if got, err := os.ReadFile(cwd); err != nil || strings.TrimSpace(string(got)) != repoRoot {
		t.Fatalf("queue cwd=%q err=%v want=%q", got, err, repoRoot)
	}
	want := `{"verdictVersion":1,"state":"candidate","runnableCount":1,"agentRunnableCount":1,"runnable":[` + string(item) + `],"agentRunnable":[` + string(item) + `],"candidate":` + string(item) + "}\n"
	if output.String() != want {
		t.Fatalf("verdict bytes=%q\nwant=%q", output.String(), want)
	}
}

func TestVerdictTimesOut(t *testing.T) {
	bin := t.TempDir()
	writeExecutable(t, filepath.Join(bin, "taskchain-task-manager"), "#!/bin/sh\nsleep 1\n")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	err := verdictWithTimeout(context.Background(), t.TempDir(), "tasks", &bytes.Buffer{}, time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "task queue timed out") {
		t.Fatalf("verdict error=%v", err)
	}
}

func TestStartAuthorizedPinUsesRepositoryRootAndExactArguments(t *testing.T) {
	bin, repoRoot := t.TempDir(), t.TempDir()
	queueArgv, queueCWD := filepath.Join(t.TempDir(), "queue-argv"), filepath.Join(t.TempDir(), "queue-cwd")
	ceArgv, ceCWD := filepath.Join(t.TempDir(), "ce-argv"), filepath.Join(t.TempDir(), "ce-cwd")
	queuePath := filepath.Join(bin, "taskchain-task-manager")
	queueScript := "#!/bin/sh\nprintf '%s\\000' \"$@\" > \"$QUEUE_ARGV\"\npwd > \"$QUEUE_CWD\"\nprintf '%s' \"$QUEUE_JSON\"\n"
	writeExecutable(t, queuePath, queueScript)
	writeExecutable(t, filepath.Join(bin, "ce"), "#!/bin/sh\nprintf '%s\\000' \"$@\" > \"$CE_ARGV\"\npwd > \"$CE_CWD\"\nprintf '%s\\n' '{\"status\":\"ACTIVE\"}'\n")
	digest := sha256.Sum256([]byte(queueScript))
	pins := artifactPins{SchemaVersion: 1, Artifacts: []artifactPin{approvedTestPin(hex.EncodeToString(digest[:]))}}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("QUEUE_ARGV", queueArgv)
	t.Setenv("QUEUE_CWD", queueCWD)
	t.Setenv("CE_ARGV", ceArgv)
	t.Setenv("CE_CWD", ceCWD)
	candidate := queueItemJSON("ISSUE-0042", "issue/ISSUE-0042.md")
	t.Setenv("QUEUE_JSON", queueJSON(candidate))
	var output bytes.Buffer
	if err := startWithPins(context.Background(), repoRoot, "feat", &output, pins); err != nil {
		t.Fatal(err)
	}
	assertFile(t, queueArgv, "queue\x00--dir\x00tasks\x00--json\x00")
	assertFile(t, ceArgv, "task\x00run-start\x00issue-42\x00--type\x00feat\x00--json\x00")
	assertFile(t, queueCWD, repoRoot+"\n")
	assertFile(t, ceCWD, repoRoot+"\n")
	if output.String() != "{\"status\":\"ACTIVE\"}\n" {
		t.Fatalf("start output=%q", output.String())
	}
}

func TestStartProductionPinDoesNotRunQueue(t *testing.T) {
	bin := t.TempDir()
	queueCalled, ceCalled := filepath.Join(t.TempDir(), "queue-called"), filepath.Join(t.TempDir(), "ce-called")
	writeExecutable(t, filepath.Join(bin, "taskchain-task-manager"), "#!/bin/sh\ntouch \"$QUEUE_CALLED\"\n")
	writeExecutable(t, filepath.Join(bin, "ce"), "#!/bin/sh\ntouch \"$CE_CALLED\"\n")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("QUEUE_CALLED", queueCalled)
	t.Setenv("CE_CALLED", ceCalled)
	err := Start(context.Background(), t.TempDir(), "feat", &bytes.Buffer{})
	want := "no authorized TaskChain binary pin"
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		want = "SHA-256 mismatch"
	}
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("Start error=%v", err)
	}
	for _, marker := range []string{queueCalled, ceCalled} {
		if _, statErr := os.Stat(marker); !os.IsNotExist(statErr) {
			t.Fatalf("unexpected subprocess marker %s: %v", marker, statErr)
		}
	}
}

func TestStartHashMismatchDoesNotRunQueueOrCE(t *testing.T) {
	bin, repoRoot := t.TempDir(), t.TempDir()
	queueCalled, ceCalled := filepath.Join(t.TempDir(), "queue-called"), filepath.Join(t.TempDir(), "ce-called")
	writeExecutable(t, filepath.Join(bin, "taskchain-task-manager"), "#!/bin/sh\ntouch \"$QUEUE_CALLED\"\n")
	writeExecutable(t, filepath.Join(bin, "ce"), "#!/bin/sh\ntouch \"$CE_CALLED\"\n")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("QUEUE_CALLED", queueCalled)
	t.Setenv("CE_CALLED", ceCalled)
	pins := artifactPins{SchemaVersion: 1, Artifacts: []artifactPin{approvedTestPin(strings.Repeat("a", 64))}}
	err := startWithPins(context.Background(), repoRoot, "feat", &bytes.Buffer{}, pins)
	if err == nil || !strings.Contains(err.Error(), "SHA-256 mismatch") {
		t.Fatalf("start error=%v", err)
	}
	for _, marker := range []string{queueCalled, ceCalled} {
		if _, err := os.Stat(marker); !os.IsNotExist(err) {
			t.Fatalf("unexpected subprocess marker %s: %v", marker, err)
		}
	}
}

func TestParseQueueRejectsDuplicateKeys(t *testing.T) {
	data := []byte(`{"outputVersion":1,"outputVersion":1,"runnableCount":0,"agentRunnableCount":0,"runnable":[],"agentRunnable":[]}`)
	if _, err := parseQueue(data); err == nil || !strings.Contains(err.Error(), "duplicate JSON object key") {
		t.Fatalf("parse error=%v", err)
	}
}

func approvedTestPin(digest string) artifactPin {
	return artifactPin{GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, SourceCommit: strings.Repeat("a", 40), SourceTree: strings.Repeat("b", 40), GoVersion: "go-test", CGOEnabled: "0", BuildCommand: "go build ./cmd/taskchain-task-manager", Status: "published-approved", Distribution: "published", SHA256: digest, MutationAuthorized: true}
}
func queueItemJSON(id, boardPath string) json.RawMessage {
	value := queueItem{Path: boardPath, ExecutionMode: "implementation", NeedsHuman: false, AllowedPaths: []string{"internal/taskqueue"}}
	value.Card.ID = id
	result, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return result
}
func queueJSON(item json.RawMessage) string {
	return `{"outputVersion":1,"runnableCount":1,"agentRunnableCount":1,"runnable":[` + string(item) + `],"agentRunnable":[` + string(item) + `]}`
}
func writeExecutable(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o700); err != nil {
		t.Fatal(err)
	}
}
func assertFile(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil || string(got) != want {
		t.Fatalf("file %s = %q err=%v want=%q", path, got, err, want)
	}
}
