package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// planProfilesYAML is the dns-bridge shape TASK-315 exists for: a compose project
// whose rust/monitoring services sit behind compose profiles, and a plan that turns
// the group on without naming its members.
const planProfilesYAML = `version: "0.1.0"
stack:
  core-compose:
    default_runner: compose
    runners:
      compose:
        files:
          - compose.yaml
default_plan: docker-dev
plans:
  docker-dev:
    entries:
      - name: core-compose
        runner: compose
        order: 10
        profiles: [rust, monitoring]
        services: [postgres, redis]
`

// TestPlanEntryProfilesDecoded locks the declaration surface: `profiles:` under a
// plan entry must survive both the JSON schema (additionalProperties:false there,
// so an undeclared key is a hard validation error, not a silently dropped one) and
// the struct decode.
func TestPlanEntryProfilesDecoded(t *testing.T) {
	if err := validateYAMLSchema([]byte(planProfilesYAML)); err != nil {
		t.Fatalf("schema rejected plans.<p>.entries[].profiles: %v", err)
	}

	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, FileName), []byte(planProfilesYAML), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	entry := cfg.Plans["docker-dev"].Entries[0]
	if got, want := strings.Join(entry.Profiles, ","), "rust,monitoring"; got != want {
		t.Fatalf("plan entry profiles = %q, want %q", got, want)
	}
	if got, want := strings.Join(entry.Services, ","), "postgres,redis"; got != want {
		t.Fatalf("plan entry services = %q, want %q (profiles must not displace services)", got, want)
	}
}

// TestPlanEntryProfilesDeepCopiedOnImport locks the subproject clone: an imported
// plan gets its own backing array, so a parent mutating the alias cannot reach into
// the child's declaration. The other slice fields on PlanEntry already do this.
func TestPlanEntryProfilesDeepCopiedOnImport(t *testing.T) {
	original := &PlanConfig{Entries: []PlanEntry{{Name: "core-compose", Profiles: []string{"rust"}}}}
	clone := cloneImportedPlan(original, nil, "child")

	clone.Entries[0].Profiles[0] = "mutated"
	if got := original.Entries[0].Profiles[0]; got != "rust" {
		t.Fatalf("imported clone shares the profiles array: original = %q, want rust", got)
	}
}

// TestPlanEntriesEqualComparesProfiles keeps profiles inside the D6 declaration
// comparison: two plans whose entries differ only in which profiles they activate
// start different containers, so they are not the same declaration.
func TestPlanEntriesEqualComparesProfiles(t *testing.T) {
	a := PlanEntry{Name: "core-compose", Profiles: []string{"rust"}}
	b := PlanEntry{Name: "core-compose", Profiles: []string{"monitoring"}}

	if planEntriesEqual(a, b) {
		t.Fatal("entries differing only in profiles compared equal")
	}
	if !planEntriesEqual(a, PlanEntry{Name: "core-compose", Profiles: []string{"rust"}}) {
		t.Fatal("identical entries compared unequal")
	}
}
