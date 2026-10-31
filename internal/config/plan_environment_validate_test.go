package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidatePlanEnvironments(t *testing.T) {
	c := &Config{
		Environments: map[string]EnvironmentProfile{"dev": {}},
		Plans: map[string]*PlanConfig{
			"good": {Environment: "dev"},
			"none": {},
			"bad":  {Environment: "dve"},
		},
	}
	errs := c.validatePlanEnvironments()
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "plan 'bad': environment 'dve'") {
		t.Fatalf("want one error naming plan bad, got %v", errs)
	}
}

// An imported plan resolves against its owner, so the importer's environments do not count.
func TestValidatePlanEnvironmentsUsesPlanOwner(t *testing.T) {
	owner := &Config{Environments: map[string]EnvironmentProfile{"child": {}}}
	c := &Config{Plans: map[string]*PlanConfig{"imported": {Environment: "child", owner: owner}}}
	if errs := c.validatePlanEnvironments(); len(errs) != 0 {
		t.Fatalf("owner declares the environment, got %v", errs)
	}
}

func TestValidateFailsOnUnknownPlanEnvironment(t *testing.T) {
	path := filepath.Join(t.TempDir(), FileName)
	yml := `version: "0.1.44"
stack:
  x:
    default_runner: script
    runners:
      script:
        up: "true"
plans:
  bad:
    environment: nope
    entries:
      - name: x
`
	if err := os.WriteFile(path, []byte(yml), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := loadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	c.filePath = path
	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "environment 'nope'") {
		t.Fatalf("Validate must refuse the plan, got %v", err)
	}
}
