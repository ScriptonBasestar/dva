package config

import (
	"strings"
	"testing"
)

func TestWarnHealthCheckRedundancy(t *testing.T) {
	c := &Config{HealthChecks: map[string]HealthCheckConfig{
		"app": {Type: "http", URL: "http://localhost:8080/health", Start: "make run", StartHint: "make run"},
	}}
	if warnings := c.warnHealthCheckRedundancy(); len(warnings) != 1 || !strings.Contains(warnings[0], "start_hint") {
		t.Fatalf("warnings = %v, want redundancy warning", warnings)
	}

	c = &Config{Stack: map[string]*LifecycleEntry{
		"infra": {HealthChecks: map[string]HealthCheckConfig{
			"db": {Type: "tcp", Address: "localhost:5432", Start: "pg_ctl start", StartHint: "pg_ctl start"},
		}},
	}}
	if warnings := c.warnHealthCheckRedundancy(); len(warnings) != 1 || !strings.Contains(warnings[0], "stack.infra.health_checks.db") {
		t.Fatalf("warnings = %v, want nested redundancy warning", warnings)
	}
}

func TestWarnUnreachableHealthChecks(t *testing.T) {
	c := &Config{HealthChecks: map[string]HealthCheckConfig{
		"api":    {Start: "make api"},
		"hinted": {StartHint: "make hinted"},
		"probe":  {URL: "http://localhost:8080"},
	}}
	warnings := c.warnUnreachableHealthChecks()
	if len(warnings) != 2 || !strings.Contains(strings.Join(warnings, "\n"), "stack.<entry>.health_checks") {
		t.Fatalf("warnings = %v, want two stack migration warnings", warnings)
	}
	if got := c.warnOrphanHealthChecks(); len(got) != 1 || !strings.Contains(got[0], "health_checks.probe") {
		t.Fatalf("orphan warnings = %v", got)
	}
}
