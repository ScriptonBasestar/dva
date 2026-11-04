package config

import (
	"reflect"
	"testing"
)

func TestValidationDiagnosticsPreserveLegacyMessagesAndOrder(t *testing.T) {
	c := &Config{Stack: map[string]*LifecycleEntry{
		"infra": {Order: 10, Compose: &ComposePluginConfig{Files: []string{"compose.yml"}}},
	}}
	diagnostics := c.ValidateDiagnostics()
	legacyOrder := c.warnLegacyStackOrder()
	noPlans := c.warnNoPlansHint()
	if len(legacyOrder) != 1 || len(noPlans) != 1 {
		t.Fatalf("fixture must produce both warning and hint: %v, %v", legacyOrder, noPlans)
	}
	want := []ValidationDiagnostic{
		{Severity: SeverityWarning, Message: legacyOrder[0]},
		{Severity: SeverityInfo, Message: noPlans[0]},
	}
	if !reflect.DeepEqual(diagnostics, want) {
		t.Fatalf("diagnostic order or severity changed: got %#v, want %#v", diagnostics, want)
	}
	if got := c.ValidateWarnings(); !reflect.DeepEqual(got, []string{legacyOrder[0], noPlans[0]}) {
		t.Fatalf("legacy messages changed: %v", got)
	}
	if got := (&Config{}).ValidateWarnings(); got != nil {
		t.Fatalf("empty legacy result changed from nil: %v", got)
	}
}

func TestValidationDiagnosticSeverityIsIndependentOfText(t *testing.T) {
	for _, tc := range []struct {
		name       string
		diagnostic ValidationDiagnostic
		want       bool
	}{
		{"info without glyph", ValidationDiagnostic{Severity: SeverityInfo, Message: "plain hint"}, false},
		{"warning with glyph", ValidationDiagnostic{Severity: SeverityWarning, Message: "ℹ warning"}, true},
		{"zero severity", ValidationDiagnostic{Message: "ℹ zero value"}, true},
		{"unknown severity", ValidationDiagnostic{Severity: DiagnosticSeverity(99), Message: "ℹ unknown"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.diagnostic.Actionable(); got != tc.want {
				t.Fatalf("Actionable() = %v, want %v", got, tc.want)
			}
		})
	}
}
