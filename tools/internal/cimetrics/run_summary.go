package cimetrics

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

const maxHostedBytes = 64 * 1024 * 1024

// HostedSummary describes the observation of an existing JSONL stream. It
// does not claim the test command's outcome or elapsed wall time.
type HostedSummary struct {
	Schema      string    `json:"schema"`
	Kind        string    `json:"kind"`
	ID          string    `json:"measurement_id"`
	Check       string    `json:"check"`
	GeneratedAt time.Time `json:"generated_at"`
	Status      string    `json:"collection_status"`
	Reason      string    `json:"collection_reason,omitempty"`
	Outcome     string    `json:"check_outcome"`
	Events      *Snapshot `json:"events,omitempty"`
	Hosted      Hosted    `json:"hosted"`
}

// SummarizeFile never runs the check and never creates local private state.
// A missing, truncated, or oversized input remains a failed/partial observation.
func SummarizeFile(check, input string) (HostedSummary, error) {
	if check != "unit" && check != "integration" {
		return HostedSummary{}, errors.New("ci-metrics: unsupported hosted check")
	}
	id, err := newID()
	if err != nil {
		return HostedSummary{}, err
	}
	r := HostedSummary{Schema: recordSchema, Kind: "hosted_jsonl_summary", ID: id, Check: check, GeneratedAt: time.Now().UTC(), Status: "complete", Outcome: "unknown", Hosted: hostedInfo()}
	root, ok := findProjectRoot()
	var allowed map[string]map[string]bool
	if ok {
		allowed, err = LoadAllowed(root)
	}
	if !ok || err != nil {
		r.Status, r.Reason = "partial", "test allowlist unavailable"
	}
	f, err := openNoFollow(input)
	if err != nil {
		r.Status, r.Reason = "failed", "input unavailable"
		return r, nil
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		r.Status, r.Reason = "failed", "input is not a regular file"
		return r, nil
	}
	p := NewParser(allowed, nil)
	n, err := io.Copy(p, io.LimitReader(f, maxHostedBytes+1))
	p.Finish()
	snap := p.Snapshot()
	r.Events = &snap
	if err != nil {
		r.Status, r.Reason = "partial", "input read failed"
	} else if n > maxHostedBytes {
		r.Status, r.Reason = "partial", "input byte limit exceeded"
	} else if snap.Status != "complete" && r.Status == "complete" {
		r.Status, r.Reason = "partial", "incomplete event stream"
	}
	return r, nil
}

func hostedInfo() Hosted {
	h := Hosted{}
	if v := os.Getenv("GITHUB_SHA"); len(v) == 40 && isHex(v) {
		h.Source = v
	}
	run := os.Getenv("GITHUB_RUN_ID")
	if len(run) > 0 && len(run) <= 20 && decimal(run) {
		h.RunID = run
	}
	if v := os.Getenv("GITHUB_RUN_ATTEMPT"); len(v) > 0 && len(v) <= 5 && decimal(v) {
		if n, e := strconv.Atoi(v); e == nil && n > 0 {
			h.Attempt = n
		}
	}
	if os.Getenv("GITHUB_JOB") == "test" {
		h.Job = "test"
	}
	return h
}

func decimal(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// WriteHostedSummary uses an exclusive link so an existing output is never
// overwritten, and temporary files contain only the sanitized summary.
func WriteHostedSummary(path string, r HostedSummary) error {
	dir := filepath.Dir(path)
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("ci-metrics: output directory unavailable")
	}
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	if len(b) > maxSummaryBytes {
		return errors.New("ci-metrics: summary byte limit exceeded")
	}
	f, err := os.CreateTemp(dir, ".ci-metrics-summary-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer func() { _ = os.Remove(name) }()
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		return err
	}
	if _, err := f.Write(append(b, '\n')); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Link(name, path)
}
