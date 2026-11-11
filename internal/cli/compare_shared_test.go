package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// Shared entries that were already up before this call (here: started by another comparison
// that declares the same shared entry) are reported as skipped, not started, and not started
// a second time.
func TestCompareUpReportsAlreadyRunningSharedAsSkipped(t *testing.T) {
	c, host, _ := compareFixture(t)
	host.running[c.FileDir()+"||api"] = true

	doc, err := captureJSON(t, func() error {
		return runCompareUp(context.Background(), c, "layout", []string{"a=main"}, false)
	})
	if err != nil {
		t.Fatalf("up: %v", err)
	}
	shared := doc["shared"].(map[string]any)
	if shared["outcome"] != outcomeSkipped {
		t.Errorf("shared outcome = %v, want %s for entries already running", shared["outcome"], outcomeSkipped)
	}
	if host.count("up:shared:") != 0 {
		t.Errorf("already running shared entries must not start again: %v", host.calls)
	}

	// Entries that were not running are still started and reported so.
	c2, host2, _ := compareFixture(t)
	doc, err = captureJSON(t, func() error {
		return runCompareUp(context.Background(), c2, "layout", []string{"a=main"}, false)
	})
	if err != nil {
		t.Fatalf("up: %v", err)
	}
	if got := doc["shared"].(map[string]any)["outcome"]; got != outcomeStarted || host2.count("up:shared:api") != 1 {
		t.Errorf("shared outcome = %v, calls = %v; want started once", got, host2.calls)
	}
}

// addSharingComparison declares a second comparison "board" that shares the fixture's api
// entry, with ports of its own.
func addSharingComparison(t *testing.T, c *config.Config) {
	t.Helper()
	ports := freePorts(t, 2)
	c.Compare["board"] = &config.CompareConfig{
		Plan: "web-dev", Shared: []string{"api"}, PerVariant: []string{"web"},
		Ports: map[string][]int{"WEB_PORT": ports},
	}
}

// A bare down of one comparison must not stop a shared entry another recorded comparison still
// uses; it says so in the result, and the last comparison's down stops it.
func TestCompareDownKeepsSharedEntriesAnotherComparisonUses(t *testing.T) {
	c, host, _ := compareFixture(t)
	addSharingComparison(t, c)
	for _, n := range []string{"layout", "board"} {
		if _, err := captureJSON(t, func() error {
			return runCompareUp(context.Background(), c, n, []string{"a=main"}, false)
		}); err != nil {
			t.Fatalf("up %s: %v", n, err)
		}
	}

	host.calls = nil
	doc, err := captureJSON(t, func() error { return runCompareDown(context.Background(), c, "layout", nil) })
	if err != nil {
		t.Fatalf("down layout: %v", err)
	}
	if host.count("down:shared") != 0 {
		t.Errorf("api is still used by board and must not stop: %v", host.calls)
	}
	shared := doc["shared"].(map[string]any)
	kept, _ := shared["kept"].(map[string]any)
	if shared["outcome"] != outcomeKept || fmt.Sprint(kept["api"]) != "[board]" {
		t.Errorf("shared = %v, want outcome %s with api kept for board", shared, outcomeKept)
	}

	// The human table says it too.
	out := captureStdout(t, func() {
		printCompareTable(os.Stdout, &compareResult{Action: "down", Compare: "layout",
			Shared: &compareSharedResult{Entries: []string{"api"}, Outcome: outcomeKept, Kept: map[string][]string{"api": {"board"}}}})
	})
	if !strings.Contains(out, "shared api kept: in use by comparison board") {
		t.Errorf("table does not name the comparison that keeps api:\n%s", out)
	}

	host.calls = nil
	if _, err := captureJSON(t, func() error { return runCompareDown(context.Background(), c, "board", nil) }); err != nil {
		t.Fatalf("down board: %v", err)
	}
	if host.count("down:shared:api") != 1 {
		t.Errorf("the last comparison's down must stop api: %v", host.calls)
	}
}

// A record of the other comparison that cannot be read still counts as in use.
func TestCompareDownKeepsSharedWhenAnotherRecordIsUnreadable(t *testing.T) {
	c, host, _ := compareFixture(t)
	addSharingComparison(t, c)
	if _, err := captureJSON(t, func() error {
		return runCompareUp(context.Background(), c, "layout", []string{"a=main"}, false)
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(c.FileDir(), config.DotDirName, config.CompareDirName, "board.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	host.calls = nil
	if _, err := captureJSON(t, func() error { return runCompareDown(context.Background(), c, "layout", nil) }); err != nil {
		t.Fatalf("down: %v", err)
	}
	if host.count("down:shared") != 0 {
		t.Errorf("an unreadable record of board must keep api: %v", host.calls)
	}
}

// A shared result's endpoint whose URL names a per_variant port is reported once per variant,
// with that variant's port, by up and by status; an endpoint the same for all stays single.
func TestCompareSharedEndpointsAreSubstitutedPerVariant(t *testing.T) {
	c, _, ports := compareFixture(t)
	check := func(doc map[string]any, action string) {
		t.Helper()
		eps, _ := doc["shared"].(map[string]any)["endpoints"].([]any)
		got := map[string]string{}
		for _, e := range eps {
			m := e.(map[string]any)
			v, _ := m["variant"].(string)
			got[m["name"].(string)+"@"+v] = m["url"].(string)
		}
		want := map[string]string{
			"api@":  "http://localhost:8080",
			"web@a": fmt.Sprintf("http://localhost:%d", ports[0]),
			"web@b": fmt.Sprintf("http://localhost:%d", ports[1]),
		}
		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("%s shared endpoints = %v, want %v", action, got, want)
		}
	}
	doc, err := captureJSON(t, func() error {
		return runCompareUp(context.Background(), c, "layout", []string{"a=main", "b=other"}, false)
	})
	if err != nil {
		t.Fatalf("up: %v", err)
	}
	check(doc, "up")
	doc, err = captureJSON(t, func() error { return runCompareStatus(context.Background(), c, "layout") })
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	check(doc, "status")
}

// The production runner interpolates the plan's endpoints with the vars it is given, and does
// not leave them in the calling checkout's environment, which the shared entries read.
func TestLifecycleRunnerEndpointsUseVarsWithoutLeakingThem(t *testing.T) {
	dir := t.TempDir()
	yml := `version: "0.1.0"
stack:
  web:
    default_runner: native
    runners:
      native:
        run: sleep 300
endpoints:
  web:
    url: http://localhost:${WEB_PORT}
plans:
  p:
    entries:
      - name: web
`
	if err := os.WriteFile(filepath.Join(dir, "dva.yml"), []byte(yml), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	oldCfg, oldEnv := cfg, env
	cfg, env = c, nil
	t.Cleanup(func() { cfg, env = oldCfg, oldEnv })

	r := lifecycleRunner{}
	eps, err := r.Endpoints(context.Background(), dir, "p", map[string]string{"WEB_PORT": "4321"})
	if err != nil {
		t.Fatal(err)
	}
	if len(eps) != 1 || eps[0].URL != "http://localhost:4321" {
		t.Fatalf("endpoints = %+v", eps)
	}
	if got := rootEnvLoad(c).env.Interpolate("${WEB_PORT}"); got != "${WEB_PORT}" {
		t.Errorf("a variant's var leaked into the calling checkout's environment: WEB_PORT = %q", got)
	}
	if _, err := r.Endpoints(context.Background(), dir, "nope", nil); err == nil || errors.Is(err, errVariantConfigUnreadable) {
		t.Errorf("unknown plan: err = %v", err)
	}
}
