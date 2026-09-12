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
	assertSkipWarnedFrom(t, verb, stderr, "")
}

// assertSkipWarnedFrom is assertSkipWarned with the composition label. Passing the child name
// is what makes the label an assertion rather than a decoration: the expected prefix is built
// from it, so dropping the label from printCompositionWarnings fails the composition tests
// without touching the leaf ones (TASK-375).
func assertSkipWarnedFrom(t *testing.T, verb, stderr, child string) {
	t.Helper()
	prefix := "warning: "
	if child != "" {
		prefix = "warning: [child: " + child + "] "
	}
	if !strings.Contains(stderr, prefix+"entry: vendor-api (optional) — skipped, directory ") {
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
	assertSkipWarnedFrom(t, "dva up <composition>", stderr, "dev")
}

func TestOptionalSkipIsReportedByCompositionStatus(t *testing.T) {
	enableDryRun(t)
	c, e := optionalSkipFixture(t)
	stderr := captureBothStreams(t, func() { _ = runCompositionStatus(c, planEnv(e), "all") })
	assertSkipWarnedFrom(t, "dva status <composition>", stderr, "dev")
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
	// No trace-leak assertion here, deliberately. A leaked ResolutionTrace could not make
	// the assertion above pass for the wrong reason: the trace renders its steps as
	// "  entry: ..." (printPlanResolution, two-space indent) while assertSkipWarned
	// requires the literal "warning: entry: ...", and only printPlanWarnings emits that
	// prefix. Guarding it would be a dead assertion that reads as a live one.
}

// incompleteEnvLoad builds the envLoad a route sees when the owner declared an env_file that
// is not there. ApplyEnvFiles rather than a hand-built EnvInputReport: the point of the test
// below is the route's behaviour on a real incomplete verdict, and a literal
// EnvInputState would pass even if the verdict stopped being reachable.
func incompleteEnvLoad(t *testing.T, c *config.Config, e *config.Environment) *envLoad {
	t.Helper()
	report := config.ApplyEnvFiles(map[string]any{"files": "absent.env", "required": true}, c.FileDir(), e)
	if !report.Incomplete() {
		t.Fatalf("fixture did not produce an incomplete env report: %+v", report)
	}
	return &envLoad{env: e, report: report}
}

// The ordering rule of TASK-375, pinned on the one verb that used to break it.
//
// runPlanUp emitted the warning after runtime.report.Err(), so a plan whose environment was
// incomplete was rejected in silence — while `dva build` and `dva status`, which warn first,
// showed the skip for the same input. The two assertions are the rule stated from both ends:
// the warning is there, and the `[plan: ...]` header — the first thing printed on the far
// side of the rejection check — is not. Only an emission that happens before the check can
// produce that pair.
func TestOptionalSkipWarnsBeforeTheVerbRejects(t *testing.T) {
	enableDryRun(t)
	c, e := optionalSkipFixture(t)
	var err error
	stderr := captureBothStreams(t, func() { err = runPlanUp(c, incompleteEnvLoad(t, c, e), "dev", nil) })
	if err == nil {
		t.Fatal("incomplete environment inputs must still reject the run")
	}
	assertSkipWarned(t, "dva up (incomplete env)", stderr)
	if strings.Contains(stderr, "[plan: ") {
		t.Errorf("the plan header describes work that never happened; stderr:\n%s", stderr)
	}
}

// The same rule on the path that does proceed: warning first, header second. Without this,
// moving the call back below the header would still pass the test above, because that one
// only ever runs on an input where the header never prints at all.
func TestOptionalSkipWarningPrecedesThePlanHeader(t *testing.T) {
	enableDryRun(t)
	c, e := optionalSkipFixture(t)
	stderr := captureBothStreams(t, func() { _ = runPlanUp(c, planEnv(e), "dev", nil) })
	assertSkipWarned(t, "dva up", stderr)
	warn, header := strings.Index(stderr, "warning: entry:"), strings.Index(stderr, "[plan: ")
	if header < 0 {
		t.Fatalf("expected the plan header on the execution path; stderr:\n%s", stderr)
	}
	if warn > header {
		t.Errorf("warning came after the plan header; stderr:\n%s", stderr)
	}
}
