package config

import (
	"strings"
	"testing"
)

func TestWarnDuplicateStackOrder(t *testing.T) {
	// Same order → warning
	c := &Config{
		Stack: map[string]*LifecycleEntry{
			"compose":      {Order: 10},
			"compose-full": {Order: 10},
		},
	}
	warnings := c.warnDuplicateStackOrder()
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
	if !strings.Contains(warnings[0], "order value 10") {
		t.Errorf("unexpected warning: %s", warnings[0])
	}

	// Different order → no warning
	c = &Config{
		Stack: map[string]*LifecycleEntry{
			"compose": {Order: 10},
			"k8s":     {Order: 20},
		},
	}
	warnings = c.warnDuplicateStackOrder()
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d", len(warnings))
	}

	// Single entry → no warning
	c = &Config{
		Stack: map[string]*LifecycleEntry{
			"compose": {Order: 10},
		},
	}
	warnings = c.warnDuplicateStackOrder()
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d", len(warnings))
	}

	// Order 0 (default) → distinct message
	c = &Config{
		Stack: map[string]*LifecycleEntry{
			"a": {Order: 0},
			"b": {Order: 0},
		},
	}
	warnings = c.warnDuplicateStackOrder()
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning for order 0, got %d", len(warnings))
	}
	if !strings.Contains(warnings[0], "default") {
		t.Errorf("expected 'default' hint in warning, got: %s", warnings[0])
	}

	// A plan that names the tied entries settles them (TASK-084 half 2).
	// docs/40-declarative-stack-and-plans.md puts order in the plan layer, so warning here would
	// tell three of dva's own examples they had not chosen a sequence their plan spells out.
	c.Plans = map[string]*PlanConfig{"local": {Entries: []PlanEntry{{Name: "a", Order: 10}, {Name: "b", Order: 20}}}}
	if warnings = c.warnDuplicateStackOrder(); len(warnings) != 0 {
		t.Errorf("a plan naming every tied entry must silence the warning, got %v", warnings)
	}

	// A plan that names only some does not settle the rest — the failure mode of the original
	// `len(c.Plans) > 0` rule, which let one planned entry hide every unplanned one. Three tied at
	// the default order, one planned: the other two still have no declared position anywhere.
	c = &Config{
		Stack: map[string]*LifecycleEntry{"a": {}, "b": {}, "c": {}},
		Plans: map[string]*PlanConfig{"local": {Entries: []PlanEntry{{Name: "a", Order: 10}}}},
	}
	warnings = c.warnDuplicateStackOrder()
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning for the entries no plan names, got %v", warnings)
	}
	if !strings.Contains(warnings[0], "entries b, c") {
		t.Errorf("warning must name only the entries no plan positions: %s", warnings[0])
	}
	if !strings.Contains(warnings[0], "dva up <plan>") {
		t.Errorf("with plans declared the warning must say which command plan order governs: %s", warnings[0])
	}

	// Same fixture with the plan removed: `a` is no longer covered, so it rejoins the list. That
	// the two assertions disagree about `a` is what proves the plan set does the filtering rather
	// than the message being fixed text. The plan clause must go too, or it advertises a section
	// the config has none of.
	c.Plans = nil
	warnings = c.warnDuplicateStackOrder()
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning without plans, got %v", warnings)
	}
	if !strings.Contains(warnings[0], "entries a, b, c") {
		t.Errorf("without plans every tied entry is unpositioned: %s", warnings[0])
	}
	if strings.Contains(warnings[0], "dva up <plan>") {
		t.Errorf("the plan clause reached a config declaring no plans: %s", warnings[0])
	}
}

// TestWarnDuplicateStackOrderModeIsolation covers entries that share an order value but
// are never live together. Order decides who starts first inside one invocation, so a
// group no invocation can hold twice has no sequence to control — telling the user to
// set explicit order values there is advice about an event that cannot happen.
func TestWarnDuplicateStackOrderModeIsolation(t *testing.T) {
	// The shape found in the wild: four compose entries left at the default order,
	// each selected by exactly one mode.
	c := &Config{
		Stack: map[string]*LifecycleEntry{
			"compose":               {},
			"compose-minimal":       {},
			"compose-observability": {},
			"compose-tracing":       {},
		},
		Modes: map[string]ModeConfig{
			"full":          {Stack: []string{"compose"}},
			"minimal":       {Stack: []string{"compose-minimal"}},
			"observability": {Stack: []string{"compose-observability"}},
			"tracing":       {Stack: []string{"compose-tracing"}},
		},
	}
	if warnings := c.warnDuplicateStackOrder(); len(warnings) != 0 {
		t.Fatalf("expected silence for mode-isolated entries, got %v", warnings)
	}

	// One mode pulling in two of them puts both in the same invocation.
	c.Modes["full"] = ModeConfig{Stack: []string{"compose", "compose-tracing"}}
	if warnings := c.warnDuplicateStackOrder(); len(warnings) != 1 {
		t.Fatalf("expected a warning when a mode holds two entries at the same order, got %v", warnings)
	}

	// A mode with no stack: filter selects every entry, so nothing is isolated.
	c.Modes["full"] = ModeConfig{Stack: []string{"compose"}}
	c.Modes["everything"] = ModeConfig{ComposeProfiles: []string{"all"}}
	if warnings := c.warnDuplicateStackOrder(); len(warnings) != 1 {
		t.Fatalf("expected a warning when a mode selects every entry, got %v", warnings)
	}

	// Suppression is per order group, not global: isolating one group must not silence
	// another group that genuinely races.
	c = &Config{
		Stack: map[string]*LifecycleEntry{
			"isolated-a": {Order: 0},
			"isolated-b": {Order: 0},
			"shared-c":   {Order: 10},
			"shared-d":   {Order: 10},
		},
		Modes: map[string]ModeConfig{
			"a": {Stack: []string{"isolated-a", "shared-c", "shared-d"}},
			"b": {Stack: []string{"isolated-b", "shared-c", "shared-d"}},
		},
	}
	warnings := c.warnDuplicateStackOrder()
	if len(warnings) != 1 {
		t.Fatalf("expected exactly the non-isolated group to warn, got %v", warnings)
	}
	if !strings.Contains(warnings[0], "shared-c, shared-d") || !strings.Contains(warnings[0], "order value 10") {
		t.Errorf("warning should name only the racing group: %s", warnings[0])
	}
}

func TestWarnDefaultModeHeavyInfra(t *testing.T) {
	svcList := func(svcs ...string) *[]string { return &svcs }

	// Default mode with kafka + monitoring → warning
	c := &Config{
		DefaultMode: "infra-only",
		Modes: map[string]ModeConfig{
			"infra-only": {
				ComposeServices: svcList("postgres", "redis", "kafka", "prometheus", "grafana", "jaeger"),
			},
		},
		Stack: map[string]*LifecycleEntry{
			"compose": {
				Compose: &ComposePluginConfig{
					Services: map[string]ServiceTagConfig{
						"postgres":   {Tags: []string{"infra", "data"}},
						"redis":      {Tags: []string{"infra", "data"}},
						"kafka":      {Tags: []string{"infra", "kafka"}},
						"prometheus": {Tags: []string{"infra", "monitoring"}},
						"grafana":    {Tags: []string{"infra", "monitoring"}},
						"jaeger":     {Tags: []string{"infra", "monitoring"}},
					},
				},
			},
		},
	}
	warnings := c.warnDefaultModeHeavyInfra()
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d: %v", len(warnings), warnings)
	}
	if !strings.Contains(warnings[0], "non-core infrastructure") {
		t.Errorf("unexpected warning: %s", warnings[0])
	}
	// Should list the heavy services
	for _, svc := range []string{"kafka", "prometheus", "grafana", "jaeger"} {
		if !strings.Contains(warnings[0], svc) {
			t.Errorf("warning should mention %q: %s", svc, warnings[0])
		}
	}

	// Default mode with only core services → no warning
	c = &Config{
		DefaultMode: "infra",
		Modes: map[string]ModeConfig{
			"infra": {
				ComposeServices: svcList("postgres", "redis"),
			},
		},
		Stack: map[string]*LifecycleEntry{
			"compose": {
				Compose: &ComposePluginConfig{
					Services: map[string]ServiceTagConfig{
						"postgres": {Tags: []string{"infra", "data"}},
						"redis":    {Tags: []string{"infra", "data"}},
					},
				},
			},
		},
	}
	warnings = c.warnDefaultModeHeavyInfra()
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings for core-only, got %d: %v", len(warnings), warnings)
	}

	// Name-based heuristic: no tags but heavy service names → warning
	c = &Config{
		DefaultMode: "infra",
		Modes: map[string]ModeConfig{
			"infra": {
				ComposeServices: svcList("postgres", "redis", "kafka", "minio"),
			},
		},
		Stack: map[string]*LifecycleEntry{
			"compose": {Compose: &ComposePluginConfig{}},
		},
	}
	warnings = c.warnDefaultModeHeavyInfra()
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning for name heuristic, got %d", len(warnings))
	}
	if !strings.Contains(warnings[0], "kafka") || !strings.Contains(warnings[0], "minio") {
		t.Errorf("warning should mention kafka and minio: %s", warnings[0])
	}

	// No default mode → no warning
	c = &Config{
		Modes: map[string]ModeConfig{
			"infra": {ComposeServices: svcList("postgres", "kafka")},
		},
	}
	warnings = c.warnDefaultModeHeavyInfra()
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings when no default_mode, got %d", len(warnings))
	}

	// compose_services nil (all services) → no warning (can't enumerate)
	c = &Config{
		DefaultMode: "infra",
		Modes: map[string]ModeConfig{
			"infra": {ComposeServices: nil},
		},
	}
	warnings = c.warnDefaultModeHeavyInfra()
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings when compose_services is nil, got %d", len(warnings))
	}
}
