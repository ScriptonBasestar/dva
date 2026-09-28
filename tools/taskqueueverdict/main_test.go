package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	value := queueItem{Path: path, ExecutionMode: mode, NeedsHuman: human, AllowedPaths: allowed}
	value.Card.ID = id
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}

func queueJSON(version, count int, runnable, agent string) string {
	return fmt.Sprintf(`{"outputVersion":%d,"runnableCount":%d,"agentRunnableCount":%d,"runnable":%s,"agentRunnable":%s}`, version, count, count, runnable, agent)
}
