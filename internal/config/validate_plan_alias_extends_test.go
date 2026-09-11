package config

import (
	"strings"
	"testing"
)

func TestValidatePlanAliasExtends(t *testing.T) {
	tests := []struct {
		name        string
		plans       map[string]*PlanConfig
		wantErrText string
	}{
		{
			name: "alias self reference",
			plans: map[string]*PlanConfig{
				"self": {Alias: "self"},
			},
			wantErrText: `plan "self": alias cannot reference itself`,
		},
		{
			name: "alias target not found",
			plans: map[string]*PlanConfig{
				"p1": {Alias: "nonexistent"},
			},
			wantErrText: `plan "p1": alias target "nonexistent" not found`,
		},
		{
			name: "alias cycle detected",
			plans: map[string]*PlanConfig{
				"p1": {Alias: "p2"},
				"p2": {Alias: "p1"},
			},
			wantErrText: `plan "p1": alias cycle detected`,
		},
		{
			name: "alias plan declares entries",
			plans: map[string]*PlanConfig{
				"target": {
					Entries: []PlanEntry{{Name: "api"}},
				},
				"p1": {
					Alias:   "target",
					Entries: []PlanEntry{{Name: "api"}},
				},
			},
			wantErrText: `plan "p1": alias plan must not declare entries`,
		},
		{
			name: "extends self reference",
			plans: map[string]*PlanConfig{
				"self": {Extends: "self"},
			},
			wantErrText: `plan "self": extends cannot reference itself`,
		},
		{
			name: "extends target not found",
			plans: map[string]*PlanConfig{
				"child": {Extends: "parent"},
			},
			wantErrText: `plan "child": extends target "parent" not found`,
		},
		{
			name: "extends cycle detected",
			plans: map[string]*PlanConfig{
				"p1": {Extends: "p2"},
				"p2": {Extends: "p1"},
			},
			wantErrText: `plan "p1": extends cycle detected at "p1"`,
		},
		{
			name: "extends target is alias",
			plans: map[string]*PlanConfig{
				"base":  {Entries: []PlanEntry{{Name: "api"}}},
				"alias": {Alias: "base"},
				"child": {Extends: "alias"},
			},
			wantErrText: `plan "child": extends target "alias" must be a concrete plan (not an alias)`,
		},
		{
			name: "extends target is composes",
			plans: map[string]*PlanConfig{
				"comp":  {Composes: []CompositionEntry{{Plan: "other"}}},
				"child": {Extends: "comp"},
			},
			wantErrText: `plan "child": extends target "comp" must be a concrete plan (not a composition plan)`,
		},
		{
			name: "cannot have both alias and extends",
			plans: map[string]*PlanConfig{
				"base":  {Entries: []PlanEntry{{Name: "api"}}},
				"child": {Alias: "base", Extends: "base"},
			},
			wantErrText: `plan "child": cannot have both alias and extends`,
		},
		{
			name: "extends mutually exclusive with composes",
			plans: map[string]*PlanConfig{
				"base":  {Entries: []PlanEntry{{Name: "api"}}},
				"child": {Extends: "base", Composes: []CompositionEntry{{Plan: "base"}}},
			},
			wantErrText: `plan "child": extends is mutually exclusive with composes`,
		},
		{
			name: "extends chain exceeds max depth of 3",
			plans: map[string]*PlanConfig{
				"p1": {Entries: []PlanEntry{{Name: "api"}}},
				"p2": {Extends: "p1"},
				"p3": {Extends: "p2"},
				"p4": {Extends: "p3"},
				"p5": {Extends: "p4"},
			},
			wantErrText: `plan "p5": extends chain exceeds maximum depth of 3`,
		},
		{
			name: "clean valid alias and extends",
			plans: map[string]*PlanConfig{
				"base": {
					Description: "Base plan",
					Environment: "dev",
					Site:        "local",
					Entries:     []PlanEntry{{Name: "api", Order: 10}},
				},
				"base-alias": {
					Description: "Alias for base",
					Alias:       "base",
				},
				"child": {
					Description: "Child plan",
					Extends:     "base",
					Entries:     []PlanEntry{{Name: "worker", Order: 20}},
				},
			},
			wantErrText: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Plans: tt.plans}
			err := cfg.validatePlanAliasExtends()
			if tt.wantErrText == "" {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErrText)
			}
			if !strings.Contains(err.Error(), tt.wantErrText) {
				t.Fatalf("expected error containing %q, got: %v", tt.wantErrText, err)
			}
		})
	}
}

func TestValidateNDStackAliasExtendsExample(t *testing.T) {
	ndStackYAML := `version: "0.1"

vars:
  ENV: dev

environment:
  LOG_LEVEL: debug

stack:
  compose:
    default_runner: compose
    runners:
      compose:
        files: [docker-compose.yml]
  proxynd:
    default_runner: native
    runners:
      native:
        build: "go build -o bin/proxynd ./cmd/proxynd"
        run: "./bin/proxynd"
  depond:
    default_runner: native
    runners:
      native:
        build: "go build -o bin/depond ./cmd/depond"
        run: "./bin/depond"
  flownd:
    default_runner: native
    runners:
      native:
        build: "go build -o bin/flownd ./cmd/flownd"
        run: "./bin/flownd"

plans:
  local-infra:
    description: "Core local infrastructure"
    environment: dev
    site: local
    entries:
      - name: compose
        runner: compose
        order: 10
        services: [postgres, redis, minio]

  infra:
    alias: local-infra
    description: "Legacy-compatible alias for local-infra"

  local-dev:
    extends: local-infra
    description: "Local development with native microservices"
    entries:
      - name: proxynd
        runner: native
        order: 20
      - name: depond
        runner: native
        order: 21
      - name: flownd
        runner: native
        order: 22

  hybrid:
    extends: local-dev
    description: "Full dev stack including monitoring tools"
    entries:
      - name: compose
        runner: compose
        order: 10
        services: [postgres, redis, minio, redis-commander, adminer, prometheus, grafana]

default_plan: local-infra

environments:
  dev:
    environment:
      ENV: dev

sites:
  local:
    vars:
      HOST: localhost
`

	cfg, warnings, err := ValidateConfigBytes([]byte(ndStackYAML))
	if err != nil {
		t.Fatalf("expected valid nd-stack config, got error: %v", err)
	}
	if len(warnings) > 0 {
		t.Fatalf("expected 0 warnings, got %d: %v", len(warnings), warnings)
	}
	if cfg.Plans["infra"].Alias != "local-infra" {
		t.Errorf("expected infra alias local-infra, got %q", cfg.Plans["infra"].Alias)
	}
	if cfg.Plans["hybrid"].Extends != "local-dev" {
		t.Errorf("expected hybrid extends local-dev, got %q", cfg.Plans["hybrid"].Extends)
	}
}

