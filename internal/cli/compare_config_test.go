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

func upConfig(t *testing.T, c *config.Config, args ...string) (map[string]any, error) {
	t.Helper()
	return captureJSON(t, func() error {
		return runCompareUp(context.Background(), c, "layout", args, false)
	})
}

// Two config variants run in one checkout as two instances, each with its own port and its own
// DVA_VARIANT, and dva creates no checkout for either.
func TestCompareUpConfigVariantsRunAsSeparateInstances(t *testing.T) {
	c, host, ports := compareFixture(t)
	doc, err := upConfig(t, c, "a=config", "b=config")
	if err != nil {
		t.Fatal(err)
	}
	for i, n := range []string{"a", "b"} {
		row := variantRow(t, doc, n)
		if row["mode"] != compare.ModeConfig || row["outcome"] != outcomeStarted {
			t.Errorf("%s = %v", n, row)
		}
		if got := host.vars["shared#"+n]; got[variantVar] != n || got["WEB_PORT"] != itoa(ports[i]) {
			t.Errorf("%s vars = %v, want %s=%s WEB_PORT=%d", n, got, variantVar, n, ports[i])
		}
	}
	if host.count("up:shared#a:web") != 1 || host.count("up:shared#b:web") != 1 || host.count("up:shared:api") != 1 {
		t.Errorf("calls = %v", host.calls)
	}
	rec, err := compare.ReadRecord(c.FileDir(), "layout")
	if err != nil {
		t.Fatal(err)
	}
	if v := rec.Variant("b"); v == nil || v.Instance != "b" || v.Path != c.FileDir() || v.ConfigDir != c.FileDir() {
		t.Errorf("record b = %+v", v)
	}
	if _, err := os.Stat(filepath.Join(c.FileDir(), ".sb/dva/variants")); err == nil {
		t.Error("config mode must not create a checkout")
	}
}

// down takes one config variant at a time: it stops that instance only, leaves the other
// running and the shared entries up, and never removes the calling checkout.
func TestCompareDownConfigVariantStopsOnlyItsInstance(t *testing.T) {
	c, host, _ := compareFixture(t)
	if _, err := upConfig(t, c, "a=config", "b=config"); err != nil {
		t.Fatal(err)
	}
	host.calls = nil
	if _, err := captureJSON(t, func() error { return runCompareDown(context.Background(), c, "layout", []string{"a"}) }); err != nil {
		t.Fatal(err)
	}
	if host.count("down:shared#a:web") != 1 || host.count("down:shared#b") != 0 || host.count("down:shared:") != 0 {
		t.Errorf("calls = %v", host.calls)
	}
	if _, err := os.Stat(filepath.Join(c.FileDir(), "dva.yml")); err != nil {
		t.Errorf("down removed the calling checkout: %v", err)
	}
	rec, err := compare.ReadRecord(c.FileDir(), "layout")
	if err != nil || rec.Variant("a") != nil || rec.Variant("b") == nil {
		t.Fatalf("record = %+v err=%v", rec, err)
	}
	host.calls = nil
	if _, err := captureJSON(t, func() error { return runCompareDown(context.Background(), c, "layout", []string{"b"}) }); err != nil {
		t.Fatal(err)
	}
	if host.count("down:shared#b:web") != 1 || host.count("down:shared:api") != 1 {
		t.Errorf("last variant must also take the shared entries down: %v", host.calls)
	}
}

// A config variant can sit beside a snapshot: only the config one is an instance.
func TestCompareUpMixesConfigAndSnapshotVariants(t *testing.T) {
	c, host, _ := compareFixture(t)
	if _, err := upConfig(t, c, "a=config", "b=other"); err != nil {
		t.Fatal(err)
	}
	if host.count("up:shared#a:web") != 1 || host.count("up:b:web") != 1 {
		t.Errorf("calls = %v", host.calls)
	}
	if _, set := host.vars["b"][variantVar]; set {
		t.Errorf("a snapshot variant must not get %s: %v", variantVar, host.vars["b"])
	}
}

// Status reads the instance's own pid file: a dead pid there marks that variant stale, a pid
// file at the ordinary path (another run's) does not, and a config variant has no ref drift.
func TestCompareStatusConfigVariantReadsItsInstancePid(t *testing.T) {
	c, _, _ := compareFixture(t)
	if _, err := upConfig(t, c, "a=config", "b=config"); err != nil {
		t.Fatal(err)
	}
	pids := filepath.Join(c.FileDir(), ".sb/dva/pids")
	write := func(rel, body string) {
		p := filepath.Join(pids, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("a/web.pid", "999999\n") // not running
	write("web.pid", "999998\n")   // the ordinary run's, not b's
	doc, err := captureJSON(t, func() error { return runCompareStatus(context.Background(), c, "layout") })
	if err != nil {
		t.Fatal(err)
	}
	a, b := variantRow(t, doc, "a"), variantRow(t, doc, "b")
	if a["stale"] != true || !strings.Contains(asJSON(a["stale_reasons"]), "999999") {
		t.Errorf("a = %v", a)
	}
	if b["stale"] == true {
		t.Errorf("b read a pid file that is not its own: %v", b)
	}
	if a["drift"] != nil || b["drift"] != nil {
		t.Errorf("config variants have no ref to drift from: %v %v", a["drift"], b["drift"])
	}
}

// A per_variant command that keeps build state in the checkout would be shared by two config
// variants. It stops before anything starts, names the entry, and is satisfied by a command
// that uses the variant var.
func TestCompareUpConfigVariantRejectsSharedBuildState(t *testing.T) {
	c, host, _ := compareFixture(t)
	native := c.Stack["web"].Runners["native"].(*config.NativeRunnerConfig)
	native.Run = "pnpm exec next dev -p $WEB_PORT"

	_, err := upConfig(t, c, "a=config", "b=config")
	if err == nil || errorCode(err) != codeComparePreflight ||
		!strings.Contains(err.Error(), "entry web runs next") || !strings.Contains(err.Error(), variantVar) {
		t.Fatalf("err = %v", err)
	}
	if len(host.calls) != 0 {
		t.Errorf("a preflight failure must start nothing: %v", host.calls)
	}
	if _, statErr := compare.ReadRecord(c.FileDir(), "layout"); statErr == nil {
		t.Error("a preflight failure must not write a record")
	}

	// The same tool in a snapshot variant is fine: it has its own checkout.
	if _, err := upConfig(t, c, "a=main", "b=other"); err != nil {
		t.Errorf("snapshot variants must not be refused: %v", err)
	}
}

func TestCompareUpConfigVariantAcceptsSplitBuildState(t *testing.T) {
	c, host, _ := compareFixture(t)
	native := c.Stack["web"].Runners["native"].(*config.NativeRunnerConfig)
	native.Run = "NEXT_DIST_DIR=.next-$DVA_VARIANT pnpm exec next dev -p $WEB_PORT"
	if _, err := upConfig(t, c, "a=config", "b=config"); err != nil {
		t.Fatalf("err = %v", err)
	}
	if host.count("up:shared#a:web") != 1 {
		t.Errorf("calls = %v", host.calls)
	}
}

// An OS DVA_VARIANT outranks every variant's own value, which would put the variants back on one
// build state while the preflight still passes; it is refused like a port var set in the OS.
func TestCompareUpConfigVariantRefusesOSVariantVar(t *testing.T) {
	c, host, _ := compareFixture(t)
	t.Setenv(variantVar, "outer")
	_, err := upConfig(t, c, "a=config", "b=config")
	if err == nil || errorCode(err) != codeComparePreflight || !strings.Contains(err.Error(), "environment sets "+variantVar+"=outer") {
		t.Fatalf("err = %v", err)
	}
	if len(host.calls) != 0 {
		t.Errorf("a preflight failure must start nothing: %v", host.calls)
	}
	// Snapshot variants never get the var, so the OS value is no conflict there.
	if _, err := upConfig(t, c, "a=main", "b=other"); err != nil {
		t.Errorf("snapshot variants must not be refused: %v", err)
	}
}

// "A" and "a" are one directory on a case-insensitive filesystem, so they cannot be two variants.
func TestParseVariantSpecsRejectsNamesDifferingByCase(t *testing.T) {
	_, err := parseVariantSpecs("layout", &config.CompareConfig{}, []string{"A=config", "a=config"})
	if err == nil || !strings.Contains(err.Error(), "twice") {
		t.Fatalf("err = %v", err)
	}
}

// The pids and log paths in the up/status JSON come from the variant's own instance directory.
func TestAnnotateProcessesReadsTheInstanceDirectory(t *testing.T) {
	dir := t.TempDir()
	write := func(path, content string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(dir, ".sb/dva/pids/a/web.pid"), "111\n")
	write(filepath.Join(dir, ".sb/dva/logs/a/web.log"), "x")
	write(filepath.Join(dir, ".sb/dva/pids/web.pid"), "999\n")

	var row compareVariantResult
	annotateProcesses(&row, dir, "a", []string{"web"})
	if row.Pids["web"] != 111 || len(row.LogPaths) != 1 || row.LogPaths[0] != filepath.Join(dir, ".sb/dva/logs/a/web.log") {
		t.Errorf("instance row = pids %v logs %v", row.Pids, row.LogPaths)
	}
	var plain compareVariantResult
	annotateProcesses(&plain, dir, "", []string{"web"})
	if plain.Pids["web"] != 999 || len(plain.LogPaths) != 0 {
		t.Errorf("ordinary row = pids %v logs %v", plain.Pids, plain.LogPaths)
	}
}
