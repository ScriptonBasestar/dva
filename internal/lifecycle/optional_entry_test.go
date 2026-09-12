package lifecycle

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// optionalEntryConfig builds a two-entry plan where only "side" is optional, so a test
// can assert that the optional entry drops out without taking the required one with it.
func optionalEntryConfig(optional bool, runner string, runnerCfg any) *config.Config {
	return &config.Config{
		Stack: map[string]*config.LifecycleEntry{
			"api": {
				Name:          "api",
				DefaultRunner: "native",
				Runners: map[string]any{
					"native": &config.NativeRunnerConfig{Run: "go run ./cmd/api"},
				},
			},
			"side": {
				Name:          "side",
				DefaultRunner: runner,
				Optional:      optional,
				Runners:       map[string]any{runner: runnerCfg},
			},
		},
		Plans: map[string]*config.PlanConfig{
			"local-dev": {Entries: []config.PlanEntry{
				{Name: "api", Runner: "native"},
				{Name: "side", Runner: runner},
			}},
		},
	}
}

func planEntryNames(t *testing.T, cfg *config.Config) []string {
	t.Helper()
	plan, err := ResolvePlan(cfg, "local-dev", nil)
	if err != nil {
		t.Fatalf("ResolvePlan: %v", err)
	}
	names := make([]string, 0, len(plan.Entries))
	for _, e := range plan.Entries {
		names = append(names, e.Name)
	}
	return names
}

// An optional entry whose directory is absent is dropped, and the rest of the plan
// still resolves — that separation is the whole point of the flag.
func TestResolvePlanSkipsOptionalEntryWithMissingDir(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "not-checked-out")
	cfg := optionalEntryConfig(true, "native", &config.NativeRunnerConfig{Run: "go run .", Dir: missing})

	names := planEntryNames(t, cfg)
	if slices.Contains(names, "side") {
		t.Errorf("optional entry with missing dir was kept: %v", names)
	}
	if !slices.Contains(names, "api") {
		t.Errorf("required entry was dropped along with the optional one: %v", names)
	}
}

// Present directory: optional changes nothing.
func TestResolvePlanKeepsOptionalEntryWhenDirExists(t *testing.T) {
	present := t.TempDir()
	cfg := optionalEntryConfig(true, "native", &config.NativeRunnerConfig{Run: "go run .", Dir: present})

	if names := planEntryNames(t, cfg); !slices.Contains(names, "side") {
		t.Errorf("optional entry with existing dir was skipped: %v", names)
	}
}

// Without optional: true a missing directory is not a resolve-time concern at all —
// the entry survives resolution and fails later, where the command actually runs.
func TestResolvePlanKeepsNonOptionalEntryWithMissingDir(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "not-checked-out")
	cfg := optionalEntryConfig(false, "native", &config.NativeRunnerConfig{Run: "go run .", Dir: missing})

	if names := planEntryNames(t, cfg); !slices.Contains(names, "side") {
		t.Errorf("non-optional entry was skipped for a missing dir: %v", names)
	}
}

// runners.process decodes into ProcessPluginConfig, not NativeRunnerConfig. A lookup
// that names only the native type silently never matches, so the optional entry is
// kept no matter what its directory says; this pins the process shape to the check.
func TestResolvePlanSkipsOptionalProcessEntryWithMissingDir(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "not-checked-out")
	cfg := optionalEntryConfig(true, "process", &config.ProcessPluginConfig{Command: "./run.sh", Dir: missing})

	if names := planEntryNames(t, cfg); slices.Contains(names, "side") {
		t.Errorf("optional process entry with missing dir was kept: %v", names)
	}
}

// An optional entry that declares no directory has nothing to test for existence, so
// it is kept rather than skipped on the strength of some unrelated path.
func TestResolvePlanKeepsOptionalEntryWithoutDir(t *testing.T) {
	cfg := optionalEntryConfig(true, "native", &config.NativeRunnerConfig{Run: "go run ."})

	if names := planEntryNames(t, cfg); !slices.Contains(names, "side") {
		t.Errorf("optional entry declaring no dir was skipped: %v", names)
	}
}

// flatOptionalEntryConfig declares the optional entry through a flat typed field
// (process:, tilt:, ...) instead of the runners: map. decodeRunnersMap only populates
// Runners from a literal runners: key, so this shape leaves Runners empty — the
// optional check has to reach the flat fields too or the flag silently does nothing.
func flatOptionalEntryConfig(mutate func(*config.LifecycleEntry)) *config.Config {
	side := &config.LifecycleEntry{Name: "side", Optional: true}
	mutate(side)
	return &config.Config{
		Stack: map[string]*config.LifecycleEntry{
			"api": {
				Name:          "api",
				DefaultRunner: "native",
				Runners: map[string]any{
					"native": &config.NativeRunnerConfig{Run: "go run ./cmd/api"},
				},
			},
			"side": side,
		},
		Plans: map[string]*config.PlanConfig{
			"local-dev": {Entries: []config.PlanEntry{
				{Name: "api", Runner: "native"},
				{Name: "side"},
			}},
		},
	}
}

// Every declaration shape that carries a directory must honour optional: true. Before
// this, only runners.native and runners.process were consulted, so an entry written as
// `process: {dir: ...}` — or using any Tier-2/3 runner — accepted the flag and ignored it.
func TestResolvePlanSkipsOptionalEntryAcrossDeclarationShapes(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "not-checked-out")

	cases := []struct {
		name   string
		mutate func(*config.LifecycleEntry)
	}{
		{"flat process", func(e *config.LifecycleEntry) {
			e.DefaultRunner = "process"
			e.Process = &config.ProcessPluginConfig{Command: "./run", Dir: missing}
		}},
		{"flat tilt", func(e *config.LifecycleEntry) {
			e.DefaultRunner = "tilt"
			e.Tilt = &config.TiltPluginConfig{Dir: missing}
		}},
		{"flat vagrant", func(e *config.LifecycleEntry) {
			e.DefaultRunner = "vagrant"
			e.Vagrant = &config.VagrantPluginConfig{Dir: missing}
		}},
		{"flat kustomize", func(e *config.LifecycleEntry) {
			e.DefaultRunner = "kustomize"
			e.Kustomize = &config.KustomizePluginConfig{Dir: missing}
		}},
		{"flat serverless", func(e *config.LifecycleEntry) {
			e.DefaultRunner = "serverless"
			e.Serverless = &config.ServerlessPluginConfig{Dir: missing}
		}},
		{"runners.tilt", func(e *config.LifecycleEntry) {
			e.DefaultRunner = "tilt"
			e.Runners = map[string]any{"tilt": &config.TiltPluginConfig{Dir: missing}}
		}},
		{"source.path", func(e *config.LifecycleEntry) {
			e.DefaultRunner = "native"
			e.Runners = map[string]any{"native": &config.NativeRunnerConfig{Run: "./run"}}
			e.Source = &config.SourceConfig{Path: missing}
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			names := planEntryNames(t, flatOptionalEntryConfig(tc.mutate))
			if slices.Contains(names, "side") {
				t.Errorf("optional entry with missing dir was kept: %v", names)
			}
			if !slices.Contains(names, "api") {
				t.Errorf("required entry was dropped alongside the optional one: %v", names)
			}
		})
	}
}

// The directory that decides an optional entry's fate belongs to the runner the plan
// actually selected. An entry can declare several, and before TASK-374 the check consulted
// them in a fixed priority order with no reference to the selection — so an entry declaring
// runners.native (with a dir) alongside runners.compose (without one) was judged on
// native's directory even when the plan ran compose, and vanished for a directory that
// nothing in the chosen execution path would ever have opened.
func TestResolvePlanOptionalEntryIsJudgedBySelectedRunner(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "not-checked-out")
	cfg := &config.Config{
		Stack: map[string]*config.LifecycleEntry{
			"side": {
				Name:     "side",
				Optional: true,
				Runners: map[string]any{
					"native":  &config.NativeRunnerConfig{Run: "go run .", Dir: missing},
					"compose": &config.ComposePluginConfig{Files: []string{"compose.side.yml"}},
				},
			},
		},
		Plans: map[string]*config.PlanConfig{
			"local-dev": {Entries: []config.PlanEntry{{Name: "side", Runner: "compose"}}},
		},
	}

	plan, err := ResolvePlan(cfg, "local-dev", nil)
	if err != nil {
		t.Fatalf("ResolvePlan: %v", err)
	}
	if len(plan.Entries) != 1 {
		t.Fatalf("entry was skipped on a directory belonging to a runner the plan did not pick; entries=%d", len(plan.Entries))
	}

	// The same entry, with the plan picking native instead, must still be skipped —
	// otherwise this test would pass just as well against a check that never fires.
	cfg.Plans["local-dev"].Entries[0].Runner = "native"
	plan, err = ResolvePlan(cfg, "local-dev", nil)
	if err != nil {
		t.Fatalf("ResolvePlan(native): %v", err)
	}
	if len(plan.Entries) != 0 {
		t.Fatalf("optional entry kept although the selected runner's dir is missing; entries=%d", len(plan.Entries))
	}
}

// A skipped entry has to be visible to someone who never passed --dry-run: ResolutionTrace
// only prints there, so the skip also goes on Warnings, which prints on every path.
func TestResolvePlanWarnsWhenOptionalEntryIsSkipped(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "not-checked-out")
	cfg := optionalEntryConfig(true, "native", &config.NativeRunnerConfig{Run: "go run .", Dir: missing})

	plan, err := ResolvePlan(cfg, "local-dev", nil)
	if err != nil {
		t.Fatalf("ResolvePlan: %v", err)
	}
	if len(plan.Warnings) != 1 {
		t.Fatalf("Warnings = %v, want exactly one skip warning", plan.Warnings)
	}
	if !strings.Contains(plan.Warnings[0], "side") || !strings.Contains(plan.Warnings[0], missing) {
		t.Errorf("warning %q names neither the entry nor the directory that caused the skip", plan.Warnings[0])
	}
	// warn() traces as well, so --dry-run stays a complete account rather than one that
	// happens to omit the entries that disappeared.
	if !slices.Contains(plan.ResolutionTrace, plan.Warnings[0]) {
		t.Errorf("skip warning is absent from ResolutionTrace: %v", plan.ResolutionTrace)
	}
}

// An entry that is kept produces no warning — a channel that prints on every path must not
// become noise on the ordinary case.
func TestResolvePlanDoesNotWarnWhenNothingIsSkipped(t *testing.T) {
	dir := t.TempDir()
	cfg := optionalEntryConfig(true, "native", &config.NativeRunnerConfig{Run: "go run .", Dir: dir})

	plan, err := ResolvePlan(cfg, "local-dev", nil)
	if err != nil {
		t.Fatalf("ResolvePlan: %v", err)
	}
	if len(plan.Warnings) != 0 {
		t.Errorf("Warnings = %v, want none", plan.Warnings)
	}
}

// optionalSkipDir answers "" both for a runner shape that declares no directory of its own
// and for a typed nil pointer boxed in an any, which is not a nil interface and panics if
// a case reads Dir off it without a guard.
func TestOptionalSkipDirFallsBackToSourcePath(t *testing.T) {
	withSource := &config.LifecycleEntry{Name: "side", Source: &config.SourceConfig{Path: "vendor/side"}}

	// compose declares no dir, so source.path decides.
	if got := optionalSkipDir(withSource, &config.ComposePluginConfig{Files: []string{"compose.yml"}}); got != "vendor/side" {
		t.Errorf("optionalSkipDir with a dirless runner = %q, want %q", got, "vendor/side")
	}
	// the selected runner's own dir wins over source.path.
	if got := optionalSkipDir(withSource, &config.NativeRunnerConfig{Dir: "app"}); got != "app" {
		t.Errorf("optionalSkipDir = %q, want the selected runner's dir %q", got, "app")
	}
	// a typed nil must not panic, and must not be read as a directory.
	var nilNative *config.NativeRunnerConfig
	if got := optionalSkipDir(&config.LifecycleEntry{Name: "side"}, nilNative); got != "" {
		t.Errorf("optionalSkipDir on a typed nil = %q, want \"\"", got)
	}
	// nothing named a directory at all: "" means "nothing to check", not "the config dir".
	if got := optionalSkipDir(&config.LifecycleEntry{Name: "side"}, &config.ComposePluginConfig{}); got != "" {
		t.Errorf("optionalSkipDir with nothing declared = %q, want \"\"", got)
	}
}

// Every other optional test builds *config.Config as a struct literal, which is exactly
// how the flat-declaration no-op stayed invisible: a hand-built Runners map is always
// populated, while the YAML decoder only fills it from a literal runners: key. This test
// goes through config.Load so the decoder decides what the entry actually looks like.
func TestResolvePlanSkipsOptionalEntryLoadedFromYAML(t *testing.T) {
	cases := []struct {
		name string
		side string
	}{
		{"runners map shape", `
  side:
    optional: true
    default_runner: process
    runners:
      process:
        command: ./side
        dir: not-checked-out
`},
		{"flat typed shape", `
  side:
    optional: true
    default_runner: process
    process:
      command: ./side
      dir: not-checked-out
`},
		// source: only accepts the compose runner in v1, and compose declares no dir of
		// its own — so this shape is precisely the one that reaches the Source.Path
		// fallback, which the old lookup could never get to.
		{"source path shape", `
  side:
    optional: true
    default_runner: compose
    source:
      path: not-checked-out
    runners:
      compose:
        files: [compose.side.yml]
`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeImportedPlanConfig(t, dir, `
version: "0.1.0"
stack:
  api:
    default_runner: process
    runners:
      process:
        command: ./api
`+tc.side+`
plans:
  local-dev:
    entries:
      - name: api
      - name: side
`)
			cfg, err := config.Load(dir)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			names := planEntryNames(t, cfg)
			if slices.Contains(names, "side") {
				t.Errorf("optional entry with missing dir was kept: %v", names)
			}
			if !slices.Contains(names, "api") {
				t.Errorf("required entry was dropped alongside the optional one: %v", names)
			}
		})
	}
}

// TASK-374 review finding #4: moving the optional check past runner resolution changed what
// optional: absorbs. A missing directory is still tolerated; a declaration that does not
// resolve is not. Nothing pinned that, so a refactor could restore the silent drop without
// any test objecting — and a silently dropped typo is unfindable by the person who made it.
func TestOptionalEntryWithUnresolvableRunnerFailsInsteadOfBeingSkipped(t *testing.T) {
	cfg := &config.Config{
		Stack: map[string]*config.LifecycleEntry{
			"side": {
				Name:     "side",
				Optional: true,
				Runners: map[string]any{
					"native": &config.NativeRunnerConfig{Run: "go run .", Dir: filepath.Join(t.TempDir(), "not-checked-out")},
				},
			},
		},
		Plans: map[string]*config.PlanConfig{
			// "nativve" is the typo this test exists for.
			"local-dev": {Entries: []config.PlanEntry{{Name: "side", Runner: "nativve"}}},
		},
	}

	plan, err := ResolvePlan(cfg, "local-dev", nil)
	if err == nil {
		t.Fatalf("an optional entry naming an undeclared runner must fail the plan, not vanish from it; entries=%d", len(plan.Entries))
	}
	if !strings.Contains(err.Error(), "side") {
		t.Errorf("the error must name the offending entry so the typo is findable, got: %v", err)
	}
}

// TASK-374 review finding #5: EntryDir absorbed resolveDir's TrimSpace, which is a behaviour
// change for its two pre-existing callers (process.go:92 native working dir, build.go:181
// build working dir), not only for the resolver. Neither was covered with whitespace input.
func TestEntryDirTrimsWhitespace(t *testing.T) {
	const configDir = "/cfg"
	for _, tc := range []struct {
		name string
		dir  string
		want string
	}{
		{"padded relative dir loses the padding", "  app  ", "/cfg/app"},
		{"blank dir is the config dir, not a blank-named subdirectory", "   ", configDir},
		{"empty dir is the config dir", "", configDir},
		{"padded absolute dir stays absolute", "  /srv/app  ", "/srv/app"},
		{"an interior space is part of the name", "my app", "/cfg/my app"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := EntryDir(configDir, tc.dir); got != tc.want {
				t.Errorf("EntryDir(%q, %q) = %q, want %q", configDir, tc.dir, got, tc.want)
			}
		})
	}
}
