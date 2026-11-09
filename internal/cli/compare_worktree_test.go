package cli

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/compare"
	"github.com/ScriptonBasestar/dva/internal/config"
)

// workWorktree checks the fixture's "other" branch out as a work worktree next to the repo, the
// way a developer's own task worktree exists before any comparison. It returns its path.
func workWorktree(t *testing.T, c *config.Config) string {
	t.Helper()
	path := filepath.Join(filepath.Dir(c.FileDir()), "wt-other")
	gitIn(t, c.FileDir(), "worktree", "add", "-q", path, "other")
	return path
}

func mustEval(t *testing.T, p string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func itoa(n int) string { return strconv.Itoa(n) }

func gitOut(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	return string(out), err
}

func asJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func upMixed(t *testing.T, c *config.Config) map[string]any {
	t.Helper()
	doc, err := captureJSON(t, func() error {
		return runCompareUp(context.Background(), c, "layout", []string{"a=main", "b=worktree:other"}, false)
	})
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func variantRow(t *testing.T, doc map[string]any, name string) map[string]any {
	t.Helper()
	for _, r := range doc["variants"].([]any) {
		if row := r.(map[string]any); row["name"] == name {
			return row
		}
	}
	t.Fatalf("no variant %q in %v", name, doc["variants"])
	return nil
}

// A snapshot and a work worktree start side by side. The worktree is used where it is: dva
// creates no checkout for it, and the record points at the provider's path.
func TestCompareUpMixesSnapshotAndWorktreeVariants(t *testing.T) {
	c, host, ports := compareFixture(t)
	wt := workWorktree(t, c)
	doc := upMixed(t, c)

	b := variantRow(t, doc, "b")
	if b["mode"] != compare.ModeWorktree || b["outcome"] != outcomeStarted {
		t.Errorf("b = %v", b)
	}
	if got, _ := filepath.EvalSymlinks(b["path"].(string)); got != mustEval(t, wt) {
		t.Errorf("b path = %v, want %s", b["path"], wt)
	}
	if a := variantRow(t, doc, "a"); a["mode"] != compare.ModeSnapshot {
		t.Errorf("a = %v", a)
	}
	if host.count("up:wt-other:web") != 1 || host.count("up:a:web") != 1 {
		t.Errorf("calls = %v", host.calls)
	}
	if host.vars["wt-other"]["WEB_PORT"] != itoa(ports[1]) {
		t.Errorf("b vars = %v, want WEB_PORT=%d", host.vars["wt-other"], ports[1])
	}
	if _, err := os.Stat(filepath.Join(c.FileDir(), ".sb/dva/variants/layout/b")); err == nil {
		t.Error("dva must not create a checkout for a worktree variant")
	}
}

// An ID the provider does not know, or a provider dva does not have, starts nothing and does
// not fall back to the calling checkout.
func TestCompareUpWorktreeLookupFailureStartsNothing(t *testing.T) {
	c, host, _ := compareFixture(t)
	_, err := captureJSON(t, func() error {
		return runCompareUp(context.Background(), c, "layout", []string{"a=main", "b=worktree:nope"}, false)
	})
	if err == nil || !strings.Contains(err.Error(), `no worktree has branch "nope"`) {
		t.Fatalf("err = %v", err)
	}
	if errorCode(err) != codeComparePreflight {
		t.Errorf("code = %q", errorCode(err))
	}
	if len(host.calls) != 0 {
		t.Errorf("a lookup failure must start nothing: %v", host.calls)
	}
	if _, statErr := compare.ReadRecord(c.FileDir(), "layout"); statErr == nil {
		t.Error("a preflight failure must not write a record")
	}
}

// down stops the worktree variant's processes and leaves its worktree and branch alone.
func TestCompareDownStopsWorktreeProcessesAndKeepsTheWorktree(t *testing.T) {
	c, host, _ := compareFixture(t)
	wt := workWorktree(t, c)
	upMixed(t, c)
	host.calls = nil
	if _, err := captureJSON(t, func() error { return runCompareDown(context.Background(), c, "layout", nil) }); err != nil {
		t.Fatal(err)
	}
	if host.count("down:wt-other:web") != 1 {
		t.Errorf("calls = %v", host.calls)
	}
	if _, err := os.Stat(wt); err != nil {
		t.Errorf("down removed the work worktree: %v", err)
	}
	out, err := gitOut(t, c.FileDir(), "worktree", "list", "--porcelain")
	if err != nil || !strings.Contains(out, "branch refs/heads/other") {
		t.Errorf("worktree no longer registered: %v %s", err, out)
	}
	if _, err := os.Stat(filepath.Join(c.FileDir(), ".sb/dva/variants/layout/a")); err == nil {
		t.Error("the snapshot variant must still be reclaimed")
	}
	if _, err := compare.ReadRecord(c.FileDir(), "layout"); err == nil {
		t.Error("record should be gone once every variant is down")
	}
}

// A work worktree is edited and committed to. A moved head is not stale and does not make a
// second up refuse; a moved path is stale.
func TestCompareStatusWorktreeHeadMovesAreFineButPathMovesAreStale(t *testing.T) {
	c, host, _ := compareFixture(t)
	wt := workWorktree(t, c)
	upMixed(t, c)

	if err := os.WriteFile(filepath.Join(wt, "edit.txt"), []byte("edit"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, wt, "add", "-A")
	gitIn(t, wt, "commit", "-q", "-m", "edit")

	doc, err := captureJSON(t, func() error { return runCompareStatus(context.Background(), c, "layout") })
	if err != nil {
		t.Fatal(err)
	}
	if b := variantRow(t, doc, "b"); b["stale"] == true || b["drift"] != nil {
		t.Errorf("a new commit in the work worktree must not be stale or drift: %v", b)
	}
	host.calls = nil
	again, err := captureJSON(t, func() error {
		return runCompareUp(context.Background(), c, "layout", []string{"a=main", "b=worktree:other"}, false)
	})
	if err != nil {
		t.Fatalf("up after a commit in the worktree: %v", err)
	}
	if b := variantRow(t, again, "b"); b["outcome"] != outcomeSkipped {
		t.Errorf("b = %v, want skipped", b)
	}

	moved := filepath.Join(filepath.Dir(wt), "wt-moved")
	gitIn(t, c.FileDir(), "worktree", "move", wt, moved)
	doc, err = captureJSON(t, func() error { return runCompareStatus(context.Background(), c, "layout") })
	if err != nil {
		t.Fatal(err)
	}
	b := variantRow(t, doc, "b")
	if b["stale"] != true || !strings.Contains(asJSON(b["stale_reasons"]), "now resolves to") {
		t.Errorf("a moved worktree must be stale with the provider's answer: %v", b)
	}
	if a := variantRow(t, doc, "a"); a["stale"] == true {
		t.Errorf("the snapshot variant is not affected: %v", a)
	}
	if _, err := captureJSON(t, func() error {
		return runCompareUp(context.Background(), c, "layout", []string{"a=main", "b=worktree:other"}, false)
	}); err == nil || !strings.Contains(err.Error(), "already started") {
		t.Errorf("up must refuse a worktree that moved since the record: %v", err)
	}
}

// status asks the provider again: a worktree that was removed is reported stale, not hidden.
func TestCompareStatusWorktreeRemovedIsStale(t *testing.T) {
	c, _, _ := compareFixture(t)
	wt := workWorktree(t, c)
	upMixed(t, c)
	gitIn(t, c.FileDir(), "worktree", "remove", "--force", wt)
	doc, err := captureJSON(t, func() error { return runCompareStatus(context.Background(), c, "layout") })
	if err != nil {
		t.Fatal(err)
	}
	b := variantRow(t, doc, "b")
	if b["stale"] != true || !strings.Contains(asJSON(b["stale_reasons"]), "provider no longer resolves") {
		t.Errorf("b = %v", b)
	}
}
