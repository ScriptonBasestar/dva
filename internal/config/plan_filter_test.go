package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// planFilterFixture is the smallest config that can carry a hook plan filter: two declared
// plans over one stack entry, plus the `up` hook the filter hangs off. `%s` takes the
// filter's plan list, which is the only thing each case varies.
//
// The compose file has to exist on disk — validateEntrySource resolves declared sources
// against the config dir — so every case writes it alongside dva.yml.
const planFilterFixture = `version: "0.1.44"
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
interaction:
  up:
    after:
      - step: seed
        run: "true"
        plans: [%s]
`

// loadPlanFilterConfig writes the fixture and its compose file, then Loads. Load rather
// than a literal &Config{}: Validate refuses a Config whose filePath is empty
// (validate.go:2), and filePath is unexported with no setter.
func loadPlanFilterConfig(t *testing.T, planList string) *Config {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "compose.yml"), []byte("services: {}\n"), 0644); err != nil {
		t.Fatalf("write compose.yml: %v", err)
	}
	return loadConfigForSchemaTest(t, dir, strings.Replace(planFilterFixture, "%s", planList, 1))
}

// TestValidateRejectsUndeclaredPlanInHookFilter is TASK-331's criterion 4 and drives
// Validate() rather than validateHookPlanFilters directly — the check is only worth
// anything if it is *wired*, and TASK-140 shipped a warning that worked and was never
// registered. Calling the helper would pass with the call in Validate deleted.
//
// The claim is specifically that this is an error, not a warning: a misspelled filter
// matches no plan, so the step is skipped on every run, and a skip is exactly what a
// *correct* filter produces. Nothing downstream can tell the two apart (docs/64 §4).
func TestValidateRejectsUndeclaredPlanInHookFilter(t *testing.T) {
	cfg := loadPlanFilterConfig(t, "desing") // the typo the check exists for

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() accepted a hook filter naming an undeclared plan; a step that can never run must not validate clean")
	}
	got := err.Error()
	for _, want := range []string{
		"interaction.up.after[0].plans",
		`"desing" is not a declared plan`,
		"Available: design, verify", // sorted, and it names the way out
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Validate() error missing %q:\n%s", want, got)
		}
	}
}

// TestValidateAcceptsDeclaredPlanInHookFilter is the control. Rejecting the typo is only
// useful if the feature it guards still validates — a check keyed off the presence of
// `plans:` rather than its contents would fail here and take the whole feature with it.
func TestValidateAcceptsDeclaredPlanInHookFilter(t *testing.T) {
	cfg := loadPlanFilterConfig(t, "design")

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() rejected a filter naming a declared plan: %v", err)
	}
}

// TestValidateReportsEveryUndeclaredPlanInOnePass pins TASK-305's one-pass contract and,
// with it, the sort. `interaction` is a map, so an unsorted result would reorder between
// runs and make `dva validate` output undiffable (TASK-128).
func TestValidateReportsEveryUndeclaredPlanInOnePass(t *testing.T) {
	cfg := loadPlanFilterConfig(t, "zzz, aaa")

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() accepted two undeclared plans in one filter")
	}
	got := err.Error()
	aaa := strings.Index(got, `"aaa"`)
	zzz := strings.Index(got, `"zzz"`)
	if aaa < 0 || zzz < 0 {
		t.Fatalf("Validate() reported only one of the two typos; both must come back in one pass:\n%s", got)
	}
	if aaa > zzz {
		t.Errorf("problems are not sorted: %q reported after %q:\n%s", "aaa", "zzz", got)
	}
}

// TestValidateNamesTheEmptyPlansCaseDifferently covers the config that has no `plans:`
// section at all. The generic message ends in "Available: <list>", which renders as a
// dangling "Available: " when the list is empty — and the author's real problem there is
// not a typo but a missing section, so the message has to say that instead.
func TestValidateNamesTheEmptyPlansCaseDifferently(t *testing.T) {
	cfg := loadConfigForSchemaTest(t, t.TempDir(), `version: "0.1.44"
interaction:
  up:
    after:
      - step: seed
        run: "true"
        plans: [design]
`)

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() accepted a plan filter in a config that declares no plans")
	}
	got := err.Error()
	if !strings.Contains(got, "declares no plans at all") {
		t.Errorf("error does not name the missing section:\n%s", got)
	}
	if strings.Contains(got, "Available:") {
		t.Errorf("error renders an empty Available list:\n%s", got)
	}
}

// TestWarnIgnoredPlanFilters covers the other half: `plans:` reaches `provision:` and
// `interaction.*.steps` only because those share ProvisionItem with hooks, and neither has
// a routed plan to filter against. The hook entry in the fixture is the load-bearing one —
// the key works there, so a check keyed off the field alone rather than off where it
// appears would tell the author to remove a filter that is doing its job.
func TestWarnIgnoredPlanFilters(t *testing.T) {
	c := &Config{
		Interaction: map[string]*InteractionCommand{
			"seed": {
				Steps: []ProvisionItem{
					{Step: "a", Run: "true", Plans: []string{"design"}},
					{Step: "b", Run: "true"}, // fine: asks for nothing it will not get
				},
			},
			"up": {
				// Honoured — must not warn.
				After: []ProvisionItem{{Step: "penpot", Run: "true", Plans: []string{"design"}}},
			},
		},
	}
	c.Provision.Profiles = map[string][]ProvisionItem{
		"default": {{Step: "bootstrap", Run: "true", Plans: []string{"design"}}},
	}

	warnings := c.warnIgnoredPlanFilters()

	want := []string{
		`interaction.seed.steps[0] "a"`,
		`provision.default[0] "bootstrap"`,
	}
	if len(warnings) != len(want) {
		t.Fatalf("got %d warnings, want %d:\n%s", len(warnings), len(want), strings.Join(warnings, "\n"))
	}
	for i, w := range want {
		if !strings.HasPrefix(warnings[i], w) {
			t.Errorf("warning %d = %q, want prefix %q", i, warnings[i], w)
		}
		if !strings.Contains(warnings[i], IgnoredPlanFilterMessage) {
			t.Errorf("warning %d does not carry the shared wording: %q", i, warnings[i])
		}
	}
}

// TestValidateWarningsReportsIgnoredPlanFilters pins the registration, not the check. The
// test above calls the helper directly, so it stays green when the line wiring it into
// ValidateWarnings is deleted — `dva validate` goes silent and nothing fails.
func TestValidateWarningsReportsIgnoredPlanFilters(t *testing.T) {
	c := &Config{
		Interaction: map[string]*InteractionCommand{
			"seed": {Steps: []ProvisionItem{{Step: "a", Run: "true", Plans: []string{"design"}}}},
		},
	}

	for _, w := range c.ValidateWarnings() {
		if strings.Contains(w, IgnoredPlanFilterMessage) {
			return
		}
	}
	t.Errorf("ValidateWarnings does not surface the ignored-plan-filter check:\n%s",
		strings.Join(c.ValidateWarnings(), "\n"))
}

// TestIgnoredPlanFilterMessageSaysTheStepStillRuns pins the one thing the wording has to
// do. "ignored" alone reads as "this line is inert", and the author moves on — but the
// consequence is the opposite of inert: the step they believed was filtered out runs on
// every invocation. That direction is what separates this message from the `parallel:`
// one, where a dropped key only costs concurrency.
func TestIgnoredPlanFilterMessageSaysTheStepStillRuns(t *testing.T) {
	if !strings.Contains(IgnoredPlanFilterMessage, "runs unfiltered") {
		t.Errorf("IgnoredPlanFilterMessage = %q, which does not say the step still runs", IgnoredPlanFilterMessage)
	}
	if !strings.Contains(IgnoredPlanFilterMessage, "hook") {
		t.Errorf("IgnoredPlanFilterMessage = %q, which does not name the path that honours the key", IgnoredPlanFilterMessage)
	}
}
