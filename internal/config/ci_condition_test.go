package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestCIConditionValidationAndRoundTrip(t *testing.T) {
	for name, mutate := range map[string]func(*CIStep){
		"valid":            func(*CIStep) {},
		"empty command":    func(s *CIStep) { s.Condition.Run = " " },
		"missing timeout":  func(s *CIStep) { s.Condition.Timeout = "" },
		"zero timeout":     func(s *CIStep) { s.Condition.Timeout = "0s" },
		"longer than step": func(s *CIStep) { s.Condition.Timeout = "31s" },
		"child":            func(s *CIStep) { s.ChildCI = CIChildIndependentRoot },
		"oci":              func(s *CIStep) { s.OCI = &CIOCI{} },
	} {
		t.Run(name, func(t *testing.T) {
			step := CIStep{Name: "unit", Run: "true", Timeout: "30s", Condition: &CICondition{Run: "printf '{}'", Timeout: "5s"}}
			mutate(&step)
			err := validateCIProfile("full", CIProfile{Timeout: "1m", MaxParallel: 1, Steps: []CIStep{step}})
			if (err == nil) != (name == "valid") {
				t.Fatalf("validation: %v", err)
			}
		})
	}
	want := CIStep{Name: "unit", Run: "true", Condition: &CICondition{Run: "printf '{}'", Timeout: "5s"}}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	var got CIStep
	if err := json.Unmarshal(data, &got); err != nil || got.Condition == nil || *got.Condition != *want.Condition {
		t.Fatalf("JSON: %s %v", data, err)
	}
	data, err = yaml.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	got = CIStep{}
	if err := yaml.Unmarshal(data, &got); err != nil || got.Condition == nil || *got.Condition != *want.Condition {
		t.Fatalf("YAML: %s %v", data, err)
	}
	want.OCI = &CIOCI{}
	if _, err := json.Marshal(want); err == nil {
		t.Fatal("OCI silently lost condition in JSON")
	}
	if _, err := yaml.Marshal(want); err == nil {
		t.Fatal("OCI silently lost condition in YAML")
	}
}

func TestLoadRejectsMalformedConditionSchema(t *testing.T) {
	for _, condition := range []string{
		"{run: 'true', timeout: 1s, extra: x}",
		"{run: '', timeout: 1s}",
		"{run: 'true'}",
		"null",
	} {
		t.Run(condition, func(t *testing.T) {
			dir := t.TempDir()
			text := "ci:\n  profiles:\n    full:\n      timeout: 1m\n      steps:\n        - name: unit\n          run: 'true'\n          condition: " + condition + "\n"
			if err := os.WriteFile(filepath.Join(dir, FileName), []byte(text), 0o600); err != nil {
				t.Fatal(err)
			}
			c, err := Load(dir)
			if err == nil {
				_, err = c.ResolveCIProfile("full")
			}
			if err == nil {
				t.Fatalf("accepted %s", strings.TrimSpace(condition))
			}
		})
	}
}

func TestJSONRejectsMalformedCondition(t *testing.T) {
	for _, value := range []string{`null`, `{"run":"true"}`, `{"run":"true","timeout":"1s","extra":1}`, `{"run":"","timeout":"1s"}`} {
		var step CIStep
		if err := json.Unmarshal([]byte(`{"name":"unit","run":"true","condition":`+value+`}`), &step); err == nil {
			t.Fatalf("accepted condition %s", value)
		}
	}
}
