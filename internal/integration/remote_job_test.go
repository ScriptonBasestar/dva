//go:build integration && (darwin || linux)

package integration

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoteJobCLIUsesExactGitHubProviderArgvAndReceipt(t *testing.T) {
	root := t.TempDir()
	git := realTool(t, "git")
	for _, args := range [][]string{{"init", root}, {"-C", root, "remote", "add", "origin", "https://github.com/acme/widget.git"}} {
		if out, err := exec.Command(git, args...).CombinedOutput(); err != nil {
			t.Fatalf("git %q: %v: %s", args, err, out)
		}
	}
	const config = `jobs:
  postgres-extensions:
    provider: github-actions
    repository: acme/widget
    ref: artifact-source
    timeout: 1m
    inputs:
      pg_version: {default: "18", values: ["17", "18"]}
    runs:
      - {name: essential, workflow: essential.yml, inputs: {pg_version: "{{input.pg_version}}", variant: essential}}
      - {name: full, workflow: full.yml, inputs: {pg_version: "{{input.pg_version}}", variant: full}}
      - {name: vector, workflow: vector.yml, inputs: {pg_version: "{{input.pg_version}}", variant: vector}}
      - {name: postgis, workflow: postgis.yml, inputs: {pg_version: "{{input.pg_version}}", variant: postgis}}
`
	if err := os.WriteFile(filepath.Join(root, "dva.yml"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}

	bin, log := t.TempDir(), filepath.Join(t.TempDir(), "gh-argv")
	gh := filepath.Join(bin, "gh")
	const fakeGH = `#!/bin/sh
for arg in "$@"; do printf '%s\n' "$arg" >> "$GH_LOG"; done
printf '%s\n' --END-- >> "$GH_LOG"
case "$*" in
  *"/commits/"*) printf '%s\n' '{"sha":"0123456789abcdef0123456789abcdef01234567"}' ;;
  *"/actions/runs/4001"*) printf '%s\n' '{"id":4001,"event":"workflow_dispatch","head_sha":"0123456789abcdef0123456789abcdef01234567","status":"completed","conclusion":"success","html_url":"https://example.test/4001","head_branch":"artifact-source","repository":{"full_name":"acme/widget"},"workflow_id":101,"run_attempt":1}' ;;
  *"/actions/runs/4002"*) printf '%s\n' '{"id":4002,"event":"workflow_dispatch","head_sha":"0123456789abcdef0123456789abcdef01234567","status":"completed","conclusion":"success","html_url":"https://example.test/4002","head_branch":"artifact-source","repository":{"full_name":"acme/widget"},"workflow_id":102,"run_attempt":1}' ;;
  *"/actions/runs/4003"*) printf '%s\n' '{"id":4003,"event":"workflow_dispatch","head_sha":"0123456789abcdef0123456789abcdef01234567","status":"completed","conclusion":"success","html_url":"https://example.test/4003","head_branch":"artifact-source","repository":{"full_name":"acme/widget"},"workflow_id":103,"run_attempt":1}' ;;
  *"/actions/runs/4004"*) printf '%s\n' '{"id":4004,"event":"workflow_dispatch","head_sha":"0123456789abcdef0123456789abcdef01234567","status":"completed","conclusion":"success","html_url":"https://example.test/4004","head_branch":"artifact-source","repository":{"full_name":"acme/widget"},"workflow_id":104,"run_attempt":1}' ;;
  *"workflows/101/dispatches"*) printf '%s\n' '{"workflow_run_id":4001,"html_url":"https://example.test/4001"}' ;;
  *"workflows/102/dispatches"*) printf '%s\n' '{"workflow_run_id":4002,"html_url":"https://example.test/4002"}' ;;
  *"workflows/103/dispatches"*) printf '%s\n' '{"workflow_run_id":4003,"html_url":"https://example.test/4003"}' ;;
  *"workflows/104/dispatches"*) printf '%s\n' '{"workflow_run_id":4004,"html_url":"https://example.test/4004"}' ;;
  *"workflows/essential.yml"*) printf '%s\n' '{"id":101}' ;;
  *"workflows/full.yml"*) printf '%s\n' '{"id":102}' ;;
  *"workflows/vector.yml"*) printf '%s\n' '{"id":103}' ;;
  *"workflows/postgis.yml"*) printf '%s\n' '{"id":104}' ;;
  *) exit 64 ;;
esac
`
	if err := os.WriteFile(gh, []byte(fakeGH), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("GH_LOG", log)
	t.Setenv("XDG_STATE_HOME", filepath.Join(t.TempDir(), "state"))

	first := runRemoteDVA(t, root, "job", "run", "postgres-extensions", "--input", "pg_version=18", "--wait")
	var runEnvelope struct {
		Result struct {
			Job struct {
				ID   string `json:"id"`
				Runs []struct {
					Name       string `json:"name"`
					RunID      int64  `json:"run_id"`
					Status     string `json:"status"`
					Conclusion string `json:"conclusion"`
				} `json:"runs"`
			} `json:"job"`
		} `json:"result"`
	}
	if err := json.Unmarshal(first, &runEnvelope); err != nil {
		t.Fatalf("job run did not write one JSON document: %v\n%s", err, first)
	}
	if runEnvelope.Result.Job.ID == "" || len(runEnvelope.Result.Job.Runs) != 4 {
		t.Fatalf("job report = %s", first)
	}
	wantIDs := map[string]int64{"essential": 4001, "full": 4002, "vector": 4003, "postgis": 4004}
	for _, run := range runEnvelope.Result.Job.Runs {
		if wantIDs[run.Name] != run.RunID || run.Status != "completed" || run.Conclusion != "success" {
			t.Fatalf("run = %+v, want completed successful exact ID", run)
		}
	}
	assertFakeGHCalls(t, readFakeGHCalls(t, log), 4, true)

	second := runRemoteDVA(t, root, "job", "status", runEnvelope.Result.Job.ID)
	var statusEnvelope struct {
		Result struct {
			Runs []struct {
				RunID int64 `json:"run_id"`
			} `json:"runs"`
		} `json:"result"`
	}
	if err := json.Unmarshal(second, &statusEnvelope); err != nil || len(statusEnvelope.Result.Runs) != 4 {
		t.Fatalf("job status = %s, err=%v", second, err)
	}
	for i, run := range statusEnvelope.Result.Runs {
		if run.RunID != int64(4001+i) {
			t.Fatalf("status run IDs = %+v", statusEnvelope.Result.Runs)
		}
	}
	assertFakeGHCalls(t, readFakeGHCalls(t, log), 4, false)
}

func runRemoteDVA(t *testing.T, root string, args ...string) []byte {
	t.Helper()
	cmd := exec.Command(dvaBinary(t), args...)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("dva %q: %v\n%s", args, err, out)
	}
	return out
}

func readFakeGHCalls(t *testing.T, file string) [][]string {
	t.Helper()
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var calls [][]string
	var call []string
	for line := range strings.SplitSeq(strings.TrimSuffix(string(b), "\n"), "\n") {
		if line == "--END--" {
			calls, call = append(calls, call), nil
			continue
		}
		call = append(call, line)
	}
	return calls
}

func assertFakeGHCalls(t *testing.T, calls [][]string, wantDispatches int, initial bool) {
	t.Helper()
	dispatches := 0
	for _, call := range calls {
		if len(call) < 8 || call[0] != "api" || !containsArgPair(call, "--hostname", "github.com") || !containsArgPair(call, "-H", "X-GitHub-Api-Version: 2026-03-10") {
			t.Fatalf("unexpected gh argv: %q", call)
		}
		for _, arg := range call {
			if strings.Contains(arg, "/dispatches") {
				if !containsArgPair(call, "-X", "POST") || !containsArgPair(call, "-f", "ref=artifact-source") || !containsArgPair(call, "-f", "inputs[pg_version]=18") {
					t.Fatalf("dispatch lost method or public inputs: %q", call)
				}
				dispatches++
			}
		}
	}
	if dispatches != wantDispatches {
		t.Fatalf("dispatch calls = %d, want %d (initial=%v): %q", dispatches, wantDispatches, initial, calls)
	}
}

func containsArgPair(args []string, key, value string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == key && args[i+1] == value {
			return true
		}
	}
	return false
}
