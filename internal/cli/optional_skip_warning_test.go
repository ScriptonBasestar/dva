package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// TASK-374 follow-up: the optional-skip warning has to reach the user from every verb that
// resolves a plan, not only the four lifecycle verbs in plan_lifecycle.go. An entry dropped
// for a missing directory was invisible outside --dry-run; wiring only `up`/`down`/`stop`/
// `restart` would have left `build`, `status`, `logs` and every composition verb exactly as
// silent as the original bug.
func optionalSkipFixture(t *testing.T) (*config.Config, *config.Environment) {
	t.Helper()
	c := loadTestConfig(t, `version: "0.1.44"
stack:
  vendor-api:
    optional: true
    default_runner: native
    runners:
      native:
        dir: vendor/api
        run: echo vendor
  infra:
    default_runner: compose
    runners:
      compose:
        files: [compose.yml]
plans:
  dev:
    entries:
      - name: vendor-api
      - name: infra
  vendor-only:
    entries:
      - name: vendor-api
  all:
    composes:
      - plan: dev
`)
	if err := os.WriteFile(filepath.Join(c.FileDir(), "compose.yml"), []byte("services:\n  api: {image: alpine}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return c, config.NewEnvironment(nil, c.FileDir(), c.FileDir())
}

// assertSkipWarned requires the entry name, the word "optional" and the missing directory to
// all be present. Asserting on the whole sentence rather than a substring keeps the test from
// passing on an unrelated line that merely mentions the entry.
func assertSkipWarned(t *testing.T, verb, stderr string) {
	t.Helper()
	if !strings.Contains(stderr, "warning: entry: vendor-api (optional) — skipped, directory ") {
		t.Errorf("%s: optional skip was not reported on the execution path; stderr:\n%s", verb, stderr)
	}
	if !strings.Contains(stderr, filepath.Join("vendor", "api")) {
		t.Errorf("%s: warning did not name the missing directory; stderr:\n%s", verb, stderr)
	}
}

func TestOptionalSkipIsReportedByPlanBuild(t *testing.T) {
	enableDryRun(t)
	c, e := optionalSkipFixture(t)
	var err error
	stderr := captureBothStreams(t, func() { err = runPlanBuild(c, planEnv(e), "dev", nil) })
	if err != nil {
		t.Fatalf("runPlanBuild: %v", err)
	}
	assertSkipWarned(t, "dva build", stderr)
}

func TestOptionalSkipIsReportedByPlanStatus(t *testing.T) {
	enableDryRun(t)
	c, e := optionalSkipFixture(t)
	stderr := captureBothStreams(t, func() { _ = runPlanStatus(c, planEnv(e), "dev") })
	assertSkipWarned(t, "dva status", stderr)
}

// `dva logs` hands off with a syscall.Exec passthrough, which the repository forbids under
// `go test` (TASK-144) because it would replace the test binary. The plan used here therefore
// contains the optional entry and nothing else: once it is skipped there is no child left to
// exec into, so the warning can be observed without tripping that guard.
func TestOptionalSkipIsReportedByPlanLogs(t *testing.T) {
	enableDryRun(t)
	c, e := optionalSkipFixture(t)
	stderr := captureBothStreams(t, func() { _ = runPlanLogs(c, planEnv(e), "vendor-only", nil) })
	assertSkipWarned(t, "dva logs", stderr)
}

// A composition plan owns no stack entries of its own — the skip happens inside a child, and
// before this change nothing walked the children's warnings, so a composition verb was silent
// even though the leaf verb for the same child was not.
func TestOptionalSkipIsReportedByCompositionUp(t *testing.T) {
	enableDryRun(t)
	c, e := optionalSkipFixture(t)
	stderr := captureBothStreams(t, func() { _ = runCompositionUp(c, planEnv(e), "all", nil) })
	assertSkipWarned(t, "dva up <composition>", stderr)
}

func TestOptionalSkipIsReportedByCompositionStatus(t *testing.T) {
	enableDryRun(t)
	c, e := optionalSkipFixture(t)
	stderr := captureBothStreams(t, func() { _ = runCompositionStatus(c, planEnv(e), "all") })
	assertSkipWarned(t, "dva status <composition>", stderr)
}

// The counterpart guard: a plan with nothing skipped must stay silent. Without this, every
// assertion above would still pass if printPlanWarnings printed unconditionally.
func TestNoWarningWhenNothingIsSkipped(t *testing.T) {
	enableDryRun(t)
	c, e := optionalSkipFixture(t)
	if err := os.MkdirAll(filepath.Join(c.FileDir(), "vendor", "api"), 0o755); err != nil {
		t.Fatal(err)
	}
	stderr := captureBothStreams(t, func() { _ = runPlanStatus(c, planEnv(e), "dev") })
	if strings.Contains(stderr, "warning: entry:") {
		t.Errorf("directory exists, so nothing should be warned about; stderr:\n%s", stderr)
	}
}

// disableDryRun is the deliberate counterpart to enableDryRun: it pins the global to false
// rather than trusting its zero value, so the test below cannot be quietly turned into a
// dry-run test by an unrelated change to how these tests are set up.
func disableDryRun(t *testing.T) {
	t.Helper()
	old := dryRun
	dryRun = false
	t.Cleanup(func() { dryRun = old })
}

// Every other test in this file runs under --dry-run, which leaves the property the card is
// actually named for untested: the skip was already visible under --dry-run before TASK-374
// (ResolutionTrace carried it), and the bug was that it was invisible *outside* --dry-run.
//
// That gap is not theoretical. Re-guarding the emission as `if dryRun { printPlanWarnings(...) }`
// — the exact bug this task exists to fix — leaves every dry-run test above still passing.
// They pin that the call exists, not that it fires on the execution path. This one does.
func TestOptionalSkipIsReportedOutsideDryRun(t *testing.T) {
	disableDryRun(t)
	c, e := optionalSkipFixture(t)
	stderr := captureBothStreams(t, func() { _ = runPlanStatus(c, planEnv(e), "dev") })
	assertSkipWarned(t, "dva status (no --dry-run)", stderr)
	// The trace is the dry-run narration and must stay out of the default output; if it
	// leaked here the assertion above would pass for the wrong reason.
	if strings.Contains(stderr, "resolution:") {
		t.Errorf("resolution trace leaked outside --dry-run; stderr:\n%s", stderr)
	}
}
