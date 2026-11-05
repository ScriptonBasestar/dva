package cli

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRemovedModeFlagsAreRejectedBeforeLifecycleLoading(t *testing.T) {
	for name, cmd := range map[string]*cobra.Command{
		"up": upCmd, "down": downCmd, "stop": stopCmd, "restart": restartCmd, "build": buildCmd,
	} {
		t.Run(name, func(t *testing.T) {
			err := cmd.RunE(cmd, []string{"--mode=legacy"})
			if err == nil || !strings.Contains(err.Error(), "removed in v0.5.0") {
				t.Fatalf("%s --mode error = %v, want removal error before config/backend work", name, err)
			}
		})
	}
}

func TestRemovedModeGuardPrecedesHooksAndRespectsTerminator(t *testing.T) {
	called := false
	cmd := &cobra.Command{RunE: func(*cobra.Command, []string) error { called = true; return nil }}
	wrapWithHooks("mode-guard-probe", cmd)

	if err := cmd.RunE(cmd, []string{"-M", "legacy"}); err == nil || !strings.Contains(err.Error(), "removed in v0.5.0") {
		t.Fatalf("wrapped command error = %v, want removed-flag error", err)
	}
	if called {
		t.Fatal("wrapped command ran despite removed flag")
	}

	if err := rejectRemovedModeFlags([]string{"--", "--mode=backend-literal"}); err != nil {
		t.Fatalf("mode literal after -- must pass through: %v", err)
	}
}

func TestRemovedModeGuardCoversDirectPlanAndCompositionPaths(t *testing.T) {
	if _, err := parsePlanFlags("up", []string{"--mode", "legacy"}); err == nil || !strings.Contains(err.Error(), "removed in v0.5.0") {
		t.Fatalf("parsePlanFlags error = %v, want removed-flag error", err)
	}
	if _, err := validateCompositionFlagScope(nil, "release", "up", []string{"-M=legacy"}); err == nil || !strings.Contains(err.Error(), "removed in v0.5.0") {
		t.Fatalf("validateCompositionFlagScope error = %v, want removed-flag error", err)
	}
}
