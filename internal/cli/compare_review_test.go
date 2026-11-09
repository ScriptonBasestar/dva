package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/compare"
	"github.com/ScriptonBasestar/dva/internal/config"
)

func upBoth(t *testing.T, c *config.Config) {
	t.Helper()
	if _, err := captureJSON(t, func() error {
		return runCompareUp(context.Background(), c, "layout", []string{"a=main", "b=other"}, false)
	}); err != nil {
		t.Fatal(err)
	}
}

// Nothing to start means the shared entries are not started either: an up that skips every
// variant would otherwise re-run the shared Up for no reason and report it as started.
func TestCompareUpWithEveryVariantRunningTouchesNothing(t *testing.T) {
	c, host, _ := compareFixture(t)
	upBoth(t, c)
	host.calls = nil
	doc, err := captureJSON(t, func() error {
		return runCompareUp(context.Background(), c, "layout", []string{"a=main", "b=other"}, false)
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, call := range host.calls {
		if strings.HasPrefix(call, "up:") {
			t.Errorf("nothing was pending, but %s ran", call)
		}
	}
	if shared, _ := doc["shared"].(map[string]any); shared["outcome"] != outcomeSkipped {
		t.Errorf("shared outcome = %v, want %s", shared["outcome"], outcomeSkipped)
	}
}

// A variant whose record is stale (its pid is dead) is started again even when the lifecycle
// still reports a running service: the record and the process disagree, so it is not skipped.
func TestCompareUpRestartsStaleVariantEvenIfStatusSaysRunning(t *testing.T) {
	c, host, _ := compareFixture(t)
	upBoth(t, c)
	rec, _ := compare.ReadRecord(c.FileDir(), "layout")
	pids := filepath.Join(rec.Variant("a").ConfigDir, config.DotDirName, config.PidsDirName)
	if err := os.MkdirAll(pids, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pids, "web.pid"), []byte("2147483646\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	host.calls = nil
	if _, err := captureJSON(t, func() error {
		return runCompareUp(context.Background(), c, "layout", []string{"a=main", "b=other"}, false)
	}); err != nil {
		t.Fatal(err)
	}
	if host.count("up:a:web") != 1 || host.count("up:b:web") != 0 {
		t.Errorf("only the stale variant restarts: %v", host.calls)
	}
}

// When the call that failed started nothing (the checkout of c could not be made), the way
// back must not be a bare `down`, which takes the running a and b down too.
func TestCompareUpFailureWithNothingTouchedSuggestsNoDown(t *testing.T) {
	c, _, _ := compareFixture(t)
	upBoth(t, c)
	snap, err := compare.SnapshotPath(c.FileDir(), "layout", "c")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(snap, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(snap, "squatter"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, upErr := captureJSON(t, func() error {
		return runCompareUp(context.Background(), c, "layout", []string{"a=main", "b=other", "c=main"}, false)
	})
	ce := asCompareError(t, upErr)
	if ce.result.variant("c").Outcome != outcomeFailed {
		t.Fatalf("c = %+v", ce.result.variant("c"))
	}
	if down, ok := ce.result.Next["down"]; ok {
		t.Errorf("next.down = %q would take down variants this call did not start", down)
	}
	if ce.result.Next["retry"] == "" {
		t.Errorf("retry must remain: %v", ce.result.Next)
	}
}

// status reports the pid and log file dva keeps for each entry of a variant.
func TestCompareStatusReportsPidAndLogPaths(t *testing.T) {
	c, _, _ := compareFixture(t)
	upBoth(t, c)
	rec, _ := compare.ReadRecord(c.FileDir(), "layout")
	base := filepath.Join(rec.Variant("a").ConfigDir, config.DotDirName)
	for dir, file := range map[string]string{config.PidsDirName: "web.pid", config.LogsDirName: "web.log"} {
		if err := os.MkdirAll(filepath.Join(base, dir), 0o755); err != nil {
			t.Fatal(err)
		}
		body := "4242\n"
		if dir == config.LogsDirName {
			body = "log\n"
		}
		if err := os.WriteFile(filepath.Join(base, dir, file), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	doc, err := captureJSON(t, func() error { return runCompareStatus(context.Background(), c, "layout") })
	if err != nil {
		t.Fatal(err)
	}
	var row map[string]any
	for _, v := range doc["variants"].([]any) {
		if m := v.(map[string]any); m["name"] == "a" {
			row = m
		}
	}
	pids, _ := row["pids"].(map[string]any)
	logs, _ := row["logs"].([]any)
	if pids["web"] != float64(4242) || len(logs) != 1 || !strings.HasSuffix(logs[0].(string), "web.log") {
		t.Errorf("pids=%v logs=%v", pids, logs)
	}
}

// A checkout that cannot load its dva.yml and is also dirty stays; the message must not claim
// the entries were stopped, because dva never got to stop them.
func TestCompareDownDoesNotClaimStoppedWhenConfigUnreadable(t *testing.T) {
	c, host, _ := compareFixture(t)
	upBoth(t, c)
	rec, _ := compare.ReadRecord(c.FileDir(), "layout")
	if err := os.WriteFile(filepath.Join(rec.Variant("b").Path, "scratch.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	host.unreadable = map[string]bool{"b": true}
	_, err := captureJSON(t, func() error { return runCompareDown(context.Background(), c, "layout", []string{"b"}) })
	if err == nil || strings.Contains(err.Error(), "entries stopped") || !strings.Contains(err.Error(), "may still run") {
		t.Fatalf("err = %v", err)
	}
}

func asCompareError(t *testing.T, err error) *compareError {
	t.Helper()
	ce, ok := err.(*compareError)
	if !ok {
		t.Fatalf("err = %v, want *compareError", err)
	}
	return ce
}
