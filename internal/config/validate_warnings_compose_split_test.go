package config

import (
	"strings"
	"testing"
)

func TestWarnMultiStackComposeSplit(t *testing.T) {
	// Multiple compose entries → warning
	c := &Config{
		Stack: map[string]*LifecycleEntry{
			"compose":      {Compose: &ComposePluginConfig{}},
			"compose-full": {Compose: &ComposePluginConfig{}},
		},
	}
	warnings := c.warnMultiStackComposeSplit()
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
	if !strings.Contains(warnings[0], "compose entries [compose, compose-full]") {
		t.Errorf("unexpected warning: %s", warnings[0])
	}

	// Single compose entry → no warning
	c = &Config{
		Stack: map[string]*LifecycleEntry{
			"compose": {Compose: &ComposePluginConfig{}},
		},
	}
	warnings = c.warnMultiStackComposeSplit()
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d", len(warnings))
	}

	// Compose + kubectl → no warning (different backends)
	c = &Config{
		Stack: map[string]*LifecycleEntry{
			"compose": {Compose: &ComposePluginConfig{}},
			"k8s":     {Kubectl: &KubectlPluginConfig{}},
		},
	}
	warnings = c.warnMultiStackComposeSplit()
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings for different backends, got %d", len(warnings))
	}
}

// TestWarnMultiStackComposeSplitModeIsolation covers the shape that replaced the
// removed modes.<name>.compose: one compose entry per mode, picked by
// modes.<name>.stack. The warning used to fire on it and tell users to consolidate,
// which modes.compose_services cannot express when the entries load different files.
func TestWarnMultiStackComposeSplitModeIsolation(t *testing.T) {
	composeStack := func() map[string]*LifecycleEntry {
		return map[string]*LifecycleEntry{
			"compose-base": {Compose: &ComposePluginConfig{Files: []string{"docker-compose.yml"}}},
			"compose-obs":  {Compose: &ComposePluginConfig{Files: []string{"docker-compose.yml", "docker-compose.obs.yml"}}},
		}
	}

	tests := []struct {
		name  string
		modes map[string]ModeConfig
		warn  bool
	}{
		{
			name: "each entry owned by one mode",
			modes: map[string]ModeConfig{
				"infra":         {Stack: []string{"compose-base"}},
				"observability": {Stack: []string{"compose-obs"}},
			},
			warn: false,
		},
		{
			name: "one mode pulls in both entries",
			modes: map[string]ModeConfig{
				"infra": {Stack: []string{"compose-base"}},
				"full":  {Stack: []string{"compose-base", "compose-obs"}},
			},
			warn: true,
		},
		{
			name: "a mode without stack: selects every entry",
			modes: map[string]ModeConfig{
				"infra": {Stack: []string{"compose-base"}},
				"full":  {ComposeProfiles: []string{"all"}},
			},
			warn: true,
		},
		{
			name: "an entry no mode claims always runs",
			modes: map[string]ModeConfig{
				"infra": {Stack: []string{"compose-base"}},
			},
			warn: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{Stack: composeStack(), Modes: tt.modes}
			warnings := c.warnMultiStackComposeSplit()
			if tt.warn && len(warnings) != 1 {
				t.Fatalf("expected a warning, got %v", warnings)
			}
			if !tt.warn && len(warnings) != 0 {
				t.Fatalf("expected silence, got %v", warnings)
			}
		})
	}
}

func TestWarnMultiStackComposeSplitPlanIsolation(t *testing.T) {
	composeStack := func() map[string]*LifecycleEntry {
		return map[string]*LifecycleEntry{
			"compose":      {Compose: &ComposePluginConfig{Files: []string{"compose.yaml"}}},
			"compose-live": {Compose: &ComposePluginConfig{Files: []string{"compose.dns-bridge.yaml"}}},
		}
	}

	tests := []struct {
		name        string
		plans       map[string]*PlanConfig
		defaultPlan string
		modes       map[string]ModeConfig
		warn        bool
	}{
		{
			name: "isolated plans with explicit default",
			plans: map[string]*PlanConfig{
				"local-infra": {Entries: []PlanEntry{{Name: "compose"}}},
				"full-live":   {Entries: []PlanEntry{{Name: "compose-live"}}},
			},
			defaultPlan: "local-infra",
			warn:        false,
		},
		{
			name: "isolated multiple plans without default have unsafe stack path",
			plans: map[string]*PlanConfig{
				"local-infra": {Entries: []PlanEntry{{Name: "compose"}}},
				"full-live":   {Entries: []PlanEntry{{Name: "compose-live"}}},
			},
			defaultPlan: "",
			warn:        true,
		},
		{
			name: "one plan selects both entries",
			plans: map[string]*PlanConfig{
				"local-infra": {Entries: []PlanEntry{{Name: "compose"}, {Name: "compose-live"}}},
			},
			warn: true,
		},
		{
			name: "unplanned compose entry has unsafe stack path",
			plans: map[string]*PlanConfig{
				"local-infra": {Entries: []PlanEntry{{Name: "compose"}}},
			},
			warn: true,
		},
		{
			name: "plans take precedence over legacy modes",
			plans: map[string]*PlanConfig{
				"local-infra": {Entries: []PlanEntry{{Name: "compose"}, {Name: "compose-live"}}},
			},
			defaultPlan: "local-infra",
			modes: map[string]ModeConfig{
				"infra": {Stack: []string{"compose"}},
				"live":  {Stack: []string{"compose-live"}},
			},
			warn: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				Stack:           composeStack(),
				Plans:           tt.plans,
				DefaultPlanName: tt.defaultPlan,
				Modes:           tt.modes,
			}
			warnings := c.warnMultiStackComposeSplit()
			if tt.warn && len(warnings) != 1 {
				t.Fatalf("expected a warning, got %v", warnings)
			}
			if !tt.warn && len(warnings) != 0 {
				t.Fatalf("expected silence, got %v", warnings)
			}
		})
	}
}

func TestWarnMultiStackComposeSplitProjectNameIsolation(t *testing.T) {
	newConfig := func(firstProject, secondProject string) *Config {
		return &Config{
			Stack: map[string]*LifecycleEntry{
				"app": {
					Compose: &ComposePluginConfig{Files: []string{"compose.yaml"}, ProjectName: firstProject},
				},
				"e2e": {
					Compose: &ComposePluginConfig{Files: []string{"compose/e2e.yaml"}, ProjectName: secondProject},
				},
			},
			Plans: map[string]*PlanConfig{
				"full": {Entries: []PlanEntry{{Name: "app"}, {Name: "e2e"}}},
			},
			DefaultPlanName: "full",
		}
	}

	t.Run("different explicit project names are independent", func(t *testing.T) {
		if warnings := newConfig("app", "isolated-e2e").warnMultiStackComposeSplit(); len(warnings) != 0 {
			t.Fatalf("expected no warning for independent compose projects, got %v", warnings)
		}
	})

	t.Run("same project name remains an overlay warning", func(t *testing.T) {
		warnings := newConfig("app", "app").warnMultiStackComposeSplit()
		if len(warnings) != 1 {
			t.Fatalf("expected one warning for same project name, got %v", warnings)
		}
		if !strings.Contains(warnings[0], "merge them into one entry") {
			t.Fatalf("same-project warning must suggest merging overlays, got %q", warnings[0])
		}
	})

	t.Run("empty project names remain conservative", func(t *testing.T) {
		if warnings := newConfig("", "").warnMultiStackComposeSplit(); len(warnings) != 1 {
			t.Fatalf("expected warning for unknown compose project names, got %v", warnings)
		}
	})

	t.Run("empty name can match an explicit project", func(t *testing.T) {
		if warnings := newConfig("", "app").warnMultiStackComposeSplit(); len(warnings) != 1 {
			t.Fatalf("expected warning for empty and explicit project names, got %v", warnings)
		}
	})

	t.Run("interpolated name can match a literal project", func(t *testing.T) {
		c := newConfig("${PROJECT_NAME}", "app")
		c.Vars = map[string]string{"PROJECT_NAME": "other"}
		if warnings := c.warnMultiStackComposeSplit(); len(warnings) != 1 {
			t.Fatalf("expected warning for interpolated and literal project names, got %v", warnings)
		}
	})

	t.Run("interpolated names resolving equal remain unknown", func(t *testing.T) {
		c := newConfig("${PROJECT_NAME}", "$PROJECT_NAME")
		c.Vars = map[string]string{"PROJECT_NAME": "app"}
		if warnings := c.warnMultiStackComposeSplit(); len(warnings) != 1 {
			t.Fatalf("expected warning for interpolated project names, got %v", warnings)
		}
	})
}

// TestWarnMultiStackComposeSplitServiceSubsetting covers the shape TASK-288 found: two
// compose entries sharing one project_name/file, each restricted by a plan entry's
// services: to a disjoint, non-empty subset (examples/service-orchestration.yml's
// infra-compose/frontend pair). Nothing overlays here — each entry is its own compose
// invocation starting different services from the same file — so the overlay-patch
// warning must not fire. Every deviation from that exact shape (empty services:,
// overlapping services:, or a third participant) falls back to the original warning.
func TestWarnMultiStackComposeSplitServiceSubsetting(t *testing.T) {
	stackWithNative := func() map[string]*LifecycleEntry {
		return map[string]*LifecycleEntry{
			"infra-compose": {DefaultRunner: "compose", Compose: &ComposePluginConfig{Files: []string{"docker-compose.yml"}, ProjectName: "orchestrator"}},
			"api":           {DefaultRunner: "native", Compose: &ComposePluginConfig{Files: []string{"docker-compose.yml"}, ProjectName: "orchestrator"}},
			"frontend":      {DefaultRunner: "compose", Compose: &ComposePluginConfig{Files: []string{"docker-compose.yml"}, ProjectName: "orchestrator"}},
		}
	}

	tests := []struct {
		name    string
		entries []PlanEntry
		warn    bool
	}{
		{
			name: "disjoint non-empty services across compose entries is silent",
			entries: []PlanEntry{
				{Name: "infra-compose", Services: []string{"postgres", "kafka"}},
				{Name: "api"},
				{Name: "frontend", Services: []string{"frontend"}},
			},
			warn: false,
		},
		{
			name: "a native runner override on a compose-configured entry excludes it too",
			entries: []PlanEntry{
				{Name: "infra-compose", Services: []string{"postgres"}},
				{Name: "api", Runner: "native"},
				{Name: "frontend", Runner: "native", Services: []string{"frontend"}},
			},
			warn: false,
		},
		{
			name: "empty services on either entry falls back to the overlay warning",
			entries: []PlanEntry{
				{Name: "infra-compose", Services: []string{"postgres"}},
				{Name: "api"},
				{Name: "frontend"},
			},
			warn: true,
		},
		{
			name: "overlapping services falls back to the overlay warning",
			entries: []PlanEntry{
				{Name: "infra-compose", Services: []string{"postgres", "kafka"}},
				{Name: "api"},
				{Name: "frontend", Services: []string{"kafka"}},
			},
			warn: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				Stack: stackWithNative(),
				Plans: map[string]*PlanConfig{
					"local-full": {Entries: tt.entries},
				},
				DefaultPlanName: "local-full",
			}
			warnings := c.warnMultiStackComposeSplit()
			if tt.warn && len(warnings) != 1 {
				t.Fatalf("expected a warning, got %v", warnings)
			}
			if !tt.warn && len(warnings) != 0 {
				t.Fatalf("expected silence, got %v", warnings)
			}
		})
	}
}

func TestWarnMultiStackComposeSplitMissingDefaultPlanSuggestsSafeDefault(t *testing.T) {
	c := &Config{
		Stack: map[string]*LifecycleEntry{
			"compose":      {Compose: &ComposePluginConfig{ProjectName: "app"}},
			"compose-live": {Compose: &ComposePluginConfig{ProjectName: "app"}},
		},
		Plans: map[string]*PlanConfig{
			"infra": {Entries: []PlanEntry{{Name: "compose"}}},
			"live":  {Entries: []PlanEntry{{Name: "compose-live"}}},
		},
	}
	warnings := c.warnMultiStackComposeSplit()
	if len(warnings) != 1 {
		t.Fatalf("expected one warning without default plan, got %v", warnings)
	}
	if !strings.Contains(warnings[0], "set default_plan") {
		t.Fatalf("missing-default warning must suggest default_plan, got %q", warnings[0])
	}
}
