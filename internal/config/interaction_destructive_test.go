package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestInteractionCommandDestructiveYAML(t *testing.T) {
	yamlContent := `
reset-db:
  description: "Reset the database"
  command: "rake db:reset"
  destructive: true
migrate:
  description: "Run migrations"
  command: "rake db:migrate"
safe-override:
  description: "Explicitly safe"
  command: "echo safe"
  destructive: false
`
	var interactions map[string]*InteractionCommand
	if err := yaml.Unmarshal([]byte(yamlContent), &interactions); err != nil {
		t.Fatalf("failed to unmarshal yaml: %v", err)
	}

	if !interactions["reset-db"].IsDestructive() {
		t.Errorf("reset-db IsDestructive() = false, want true")
	}
	if interactions["migrate"].IsDestructive() {
		t.Errorf("migrate IsDestructive() = true, want false (omitted)")
	}
	if interactions["safe-override"].IsDestructive() {
		t.Errorf("safe-override IsDestructive() = true, want false (explicit false)")
	}
}

func TestMergeInteractionCommandDestructive(t *testing.T) {
	tTrue := true
	tFalse := false

	tests := []struct {
		name     string
		base     *bool
		other    *bool
		expected bool
	}{
		{
			name:     "base nil, other true -> true",
			base:     nil,
			other:    &tTrue,
			expected: true,
		},
		{
			name:     "base true, other nil -> preserved true",
			base:     &tTrue,
			other:    nil,
			expected: true,
		},
		{
			name:     "base true, other false -> overridden false",
			base:     &tTrue,
			other:    &tFalse,
			expected: false,
		},
		{
			name:     "base false, other true -> overridden true",
			base:     &tFalse,
			other:    &tTrue,
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			baseCmd := &InteractionCommand{Destructive: tc.base}
			otherCmd := &InteractionCommand{Destructive: tc.other}
			merged, err := mergeInteractionCommand(baseCmd, otherCmd)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if merged.IsDestructive() != tc.expected {
				t.Errorf("merged.IsDestructive() = %v, want %v", merged.IsDestructive(), tc.expected)
			}
		})
	}
}

func TestInteractionCommandDestructiveSchemaValidation(t *testing.T) {
	validConfig := `
version: "0.1.0"
interaction:
  db-reset:
    description: "Reset database"
    command: "bin/reset"
    destructive: true
`
	if err := validateYAMLSchema([]byte(validConfig)); err != nil {
		t.Fatalf("valid config failed schema validation: %v", err)
	}

	invalidConfig := `
version: "0.1.0"
interaction:
  db-reset:
    description: "Reset database"
    command: "bin/reset"
    destructive: "not-a-boolean"
`
	if err := validateYAMLSchema([]byte(invalidConfig)); err == nil {
		t.Fatal("expected schema validation error for non-boolean destructive field, got nil")
	}
}
