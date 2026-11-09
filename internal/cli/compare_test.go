package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/compare"
	"github.com/ScriptonBasestar/dva/internal/config"
	"github.com/ScriptonBasestar/dva/internal/lifecycle"
	"github.com/ScriptonBasestar/dva/internal/output"
)

// fakeHost records every call compare makes to the lifecycle and fails the ones a test names.
// up marks the (dir, entry) pair running, so a later Status sees what Up started.
type fakeHost struct {
	calls      []string
	failUp     map[string]bool  // dir base name → Up fails
	unreadable map[string]bool  // dir base name → Down finds no loadable dva.yml
	onUp       func(dir string) // runs when an Up succeeds, to leave files a real start would
	running    map[string]bool  // "dir|entry"
	vars       map[string]map[string]string
}

func newFakeHost() *fakeHost {
	return &fakeHost{failUp: map[string]bool{}, running: map[string]bool{}, vars: map[string]map[string]string{}}
}

func (f *fakeHost) tag(dir string) string {
	if filepath.Base(dir) == "repo" {
		return "shared"
	}
	return filepath.Base(dir)
}

func (f *fakeHost) Up(_ context.Context, dir, _ string, vars map[string]string, names []string, _ bool) (*compareObserved, error) {
	f.calls = append(f.calls, "up:"+f.tag(dir)+":"+strings.Join(names, ","))
	f.vars[f.tag(dir)] = vars
	if f.failUp[f.tag(dir)] {
		return nil, errors.New("simulated start failure")
	}
	for _, n := range names {
		f.running[dir+"|"+n] = true
	}
	if f.onUp != nil {
		f.onUp(dir)
	}
	return f.observe(dir, names), nil
}

func (f *fakeHost) Down(_ context.Context, dir, _ string, _ map[string]string, names []string) error {
	f.calls = append(f.calls, "down:"+f.tag(dir)+":"+strings.Join(names, ","))
	if f.unreadable[f.tag(dir)] {
		return fmt.Errorf("load %s: %w", dir, errVariantConfigUnreadable)
	}
	for _, n := range names {
		delete(f.running, dir+"|"+n)
	}
	return nil
}

func (f *fakeHost) Status(_ context.Context, dir, _ string, _ map[string]string, names []string) (*compareObserved, error) {
	return f.observe(dir, names), nil
}

func (f *fakeHost) observe(dir string, names []string) *compareObserved {
	st := &lifecycle.AggregatedStatus{}
	for _, n := range names {
		state := "stopped"
		if f.running[dir+"|"+n] {
			state = "running"
		}
		st.Entries = append(st.Entries, lifecycle.EntryStatus{Name: n, Services: []lifecycle.ServiceStatus{{Name: n, State: state}}})
	}
	return &compareObserved{Status: st}
}

func (f *fakeHost) count(prefix string) int {
	n := 0
	for _, c := range f.calls {
		if strings.HasPrefix(c, prefix) {
			n++
		}
	}
	return n
}

const compareFixtureYML = `version: "0.1.0"
stack:
  api:
    default_runner: native
    runners:
      native:
        run: go run ./cmd/api
  web:
    default_runner: native
    runners:
      native:
        run: go run ./cmd/web
plans:
  web-dev:
    entries:
      - name: api
      - name: web
compare:
  layout:
    plan: web-dev
    shared: [api]
    per_variant: [web]
    ports:
      WEB_PORT: [%d, %d, %d]
`

func freePorts(t *testing.T, n int) []int {
	t.Helper()
	var ports []int
	var ls []net.Listener
	for range n {
		l, err := net.Listen("tcp", "0.0.0.0:0")
		if err != nil {
			t.Fatal(err)
		}
		ls = append(ls, l)
		ports = append(ports, l.Addr().(*net.TCPAddr).Port)
	}
	for _, l := range ls {
		_ = l.Close()
	}
	return ports
}

func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	full := append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@example.com"}, args...)
	if out, err := exec.Command("git", full...).CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
}

// compareFixture is a git repository with the declaration above, a second branch "other" one
// commit ahead of main, and a fake lifecycle host. It returns the loaded config.
func compareFixture(t *testing.T) (*config.Config, *fakeHost, []int) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	ports := freePorts(t, 3)
	dir := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	gitIn(t, dir, "init", "-q", "-b", "main")
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".gitignore", ".sb/\n")
	write("dva.yml", fmt.Sprintf(compareFixtureYML, ports[0], ports[1], ports[2]))
	gitIn(t, dir, "add", "-A")
	gitIn(t, dir, "commit", "-q", "-m", "initial")
	gitIn(t, dir, "checkout", "-q", "-b", "other")
	write("marker.txt", "other\n")
	gitIn(t, dir, "add", "-A")
	gitIn(t, dir, "commit", "-q", "-m", "other")
	gitIn(t, dir, "checkout", "-q", "main")

	c, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	host := newFakeHost()
	old, oldDry, oldJSON := compareHost, dryRun, jsonOutput
	output.ResetStdoutDocument()
	compareHost, dryRun, jsonOutput = host, false, false
	t.Cleanup(func() { compareHost, dryRun, jsonOutput = old, oldDry, oldJSON; output.ResetStdoutDocument() })
	return c, host, ports
}

func captureJSON(t *testing.T, fn func() error) (map[string]any, error) {
	t.Helper()
	old := jsonOutput
	jsonOutput = true
	defer func() { jsonOutput = old }()
	var err error
	out := captureStdout(t, func() { err = fn() })
	if len(bytes.TrimSpace([]byte(out))) == 0 {
		return nil, err
	}
	var doc map[string]any
	if jerr := json.Unmarshal([]byte(out), &doc); jerr != nil {
		t.Fatalf("stdout is not one JSON document: %v\n%s", jerr, out)
	}
	return doc, err
}

func TestCompareUpStartsSharedOnceAndEachVariantOnItsOwnPort(t *testing.T) {
	c, host, ports := compareFixture(t)
	doc, err := captureJSON(t, func() error {
		return runCompareUp(context.Background(), c, "layout", []string{"a=main", "b=other"}, false)
	})
	if err != nil {
		t.Fatalf("up: %v", err)
	}
	if host.count("up:shared:api") != 1 {
		t.Errorf("shared entries must start exactly once, calls = %v", host.calls)
	}
	variants := doc["variants"].([]any)
	if len(variants) != 2 {
		t.Fatalf("variants = %v", variants)
	}
	for i, want := range []int{ports[0], ports[1]} {
		row := variants[i].(map[string]any)
		if row["outcome"] != outcomeStarted {
			t.Errorf("variant %d outcome = %v", i, row["outcome"])
		}
		if got := int(row["ports"].(map[string]any)["WEB_PORT"].(float64)); got != want {
			t.Errorf("variant %d WEB_PORT = %d, want list position %d = %d", i, got, i, want)
		}
	}
	if host.vars["a"]["WEB_PORT"] != fmt.Sprint(ports[0]) || host.vars["b"]["WEB_PORT"] != fmt.Sprint(ports[1]) {
		t.Errorf("per_variant vars not handed to each checkout: %v", host.vars)
	}
	rec, err := compare.ReadRecord(c.FileDir(), "layout")
	if err != nil || len(rec.Variants) != 2 {
		t.Fatalf("record = %+v, %v", rec, err)
	}
	if rec.Variant("a").Commit == rec.Variant("b").Commit {
		t.Errorf("a and b were pinned to the same commit")
	}
}

func TestCompareUpPartialFailureLeavesStartedAndRetryStartsOnlyMissing(t *testing.T) {
	c, host, _ := compareFixture(t)
	args := []string{"a=main", "b=other"}
	host.failUp["b"] = true

	doc, err := captureJSON(t, func() error { return runCompareUp(context.Background(), c, "layout", args, false) })
	if err == nil {
		t.Fatal("a failing variant must fail the command")
	}
	if errorCode(err) != codeComparePartial {
		t.Errorf("code = %q, want %s", errorCode(err), codeComparePartial)
	}
	if doc != nil {
		t.Errorf("a failed up must leave stdout to the failure envelope, got %v", doc)
	}
	var ce *compareError
	if !errors.As(err, &ce) || ce.result.variant("a").Outcome != outcomeStarted || ce.result.variant("b").Outcome != outcomeFailed {
		t.Fatalf("result = %+v", ce.result)
	}
	if ce.result.Next["down"] == "" || !strings.Contains(ce.result.Next["retry"], "b=other") {
		t.Errorf("failure must name the way back: %v", ce.result.Next)
	}
	rec, _ := compare.ReadRecord(c.FileDir(), "layout")
	if rec == nil || rec.Variant("a") == nil || rec.Variant("b") == nil {
		t.Fatalf("both variants must be recorded so down can reclaim them: %+v", rec)
	}

	host.failUp["b"] = false
	host.calls = nil
	if _, err := captureJSON(t, func() error { return runCompareUp(context.Background(), c, "layout", args, false) }); err != nil {
		t.Fatalf("retry: %v", err)
	}
	if host.count("up:a:") != 0 {
		t.Errorf("a was already running and must not start again: %v", host.calls)
	}
	if host.count("up:b:") != 1 {
		t.Errorf("b must start on retry: %v", host.calls)
	}
}

func TestCompareUpPreflightCreatesNothing(t *testing.T) {
	cases := []struct {
		name string
		prep func(t *testing.T, ports []int)
		args []string
		want string
	}{
		{"busy port", func(t *testing.T, ports []int) {
			l, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", ports[1]))
			if err != nil {
				t.Skipf("cannot hold port: %v", err)
			}
			t.Cleanup(func() { _ = l.Close() })
		}, []string{"a=main", "b=other"}, "already in use"},
		{"short port list", nil, []string{"a=main", "b=other", "c=main", "d=other"}, "ports"},
		{"env override", func(t *testing.T, ports []int) { t.Setenv("WEB_PORT", "1") }, []string{"a=main"}, "environment sets WEB_PORT"},
		{"bad ref", nil, []string{"a=no-such-ref"}, "no-such-ref"},
		{"provider target", nil, []string{"a=worktree:TASK-1"}, "provider target"},
		{"bad name", nil, []string{"../x=main"}, "must start with a letter"},
		{"duplicate", nil, []string{"a=main", "a=other"}, "given twice"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, host, ports := compareFixture(t)
			if tc.prep != nil {
				tc.prep(t, ports)
			}
			_, err := captureJSON(t, func() error { return runCompareUp(context.Background(), c, "layout", tc.args, false) })
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want it to contain %q", err, tc.want)
			}
			if errorCode(err) != codeComparePreflight {
				t.Errorf("code = %q", errorCode(err))
			}
			if len(host.calls) != 0 {
				t.Errorf("preflight failure must start nothing: %v", host.calls)
			}
			if _, statErr := os.Stat(filepath.Join(c.FileDir(), ".sb/dva/variants")); statErr == nil {
				t.Errorf("preflight failure must create no checkout")
			}
			if _, rerr := compare.ReadRecord(c.FileDir(), "layout"); rerr == nil {
				t.Errorf("preflight failure must write no record")
			}
		})
	}
}

func TestCompareUpUnknownNameListsDeclared(t *testing.T) {
	c, _, _ := compareFixture(t)
	err := runCompareUp(context.Background(), c, "nope", []string{"a=main"}, false)
	if err == nil || !strings.Contains(err.Error(), "declared: layout") {
		t.Fatalf("err = %v", err)
	}
}

func TestCompareUpDryRunPlansWithoutSideEffects(t *testing.T) {
	c, host, _ := compareFixture(t)
	dryRun = true
	doc, err := captureJSON(t, func() error {
		return runCompareUp(context.Background(), c, "layout", []string{"a=main", "b=other"}, false)
	})
	if err != nil {
		t.Fatal(err)
	}
	if doc["dry_run"] != true {
		t.Errorf("dry_run = %v", doc["dry_run"])
	}
	for _, v := range doc["variants"].([]any) {
		if v.(map[string]any)["outcome"] != outcomePlanned {
			t.Errorf("row = %v", v)
		}
	}
	if len(host.calls) != 0 {
		t.Errorf("dry-run started something: %v", host.calls)
	}
	if _, err := os.Stat(filepath.Join(c.FileDir(), ".sb/dva/variants")); err == nil {
		t.Errorf("dry-run created a checkout")
	}
}

func TestCompareUpRefusesChangedSourceWithoutDown(t *testing.T) {
	c, _, _ := compareFixture(t)
	if _, err := captureJSON(t, func() error {
		return runCompareUp(context.Background(), c, "layout", []string{"a=main"}, false)
	}); err != nil {
		t.Fatal(err)
	}
	_, err := captureJSON(t, func() error {
		return runCompareUp(context.Background(), c, "layout", []string{"a=other"}, false)
	})
	if err == nil || !strings.Contains(err.Error(), "dva compare down layout a") {
		t.Fatalf("err = %v", err)
	}
}

func TestCompareDownStopsEntriesBeforeRemovingCheckoutsAndSharedLast(t *testing.T) {
	c, host, _ := compareFixture(t)
	if _, err := captureJSON(t, func() error {
		return runCompareUp(context.Background(), c, "layout", []string{"a=main", "b=other"}, false)
	}); err != nil {
		t.Fatal(err)
	}
	rec, _ := compare.ReadRecord(c.FileDir(), "layout")
	pathA := rec.Variant("a").Path
	host.calls = nil
	doc, err := captureJSON(t, func() error { return runCompareDown(context.Background(), c, "layout", nil) })
	if err != nil {
		t.Fatalf("down: %v", err)
	}
	want := []string{"down:a:web", "down:b:web", "down:shared:api"}
	if strings.Join(host.calls, " ") != strings.Join(want, " ") {
		t.Errorf("order = %v, want %v", host.calls, want)
	}
	for _, v := range doc["variants"].([]any) {
		if v.(map[string]any)["outcome"] != outcomeStopped {
			t.Errorf("row = %v", v)
		}
	}
	if _, err := os.Stat(pathA); err == nil {
		t.Errorf("checkout %s still exists", pathA)
	}
	if _, err := compare.ReadRecord(c.FileDir(), "layout"); err == nil {
		t.Errorf("record must be gone once nothing remains")
	}
}

func TestCompareDownOneVariantKeepsSharedAndRecord(t *testing.T) {
	c, host, _ := compareFixture(t)
	if _, err := captureJSON(t, func() error {
		return runCompareUp(context.Background(), c, "layout", []string{"a=main", "b=other"}, false)
	}); err != nil {
		t.Fatal(err)
	}
	host.calls = nil
	if _, err := captureJSON(t, func() error { return runCompareDown(context.Background(), c, "layout", []string{"b"}) }); err != nil {
		t.Fatal(err)
	}
	if strings.Join(host.calls, " ") != "down:b:web" {
		t.Errorf("calls = %v", host.calls)
	}
	rec, err := compare.ReadRecord(c.FileDir(), "layout")
	if err != nil || rec.Variant("a") == nil || rec.Variant("b") != nil {
		t.Errorf("record = %+v, %v", rec, err)
	}
}

func TestCompareDownRefusesDirtySnapshotAndSaysWhatRemains(t *testing.T) {
	c, host, _ := compareFixture(t)
	if _, err := captureJSON(t, func() error {
		return runCompareUp(context.Background(), c, "layout", []string{"a=main"}, false)
	}); err != nil {
		t.Fatal(err)
	}
	rec, _ := compare.ReadRecord(c.FileDir(), "layout")
	if err := os.WriteFile(filepath.Join(rec.Variant("a").Path, "scratch.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	doc, err := captureJSON(t, func() error { return runCompareDown(context.Background(), c, "layout", nil) })
	if err == nil || errorCode(err) != codeCompareDown {
		t.Fatalf("err = %v (%s)", err, errorCode(err))
	}
	if doc != nil {
		t.Errorf("failure must leave stdout to the envelope: %v", doc)
	}
	var ce *compareError
	if !errors.As(err, &ce) || ce.result.variant("a").Outcome != outcomeFailed || ce.result.Shared.Outcome != outcomeNotAttempted {
		t.Fatalf("result = %+v", ce.result)
	}
	if host.count("down:shared") != 0 {
		t.Errorf("shared entries must not stop while a variant remains: %v", host.calls)
	}
	if _, err := os.Stat(filepath.Join(rec.Variant("a").Path, "scratch.txt")); err != nil {
		t.Errorf("dirty checkout was touched: %v", err)
	}
	if after, _ := compare.ReadRecord(c.FileDir(), "layout"); after == nil || after.Variant("a") == nil {
		t.Errorf("record must keep the variant that is still there")
	}
}

func TestCompareStatusReadsOnlyAndReportsDrift(t *testing.T) {
	c, host, _ := compareFixture(t)
	if _, err := captureJSON(t, func() error {
		return runCompareUp(context.Background(), c, "layout", []string{"a=main"}, false)
	}); err != nil {
		t.Fatal(err)
	}
	gitIn(t, c.FileDir(), "checkout", "-q", "-b", "mv", "main")
	if err := os.WriteFile(filepath.Join(c.FileDir(), "m.txt"), []byte("m"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, c.FileDir(), "add", "-A")
	gitIn(t, c.FileDir(), "commit", "-q", "-m", "move")
	gitIn(t, c.FileDir(), "branch", "-f", "main", "mv")
	gitIn(t, c.FileDir(), "checkout", "-q", "main")

	host.calls = nil
	doc, err := captureJSON(t, func() error { return runCompareStatus(context.Background(), c, "layout") })
	if err != nil {
		t.Fatal(err)
	}
	for _, call := range host.calls {
		if strings.HasPrefix(call, "up:") || strings.HasPrefix(call, "down:") {
			t.Errorf("status must only read, called %s", call)
		}
	}
	row := doc["variants"].([]any)[0].(map[string]any)
	if row["outcome"] != outcomeObserved || row["drift"] == nil {
		t.Errorf("row = %v", row)
	}
}

func TestCompareStatusWithoutRecordFails(t *testing.T) {
	c, _, _ := compareFixture(t)
	err := runCompareStatus(context.Background(), c, "layout")
	if err == nil || errorCode(err) != codeComparePreflight {
		t.Fatalf("err = %v", err)
	}
}

func TestCompareFailureEnvelopeCarriesCodeAndResult(t *testing.T) {
	c, host, _ := compareFixture(t)
	host.failUp["b"] = true
	old := jsonOutput
	jsonOutput = true
	defer func() { jsonOutput = old }()
	var err error
	out := captureStdout(t, func() {
		err = runCompareUp(context.Background(), c, "layout", []string{"a=main", "b=other"}, false)
		emitFailureJSONFor(err)
	})
	var doc struct {
		Error struct {
			Message  string         `json:"message"`
			ExitCode int            `json:"exit_code"`
			Code     string         `json:"code"`
			Result   map[string]any `json:"result"`
		} `json:"error"`
	}
	if jerr := json.Unmarshal([]byte(out), &doc); jerr != nil {
		t.Fatalf("stdout must be exactly the one envelope: %v\n%s", jerr, out)
	}
	if doc.Error.Code != codeComparePartial || doc.Error.ExitCode != 1 || doc.Error.Message == "" {
		t.Errorf("envelope = %+v", doc.Error)
	}
	if doc.Error.Result["compare"] != "layout" || len(doc.Error.Result["variants"].([]any)) != 2 {
		t.Errorf("result = %v", doc.Error.Result)
	}
}

// A variant whose dva.yml never loaded was never started, so down must still reclaim it;
// otherwise the command printed after a failed up could never finish.
func TestCompareDownReclaimsVariantWhoseConfigCannotLoad(t *testing.T) {
	c, host, _ := compareFixture(t)
	if _, err := captureJSON(t, func() error {
		return runCompareUp(context.Background(), c, "layout", []string{"a=main", "b=other"}, false)
	}); err != nil {
		t.Fatal(err)
	}
	host.unreadable = map[string]bool{"b": true}
	if _, err := captureJSON(t, func() error { return runCompareDown(context.Background(), c, "layout", []string{"b"}) }); err != nil {
		t.Fatalf("down of an unloadable variant failed: %v", err)
	}
	rec, err := compare.ReadRecord(c.FileDir(), "layout")
	if err != nil || rec.Variant("b") != nil {
		t.Errorf("b still recorded: %+v, %v", rec, err)
	}
}
