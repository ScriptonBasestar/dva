package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// routeChildYAML is the `engine/` child every case in this file routes into: one key its
// own validator rejects (`status` is a reserved built-in) and one it accepts (`test`). The
// pair is the point — a check that refused the whole subproject rather than the one key
// would pass every rejection case here and fail only on `test`.
const routeChildYAML = "interaction:\n  status:\n    command: echo CHILD-STATUS\n  test:\n    command: echo CHILD-TEST\n"

// writeParentAndChild lays down a parent dva.yml in a fresh temp dir plus the routeChildYAML
// `engine/` child, chdirs into the parent, and returns its path.
//
// No `version:` key on purpose — it declares the *minimum dva version*, not a schema
// version, so a value like "1.0" makes the load fail with an upgrade demand, and that
// failure is fatal inside mustLoadConfig (os.Exit(1)), which takes the whole test binary
// with it. Same trap unroutable_namespace_test.go records.
func writeParentAndChild(t *testing.T, parentYAML string) string {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "dva.yml"), []byte(parentYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	childDir := filepath.Join(dir, "engine")
	if err := os.MkdirAll(childDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(childDir, "dva.yml"), []byte(routeChildYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	// loadConfig caches into the package-level `cfg`, so without this every subtest after
	// the first routes against the previous one's parent config — whose FileDir points at a
	// TempDir that has already been removed. The established idiom (root_test.go,
	// hooks_test.go) is to clear it on both sides.
	cfg = nil
	t.Cleanup(func() { cfg = nil })
	return dir
}

// TestSubprojectParentRouteRejectsChildInvalidKey covers TASK-263 §3 decision (b) across
// all three parent address forms.
//
// The rule is that the parent must not offer an address the child would refuse. `status` is
// the case that made it necessary: measured on v0.1.48, `dva run --project engine status`
// printed CHILD-STATUS from a parent whose child's own `dva config validate` exits 1 on that
// very key. The parent was the only place the key worked, which leaves "which validator is
// authoritative" answered differently depending on where you stand.
//
// One test for three forms because they are one rule, and because two of them share
// runSubprojectCommand while the third does not — a per-form test file would let the import
// leg rot without anything going red.
//
// Each form is driven through its real entry point rather than by calling
// runSubprojectCommand with the two halves pre-split. That direct call is what
// unroutable_namespace_test.go records as the weakness worth avoiding: it hardcodes the
// split and so passes no matter what run.go does with the colon.
func TestSubprojectParentRouteRejectsChildInvalidKey(t *testing.T) {
	const declOnlyParent = "subprojects:\n  engine:\n    path: engine\n"

	assertRejected := func(t *testing.T, err error, form string) {
		t.Helper()
		if err == nil {
			t.Fatalf("%s reached the child's `status`; the child's own validator rejects that key", form)
		}
		if !strings.Contains(err.Error(), "rejected by `dva config validate` inside subproject `engine`") {
			t.Errorf("%s failed for some other reason: %v", form, err)
		}
	}

	t.Run("--project form", func(t *testing.T) {
		writeParentAndChild(t, declOnlyParent)

		// projectName is the package var the --project flag writes into, so setting it here
		// is what the flag does; RunE reads it as `resolvedProject`.
		prev := projectName
		projectName = "engine"
		t.Cleanup(func() { projectName = prev })

		assertRejected(t, runCmd.RunE(runCmd, []string{"status"}), "`dva run --project engine status`")
	})

	t.Run("colon shorthand form", func(t *testing.T) {
		writeParentAndChild(t, declOnlyParent)

		prev := projectName
		projectName = ""
		t.Cleanup(func() { projectName = prev })

		assertRejected(t, runCmd.RunE(runCmd, []string{"engine:status"}), "`dva engine:status`")
	})

	t.Run("slash import form", func(t *testing.T) {
		// Not driven through RunE: this form fails at config load, and mustLoadConfig turns a
		// load failure into os.Exit(1), which would end the test binary rather than the test.
		dir := writeParentAndChild(t,
			"subprojects:\n  engine:\n    path: engine\n    import:\n      interactions:\n        - status\n")

		_, err := config.Load(dir)
		assertRejected(t, err, "an `engine/status` import")
	})

	// The healthy key still routes, through the form that shares the checked path. Without
	// this the cheapest way to pass the three legs above is to refuse every subproject route.
	//
	// Under --explain, because the check under test runs before anything executes and the
	// execution itself is out of reach here: LocalRunner ends in syscall.Exec, which
	// dvaexec.ExecReplace panics on under `go test` rather than replacing the test binary
	// (TASK-144). Reaching Explain is exactly the assertion — it is past the rejection.
	t.Run("a key the child accepts still routes", func(t *testing.T) {
		writeParentAndChild(t, declOnlyParent)

		prevProject, prevDry := projectName, dryRun
		projectName, dryRun = "engine", true
		t.Cleanup(func() { projectName, dryRun = prevProject, prevDry })

		if err := runCmd.RunE(runCmd, []string{"test"}); err != nil {
			t.Fatalf("`dva run --project engine test --explain` failed: %v", err)
		}
	})
}
