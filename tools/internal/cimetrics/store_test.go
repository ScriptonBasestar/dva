//go:build darwin || linux || freebsd || netbsd || openbsd

package cimetrics

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var errRetentionBusy = errors.New("metrics retention lock busy")

func pruneFinished(dir string, now time.Time) error {
	lock, err := lockRetention(dir)
	if err != nil {
		return err
	}
	if lock == nil {
		return errRetentionBusy
	}
	defer unlockRetention(lock)
	return pruneFinishedLocked(dir, now)
}

func testRecord(id string, at time.Time) Record {
	return Record{Schema: recordSchema, ID: id, Check: "unit", StartedAt: at.Add(-time.Second), FinishedAt: at, Status: "complete", Outcome: "succeeded", Metadata: unknownMetadata()}
}

func writeActive(t *testing.T, dir, id string, started time.Time) *os.File {
	t.Helper()
	f, err := openExclusive(filepath.Join(dir, id+".active.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	header := fmt.Sprintf("{\"schema\":%q,\"measurement_id\":%q,\"check\":\"unit\",\"started_at\":%q}\n", recordSchema, id, started.UTC().Format(time.RFC3339Nano))
	if _, err := f.WriteString(header); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestMetricsRetentionProtectsActiveAndRecoversAbandoned(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	old := now.Add(-31 * 24 * time.Hour)
	oldID := strings.Repeat("a", 32)
	liveID := strings.Repeat("b", 32)
	abandonedID := strings.Repeat("c", 32)
	unknownID := strings.Repeat("d", 32)
	symlinkID := strings.Repeat("e", 32)
	if err := writeFinal(dir, testRecord(oldID, old)); err != nil {
		t.Fatal(err)
	}
	oldEvents := filepath.Join(dir, oldID+".events.jsonl")
	if err := os.WriteFile(oldEvents, fmt.Appendf(nil, "{\"schema\":%q,\"measurement_id\":%q}\n", recordSchema, oldID), 0o600); err != nil {
		t.Fatal(err)
	}
	live := writeActive(t, dir, liveID, now.Add(-time.Minute))
	locked, err := tryLockFile(live)
	if err != nil || !locked {
		t.Fatalf("active lock: %v %v", locked, err)
	}
	defer func() { unlockFile(live); _ = live.Close() }()
	abandoned := writeActive(t, dir, abandonedID, now.Add(-time.Minute))
	if err := abandoned.Close(); err != nil {
		t.Fatal(err)
	}
	unknown := filepath.Join(dir, unknownID+".json")
	if err := os.WriteFile(unknown, []byte(`{"schema":"other"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "do-not-touch")
	if err := os.WriteFile(target, []byte("sentinel"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, symlinkID+".json")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err := pruneFinished(dir, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{filepath.Join(dir, oldID+".json"), oldEvents, filepath.Join(dir, abandonedID+".active.jsonl")} {
		if _, err := os.Lstat(p); !os.IsNotExist(err) {
			t.Fatalf("expected removed %s: %v", p, err)
		}
	}
	for _, p := range []string{filepath.Join(dir, liveID+".active.jsonl"), unknown, link, target, filepath.Join(dir, abandonedID+".events.jsonl")} {
		if _, err := os.Lstat(p); err != nil {
			t.Fatalf("expected preserved %s: %v", p, err)
		}
	}
	b, err := os.ReadFile(filepath.Join(dir, abandonedID+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var rec Record
	if err := json.Unmarshal(b, &rec); err != nil {
		t.Fatal(err)
	}
	if rec.Status != "partial" || rec.Outcome != "unknown" || rec.ExitCode != nil {
		t.Fatalf("abandoned run was misclassified: %+v", rec)
	}
	if b, err := os.ReadFile(target); err != nil || string(b) != "sentinel" {
		t.Fatalf("symlink target changed: %q %v", b, err)
	}
}

func TestMetricsRetentionEnforcesCountAndRespectsLock(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	for i := range maxFinishedCount + 1 {
		id := fmt.Sprintf("%032x", i+1)
		if err := writeFinal(dir, testRecord(id, now.Add(-time.Duration(maxFinishedCount+1-i)*time.Second))); err != nil {
			t.Fatal(err)
		}
	}
	lock, err := lockRetention(dir)
	if err != nil || lock == nil {
		t.Fatalf("lock: %v", err)
	}
	if err := pruneFinished(dir, now); !errors.Is(err, errRetentionBusy) {
		t.Fatalf("expected busy lock, got %v", err)
	}
	oldest := filepath.Join(dir, fmt.Sprintf("%032x.json", 1))
	if _, err := os.Stat(oldest); err != nil {
		t.Fatalf("competing cleanup ignored lock: %v", err)
	}
	unlockRetention(lock)
	if err := pruneFinished(dir, now); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(oldest); !os.IsNotExist(err) {
		t.Fatalf("oldest record not removed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, fmt.Sprintf("%032x.json", maxFinishedCount+1))); err != nil {
		t.Fatalf("newest record removed: %v", err)
	}
}

func TestMetricsStateRejectsSymlinkAndNonPrivateDirectory(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DVA_PROJECT_CI_METRICS_DIR", link)
	if _, err := metricsDirectory(strings.Repeat("a", 64)); err == nil {
		t.Fatal("accepted symlink state root")
	}
	if entries, err := os.ReadDir(target); err != nil || len(entries) != 0 {
		t.Fatalf("wrote through symlink: %v %v", entries, err)
	}
	public := filepath.Join(root, "public")
	if err := os.Mkdir(public, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(public, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DVA_PROJECT_CI_METRICS_DIR", public)
	if _, err := metricsDirectory(strings.Repeat("b", 64)); err == nil {
		t.Fatal("accepted non-private state root")
	}
}

func TestMetricsStateAcceptsPrivateOverride(t *testing.T) {
	state := t.TempDir()
	if err := os.Chmod(state, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DVA_PROJECT_CI_METRICS_DIR", state)
	path, err := metricsDirectory(strings.Repeat("a", 64))
	if err != nil {
		t.Fatalf("private state rejected: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestMetricsFinalizationDefersWhileRetentionLocked(t *testing.T) {
	base := t.TempDir()
	if err := os.Chmod(base, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DVA_PROJECT_CI_METRICS_DIR", base)
	root, ok := findProjectRoot()
	if !ok {
		t.Fatal("project root unavailable")
	}
	key, ok := repositoryKey(root)
	if !ok {
		t.Fatal("repository identity unavailable")
	}
	dir, err := metricsDirectory(key)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	for i := 1; i <= maxFinishedCount; i++ {
		if err := writeFinal(dir, testRecord(fmt.Sprintf("%032x", i), now.Add(-time.Duration(i)*time.Second))); err != nil {
			t.Fatal(err)
		}
	}
	lock, err := lockRetention(dir)
	if err != nil || lock == nil {
		t.Fatalf("lock: %v", err)
	}
	id := strings.Repeat("f", 32)
	c := newCollector(root, "unit", id, now)
	if c.finish(testRecord(id, now)) {
		t.Fatal("busy finalization reported complete")
	}
	select {
	case <-c.done:
	case <-time.After(2 * time.Second):
		t.Fatal("collector did not finish")
	}
	if _, err := os.Stat(filepath.Join(dir, id+".json")); !os.IsNotExist(err) {
		t.Fatalf("published while lock held: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, id+".active.jsonl")); err != nil {
		t.Fatalf("missing deferred record: %v", err)
	}
	unlockRetention(lock)
	if err := pruneFinished(dir, time.Now()); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, id+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var recovered Record
	if err := json.Unmarshal(b, &recovered); err != nil {
		t.Fatal(err)
	}
	if recovered.Status != "partial" || recovered.Outcome != "unknown" {
		t.Fatalf("wrong recovery: %+v", recovered)
	}
	entries, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil || len(entries) != maxFinishedCount {
		t.Fatalf("retention count %d: %v", len(entries), err)
	}
}
