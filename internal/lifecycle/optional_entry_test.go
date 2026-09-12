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
