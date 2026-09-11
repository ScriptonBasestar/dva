package agentdeny

import (
	"encoding/json"
	"os"
	"slices"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
	"github.com/ScriptonBasestar/dva/internal/skillinstall"
)

func TestProjectScopeDestructivePatternsProjection(t *testing.T) {
	tTrue := true
	tFalse := false

	cfg := &config.Config{
		Interaction: map[string]*config.InteractionCommand{
			"db": {
				Description: "Database operations",
				Destructive: &tTrue,
				Subcommands: map[string]*config.InteractionCommand{
					"reset": {
						Description: "Reset DB",
						Command:     "rake db:reset",
					},
					"migrate": {
						Description: "Run migrations",
						Command:     "rake db:migrate",
						Destructive: &tFalse,
					},
				},
			},
			"cache": {
				Description: "Cache operations",
				Subcommands: map[string]*config.InteractionCommand{
					"flush": {
						Description: "Flush cache",
						Command:     "redis-cli flushall",
						Destructive: &tTrue,
					},
				},
			},
			"status": {
				Description: "Safe status check",
				Command:     "echo ok",
			},
		},
	}

	patterns := DestructiveInteractionPatterns(cfg)
	expected := []string{
		"Bash(dva cache flush *)",
		"Bash(dva db *)",
		"Bash(dva db reset *)",
		"Bash(dva run cache flush *)",
		"Bash(dva run db *)",
		"Bash(dva run db reset *)",
	}

	for _, want := range expected {
		if !slices.Contains(patterns, want) {
			t.Errorf("DestructiveInteractionPatterns missing %q; got: %v", want, patterns)
		}
	}
	for _, notWant := range []string{
		"Bash(dva status *)",
		"Bash(dva run status *)",
		"Bash(dva db migrate *)",
		"Bash(dva run db migrate *)",
	} {
		if slices.Contains(patterns, notWant) {
			t.Errorf("DestructiveInteractionPatterns contains safe command %q", notWant)
		}
	}

	t.Run("Install in project scope writes destructive patterns to settings.json", func(t *testing.T) {
		options, root := testOptions(t)
		options.Config = cfg
		options.Scope = skillinstall.ScopeProject

		result, err := Install(options)
		if err != nil {
			t.Fatalf("Install failed: %v", err)
		}
		if len(result.Destinations) != 1 || result.Destinations[0].Status != "installed" {
			t.Fatalf("unexpected destination result: %+v", result)
		}

		contents, err := os.ReadFile(settingsPath(root))
		if err != nil {
			t.Fatalf("failed to read settings file: %v", err)
		}
		var doc struct {
			Permissions struct {
				Deny []string `json:"deny"`
			} `json:"permissions"`
		}
		if err := json.Unmarshal(contents, &doc); err != nil {
			t.Fatalf("invalid json: %v", err)
		}

		for _, want := range expected {
			if !slices.Contains(doc.Permissions.Deny, want) {
				t.Errorf("settings.json missing pattern %q; got: %v", want, doc.Permissions.Deny)
			}
		}

		// Status reports installed
		statusResult, err := Status(options)
		if err != nil {
			t.Fatalf("Status failed: %v", err)
		}
		if statusResult.Destinations[0].Status != "installed" {
			t.Errorf("Status got %q, want 'installed'", statusResult.Destinations[0].Status)
		}

		// Uninstall cleanly removes patterns
		unres, err := Uninstall(options)
		if err != nil {
			t.Fatalf("Uninstall failed: %v", err)
		}
		if unres.Destinations[0].Status != "uninstalled" {
			t.Errorf("Uninstall got %q, want 'uninstalled'", unres.Destinations[0].Status)
		}
	})

	t.Run("User scope does not include project destructive patterns", func(t *testing.T) {
		options, _ := testOptions(t)
		options.Config = cfg
		options.Scope = skillinstall.ScopeUser

		wantPatterns := desiredPatterns(options)
		for _, p := range expected {
			if slices.Contains(wantPatterns, p) {
				t.Errorf("user scope should not contain project pattern %q", p)
			}
		}
	})
}
