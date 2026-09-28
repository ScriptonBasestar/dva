package taskqueue

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// These are the former taskqueueverdict tests, retained in the package that
// now owns validation, classification, pinning, and CE invocation.
func TestClassifyStates(t *testing.T) {
	implementation := legacyItem("TASK-1", "doing/1.md", "implementation", false, "internal/cli")
	human := legacyItem("TASK-2", "todo/2.md", "external", true)
	for _, tc := range []struct {
		name, state     string
		runnable, agent []json.RawMessage
		candidate       bool
	}{
		{name: "empty", state: "empty"},
		{name: "human required", state: "human_required", runnable: []json.RawMessage{human}},
		{name: "candidate", state: "candidate", runnable: []json.RawMessage{implementation}, agent: []json.RawMessage{implementation}, candidate: true},
		{name: "selection required", state: "selection_required", runnable: []json.RawMessage{implementation, legacyItem("TASK-3", "todo/3.md", "implementation", false, "internal/config")}, agent: []json.RawMessage{implementation, legacyItem("TASK-3", "todo/3.md", "implementation", false, "internal/config")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := classify(queue{OutputVersion: 1, Runnable: tc.runnable, AgentRunnable: tc.agent, RunnableCount: len(tc.runnable), AgentRunnableCount: len(tc.agent)})
			if v.State != tc.state || v.VerdictVersion != 1 || v.RunnableCount != len(tc.runnable) || v.AgentRunnableCount != len(tc.agent) {
				t.Fatalf("verdict=%+v", v)
			}
			if tc.candidate != !bytes.Equal(v.Candidate, []byte("null")) {
				t.Fatalf("candidate=%s", v.Candidate)
			}
			if tc.candidate && !bytes.Equal(v.Candidate, implementation) {
				t.Fatalf("candidate=%s want=%s", v.Candidate, implementation)
			}
		})
	}
}

func TestParseQueueRejectsInvalidInvariants(t *testing.T) {
	implementation := string(legacyItem("TASK-1", "doing/1.md", "implementation", false, "internal/cli"))
	human := string(legacyItem("TASK-2", "todo/2.md", "external", true))
	valid := legacyQueueJSON(1, 1, 1, "["+implementation+"]", "["+implementation+"]")
	for _, tc := range []struct{ name, value string }{
		{"malformed", "{"}, {"trailing value", valid + " {}"}, {"unsupported version", legacyQueueJSON(2, 1, 1, "["+implementation+"]", "["+implementation+"]")},
		{"duplicate top key", `{"outputVersion":1,"outputVersion":2,"runnableCount":0,"agentRunnableCount":0,"runnable":[],"agentRunnable":[]}`},
		{"duplicate nested key", `{"outputVersion":1,"runnableCount":1,"agentRunnableCount":1,"runnable":[{"path":"todo/TASK-1.md","card":{"id":"TASK-1","id":"TASK-2"},"needsHuman":false,"executionMode":"implementation","allowedPaths":["internal/cli"]}],"agentRunnable":[{"path":"todo/TASK-1.md","card":{"id":"TASK-1","id":"TASK-2"},"needsHuman":false,"executionMode":"implementation","allowedPaths":["internal/cli"]}]}`},
		{"null runnable", legacyQueueJSON(1, 0, 0, "null", "[]")}, {"null count", `{"outputVersion":1,"runnableCount":null,"agentRunnableCount":0,"runnable":[],"agentRunnable":[]}`},
		{"count mismatch", legacyQueueJSON(1, 2, 1, "["+implementation+"]", "["+implementation+"]")}, {"omitted projection", `{"outputVersion":1,"runnableCount":1,"agentRunnableCount":0,"runnable":[` + implementation + `],"agentRunnable":[]}`},
		{"duplicate id", legacyQueueJSON(1, 2, 0, "["+implementation+","+implementation+"]", "[]")},
		{"invalid id", legacyQueueJSON(1, 1, 0, "["+string(legacyItem("not-a-task", "todo/1.md", "implementation", false, "internal/cli"))+"]", "[]")},
		{"non queue id", legacyQueueJSON(1, 1, 0, "["+string(legacyItem("PLAN-1", "todo/1.md", "implementation", false, "internal/cli"))+"]", "[]")},
		{"alias id", `{"outputVersion":1,"runnableCount":2,"agentRunnableCount":0,"runnable":[` + string(legacyItem("TASK-01", "todo/01.md", "implementation", true, "internal/cli")) + `,` + string(legacyItem("TASK-1", "todo/1.md", "implementation", true, "internal/cli")) + `],"agentRunnable":[]}`},
		{"unsafe board path", legacyQueueJSON(1, 1, 0, "["+string(legacyItem("TASK-1", "../../outside", "implementation", false, "internal/cli"))+"]", "[]")},
		{"windows board path", legacyQueueJSON(1, 1, 0, "["+string(legacyItem("TASK-1", "C:/outside", "implementation", false, "internal/cli"))+"]", "[]")},
		{"agent absent", legacyQueueJSON(1, 1, 1, "["+implementation+"]", "["+human+"]")},
		{"agent human", legacyQueueJSON(1, 1, 1, "["+human+"]", "["+human+"]")},
		{"agent external", legacyQueueJSON(1, 1, 1, "["+string(legacyItem("TASK-4", "todo/4.md", "external", false))+"]", "["+string(legacyItem("TASK-4", "todo/4.md", "external", false))+"]")},
		{"unsafe allowed", legacyQueueJSON(1, 1, 1, "["+string(legacyItem("TASK-5", "todo/5.md", "implementation", false, "../secret"))+"]", "["+string(legacyItem("TASK-5", "todo/5.md", "implementation", false, "../secret"))+"]")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parseQueue([]byte(tc.value)); err == nil {
				t.Fatal("parse accepted invalid queue")
			}
		})
	}
}

func TestLoadQueueForwardsExactInvocationAndDoesNotMutateBoard(t *testing.T) {
	bin, repoRoot, board := t.TempDir(), t.TempDir(), t.TempDir()
	argv := filepath.Join(t.TempDir(), "argv")
	writeExecutable(t, filepath.Join(bin, "taskchain-task-manager"), "#!/bin/sh\nprintf '%s\\000' \"$@\" > \"$QUEUE_ARGV\"\nprintf '%s' \"$QUEUE_JSON\"\n")
	boardFile := filepath.Join(board, "unchanged")
	if err := os.WriteFile(boardFile, []byte("board bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(boardFile)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("QUEUE_ARGV", argv)
	implementation := string(legacyItem("TASK-1", "doing/1.md", "implementation", false, "internal/cli"))
	t.Setenv("QUEUE_JSON", legacyQueueJSON(1, 1, 1, "["+implementation+"]", "["+implementation+"]"))
	if _, err := loadQueueBinary(context.Background(), repoRoot, board, "taskchain-task-manager"); err != nil {
		t.Fatal(err)
	}
	assertFile(t, argv, "queue\x00--dir\x00"+board+"\x00--json\x00")
	after, err := os.ReadFile(boardFile)
	if err != nil || !bytes.Equal(after, before) {
		t.Fatalf("board changed=%q err=%v", after, err)
	}
}

func TestStartRefusesNonCandidateAndPreservesCEFailureOutput(t *testing.T) {
	bin, repoRoot := t.TempDir(), t.TempDir()
	queueArgs, ceArgs := filepath.Join(t.TempDir(), "queue"), filepath.Join(t.TempDir(), "ce")
	queueScript := "#!/bin/sh\nprintf '%s\\000' \"$@\" > \"$QUEUE_ARGV\"\nprintf '%s' \"$QUEUE_JSON\"\n"
	writeExecutable(t, filepath.Join(bin, "taskchain-task-manager"), queueScript)
	writeExecutable(t, filepath.Join(bin, "ce"), "#!/bin/sh\nprintf '%s\\000' \"$@\" > \"$CE_ARGV\"\nprintf '%s' \"$CE_STDOUT\"\nprintf '%s' \"$CE_STDERR\" >&2\nexit \"${CE_EXIT:-0}\"\n")
	digest := sha256.Sum256([]byte(queueScript))
	pins := artifactPins{SchemaVersion: 1, Artifacts: []artifactPin{approvedTestPin(hex.EncodeToString(digest[:]))}}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("QUEUE_ARGV", queueArgs)
	t.Setenv("CE_ARGV", ceArgs)
	candidate := string(legacyItem("ISSUE-0042", "issue/42.md", "implementation", false, "internal/taskqueue"))
	human := string(legacyItem("TASK-2", "todo/2.md", "external", true))
	for _, tc := range []struct {
		name, typ, queue, out, stderr, exit, errText string
		callCE                                       bool
	}{
		{"candidate failure", "docs", legacyQueueJSON(1, 1, 1, "["+candidate+"]", "["+candidate+"]"), "{\"status\":\"BLOCKED\"}\n", "blocked\n", "1", "CE run-start failed", true},
		{"human required", "fix", legacyQueueJSON(1, 1, 0, "["+human+"]", "[]"), "", "", "", `queue verdict is "human_required"`, false},
		{"empty", "test", legacyQueueJSON(1, 0, 0, "[]", "[]"), "", "", "", `queue verdict is "empty"`, false},
		{"selection", "chore", legacyQueueJSON(1, 2, 2, "["+candidate+","+string(legacyItem("TASK-43", "todo/43.md", "implementation", false, "internal/taskqueue"))+"]", "["+candidate+","+string(legacyItem("TASK-43", "todo/43.md", "implementation", false, "internal/taskqueue"))+"]"), "", "", "", `queue verdict is "selection_required"`, false},
		{"invalid type", "deploy", legacyQueueJSON(1, 1, 1, "["+candidate+"]", "["+candidate+"]"), "", "", "", "invalid task branch type", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_ = os.Remove(queueArgs)
			_ = os.Remove(ceArgs)
			t.Setenv("QUEUE_JSON", tc.queue)
			t.Setenv("CE_STDOUT", tc.out)
			t.Setenv("CE_STDERR", tc.stderr)
			t.Setenv("CE_EXIT", tc.exit)
			var output bytes.Buffer
			err := startWithPins(context.Background(), repoRoot, tc.typ, &output, pins)
			if err == nil || !strings.Contains(err.Error(), tc.errText) {
				t.Fatalf("error=%v want=%q", err, tc.errText)
			}
			if output.String() != tc.out {
				t.Fatalf("output=%q want=%q", output.String(), tc.out)
			}
			_, ceErr := os.Stat(ceArgs)
			if (ceErr == nil) != tc.callCE {
				t.Fatalf("CE called=%t", ceErr == nil)
			}
		})
	}
}

func TestSafeAllowedPath(t *testing.T) {
	for _, tc := range []struct {
		path string
		want bool
	}{{"internal/cli", true}, {"dva.yml", true}, {"docs/{draft}.md", true}, {"", false}, {"tasks/card.md", false}, {"/absolute", false}, {"../parent", false}, {"a/../b", false}, {"a/*", false}, {"a\\b", false}, {"a b", false}, {"C:/drive", false}, {"a/./b", false}, {"a//b", false}} {
		if got := safeAllowedPath(tc.path); got != tc.want {
			t.Errorf("safeAllowedPath(%q)=%t want=%t", tc.path, got, tc.want)
		}
	}
}

func TestPinSnapshotBindsSelectedBytes(t *testing.T) {
	bin := t.TempDir()
	first := filepath.Join(bin, "taskchain-task-manager")
	verified := []byte("#!/bin/sh\nprintf verified\n")
	writeExecutable(t, first, string(verified))
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	sha := sha256.Sum256(verified)
	snapshot, cleanup, err := pinnedQueueBinary(artifactPin{GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, SHA256: hex.EncodeToString(sha[:]), MutationAuthorized: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(first, []byte("#!/bin/sh\nprintf substituted\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command(snapshot).Output()
	if err != nil || string(output) != "verified" {
		t.Fatalf("snapshot=%q err=%v", output, err)
	}
	if err := cleanup(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(snapshot); !os.IsNotExist(err) {
		t.Fatalf("snapshot remains=%v", err)
	}
	if _, _, err := pinnedQueueBinary(artifactPin{GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, SHA256: hex.EncodeToString(sha[:])}); err == nil || !strings.Contains(err.Error(), "SHA-256 mismatch") {
		t.Fatalf("substitution error=%v", err)
	}
}

func TestPinFirstPathResultCannotFallThrough(t *testing.T) {
	firstDir, secondDir := t.TempDir(), t.TempDir()
	writeExecutable(t, filepath.Join(firstDir, "taskchain-task-manager"), "#!/bin/sh\nprintf bad\n")
	good := []byte("#!/bin/sh\nprintf good\n")
	writeExecutable(t, filepath.Join(secondDir, "taskchain-task-manager"), string(good))
	t.Setenv("PATH", firstDir+string(os.PathListSeparator)+secondDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	sha := sha256.Sum256(good)
	if _, _, err := pinnedQueueBinary(artifactPin{GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, SHA256: hex.EncodeToString(sha[:])}); err == nil || !strings.Contains(err.Error(), filepath.Join(firstDir, "taskchain-task-manager")) {
		t.Fatalf("first PATH mismatch=%v", err)
	}
}

func TestPinSymlinkRetargetKeepsSnapshot(t *testing.T) {
	bin := t.TempDir()
	verified, substituted, link := filepath.Join(bin, "verified"), filepath.Join(bin, "substituted"), filepath.Join(bin, "taskchain-task-manager")
	verifiedBytes := []byte("#!/bin/sh\nprintf verified\n")
	writeExecutable(t, verified, string(verifiedBytes))
	writeExecutable(t, substituted, "#!/bin/sh\nprintf substituted\n")
	if err := os.Symlink(verified, link); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	sha := sha256.Sum256(verifiedBytes)
	snapshot, cleanup, err := pinnedQueueBinary(artifactPin{GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, SHA256: hex.EncodeToString(sha[:])})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(substituted, link); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command(snapshot).Output()
	if err != nil || string(output) != "verified" {
		t.Fatalf("output=%q err=%v", output, err)
	}
	if err := cleanup(); err != nil {
		t.Fatal(err)
	}
}

func TestPinNoPlatformOrMalformedManifest(t *testing.T) {
	for _, pins := range []artifactPins{{SchemaVersion: 1, Artifacts: []artifactPin{{GOOS: "unsupported", GOARCH: runtime.GOARCH, MutationAuthorized: true}}}, {SchemaVersion: 0}, {SchemaVersion: 1, Artifacts: []artifactPin{{GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, MutationAuthorized: true, SHA256: "wrong"}}}} {
		if _, err := pins.activeForPlatform(); err == nil {
			t.Fatalf("invalid pin accepted=%+v", pins)
		}
	}
}

func TestPinCandidateCannotBeActivatedByBooleanAlone(t *testing.T) {
	pins := embeddedPins
	if pins.loadErr != nil {
		t.Fatal(pins.loadErr)
	}
	pins.Artifacts = append([]artifactPin(nil), pins.Artifacts...)
	for i := range pins.Artifacts {
		if pins.Artifacts[i].GOOS == runtime.GOOS && pins.Artifacts[i].GOARCH == runtime.GOARCH {
			pins.Artifacts[i].MutationAuthorized = true
			if _, err := pins.activeForPlatform(); err == nil || !strings.Contains(err.Error(), "lacks approved release/build provenance") {
				t.Fatalf("candidate activated=%v", err)
			}
			return
		}
	}
	if _, err := pins.activeForPlatform(); err == nil {
		t.Fatal("unlisted platform accepted")
	}
}

func legacyItem(id, boardPath, mode string, human bool, allowed ...string) json.RawMessage {
	if allowed == nil {
		allowed = []string{}
	}
	value := queueItem{Path: boardPath, ExecutionMode: mode, NeedsHuman: human, AllowedPaths: allowed}
	value.Card.ID = id
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}
func legacyQueueJSON(version, runnableCount, agentCount int, runnable, agent string) string {
	return `{"outputVersion":` + strconv.Itoa(version) + `,"runnableCount":` + strconv.Itoa(runnableCount) + `,"agentRunnableCount":` + strconv.Itoa(agentCount) + `,"runnable":` + runnable + `,"agentRunnable":` + agent + `}`
}
