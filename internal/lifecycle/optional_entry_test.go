package lifecycle

import (
	"path/filepath"
	"slices"
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

// An entry declaring two directory-bearing runners must resolve to the same directory on
// every run. Runners is a map, so an unsorted walk would pick either one at random.
func TestOptionalEntryDirIsDeterministicAcrossRunners(t *testing.T) {
	entry := &config.LifecycleEntry{
		Name: "side",
		Runners: map[string]any{
			"native":  &config.NativeRunnerConfig{Dir: "a-native"},
			"process": &config.ProcessPluginConfig{Dir: "z-process"},
			"tilt":    &config.TiltPluginConfig{Dir: "m-tilt"},
		},
	}
	for i := range 50 {
		if got := optionalEntryDir(entry); got != "a-native" {
			t.Fatalf("iteration %d: optionalEntryDir = %q, want %q", i, got, "a-native")
		}
	}
}

// Nil flat fields are typed nil pointers once boxed in an any, which is not a nil
// interface — reading Dir off one without a guard panics.
func TestOptionalEntryDirIgnoresNilFlatFields(t *testing.T) {
	if got := optionalEntryDir(&config.LifecycleEntry{Name: "side"}); got != "" {
		t.Errorf("optionalEntryDir on a bare entry = %q, want \"\"", got)
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
