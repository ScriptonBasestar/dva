package config

import (
	"strings"
	"testing"
)

func TestWarnUnresolvedEnvVars(t *testing.T) {
	c := &Config{
		Environment: map[string]string{
			"OK":   "value",
			"BAD":  "${MISSING_VAR}",
			"NOPE": "$MISSING_VAR_TOO",
			"GOOD": "${EXISTING_VAR}",
		},
	}
	env := NewEnvironment(c.Environment, ".", ".")
	env.Vars["EXISTING_VAR"] = "exists"

	warnings := c.warnUnresolvedEnvVars(env, false)
	if len(warnings) != 2 {
		t.Fatalf("expected 2 warnings, got %d", len(warnings))
	}
	// Warnings are sorted
	if !strings.Contains(warnings[0], "environment.BAD:") {
		t.Errorf("unexpected warning text: %s", warnings[0])
	}
	if !strings.Contains(warnings[1], "environment.NOPE:") {
		t.Errorf("unexpected warning text: %s", warnings[1])
	}
}

func TestWarnSuspiciousEnvPatterns(t *testing.T) {
	c := &Config{
		Environment: map[string]string{
			"DEFAULT":  "${VAR:-default}",
			"DEFAULT2": "${VAR-default}",
			"NESTED":   "${VAR:-${OTHER}:5432}",
			"ALT":      "${VAR:+alt}",
			"OP":       "${VAR:=ok}",
			"SPECIAL":  "count is $#",
			"GOOD":     "${VAR} and $VAR2",
		},
	}

	warnings := c.warnSuspiciousEnvPatterns()
	// `${VAR:-default}` and `${VAR-default}` are supported since TASK-303 and must not be
	// reported; ALT, OP and SPECIAL still are.
	if len(warnings) != 3 {
		t.Fatalf("expected 3 warnings, got %d: %v", len(warnings), warnings)
	}
	if !strings.Contains(warnings[0], "environment.ALT:") {
		t.Errorf("unexpected warning text: %s", warnings[0])
	}
	if !strings.Contains(warnings[1], "environment.OP:") {
		t.Errorf("unexpected warning text: %s", warnings[1])
	}
	if !strings.Contains(warnings[2], "environment.SPECIAL:") {
		t.Errorf("unexpected warning text: %s", warnings[2])
	}
}
