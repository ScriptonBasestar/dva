package config

import (
	"strings"
	"testing"
)

// primaryComposeConfig builds a stack of compose entries, marking the named ones primary.
// Orders are set so that the implicit order/name rule would pick "aardvark", letting a
// test tell an explicit primary apart from the inferred one.
func primaryComposeConfig(primaries ...string) *Config {
	names := []string{"aardvark", "monkey", "zebra"}
	stack := make(map[string]*LifecycleEntry, len(names))
	for i, n := range names {
		e := &LifecycleEntry{
			Name:    n,
			Order:   i,
			Compose: &ComposePluginConfig{Files: []string{n + ".yml"}},
		}
		for _, p := range primaries {
			if p == n {
				e.Primary = true
			}
		}
		stack[n] = e
	}
	return &Config{Stack: stack}
}

// An explicit primary: true overrides the order/name inference that would otherwise win.
func TestPrimaryComposeEntryExplicitBeatsOrder(t *testing.T) {
	cfg := primaryComposeConfig("zebra")
	if e := cfg.PrimaryComposeEntry(); e == nil || e.Name != "zebra" {
		t.Fatalf("PrimaryComposeEntry() = %v, want zebra", e)
	}
}

// With no entry marked, the pre-existing order/name inference still decides.
func TestPrimaryComposeEntryFallsBackToOrder(t *testing.T) {
	cfg := primaryComposeConfig()
	if e := cfg.PrimaryComposeEntry(); e == nil || e.Name != "aardvark" {
		t.Fatalf("PrimaryComposeEntry() = %v, want aardvark", e)
	}
}

// Config.Stack is a map, so a selection that takes "the first primary seen" varies with
// Go's randomized iteration order. Repeating the call makes that nondeterminism visible:
// every call must agree, and on the name warnMultiplePrimaryCompose reports as used.
func TestPrimaryComposeEntryMultiplePrimariesIsDeterministic(t *testing.T) {
	cfg := primaryComposeConfig("zebra", "monkey")
	for i := range 50 {
		e := cfg.PrimaryComposeEntry()
		if e == nil || e.Name != "monkey" {
			t.Fatalf("call %d: PrimaryComposeEntry() = %v, want monkey on every call", i, e)
		}
	}
}

// A non-compose entry marked primary must not hijack the compose selection.
func TestPrimaryComposeEntryIgnoresNonComposePrimary(t *testing.T) {
	cfg := primaryComposeConfig()
	cfg.Stack["worker"] = &LifecycleEntry{Name: "worker", Primary: true, Process: &ProcessPluginConfig{Command: "./worker"}}
	if e := cfg.PrimaryComposeEntry(); e == nil || e.Name != "aardvark" {
		t.Fatalf("PrimaryComposeEntry() = %v, want aardvark", e)
	}
}

func TestWarnMultiplePrimaryCompose(t *testing.T) {
	t.Run("one primary is silent", func(t *testing.T) {
		if got := primaryComposeConfig("zebra").warnMultiplePrimaryCompose(); len(got) != 0 {
			t.Fatalf("warnMultiplePrimaryCompose() = %v, want none", got)
		}
	})
	t.Run("no primary is silent", func(t *testing.T) {
		if got := primaryComposeConfig().warnMultiplePrimaryCompose(); len(got) != 0 {
			t.Fatalf("warnMultiplePrimaryCompose() = %v, want none", got)
		}
	})
	t.Run("names the entry actually used", func(t *testing.T) {
		cfg := primaryComposeConfig("zebra", "monkey")
		got := cfg.warnMultiplePrimaryCompose()
		if len(got) != 1 {
			t.Fatalf("warnMultiplePrimaryCompose() = %v, want one warning", got)
		}
		// The warning tells the user which entry wins; if it named a different one than
		// PrimaryComposeEntry returns, the advice would send them to the wrong entry.
		used := cfg.PrimaryComposeEntry()
		if used == nil || !strings.Contains(got[0], used.Name) {
			t.Errorf("warning %q does not name the entry in use (%v)", got[0], used)
		}
		for _, n := range []string{"monkey", "zebra"} {
			if !strings.Contains(got[0], n) {
				t.Errorf("warning %q omits conflicting entry %q", got[0], n)
			}
		}
	})
}
