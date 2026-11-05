package config

import (
	"strings"
	"testing"
)

func TestWarnDuplicateStackOrder(t *testing.T) {
	c := &Config{Stack: map[string]*LifecycleEntry{
		"compose":      {Order: 10},
		"compose-full": {Order: 10},
	}}
	warnings := c.warnDuplicateStackOrder()
	if len(warnings) != 1 || !strings.Contains(warnings[0], "order value 10") {
		t.Fatalf("warnings = %v, want one order warning", warnings)
	}

	c = &Config{Stack: map[string]*LifecycleEntry{
		"compose": {Order: 10},
		"k8s":     {Order: 20},
	}}
	if warnings = c.warnDuplicateStackOrder(); len(warnings) != 0 {
		t.Fatalf("warnings = %v, want no different-order warning", warnings)
	}

	c = &Config{
		Stack: map[string]*LifecycleEntry{"a": {}, "b": {}, "c": {}},
		Plans: map[string]*PlanConfig{"local": {Entries: []PlanEntry{{Name: "a", Order: 10}}}},
	}
	warnings = c.warnDuplicateStackOrder()
	if len(warnings) != 1 || !strings.Contains(warnings[0], "entries b, c") || !strings.Contains(warnings[0], "dva up <plan>") {
		t.Fatalf("warnings = %v, want plan-aware unpositioned entries", warnings)
	}

	c.Plans = nil
	warnings = c.warnDuplicateStackOrder()
	if len(warnings) != 1 || !strings.Contains(warnings[0], "entries a, b, c") || strings.Contains(warnings[0], "dva up <plan>") {
		t.Fatalf("warnings = %v, want no-plan warning", warnings)
	}
}
