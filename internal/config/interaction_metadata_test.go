package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestIntentionalShadowAcknowledgesOnlyLocalSubprojectCollision(t *testing.T) {
	yes, no := true, false
	for _, tc := range []struct {
		name     string
		flag     *bool
		owner    *Config
		warnings int
	}{
		{name: "unacknowledged", warnings: 1},
		{name: "acknowledged", flag: &yes},
		{name: "explicit false", flag: &no, warnings: 1},
		{name: "child declaration cannot acknowledge parent namespace", flag: &yes, owner: &Config{}, warnings: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{
				Subprojects: map[string]SubprojectConfig{"engine": {Path: "engine"}},
				Interaction: map[string]*InteractionCommand{"engine:test": {Command: "true", IntentionalShadow: tc.flag, owner: tc.owner}},
			}
			if got := cfg.warnLiteralKeyShadowsSubproject(); len(got) != tc.warnings {
				t.Fatalf("shadow warnings = %v, want %d", got, tc.warnings)
			}
		})
	}
}

func TestIntentionalShadowStaleAndNestedDeclarationsWarn(t *testing.T) {
	yes := true
	cfg := &Config{Interaction: map[string]*InteractionCommand{
		"stale:test": {Command: "true", IntentionalShadow: &yes},
		"group": {Subcommands: map[string]*InteractionCommand{
			"engine:test": {Command: "true", IntentionalShadow: &yes},
		}},
	}, Subprojects: map[string]SubprojectConfig{"engine": {Path: "engine"}}}
	warnings := cfg.warnUnusedIntentionalShadows()
	if len(warnings) != 2 || !strings.Contains(strings.Join(warnings, "\n"), "group.subcommands.engine:test") {
		t.Fatalf("stale or nested intention was silently ignored: %v", warnings)
	}
}

func TestMergeInteractionMetadataPreservesExplicitFalseAndEmptyAliases(t *testing.T) {
	var base, overlay InteractionCommand
	if err := yaml.Unmarshal([]byte("command: true\nintentional_shadow: true\nsuggestion_aliases:\n  makefile: [legacy-build]\n"), &base); err != nil {
		t.Fatal(err)
	}
	if err := yaml.Unmarshal([]byte("intentional_shadow: false\nsuggestion_aliases:\n  makefile: []\n"), &overlay); err != nil {
		t.Fatal(err)
	}
	merged, err := mergeInteractionCommand(&base, &overlay)
	if err != nil {
		t.Fatal(err)
	}
	if merged.IntentionalShadow == nil || *merged.IntentionalShadow || merged.SuggestionAliases == nil || len(merged.SuggestionAliases.Makefile) != 0 {
		t.Fatalf("explicit false/empty metadata did not replace inherited values: %+v", merged)
	}
	if merged.Command != "true" {
		t.Fatalf("metadata changed executable command: %q", merged.Command)
	}
}

func TestLoadRejectsDuplicateMakefileAliasOwnership(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(`interaction:
  build-new:
    runner: local
    command: "true"
    suggestion_aliases: {makefile: [legacy-build]}
  test-new:
    runner: local
    command: "true"
    suggestion_aliases: {makefile: [legacy-build]}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), "legacy-build") || !strings.Contains(err.Error(), "exactly one owning interaction") {
		t.Fatalf("duplicate target ownership error = %v", err)
	}
}

func TestMakefileAliasOwnershipSeparatesChildNamespace(t *testing.T) {
	child := &Config{}
	cfg := &Config{Interaction: map[string]*InteractionCommand{
		"local-build": {Command: "true", SuggestionAliases: &SuggestionAliasConfig{Makefile: []string{"legacy-build"}}},
		"child/build": {Command: "true", owner: child, SuggestionAliases: &SuggestionAliasConfig{Makefile: []string{"legacy-build"}}},
	}}
	if err := cfg.validateSuggestionAliases(); err != nil {
		t.Fatalf("child metadata conflicted with parent's target namespace: %v", err)
	}
}
