package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// planScopedHookConfig declares two plans over one stack entry and hangs three after-hooks
// off `up`: one bound to each plan, and one unfiltered.
//
// The fixture is what TASK-331 was filed against, reduced: careerarchive-devbox had a
// Penpot seed hook written for its `design` plan, and adding a second plan made `dva up
// verify` fail on it while every entry came up healthy. The unfiltered third hook is not
// decoration — it is the compatibility half of the claim, and a filter implementation that
// dropped it would still pass a test that only asserted the filtered two.
const planScopedHookConfig = `version: "0.1.22"
stack:
  compose:
    default_runner: compose
    order: 10
    runners:
      compose:
        files: [compose.yml]
plans:
  design:
    entries:
      - name: compose
  verify:
    entries:
      - name: compose
`

// hookMarkers gives each hook step a distinct file to touch, so "which hooks ran" is read
// off the filesystem rather than off parsed output. Output assertions would couple the test
// to the log format that TASK-375 has already had to re-pin once.
type hookMarkers struct {
	dir                    string
	design, verify, always string
}

// beforeMarkers is the same idea for the `before` phase. A separate fixture rather than more
// steps in the shared one: the after-phase tests assert on counts and on stderr, and folding
// two phases into one config would make each of them pass for reasons the other supplied.
type beforeMarkers struct {
	design, always string
}

func newHookMarkers(t *testing.T) hookMarkers {
	t.Helper()
	dir := t.TempDir()
	return hookMarkers{
		dir:    dir,
		design: filepath.Join(dir, "design"),
		verify: filepath.Join(dir, "verify"),
		always: filepath.Join(dir, "always"),
	}
}

func (m hookMarkers) config() string {
	return planScopedHookConfig + fmt.Sprintf(`interaction:
  up:
    after:
      - step: Seed the design stack
        plans: [design]
        run: "touch %s"
      - step: Seed the verify stack
        plans: [verify]
        run: "touch %s"
      - step: Warm the shared cache
        run: "touch %s"
`, m.design, m.verify, m.always)
}

func newBeforeMarkers(t *testing.T) beforeMarkers {
	t.Helper()
	dir := t.TempDir()
	return beforeMarkers{design: filepath.Join(dir, "b-design"), always: filepath.Join(dir, "b-always")}
}

func (m beforeMarkers) config() string {
	return planScopedHookConfig + fmt.Sprintf(`interaction:
  up:
    before:
      - step: Provision the design secrets
        plans: [design]
        run: "touch %s"
      - step: Check the docker daemon
        run: "touch %s"
`, m.design, m.always)
}

func (m hookMarkers) ran(t *testing.T, path string) bool {
	t.Helper()
	_, err := os.Stat(path)
	return err == nil
}

// runUpWithHooks drives the wrapped `up` RunE with args and returns what the hook loop wrote
// to stderr.
func runUpWithHooks(t *testing.T, cfgText string, args []string) string {
	t.Helper()
	c := loadTestConfig(t, cfgText)

	oldCfg, oldEnv, oldDryRun := cfg, env, dryRun
	cfg = c
	env = planEnv(config.NewEnvironment(nil, c.FileDir(), c.FileDir()))
	dryRun = false
	t.Cleanup(func() { cfg, env, dryRun = oldCfg, oldEnv, oldDryRun })

	cmd := &cobra.Command{RunE: func(_ *cobra.Command, _ []string) error { return nil }}
	wrapWithHooks("up", cmd)

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	err := cmd.RunE(cmd, args)
	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	if _, copyErr := buf.ReadFrom(r); copyErr != nil {
		t.Fatalf("read stderr: %v", copyErr)
	}
	if err != nil {
		t.Fatalf("dva up %v: %v\nstderr:\n%s", args, err, buf.String())
	}
	return buf.String()
}

// TestPlanScopedHooks_FilteredHookDoesNotRunOnAnotherPlan is TASK-331's second acceptance
// criterion.
func TestPlanScopedHooks_FilteredHookDoesNotRunOnAnotherPlan(t *testing.T) {
	m := newHookMarkers(t)
	out := runUpWithHooks(t, m.config(), []string{"verify"})

	if m.ran(t, m.design) {
		t.Errorf("the design-only hook ran under `dva up verify`\nstderr:\n%s", out)
	}
	if !m.ran(t, m.verify) {
		t.Errorf("the verify hook did not run under `dva up verify`\nstderr:\n%s", out)
	}
	// A skipped step that says nothing is the failure mode this key introduces, so the
	// announcement is part of the contract and not a log-format detail (docs/64 §3).
	if !strings.Contains(out, "Seed the design stack") || !strings.Contains(out, "skipped") {
		t.Errorf("skipped hook was not announced on stderr:\n%s", out)
	}
}

// TestPlanScopedHooks_BeforePhaseIsFilteredToo exists because every other test in this file
// drives `after:`, and the three phases are filtered by three separate call sites in
// wrapWithHooks. Verified by reverting only the `before` call site to the unfiltered lists:
// the rest of the suite stayed green while a design-only hook fired on `dva up verify` —
// which is TASK-331's original defect, one phase over.
//
// `before` is also the phase where the defect costs the most. An after-hook fails a command
// that already did its work; a before-hook fails it before the stack comes up at all.
func TestPlanScopedHooks_BeforePhaseIsFilteredToo(t *testing.T) {
	m := newBeforeMarkers(t)
	out := runUpWithHooks(t, m.config(), []string{"verify"})

	if _, err := os.Stat(m.design); err == nil {
		t.Errorf("the design-only before-hook ran under `dva up verify`\nstderr:\n%s", out)
	}
	if _, err := os.Stat(m.always); err != nil {
		t.Errorf("the unfiltered before-hook did not run under `dva up verify`\nstderr:\n%s", out)
	}
	// The announcement is per-phase too: reportSkippedHookSteps is called three times, and
	// dropping the `before` call leaves a step that silently does not run.
	if !strings.Contains(out, "[hook:before:up]") || !strings.Contains(out, "Provision the design secrets") {
		t.Errorf("the skipped before-hook was not announced on stderr:\n%s", out)
	}
}

// TestPlanScopedHooks_UnfilteredHookRunsOnEveryPlan is TASK-331's third acceptance criterion:
// the compatibility guarantee that every dva.yml written before `plans:` existed keeps
// behaving as it did.
func TestPlanScopedHooks_UnfilteredHookRunsOnEveryPlan(t *testing.T) {
	for _, plan := range []string{"design", "verify"} {
		t.Run(plan, func(t *testing.T) {
			m := newHookMarkers(t)
			out := runUpWithHooks(t, m.config(), []string{plan})
			if !m.ran(t, m.always) {
				t.Errorf("the unfiltered hook did not run under `dva up %s`\nstderr:\n%s", plan, out)
			}
		})
	}
}

// TestPlanScopedHooks_FilteredHookSkipsWhenNoPlanRoutes pins the third row of docs/64 §3's
// table. It is the row a reader is most likely to expect the other way round, so it gets its
// own test rather than a branch inside another.
func TestPlanScopedHooks_FilteredHookSkipsWhenNoPlanRoutes(t *testing.T) {
	m := newHookMarkers(t)
	out := runUpWithHooks(t, m.config(), nil)

	if m.ran(t, m.design) || m.ran(t, m.verify) {
		t.Errorf("a plan-filtered hook ran with no routed plan\nstderr:\n%s", out)
	}
	if !m.ran(t, m.always) {
		t.Errorf("the unfiltered hook did not run with no routed plan\nstderr:\n%s", out)
	}
	if !strings.Contains(out, "running plan is none") {
		t.Errorf("skip line did not name the absent route:\n%s", out)
	}
}

// TestPlanScopedHooks_DefaultPlanRoutesTheFilter pins that the filter compares against the
// plan that will actually run, not against the argv the user typed. With `default_plan`, a
// bare `dva up` routes to a plan and the filter for that plan must fire.
func TestPlanScopedHooks_DefaultPlanRoutesTheFilter(t *testing.T) {
	m := newHookMarkers(t)
	out := runUpWithHooks(t, "default_plan: design\n"+m.config(), nil)

	if !m.ran(t, m.design) {
		t.Errorf("default_plan did not route the filter; the design hook was skipped\nstderr:\n%s", out)
	}
	if m.ran(t, m.verify) {
		t.Errorf("the verify hook ran under default_plan: design\nstderr:\n%s", out)
	}
}

// TestPlanScopedHooks_FilteredReplaceFallsBackToBuiltin is the ordering claim in
// wrapWithHooks: filtering happens before `len(replace) > 0` is measured.
//
// Measured against the inverted implementation — filtering after that test — the built-in
// never runs and neither does any replace step, so `dva up design` succeeds having done
// nothing at all. That is the one outcome `plans:` cannot be allowed to produce, because no
// declaration in the file asks for it.
func TestPlanScopedHooks_FilteredReplaceFallsBackToBuiltin(t *testing.T) {
	dir := t.TempDir()
	replaced := filepath.Join(dir, "replaced")
	c := loadTestConfig(t, planScopedHookConfig+fmt.Sprintf(`interaction:
  up:
    replace:
      - step: Bring up the design stack by hand
        plans: [verify]
        run: "touch %s"
`, replaced))

	oldCfg, oldEnv, oldDryRun := cfg, env, dryRun
	cfg = c
	env = planEnv(config.NewEnvironment(nil, c.FileDir(), c.FileDir()))
	dryRun = false
	t.Cleanup(func() { cfg, env, dryRun = oldCfg, oldEnv, oldDryRun })

	builtinRan := false
	cmd := &cobra.Command{RunE: func(_ *cobra.Command, _ []string) error {
		builtinRan = true
		return nil
	}}
	wrapWithHooks("up", cmd)

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	err := cmd.RunE(cmd, []string{"design"})
	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	if _, copyErr := buf.ReadFrom(r); copyErr != nil {
		t.Fatalf("read stderr: %v", copyErr)
	}
	out := buf.String()

	if err != nil {
		t.Fatalf("dva up design: %v\nstderr:\n%s", err, out)
	}
	if !builtinRan {
		t.Error("a replace list filtered away for this plan suppressed the built-in instead of yielding to it")
	}
	if _, statErr := os.Stat(replaced); statErr == nil {
		t.Error("the verify-only replace step ran under `dva up design`")
	}
	// The replace phase has its own announcement call, and this is the phase where silence
	// misleads most: the output is the built-in's, indistinguishable from a config that
	// declared no replace at all.
	if !strings.Contains(out, "[hook:replace:up]") || !strings.Contains(out, "Bring up the design stack by hand") {
		t.Errorf("the skipped replace step was not announced on stderr:\n%s", out)
	}
}

// TestSkipLineExamplesInDocsMatchTheRenderedFormat pins docs/64 and USAGE.md to the code
// that produces the line, rather than to a second copy of the string.
//
// review-331 F5 found the docs/64 example carrying a `[1/2]` progress counter the code never
// emits. A grep-for-the-literal binding would not have caught it — that stores the wrong
// string twice. Rendering the expected line through the real printer means editing either
// half of the format (config.StepsForPlan's suffix or reportSkippedHookSteps' prefix) fails
// here instead of silently desyncing both documents.
func TestSkipLineExamplesInDocsMatchTheRenderedFormat(t *testing.T) {
	// The exact step and plan names the two documents use in their examples.
	for _, tc := range []struct {
		doc, step, filter, running string
	}{
		{"docs/64-plan-scoped-interaction-hooks.md", "Seed the local Penpot account and Stage 0 file", "design", "verify"},
		{"USAGE.md", "Penpot 계정 시드", "design", "verify"},
	} {
		t.Run(tc.doc, func(t *testing.T) {
			body, err := os.ReadFile(filepath.Join("..", "..", tc.doc))
			if err != nil {
				t.Fatalf("reading %s: %v", tc.doc, err)
			}

			_, skipped := config.StepsForPlan(
				[]config.ProvisionItem{{Step: tc.step, Run: "true", Plans: []string{tc.filter}}},
				tc.running,
			)
			if len(skipped) != 1 {
				t.Fatalf("fixture produced %d skip lines, want 1; the comparison would be vacuous", len(skipped))
			}

			rendered := strings.TrimSpace(captureStderr(t, func() {
				reportSkippedHookSteps("after", "up", skipped)
			}))
			if rendered == "" {
				t.Fatal("the printer rendered nothing; the comparison would be vacuous")
			}
			if !strings.Contains(string(body), rendered) {
				t.Errorf("%s does not contain the line this printer renders;\n want: %s", tc.doc, rendered)
			}
		})
	}
}

// captureStderr runs fn with os.Stderr redirected and returns what it wrote.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = w
	fn()
	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("read stderr: %v", err)
	}
	return buf.String()
}
