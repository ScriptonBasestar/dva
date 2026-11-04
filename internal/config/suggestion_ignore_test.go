package config

import (
	"encoding/json"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSuggestionIgnoreRuleLegacyScalarRoundTrips(t *testing.T) {
	var rules []SuggestionIgnoreRule
	if err := yaml.Unmarshal([]byte("- release-*\n"), &rules); err != nil {
		t.Fatalf("unmarshal yaml: %v", err)
	}
	if len(rules) != 1 || rules[0].Pattern != "release-*" || rules[0].Reason != "" || rules[0].MaxMatches != nil {
		t.Fatalf("legacy YAML rule = %+v, want scalar-compatible rule", rules)
	}
	yamlOut, err := yaml.Marshal(rules)
	if err != nil {
		t.Fatalf("marshal yaml: %v", err)
	}
	if got := string(yamlOut); got != "- release-*\n" {
		t.Fatalf("legacy YAML round trip = %q, want scalar spelling", got)
	}

	if err := json.Unmarshal([]byte(`["release-*"]`), &rules); err != nil {
		t.Fatalf("unmarshal json: %v", err)
	}
	jsonOut, err := json.Marshal(rules)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	if got := string(jsonOut); got != `["release-*"]` {
		t.Fatalf("legacy JSON round trip = %s, want scalar spelling", got)
	}
}

func TestSuggestionIgnoreRuleObjectRoundTrips(t *testing.T) {
	var rules []SuggestionIgnoreRule
	if err := yaml.Unmarshal([]byte("- pattern: release-*\n  reason: release automation\n  max_matches: 3\n"), &rules); err != nil {
		t.Fatalf("unmarshal yaml object: %v", err)
	}
	if len(rules) != 1 || rules[0].Pattern != "release-*" || rules[0].Reason != "release automation" || rules[0].MaxMatches == nil || *rules[0].MaxMatches != 3 {
		t.Fatalf("object YAML rule = %+v, want pattern/reason/cap", rules)
	}
	jsonOut, err := json.Marshal(rules)
	if err != nil {
		t.Fatalf("marshal object JSON: %v", err)
	}
	if got := string(jsonOut); got != `[{"pattern":"release-*","reason":"release automation","max_matches":3}]` {
		t.Fatalf("object JSON round trip = %s", got)
	}
}

func TestSuggestionIgnoreRuleRejectsIncompleteObject(t *testing.T) {
	for _, input := range []string{
		"- pattern: build\n",
		"- pattern: build\n  reason: '   '\n",
		"- pattern: build-*\n  reason: CI only\n",
		"- pattern: build-*\n  reason: CI only\n  max_matches: 0\n",
	} {
		var rules []SuggestionIgnoreRule
		if err := yaml.Unmarshal([]byte(input), &rules); err == nil {
			t.Fatalf("YAML %q loaded, want object contract error", input)
		}
	}
}

func TestSuggestionIgnoreRuleMatches(t *testing.T) {
	rule := SuggestionIgnoreRule{Pattern: "test-*"}
	if !rule.Matches("test-unit") || rule.Matches("build") {
		t.Fatalf("Matches did not preserve path.Match semantics")
	}
	if (SuggestionIgnoreRule{Pattern: "["}).Matches("anything") {
		t.Fatal("invalid legacy pattern must remain a non-match")
	}
}

func TestSuggestionIgnoreRuleJSONRejectsObjectWithoutReason(t *testing.T) {
	var rule SuggestionIgnoreRule
	err := json.Unmarshal([]byte(`{"pattern":"build"}`), &rule)
	if err == nil || !strings.Contains(err.Error(), "reason") {
		t.Fatalf("JSON object error = %v, want missing reason", err)
	}
}

func TestSuggestionIgnoreRuleSchemaContract(t *testing.T) {
	for _, tc := range []struct {
		name    string
		entry   string
		wantErr bool
	}{
		{name: "legacy scalar", entry: `"release-*"`},
		{name: "exact object with reason", entry: "pattern: build\n    reason: covered by native build"},
		{name: "glob object with positive cap", entry: "pattern: test-*\n    reason: CI matrix\n    max_matches: 2"},
		{name: "glob lacks cap", entry: "pattern: test-*\n    reason: CI matrix", wantErr: true},
		{name: "whitespace reason", entry: "pattern: build\n    reason: '   '", wantErr: true},
		{name: "zero cap", entry: "pattern: test-*\n    reason: CI matrix\n    max_matches: 0", wantErr: true},
		{name: "unknown object field", entry: "pattern: build\n    reason: CI matrix\n    because: legacy", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			document := "suggestion_ignore:\n  - " + tc.entry + "\n"
			err := validateYAMLSchema([]byte(document))
			if tc.wantErr {
				if err == nil {
					t.Fatalf("validateYAMLSchema accepted invalid entry:\n%s", document)
				}
				return
			}
			if err != nil {
				t.Fatalf("validateYAMLSchema rejected valid entry:\n%s\nerror: %v", document, err)
			}
		})
	}
}
