package config

import (
	"slices"
	"sort"
	"strings"
	"testing"
)

func TestWarnHealthCheckRedundancy(t *testing.T) {
	// Both start and start_hint → warning
	c := &Config{
		HealthChecks: map[string]HealthCheckConfig{
			"app": {
				Type:      "http",
				URL:       "http://localhost:8080/health",
				Start:     "make run",
				StartHint: "make run",
			},
		},
		Stack: make(map[string]*LifecycleEntry),
	}
	warnings := c.warnHealthCheckRedundancy()
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
	if !strings.Contains(warnings[0], "start_hint") {
		t.Errorf("unexpected warning: %s", warnings[0])
	}

	// Only start → no warning
	c = &Config{
		HealthChecks: map[string]HealthCheckConfig{
			"app": {Type: "http", Start: "make run"},
		},
		Stack: make(map[string]*LifecycleEntry),
	}
	warnings = c.warnHealthCheckRedundancy()
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d", len(warnings))
	}

	// Only start_hint → no warning
	c = &Config{
		HealthChecks: map[string]HealthCheckConfig{
			"app": {Type: "http", StartHint: "make run"},
		},
		Stack: make(map[string]*LifecycleEntry),
	}
	warnings = c.warnHealthCheckRedundancy()
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d", len(warnings))
	}
}

func TestWarnHealthCheckRedundancy_StackNested(t *testing.T) {
	c := &Config{
		HealthChecks: make(map[string]HealthCheckConfig),
		Stack: map[string]*LifecycleEntry{
			"infra": {
				HealthChecks: map[string]HealthCheckConfig{
					"db": {
						Type:      "tcp",
						Address:   "localhost:5432",
						Start:     "pg_ctl start",
						StartHint: "pg_ctl start",
					},
				},
			},
		},
	}
	warnings := c.warnHealthCheckRedundancy()
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning for nested health check, got %d", len(warnings))
	}
	if !strings.Contains(warnings[0], "stack.infra.health_checks.db") {
		t.Errorf("unexpected warning: %s", warnings[0])
	}
}

// TestWarnUnreachableHealthChecks covers TASK-179: top-level health_checks.start /
// start_hint only run when a modes.*.health_checks entry names them. Nested
// stack.*.health_checks is a different field and must stay silent here.
func TestWarnUnreachableHealthChecks(t *testing.T) {
	t.Run("start without modes warns", func(t *testing.T) {
		c := &Config{
			HealthChecks: map[string]HealthCheckConfig{
				"api":  {Type: "http", URL: "http://localhost:1", Start: "make run"},
				"web":  {Type: "http", URL: "http://localhost:2", StartHint: "start web"},
				"both": {Type: "tcp", Address: "localhost:3", Start: "up", StartHint: "up by hand"},
				// readiness-only: no start/start_hint → no warning from this pass
				"probe": {Type: "http", URL: "http://localhost:4"},
			},
		}
		warnings := c.warnUnreachableHealthChecks()
		if len(warnings) != 3 {
			t.Fatalf("expected 3 warnings, got %d: %v", len(warnings), warnings)
		}
		joined := strings.Join(warnings, "\n")
		for _, name := range []string{"api", "web", "both"} {
			if !strings.Contains(joined, "health_checks."+name) {
				t.Errorf("missing warning for %s in %v", name, warnings)
			}
		}
		if strings.Contains(joined, "probe") {
			t.Errorf("readiness-only probe must not warn: %v", warnings)
		}
		if !strings.Contains(joined, "no modes.*.health_checks") {
			t.Errorf("warning should say why it will not run: %v", warnings)
		}
	})

	t.Run("referenced by modes is silent", func(t *testing.T) {
		c := &Config{
			HealthChecks: map[string]HealthCheckConfig{
				"api": {Type: "http", Start: "make run"},
				"web": {Type: "http", StartHint: "start web"},
			},
			Modes: map[string]ModeConfig{
				"dev": {HealthChecks: []string{"api", "web"}},
			},
		}
		warnings := c.warnUnreachableHealthChecks()
		if len(warnings) != 0 {
			t.Fatalf("expected 0 warnings when modes reference checks, got %v", warnings)
		}
	})

	t.Run("partial mode coverage warns only unreferenced", func(t *testing.T) {
		c := &Config{
			HealthChecks: map[string]HealthCheckConfig{
				"api":    {Start: "make api"},
				"worker": {Start: "make worker"},
			},
			Modes: map[string]ModeConfig{
				"dev": {HealthChecks: []string{"api"}},
			},
		}
		warnings := c.warnUnreachableHealthChecks()
		if len(warnings) != 1 {
			t.Fatalf("expected 1 warning, got %d: %v", len(warnings), warnings)
		}
		if !strings.Contains(warnings[0], "health_checks.worker") {
			t.Errorf("expected worker warning, got %s", warnings[0])
		}
		if strings.Contains(warnings[0], "health_checks.api") {
			t.Errorf("api is referenced and must not warn: %s", warnings[0])
		}
	})

	t.Run("stack nested start draws no warning", func(t *testing.T) {
		c := &Config{
			Stack: map[string]*LifecycleEntry{
				"infra": {
					HealthChecks: map[string]HealthCheckConfig{
						"db": {Type: "tcp", Address: "localhost:5432", Start: "pg_ctl start"},
					},
				},
			},
		}
		warnings := c.warnUnreachableHealthChecks()
		if len(warnings) != 0 {
			t.Fatalf("nested stack health_checks must not warn, got %v", warnings)
		}
	})

	t.Run("mode references missing health_checks name", func(t *testing.T) {
		c := &Config{
			HealthChecks: map[string]HealthCheckConfig{
				"api": {Start: "make run"},
			},
			Modes: map[string]ModeConfig{
				"dev": {HealthChecks: []string{"api", "missing-svc"}},
			},
		}
		warnings := c.warnUnreachableHealthChecks()
		if len(warnings) != 1 {
			t.Fatalf("expected 1 dangling-reference warning, got %d: %v", len(warnings), warnings)
		}
		if !strings.Contains(warnings[0], "modes.dev.health_checks") || !strings.Contains(warnings[0], "missing-svc") {
			t.Errorf("unexpected warning: %s", warnings[0])
		}
	})

	t.Run("warnings are order-stable", func(t *testing.T) {
		c := &Config{
			HealthChecks: map[string]HealthCheckConfig{
				"alpha":   {Start: "a"},
				"bravo":   {Start: "b"},
				"charlie": {StartHint: "c"},
			},
			Modes: map[string]ModeConfig{
				"m1": {HealthChecks: []string{"nope-a"}},
				"m2": {HealthChecks: []string{"nope-b"}},
			},
		}
		first := c.warnUnreachableHealthChecks()
		// 3 unreferenced + 2 dangling
		if len(first) != 5 {
			t.Fatalf("expected 5 warnings, got %d: %v", len(first), first)
		}
		if !sort.StringsAreSorted(first) {
			t.Errorf("warnings are not sorted: %v", first)
		}
		for i := range 50 {
			if got := c.warnUnreachableHealthChecks(); !slices.Equal(got, first) {
				t.Fatalf("run %d differs from run 0:\n first: %v\n got:   %v", i+1, first, got)
			}
		}
	})
}
