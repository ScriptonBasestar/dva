package lifecycle

import (
	"slices"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func TestResolvePlanAlias(t *testing.T) {
	cfg := &config.Config{
		Plans: map[string]*config.PlanConfig{
			"base": {
				Description: "Concrete plan",
				Environment: "dev",
				Site:        "local",
			},
			"alias1": {
				Alias: "base",
			},
			"alias2": {
				Alias: "alias1",
			},
			"self": {
				Alias: "self",
			},
			"cycle-a": {
				Alias: "cycle-b",
			},
			"cycle-b": {
				Alias: "cycle-a",
			},
			"missing": {
				Alias: "does-not-exist",
			},
		},
	}

	t.Run("concrete plan returns self", func(t *testing.T) {
		name, plan, err := ResolvePlanAlias(cfg, "base")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if name != "base" || plan.Description != "Concrete plan" {
			t.Errorf("expected base, got %s: %+v", name, plan)
		}
	})

	t.Run("single alias resolves to target", func(t *testing.T) {
		name, plan, err := ResolvePlanAlias(cfg, "alias1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if name != "base" || plan.Description != "Concrete plan" {
			t.Errorf("expected base, got %s: %+v", name, plan)
		}
	})

	t.Run("chained alias resolves to ultimate target", func(t *testing.T) {
		name, plan, err := ResolvePlanAlias(cfg, "alias2")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if name != "base" || plan.Description != "Concrete plan" {
			t.Errorf("expected base, got %s: %+v", name, plan)
		}
	})

	t.Run("self-reference fails", func(t *testing.T) {
		_, _, err := ResolvePlanAlias(cfg, "self")
		if err == nil {
			t.Fatal("expected error for self-reference, got nil")
		}
	})

	t.Run("cycle fails", func(t *testing.T) {
		_, _, err := ResolvePlanAlias(cfg, "cycle-a")
		if err == nil {
			t.Fatal("expected error for cycle, got nil")
		}
	})

	t.Run("missing target fails", func(t *testing.T) {
		_, _, err := ResolvePlanAlias(cfg, "missing")
		if err == nil {
			t.Fatal("expected error for missing target, got nil")
		}
	})
}

func TestMergePlanExtends(t *testing.T) {
	cfg := &config.Config{
		Plans: map[string]*config.PlanConfig{
			"parent": {
				Description:  "Parent plan",
				Environment:  "dev",
				Site:         "local",
				EndpointTags: []string{"web"},
				Vars: map[string]string{
					"VAR_A": "parent-a",
					"VAR_B": "parent-b",
				},
				Entries: []config.PlanEntry{
					{
						Name:     "compose-app",
						Runner:   "compose",
						Order:    10,
						Services: []string{"postgres", "redis"},
					},
					{
						Name:   "api",
						Runner: "native",
						Order:  20,
					},
				},
			},
			"child": {
				Extends:      "parent",
				Description:  "Child override",
				Environment:  "stg",
				EndpointTags: []string{"api"},
				Vars: map[string]string{
					"VAR_B": "child-b",
					"VAR_C": "child-c",
				},
				Entries: []config.PlanEntry{
					{
						Name:     "compose-app",
						Runner:   "compose",
						Order:    15,
						Services: []string{"postgres", "redis", "minio"},
					},
					{
						Name:   "worker",
						Runner: "native",
						Order:  30,
					},
				},
			},
		},
	}

	merged, err := MergePlanExtends(cfg, "child", cfg.Plans["child"])
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 1. Scalar overrides
	if merged.Description != "Child override" {
		t.Errorf("description: expected 'Child override', got %q", merged.Description)
	}
	if merged.Environment != "stg" {
		t.Errorf("environment: expected 'stg', got %q", merged.Environment)
	}
	if merged.Site != "local" { // parent's site inherited
		t.Errorf("site: expected 'local', got %q", merged.Site)
	}
	if !slices.Equal(merged.EndpointTags, []string{"api"}) {
		t.Errorf("endpoint_tags: expected ['api'], got %v", merged.EndpointTags)
	}

	// 2. Vars merged
	if merged.Vars["VAR_A"] != "parent-a" {
		t.Errorf("VAR_A: expected 'parent-a', got %q", merged.Vars["VAR_A"])
	}
	if merged.Vars["VAR_B"] != "child-b" {
		t.Errorf("VAR_B: expected 'child-b', got %q", merged.Vars["VAR_B"])
	}
	if merged.Vars["VAR_C"] != "child-c" {
		t.Errorf("VAR_C: expected 'child-c', got %q", merged.Vars["VAR_C"])
	}

	// 3. Entries merged by name (replacement, not union; new appended)
	if len(merged.Entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(merged.Entries))
	}
	// First entry: compose-app (replaced by child)
	e0 := merged.Entries[0]
	if e0.Name != "compose-app" || e0.Order != 15 || len(e0.Services) != 3 || e0.Services[2] != "minio" {
		t.Errorf("entry 0: expected child replaced compose-app, got %+v", e0)
	}
	// Second entry: api (inherited from parent)
	e1 := merged.Entries[1]
	if e1.Name != "api" || e1.Order != 20 {
		t.Errorf("entry 1: expected inherited api, got %+v", e1)
	}
	// Third entry: worker (appended by child)
	e2 := merged.Entries[2]
	if e2.Name != "worker" || e2.Order != 30 {
		t.Errorf("entry 2: expected appended worker, got %+v", e2)
	}
}
