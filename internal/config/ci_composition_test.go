package config

import (
	"encoding/json"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestCIChildCompositionDeclaration(t *testing.T) {
	base := "ci:\n  profiles:\n    full:\n      timeout: 1m\n      steps:\n        - name: native\n          run: true\n"
	for _, tc := range []struct {
		name, field string
		valid       bool
	}{
		{"omitted", "", true},
		{"opt-in", "          child_ci: independent-root\n", true},
		{"unknown", "          child_ci: reentrant\n", false},
		{"boolean", "          child_ci: true\n", false},
		{"number", "          child_ci: 1\n", false},
		{"null", "          child_ci: null\n", false},
		{"empty", "          child_ci: ''\n", false},
		{"whitespace", "          child_ci: ' independent-root'\n", false},
		{"reserved", "          environment: {DVA_CI_COMPOSITION: forged}\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := loadCIConfig(t, base+tc.field)
			if (err == nil) != tc.valid {
				t.Fatalf("load result: %v; valid=%t", err, tc.valid)
			}
		})
	}
}

func TestCIChildCompositionSerialization(t *testing.T) {
	step := CIStep{Name: "native", Run: "true", ChildCI: CIChildIndependentRoot}
	data, err := json.Marshal(step)
	if err != nil || !strings.Contains(string(data), `"child_ci":"independent-root"`) {
		t.Fatalf("JSON: %s %v", data, err)
	}
	var decoded CIStep
	if err := json.Unmarshal(data, &decoded); err != nil || decoded.ChildCI != step.ChildCI {
		t.Fatalf("round trip: %#v %v", decoded, err)
	}
	step.ChildCI = ""
	data, err = json.Marshal(step)
	if err != nil || strings.Contains(string(data), "child_ci") {
		t.Fatalf("omission changed legacy JSON: %s %v", data, err)
	}
	step.OCI, step.ChildCI = &CIOCI{}, CIChildIndependentRoot
	if _, err := json.Marshal(step); err == nil {
		t.Fatal("OCI must not silently discard child_ci while serializing JSON")
	}
	if _, err := yaml.Marshal(step); err == nil {
		t.Fatal("OCI must not silently discard child_ci while serializing YAML")
	}
	cfg := &Config{CI: &CIConfig{Profiles: map[string]CIProfile{"full": {
		Timeout: "1m", Steps: []CIStep{step},
	}}}}
	if _, err := cfg.ResolveCIProfile("full"); err == nil || !strings.Contains(err.Error(), "native") {
		t.Fatalf("OCI composition resolve error: %v", err)
	}
}

func TestCIChildCompositionMergedAndJSONPresence(t *testing.T) {
	for _, value := range []string{"null", "''", "true", "7"} {
		var step CIStep
		input := "<<: &base {name: native, run: 'true', child_ci: " + value + "}"
		if err := yaml.Unmarshal([]byte(input), &step); err == nil {
			t.Fatalf("merged invalid presence accepted: %s", input)
		}
	}
	for _, value := range []string{"null", `""`, "true", "7"} {
		var step CIStep
		if err := json.Unmarshal([]byte(`{"name":"native","run":"true","child_ci":`+value+`}`), &step); err == nil {
			t.Fatalf("invalid JSON presence accepted: %s", value)
		}
	}
}
