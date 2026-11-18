//go:build darwin || linux || freebsd || netbsd || openbsd

package cimetrics

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestProgressRetentionNoisyTests(t *testing.T) {
	base := t.TempDir()
	if err := os.Chmod(base, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DVA_PROJECT_CI_METRICS_DIR", base)
	root, ok := findProjectRoot()
	if !ok {
		t.Fatal("no root")
	}
	id := strings.Repeat("9", 32)
	now := time.Now().UTC()
	c := newCollector(root, "unit", id, now)
	p := NewParser(nil, c.emit)
	for i := 0; i < 12000; i++ {
		for _, action := range []string{"run", "pass"} {
			_, _ = fmt.Fprintf(p, "{\"Action\":%q,\"Package\":\"example/pkg\",\"Test\":%q}\n", action, fmt.Sprintf("Test%d", i))
		}
	}
	_, _ = fmt.Fprintln(p, `{"Action":"pass","Package":"example/pkg"}`)
	p.Finish()
	snap := p.Snapshot()
	r := testRecord(id, now)
	r.Events = &snap
	if !c.finish(r) {
		t.Fatalf("intentional compaction reported as collection loss: failed=%v dropped=%v snapshot=%+v", c.failed.Load(), c.dropped.Load(), snap.Reasons)
	}
	key, _ := repositoryKey(root)
	dir, err := metricsDirectory(key)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, id+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Schema   string `json:"schema"`
		Status   string `json:"collection_status"`
		Progress *struct {
			Mode         string `json:"mode"`
			DetailStatus string `json:"detail_status"`
			Omitted      int    `json:"omitted"`
			QueueDropped int    `json:"queue_dropped"`
		} `json:"progress_retention"`
		Events Snapshot `json:"events"`
	}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	events, err := os.ReadFile(filepath.Join(dir, id+".events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	lines := bytes.Split(bytes.TrimSpace(events), []byte("\n"))
	var footer struct {
		Type    string         `json:"type"`
		Outcome string         `json:"check_outcome"`
		Counts  map[string]int `json:"counts"`
	}
	if err := json.Unmarshal(lines[len(lines)-1], &footer); err != nil || footer.Type != "final" || footer.Outcome != "succeeded" || footer.Counts["pass"] != 12001 {
		t.Fatalf("bad footer: %+v %v", footer, err)
	}
	if len(events) > maxProgressBytes {
		t.Fatalf("progress exceeds cap: %d", len(events))
	}
	if got.Schema != recordSchema || got.Status != "complete" || got.Progress == nil || got.Progress.Mode != "package-milestones-and-failures" || got.Progress.DetailStatus != "compacted" || got.Progress.Omitted != 24000 || got.Progress.QueueDropped != 0 || !got.Events.TimingsTruncated || got.Events.Counts["pass"] != 12001 {
		t.Fatalf("unexpected summary: %+v", got)
	}
}

func TestProgressQueueLossIsCounted(t *testing.T) {
	c := &collector{events: make(chan Progress, 1), stop: make(chan struct{})}
	c.emit(Progress{Action: "start"})
	c.emit(Progress{Action: "fail"})
	if c.queueDropped.Load() != 1 || !c.dropped.Load() {
		t.Fatal("queue overflow not reported")
	}
}

func TestProgressWriterLimitsAndErrors(t *testing.T) {
	var buf bytes.Buffer
	stats := &ProgressRetention{}
	w := progressWriter{w: &buf, stats: stats}
	payload := strings.Repeat("x", maxProgressBytes-progressFooterReserve-3)
	if !w.write(payload, maxProgressBytes-progressFooterReserve) {
		t.Fatal("exact boundary rejected")
	}
	if w.size != maxProgressBytes-progressFooterReserve {
		t.Fatalf("size %d", w.size)
	}
	if w.write("x", maxProgressBytes-progressFooterReserve) || stats.ByteDropped != 1 {
		t.Fatal("byte loss not counted")
	}
	if !w.write(map[string]string{"type": "final"}, maxProgressBytes) {
		t.Fatal("reserved footer rejected")
	}
	if w.write(math.NaN(), maxProgressBytes) || stats.MarshalFailures != 1 {
		t.Fatal("marshal failure not counted")
	}
	w.w = errorWriter{}
	if w.write("x", maxProgressBytes) || stats.WriteFailures != 1 {
		t.Fatal("write failure not counted")
	}
}

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

func TestLegacyRecordRetentionUnknown(t *testing.T) {
	var r Record
	if err := json.Unmarshal([]byte(`{"schema":"dva-project-ci-metrics/v1","collection_status":"complete"}`), &r); err != nil {
		t.Fatal(err)
	}
	if r.ProgressRetention != nil {
		t.Fatal("legacy absence must remain unknown")
	}
}

func TestLegacyDiskInventoryPruneAndRecovery(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	liveID := strings.Repeat("a", 32)
	oldID := strings.Repeat("b", 32)
	activeID := strings.Repeat("c", 32)
	unknownID := strings.Repeat("d", 32)
	makeLegacy := func(id string, at time.Time) []byte {
		r := testRecord(id, at)
		r.Schema = legacyRecordSchema
		b, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	live := makeLegacy(liveID, now.Add(-time.Hour))
	for id, b := range map[string][]byte{liveID: live, oldID: makeLegacy(oldID, now.Add(-31*24*time.Hour)), unknownID: []byte(`{"schema":"other"}`)} {
		if err := os.WriteFile(filepath.Join(dir, id+".json"), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []string{liveID, oldID} {
		b := fmt.Appendf(nil, "{\"schema\":%q,\"measurement_id\":%q}\n", legacyRecordSchema, id)
		if err := os.WriteFile(filepath.Join(dir, id+".events.jsonl"), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	header := fmt.Appendf(nil, "{\"schema\":%q,\"measurement_id\":%q,\"check\":\"unit\",\"started_at\":%q}\n", legacyRecordSchema, activeID, now.Add(-time.Minute).Format(time.RFC3339Nano))
	if err := os.WriteFile(filepath.Join(dir, activeID+".active.jsonl"), header, 0600); err != nil {
		t.Fatal(err)
	}
	if err := pruneFinished(dir, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, liveID+".json"))
	if err != nil || !bytes.Equal(got, live) {
		t.Fatalf("legacy record changed: %v", err)
	}
	var r Record
	if err := json.Unmarshal(got, &r); err != nil || r.Schema != legacyRecordSchema || r.ProgressRetention != nil {
		t.Fatalf("legacy decode: %+v %v", r, err)
	}
	if _, err := os.Stat(filepath.Join(dir, oldID+".json")); !os.IsNotExist(err) {
		t.Fatalf("expired v1 not pruned: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, oldID+".events.jsonl")); !os.IsNotExist(err) {
		t.Fatalf("expired v1 events not pruned: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, unknownID+".json")); err != nil {
		t.Fatalf("unknown deleted: %v", err)
	}
	got, err = os.ReadFile(filepath.Join(dir, activeID+".json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(got, &r); err != nil || r.Schema != legacyRecordSchema || r.Status != "partial" || r.Outcome != "unknown" || r.ProgressRetention != nil {
		t.Fatalf("legacy recovery: %+v %v", r, err)
	}
	recovered, err := os.ReadFile(filepath.Join(dir, activeID+".events.jsonl"))
	if err != nil || !bytes.Equal(recovered, header) {
		t.Fatalf("legacy progress changed: %v", err)
	}
}
