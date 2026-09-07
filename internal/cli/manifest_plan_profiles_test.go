package cli

import (
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// TestManifestPlanEntryCarriesProfiles keeps `dva show`'s machine-readable manifest
// honest about what a plan will run: a profile selection changes the containers that
// come up, so a manifest that omits it describes a different plan than the one on disk.
func TestManifestPlanEntryCarriesProfiles(t *testing.T) {
	cfg := &config.Config{
		Stack: map[string]*config.LifecycleEntry{
			"core-compose": {
				Name:          "core-compose",
				DefaultRunner: "compose",
				Runners: map[string]any{
					"compose": &config.ComposePluginConfig{Files: []string{"compose.yaml"}},
				},
			},
		},
		Plans: map[string]*config.PlanConfig{
			"docker-dev": {Entries: []config.PlanEntry{{
				Name: "core-compose", Runner: "compose", Profiles: []string{"rust", "monitoring"},
			}}},
		},
	}

	plans := buildManifestPlans(cfg)
	plan, ok := plans["docker-dev"]
	if !ok {
		t.Fatalf("plan missing from manifest: %v", plans)
	}
	if plan.ResolutionError != "" {
		t.Fatalf("plan failed to resolve: %s", plan.ResolutionError)
	}
	if len(plan.Entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(plan.Entries))
	}
	if got, want := strings.Join(plan.Entries[0].Profiles, ","), "rust,monitoring"; got != want {
		t.Fatalf("manifest entry profiles = %q, want %q", got, want)
	}
}
