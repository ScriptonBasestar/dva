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
	return writeParentAndChildYAML(t, parentYAML, routeChildYAML)
}

// writeParentAndChildYAML is writeParentAndChild for the one case that needs a child other
// than routeChildYAML. Everything below the child's bytes — the chdir and the `cfg` reset on
// both sides — is identical and lives here so it cannot drift between the two entry points.
func writeParentAndChildYAML(t *testing.T, parentYAML, childYAML string) string {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "dva.yml"), []byte(parentYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	childDir := filepath.Join(dir, "engine")
	if err := os.MkdirAll(childDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(childDir, "dva.yml"), []byte(childYAML), 0o644); err != nil {
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

	// dryRun on both rejection legs, for the failure mode rather than the assertion. The
	// check under test runs before dryRun is consulted, so this cannot weaken either leg —
	// but when one of them breaks, execution falls through to LocalRunner, whose syscall.Exec
	// makes dvaexec.ExecReplace panic under `go test` (TASK-144). A panic ends the whole test
	// binary, so the first broken leg is the only one anybody sees and the second never runs.
	// Measured: deleting the check in run.go reported only `--project form` until this was
	// set, and reports both legs as clean FAILs with it.
	t.Run("--project form", func(t *testing.T) {
		writeParentAndChild(t, declOnlyParent)

		// projectName is the package var the --project flag writes into, so setting it here
		// is what the flag does; RunE reads it as `resolvedProject`.
		prevProject, prevDry := projectName, dryRun
		projectName, dryRun = "engine", true
		t.Cleanup(func() { projectName, dryRun = prevProject, prevDry })

		assertRejected(t, runCmd.RunE(runCmd, []string{"status"}), "`dva run --project engine status`")
	})

	t.Run("colon shorthand form", func(t *testing.T) {
		writeParentAndChild(t, declOnlyParent)

		prevProject, prevDry := projectName, dryRun
		projectName, dryRun = "", true
		t.Cleanup(func() { projectName, dryRun = prevProject, prevDry })

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

// taggedChildYAML is routeChildYAML with the rejected key carrying a tag, so a parent can
// exclude exactly the key whose rejection is under test.
const taggedChildYAML = "interaction:\n  status:\n    command: echo CHILD-STATUS\n    tags:\n      - infra\n  test:\n    command: echo CHILD-TEST\n"

// TestSubprojectParentRouteExcludedKeyStaysNotFound pins the order between the parent's
// exclude_tags filter and the child's own rejection, which is a choice rather than an
// accident and was made silently the first time.
//
// `exclude_tags` is the parent saying it does not offer the key, and `dva ls --project`
// applies the filter and omits it. With the rejection checked first, one spelling got two
// answers from the same binary: `ls --project engine` did not list `status` while
// `run --project engine status` reported it as existing-but-refused. That is the shape
// LiteralKeyWins' own comment rules out, so the filter decides first and an excluded key is
// "not found" on both surfaces.
//
// Both legs run against the same child. A test that only proved the excluded key says "not
// found" would also pass if the rejection had simply been deleted; the second leg is what
// separates "the filter came first" from "the check is gone".
func TestSubprojectParentRouteExcludedKeyStaysNotFound(t *testing.T) {
	const rejection = "rejected by `dva config validate` inside subproject `engine`"

	t.Run("excluded reserved key reports not found", func(t *testing.T) {
		writeParentAndChildYAML(t,
			"subprojects:\n  engine:\n    path: engine\n    exclude_tags:\n      - infra\n",
			taggedChildYAML)

		prevProject, prevDry := projectName, dryRun
		projectName, dryRun = "engine", true
		t.Cleanup(func() { projectName, dryRun = prevProject, prevDry })

		err := runCmd.RunE(runCmd, []string{"status"})
		if err == nil {
			t.Fatal("`dva run --project engine status` succeeded; the parent excluded that key")
		}
		if !strings.Contains(err.Error(), "not found in subproject `engine`") {
			t.Errorf("excluded key did not report not-found: %v", err)
		}
		if strings.Contains(err.Error(), rejection) {
			t.Errorf("excluded key leaked the child's rejection, which names a key the parent does not offer: %v", err)
		}
	})

	t.Run("the same key still gets the rejection when not excluded", func(t *testing.T) {
		writeParentAndChildYAML(t, "subprojects:\n  engine:\n    path: engine\n", taggedChildYAML)

		prevProject, prevDry := projectName, dryRun
		projectName, dryRun = "engine", true
		t.Cleanup(func() { projectName, dryRun = prevProject, prevDry })

		err := runCmd.RunE(runCmd, []string{"status"})
		if err == nil {
			t.Fatal("`dva run --project engine status` reached the child's `status`")
		}
		if !strings.Contains(err.Error(), rejection) {
			t.Errorf("offered key lost the child's rejection: %v", err)
		}
	})
}
