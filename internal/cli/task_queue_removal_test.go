package cli

import (
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// Task tooling remains usable as a project interaction without a built-in
// command taking its name or advertising a DVA-owned task lifecycle.
func TestTaskQueueNamesBelongToProjectInteractions(t *testing.T) {
	for _, name := range []string{"task-queue", "task-queue-verdict", "task-queue-start"} {
		t.Run(name, func(t *testing.T) {
			for _, command := range rootCmd.Commands() {
				if command.Name() == name {
					t.Fatalf("task command %q is registered as a built-in", name)
				}
			}
			cfg := &config.Config{Interaction: map[string]*config.InteractionCommand{
				name: {Command: "external-task-tool", Runner: "local"},
			}}
			if conflicts := config.ValidateReservedCommands(cfg.Interaction); len(conflicts) != 0 {
				t.Fatalf("project task interaction has reserved-name conflicts: %v", conflicts)
			}
			manifest := buildManifest(cfg)
			if _, exists := manifest.StaticCommands[name]; exists {
				t.Fatalf("manifest advertises %q as a built-in", name)
			}
		})
	}
}
