package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClassifyStates(t *testing.T) {
	implementation := item("TASK-1", "tasks/doing/1.md", "internal/cli", false)
	human := item("TASK-2", "tasks/todo/2.md", "", true)
	for _, tc := range []struct {
		name      string
		runnable  []json.RawMessage
		agent     []json.RawMessage
		state     string
		candidate bool
	}{
		{name: "empty", state: "empty"},
		{name: "human required", runnable: []json.RawMessage{human}, state: "human_required"},
		{name: "candidate", runnable: []json.RawMessage{implementation}, agent: []json.RawMessage{implementation}, state: "candidate", candidate: true},
		{name: "selection required", runnable: []json.RawMessage{implementation, item("TASK-3", "tasks/todo/3.md", "internal/config", false)}, agent: []json.RawMessage{implementation, item("TASK-3", "tasks/todo/3.md", "internal/config", false)}, state: "selection_required"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q := queue{OutputVersion: 1, Runnable: tc.runnable, AgentRunnable: tc.agent, RunnableCount: len(tc.runnable), AgentRunnableCount: len(tc.agent)}
			v := classify(q)
			if v.State != tc.state || v.VerdictVersion != 1 || v.RunnableCount != len(tc.runnable) || v.AgentRunnableCount != len(tc.agent) {
				t.Fatalf("verdict = %+v", v)
			}
			if tc.candidate != !bytes.Equal(v.Candidate, []byte("null")) {
				t.Fatalf("candidate = %s, candidate present = %t", v.Candidate, tc.candidate)
			}
			if tc.candidate && !bytes.Equal(v.Candidate, implementation) {
				t.Fatalf("candidate = %s, want %s", v.Candidate, implementation)
			}
		})
	}
}

func TestParseQueueRejectsInvalidInvariants(t *testing.T) {
	implementation := string(item("TASK-1", "tasks/doing/1.md", "internal/cli", false))
	human := string(item("TASK-2", "tasks/todo/2.md", "", true))
	valid := queueJSON(1, 1, "["+implementation+"]", "["+implementation+"]")
	for _, tc := range []struct {
		name string
		json string
	}{
		{name: "malformed", json: "{"},
		{name: "trailing value", json: valid + " {}"},
		{name: "unsupported version", json: queueJSON(2, 1, "["+implementation+"]", "["+implementation+"]")},
		{name: "duplicate top level key", json: `{"outputVersion":1,"outputVersion":2,"runnableCount":0,"agentRunnableCount":0,"runnable":[],"agentRunnable":[]}`},
		{name: "duplicate candidate key", json: `{"outputVersion":1,"runnableCount":1,"agentRunnableCount":1,"runnable":[{"path":"todo/TASK-1.md","card":{"id":"TASK-1","id":"TASK-2"},"needsHuman":false,"executionMode":"implementation","allowedPaths":["internal/cli"]}],"agentRunnable":[{"path":"todo/TASK-1.md","card":{"id":"TASK-1","id":"TASK-2"},"needsHuman":false,"executionMode":"implementation","allowedPaths":["internal/cli"]}]}`},
		{name: "null runnable", json: queueJSON(1, 0, "null", "[]")},
		{name: "null count", json: `{"outputVersion":1,"runnableCount":null,"agentRunnableCount":0,"runnable":[],"agentRunnable":[]}`},
		{name: "count mismatch", json: queueJSON(1, 2, "["+implementation+"]", "["+implementation+"]")},
		{name: "omitted agent projection", json: `{"outputVersion":1,"runnableCount":1,"agentRunnableCount":0,"runnable":[` + implementation + `],"agentRunnable":[]}`},
		{name: "duplicate runnable id", json: queueJSON(1, 2, "["+implementation+","+implementation+"]", "[]")},
		{name: "invalid card id", json: queueJSON(1, 1, "["+string(item("not-a-task", "todo/1.md", "implementation", false, "internal/cli"))+"]", "[]")},
		{name: "non-queue card id", json: queueJSON(1, 1, "["+string(item("PLAN-1", "todo/1.md", "implementation", false, "internal/cli"))+"]", "[]")},
		{name: "alias duplicate id", json: `{"outputVersion":1,"runnableCount":2,"agentRunnableCount":0,"runnable":[` + string(item("TASK-01", "todo/01.md", "implementation", true, "internal/cli")) + `,` + string(item("TASK-1", "todo/1.md", "implementation", true, "internal/cli")) + `],"agentRunnable":[]}`},
		{name: "unsafe board path", json: queueJSON(1, 1, "["+string(item("TASK-1", "../../outside", "implementation", false, "internal/cli"))+"]", "[]")},
		{name: "Windows drive board path", json: queueJSON(1, 1, "["+string(item("TASK-1", "C:/outside", "implementation", false, "internal/cli"))+"]", "[]")},
		{name: "agent item absent from runnable", json: queueJSON(1, 1, "["+implementation+"]", "["+human+"]")},
		{name: "agent needs human", json: queueJSON(1, 1, "["+human+"]", "["+human+"]")},
		{name: "agent external", json: queueJSON(1, 1, "["+string(item("TASK-4", "tasks/todo/4.md", "external", false))+"]", "["+string(item("TASK-4", "tasks/todo/4.md", "external", false))+"]")},
		{name: "unsafe allowed path", json: queueJSON(1, 1, "["+string(item("TASK-5", "tasks/todo/5.md", "implementation", false, "../secret"))+"]", "["+string(item("TASK-5", "tasks/todo/5.md", "implementation", false, "../secret"))+"]")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parseQueue([]byte(tc.json)); err == nil {
				t.Fatalf("parseQueue(%s) succeeded", tc.json)
			}
		})
	}
}

func TestLoadQueueForwardsExactInvocationAndDoesNotMutateBoard(t *testing.T) {
	bin := t.TempDir()
	argv := filepath.Join(t.TempDir(), "argv")
	stub := filepath.Join(bin, "taskchain-task-manager")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\nprintf '%s\\000' \"$@\" > \"$TASK_QUEUE_ARGV\"\nprintf '%s' \"$TASK_QUEUE_JSON\"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	board := t.TempDir()
	boardFile := filepath.Join(board, "unchanged")
	if err := os.WriteFile(boardFile, []byte("board bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(boardFile)
	if err != nil {
		t.Fatal(err)
	}
	implementation := string(item("TASK-1", "tasks/doing/1.md", "implementation", false, "internal/cli"))
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("TASK_QUEUE_ARGV", argv)
	t.Setenv("TASK_QUEUE_JSON", queueJSON(1, 1, "["+implementation+"]", "["+implementation+"]"))
	if _, err := loadQueue(context.Background(), board); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(argv); err != nil || !bytes.Equal(got, []byte("queue\x00--dir\x00"+board+"\x00--json\x00")) {
		t.Fatalf("argv = %q, err=%v", got, err)
	}
	if after, err := os.ReadFile(boardFile); err != nil || !bytes.Equal(after, before) {
		t.Fatalf("board changed: %q, err=%v", after, err)
	}
}

func TestExecuteStartTypeBridge(t *testing.T) {
	bin := t.TempDir()
	queueArgs := filepath.Join(t.TempDir(), "queue-argv")
	ceArgs := filepath.Join(t.TempDir(), "ce-argv")
	ceCWD := filepath.Join(t.TempDir(), "ce-cwd")
	queueStub := filepath.Join(bin, "taskchain-task-manager")
	ceStub := filepath.Join(bin, "ce")
	if err := os.WriteFile(queueStub, []byte("#!/bin/sh\nprintf '%s\\000' \"$@\" > \"$TASK_QUEUE_ARGV\"\nprintf '%s' \"$TASK_QUEUE_JSON\"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	queueBytes, err := os.ReadFile(queueStub)
	if err != nil {
		t.Fatal(err)
	}
	queueDigest := sha256.Sum256(queueBytes)
	testPins := artifactPins{SchemaVersion: 1, Artifacts: []artifactPin{approvedTestPin(hex.EncodeToString(queueDigest[:]))}}
	if err := os.WriteFile(ceStub, []byte("#!/bin/sh\nprintf '%s\\000' \"$@\" >> \"$CE_ARGV\"\npwd > \"$CE_CWD\"\nprintf '%s' \"$CE_STDOUT\"\nprintf '%s' \"$CE_STDERR\" >&2\nexit \"${CE_EXIT:-0}\"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("TASK_QUEUE_ARGV", queueArgs)
	t.Setenv("CE_ARGV", ceArgs)
	t.Setenv("CE_CWD", ceCWD)

	candidate := string(item("ISSUE-0042", "tasks/issue/42.md", "implementation", false, "tools/taskqueueverdict"))
	secondCandidate := string(item("TASK-43", "tasks/todo/43.md", "implementation", false, "tools/taskqueueverdict"))
	human := string(item("TASK-2", "tasks/todo/2.md", "external", true))
	for _, tc := range []struct {
		name         string
		dir          string
		startType    string
		queueJSON    string
		ceStdout     string
		ceStderr     string
		ceExit       string
		wantErr      bool
		wantErrText  string
		wantCE       bool
		wantOutput   string
		queueInvoked bool
	}{
		{
			name:         "read only without type",
			queueJSON:    queueJSON(1, 1, "["+candidate+"]", "["+candidate+"]"),
			wantOutput:   `"state":"candidate"`,
			queueInvoked: true,
		},
		{
			name:         "candidate starts normalized issue key",
			startType:    "feat",
			queueJSON:    queueJSON(1, 1, "["+candidate+"]", "["+candidate+"]"),
			ceStdout:     "{\"schemaVersion\":2,\"status\":\"ACTIVE\"}\n",
			wantCE:       true,
			wantOutput:   "{\"schemaVersion\":2,\"status\":\"ACTIVE\"}\n",
			queueInvoked: true,
		},
		{
			name:         "human required does not start",
			startType:    "fix",
			queueJSON:    queueJSONCounts(1, 1, 0, "["+human+"]", "[]"),
			wantErr:      true,
			wantErrText:  `queue verdict is "human_required"`,
			queueInvoked: true,
		},
		{
			name:         "empty does not start",
			startType:    "test",
			queueJSON:    queueJSON(1, 0, "[]", "[]"),
			wantErr:      true,
			wantErrText:  `queue verdict is "empty"`,
			queueInvoked: true,
		},
		{
			name:         "selection required does not start",
			startType:    "chore",
			queueJSON:    queueJSON(1, 2, "["+candidate+","+secondCandidate+"]", "["+candidate+","+secondCandidate+"]"),
			wantErr:      true,
			wantErrText:  `queue verdict is "selection_required"`,
			queueInvoked: true,
		},
		{
			name:         "invalid queue does not start",
			startType:    "perf",
			queueJSON:    "{",
			wantErr:      true,
			queueInvoked: true,
		},
		{
			name:         "invalid type does not query queue",
			startType:    "deploy",
			queueJSON:    queueJSON(1, 1, "["+candidate+"]", "["+candidate+"]"),
			wantErr:      true,
			queueInvoked: false,
		},
		{
			name:         "alternate board does not query queue or start CE",
			dir:          "../other-board",
			startType:    "feat",
			queueJSON:    queueJSON(1, 1, "["+candidate+"]", "["+candidate+"]"),
			wantErr:      true,
			queueInvoked: false,
		},
		{
			name:         "CE failure preserves structured response",
			startType:    "docs",
			queueJSON:    queueJSON(1, 1, "["+candidate+"]", "["+candidate+"]"),
			ceStdout:     "{\"status\":\"BLOCKED\",\"receipt\":{\"worktree\":\"/tmp/created\"}}\n",
			ceStderr:     "blocked\n",
			ceExit:       "1",
			wantErr:      true,
			wantCE:       true,
			wantOutput:   "{\"status\":\"BLOCKED\",\"receipt\":{\"worktree\":\"/tmp/created\"}}\n",
			queueInvoked: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_ = os.Remove(queueArgs)
			_ = os.Remove(ceArgs)
			_ = os.Remove(ceCWD)
			t.Setenv("TASK_QUEUE_JSON", tc.queueJSON)
			t.Setenv("CE_STDOUT", tc.ceStdout)
			t.Setenv("CE_STDERR", tc.ceStderr)
			t.Setenv("CE_EXIT", tc.ceExit)
			var output bytes.Buffer
			dir := tc.dir
			if dir == "" {
				dir = "tasks"
			}
			err := executeWithPins(context.Background(), dir, tc.startType, &output, testPins)
			if (err != nil) != tc.wantErr {
				t.Fatalf("execute error = %v, want error = %t", err, tc.wantErr)
			}
			if tc.wantErrText != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErrText)) {
				t.Fatalf("execute error = %v, want containing %q", err, tc.wantErrText)
			}
			if tc.wantErr && tc.wantOutput == "" && output.Len() != 0 {
				t.Fatalf("error output = %q, want no success output", output.String())
			}
			if tc.wantOutput != "" && !strings.Contains(output.String(), tc.wantOutput) {
				t.Fatalf("output = %q, want containing %q", output.String(), tc.wantOutput)
			}
			_, queueErr := os.Stat(queueArgs)
			if (queueErr == nil) != tc.queueInvoked {
				t.Fatalf("queue invoked = %t, want %t (err=%v)", queueErr == nil, tc.queueInvoked, queueErr)
			}
			gotCE, ceErr := os.ReadFile(ceArgs)
			if (ceErr == nil) != tc.wantCE {
				t.Fatalf("CE invoked = %t, want %t (err=%v)", ceErr == nil, tc.wantCE, ceErr)
			}
			if tc.wantCE {
				want := "task\x00run-start\x00issue-42\x00--type\x00" + tc.startType + "\x00--json\x00"
				if string(gotCE) != want {
					t.Fatalf("CE argv = %q, want %q", gotCE, want)
				}
				gotCWD, err := os.ReadFile(ceCWD)
				if err != nil {
					t.Fatal(err)
				}
				wantCWD, err := os.Getwd()
				if err != nil {
					t.Fatal(err)
				}
				if strings.TrimSpace(string(gotCWD)) != wantCWD {
					t.Fatalf("CE cwd = %q, want %q", gotCWD, wantCWD)
				}
			}
		})
	}
}

func TestSafeAllowedPath(t *testing.T) {
	for _, tc := range []struct {
		path string
		want bool
	}{
		{"internal/cli", true}, {"dva.yml", true}, {"docs/{draft}.md", true}, {"", false}, {"tasks/card.md", false}, {"/absolute", false}, {"../parent", false}, {"a/../b", false}, {"a/*", false}, {"a\\b", false}, {"a b", false}, {"C:/drive", false}, {"a/./b", false}, {"a//b", false},
	} {
		if got := safeAllowedPath(tc.path); got != tc.want {
			t.Errorf("safeAllowedPath(%q) = %t, want %t", tc.path, got, tc.want)
		}
	}
}

func item(id, path, mode string, human bool, allowed ...string) json.RawMessage {
	if allowed == nil {
		allowed = []string{}
	}
	value := queueItem{Path: path, ExecutionMode: mode, NeedsHuman: human, AllowedPaths: allowed}
	value.Card.ID = id
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}

func queueJSON(version, count int, runnable, agent string) string {
	return queueJSONCounts(version, count, count, runnable, agent)
}

func queueJSONCounts(version, runnableCount, agentCount int, runnable, agent string) string {
	return fmt.Sprintf(`{"outputVersion":%d,"runnableCount":%d,"agentRunnableCount":%d,"runnable":%s,"agentRunnable":%s}`, version, runnableCount, agentCount, runnable, agent)
}
