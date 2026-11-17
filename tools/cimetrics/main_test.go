package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMetricsHostedSummaryIsSanitizedAndKeepsCheckUnknown(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "test.jsonl")
	output := filepath.Join(dir, "summary.json")
	secret := "sensitive-fixture-token"
	content := `{"Action":"run","Package":"github.com/ScriptonBasestar/dva/tools/cimetrics","Test":"TestMetricsHostedSummaryIsSanitizedAndKeepsCheckUnknown/` + secret + `"}` + "\n" +
		`{"Action":"output","Package":"github.com/ScriptonBasestar/dva/tools/cimetrics","Output":"` + secret + `"}` + "\n"
	if err := os.WriteFile(input, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_RUN_ID", "123456")
	t.Setenv("GITHUB_RUN_ATTEMPT", "2")
	t.Setenv("GITHUB_JOB", "test")
	t.Setenv("GITHUB_SHA", strings.Repeat("a", 40))
	var out, stderr bytes.Buffer
	r := execute([]string{"summarize", "--check", "unit", "--input", input, "--output", output}, &out, &stderr)
	if r.code != 0 {
		t.Fatalf("summarize: %+v %s", r, stderr.String())
	}
	b, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), secret) || strings.Contains(string(b), dir) {
		t.Fatalf("summary leaks input: %s", b)
	}
	var got struct {
		Outcome string `json:"check_outcome"`
		Status  string `json:"collection_status"`
		Hosted  struct {
			RunID   string `json:"run_id"`
			Attempt int    `json:"attempt"`
			Job     string `json:"job"`
			Source  string `json:"source_sha"`
		} `json:"hosted"`
	}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.Outcome != "unknown" || got.Status != "partial" || got.Hosted.RunID != "123456" || got.Hosted.Attempt != 2 || got.Hosted.Job != "test" || got.Hosted.Source != strings.Repeat("a", 40) {
		t.Fatalf("incorrect summary: %+v", got)
	}
	info, err := os.Stat(output)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("summary permissions: %v %v", info, err)
	}
	if r = execute([]string{"summarize", "--check", "unit", "--input", input, "--output", output}, &out, &stderr); r.code != 0 {
		t.Fatalf("auxiliary storage error changed verdict: %+v", r)
	}
	after, err := os.ReadFile(output)
	if err != nil || !bytes.Equal(b, after) {
		t.Fatal("existing summary was overwritten")
	}
	if !strings.Contains(stderr.String(), "summary storage degraded") {
		t.Fatalf("storage error hidden: %s", stderr.String())
	}
}

func TestMetricsHostedMissingInputProducesFailedObservationWithoutGateFailure(t *testing.T) {
	output := filepath.Join(t.TempDir(), "summary.json")
	var out, stderr bytes.Buffer
	r := execute([]string{"summarize", "--check", "integration", "--input", filepath.Join(t.TempDir(), "missing"), "--output", output}, &out, &stderr)
	if r.code != 0 {
		t.Fatalf("supplementary input failure changed check outcome: %+v", r)
	}
	b, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Status  string `json:"collection_status"`
		Outcome string `json:"check_outcome"`
	}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.Status != "failed" || got.Outcome != "unknown" {
		t.Fatalf("missing input misclassified: %+v", got)
	}
	if !strings.Contains(stderr.String(), "partial or unavailable") {
		t.Fatalf("unavailable summary hidden: %s", stderr.String())
	}
}
