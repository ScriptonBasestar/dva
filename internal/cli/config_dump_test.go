package cli

import (
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func TestConfigSchemaViewUsesYAMLFieldNames(t *testing.T) {
	c := &config.Config{
		Version: "0.1.44",
		Provision: config.ProvisionConfig{
			DefaultProfile: "setup",
			Profiles: map[string][]config.ProvisionItem{
				"setup": {{Step: "Install deps", Run: "npm install"}},
			},
		},
	}

	view, err := configSchemaView(c)
	if err != nil {
		t.Fatalf("configSchemaView error: %v", err)
	}
	root, ok := view.(map[string]any)
	if !ok {
		t.Fatalf("view type = %T, want map[string]any", view)
	}
	if _, exists := root["Provision"]; exists {
		t.Fatal("view contains Go field name Provision")
	}
	provision, ok := root["provision"].(map[string]any)
	if !ok {
		t.Fatalf("provision type = %T, want map[string]any", root["provision"])
	}
	if got := provision["default_profile"]; got != "setup" {
		t.Fatalf("default_profile = %v, want setup", got)
	}
	if _, exists := provision["setup"]; !exists {
		t.Fatal("schema view missing setup profile")
	}
}

// TestConfigSchemaViewShowsCompare: `dva config show` renders a compare declaration under its
// yaml key with yaml field names (TASK-589).
func TestConfigSchemaViewShowsCompare(t *testing.T) {
	c := &config.Config{
		Compare: map[string]*config.CompareConfig{
			"web-layout": {
				Plan:       "web-dev",
				Shared:     []string{"api", "db"},
				PerVariant: []string{"web"},
				Ports:      map[string][]int{"WEB_PORT": {3001, 3002}},
				Variants:   map[string]string{"a": "main"},
			},
		},
	}
	view, err := configSchemaView(c)
	if err != nil {
		t.Fatalf("configSchemaView error: %v", err)
	}
	root := view.(map[string]any)
	compare, ok := root["compare"].(map[string]any)
	if !ok {
		t.Fatalf("compare type = %T, want map[string]any", root["compare"])
	}
	entry, ok := compare["web-layout"].(map[string]any)
	if !ok {
		t.Fatalf("compare.web-layout type = %T", compare["web-layout"])
	}
	for _, key := range []string{"plan", "shared", "per_variant", "ports", "variants"} {
		if _, exists := entry[key]; !exists {
			t.Errorf("compare.web-layout missing %q: %v", key, entry)
		}
	}
}
