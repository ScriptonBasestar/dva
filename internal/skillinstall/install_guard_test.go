package skillinstall

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/skillclaim"
)

func TestDryRunDoesNotMutate(t *testing.T) {
	t.Parallel()
	options := testOptions(t, ScopeUser, RuntimeClaudeCode)
	options.DryRun = true
	result, err := Install(options)
	if err != nil {
		t.Fatal(err)
	}
	if result.Destinations[0].Status != "would-install" {
		t.Fatalf("status = %s", result.Destinations[0].Status)
	}
	if _, err := os.Stat(result.Destinations[0].Destination); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("dry-run created destination: %v", err)
	}
	if _, err := os.Stat(options.StateRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("dry-run created state: %v", err)
	}
	status, err := Status(options)
	if err != nil {
		t.Fatal(err)
	}
	if status.Destinations[0].Status != "absent" {
		t.Fatalf("status after dry-run = %s", status.Destinations[0].Status)
	}
}

func TestAbsentUninstallDryRunDoesNotCreateClaimState(t *testing.T) {
	t.Parallel()
	options := testOptions(t, ScopeProject, RuntimeCodex)
	options.DryRun = true
	result, err := Uninstall(options)
	if err != nil {
		t.Fatal(err)
	}
	if result.Destinations[0].Status != "not-installed" {
		t.Fatalf("absent uninstall dry-run = %s", result.Destinations[0].Status)
	}
	if _, err := os.Lstat(filepath.Join(filepath.Dir(options.StateRoot), "agent-skills")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("absent uninstall dry-run created persistent state: %v", err)
	}
	if _, err := os.Lstat(options.ProjectRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("absent uninstall dry-run created project state: %v", err)
	}
}

func TestUninstallDryRunDoesNotMutate(t *testing.T) {
	t.Parallel()
	options := testOptions(t, ScopeUser, RuntimeClaudeCode)
	installed, err := Install(options)
	if err != nil {
		t.Fatal(err)
	}
	destination := installed.Destinations[0].Destination
	receiptFile := receiptPath(options.StateRoot, destination)
	receiptBefore, err := os.ReadFile(receiptFile)
	if err != nil {
		t.Fatal(err)
	}
	options.DryRun = true
	result, err := Uninstall(options)
	if err != nil {
		t.Fatal(err)
	}
	if result.Destinations[0].Status != "would-uninstall" {
		t.Fatalf("status = %s", result.Destinations[0].Status)
	}
	if _, err := os.Stat(filepath.Join(destination, "dva", "SKILL.md")); err != nil {
		t.Fatalf("dry-run removed skill: %v", err)
	}
	receiptAfter, err := os.ReadFile(receiptFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(receiptBefore, receiptAfter) {
		t.Fatal("dry-run changed receipt")
	}
}

func TestCollisionAndDriftAreRefused(t *testing.T) {
	t.Parallel()
	options := testOptions(t, ScopeProject, RuntimeGrok)
	_, destinations, err := resolve(options)
	if err != nil {
		t.Fatal(err)
	}
	collision := filepath.Join(destinations[0].path, "dva")
	if err := os.MkdirAll(collision, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(options); err == nil {
		t.Fatal("collision install succeeded")
	}
	if err := os.RemoveAll(collision); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(options); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destinations[0].path, "dva", "SKILL.md"), []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Uninstall(options); err == nil {
		t.Fatal("drifted uninstall succeeded")
	}
	status, err := Status(options)
	if err != nil {
		t.Fatal(err)
	}
	if status.Destinations[0].Status != "drifted" {
		t.Fatalf("status = %s", status.Destinations[0].Status)
	}
}

func TestMultiDestinationOperationsPreflightBeforeMutation(t *testing.T) {
	t.Parallel()
	t.Run("install collision", func(t *testing.T) {
		options := testOptions(t, ScopeProject, RuntimeClaudeCode, RuntimeGrok)
		_, destinations, err := resolve(options)
		if err != nil {
			t.Fatal(err)
		}
		if len(destinations) != 2 {
			t.Fatalf("destinations = %v", destinations)
		}
		laterCollision := filepath.Join(destinations[1].path, "dva")
		if err := os.MkdirAll(laterCollision, 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := Install(options); err == nil {
			t.Fatal("multi-destination install ignored later collision")
		}
		if _, err := os.Stat(filepath.Join(destinations[0].path, "dva")); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("earlier destination mutated before collision: %v", err)
		}
	})

	t.Run("uninstall drift", func(t *testing.T) {
		options := testOptions(t, ScopeProject, RuntimeClaudeCode, RuntimeGrok)
		installed, err := Install(options)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(installed.Destinations[1].Destination, "dva", "SKILL.md"), []byte("drift"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Uninstall(options); err == nil {
			t.Fatal("multi-destination uninstall ignored later drift")
		}
		if _, err := os.Stat(filepath.Join(installed.Destinations[0].Destination, "dva", "SKILL.md")); err != nil {
			t.Fatalf("earlier destination removed before drift was found: %v", err)
		}
	})

	t.Run("claimed bundle expansion collision", func(t *testing.T) {
		options := testOptions(t, ScopeProject, RuntimeClaudeCode, RuntimeGrok)
		_, destinations, err := resolve(options)
		if err != nil {
			t.Fatal(err)
		}
		if len(destinations) != 2 {
			t.Fatalf("destinations = %v", destinations)
		}
		if _, err := Install(options); err != nil {
			t.Fatal(err)
		}
		records := make(map[string]receipt, len(destinations))
		for _, target := range destinations {
			records[target.path] = reduceClaimedBundleAtTarget(t, options, target)
		}
		laterCollision := filepath.Join(claimDestination(destinations[1], "dva-ci"), "foreign-marker")
		if err := os.MkdirAll(filepath.Dir(laterCollision), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(laterCollision, []byte("foreign"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Install(options); err == nil || !strings.Contains(err.Error(), "newly bundled DVA skill") {
			t.Fatalf("multi-destination claimed expansion ignored later collision: %v", err)
		}
		assertClaimedBundleState(t, options, destinations[0], records[destinations[0].path])
		if _, err := os.Stat(claimDestination(destinations[0], "dva-ci")); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("earlier claimed destination upgraded before later collision: %v", err)
		}
	})

	t.Run("legacy bundle expansion collision", func(t *testing.T) {
		options := testOptions(t, ScopeProject, RuntimeClaudeCode, RuntimeGrok)
		_, destinations, err := resolve(options)
		if err != nil {
			t.Fatal(err)
		}
		if len(destinations) != 2 {
			t.Fatalf("destinations = %v", destinations)
		}
		if _, err := Install(options); err != nil {
			t.Fatal(err)
		}
		for _, target := range destinations {
			record := reduceClaimedBundleAtTarget(t, options, target)
			record.Schema, record.Format = 1, ""
			if err := writeReceipt(receiptPath(options.StateRoot, target.path), record); err != nil {
				t.Fatal(err)
			}
			claims, err := projectedClaims(target, options.Scope, record.Runtimes, skillBundle{files: record.Files}, skillclaim.StateActive, "legacy-preflight")
			if err != nil {
				t.Fatal(err)
			}
			for _, claim := range claims {
				if err := os.Remove(skillclaim.Path(filepath.Dir(options.StateRoot), claim.Destination)); err != nil {
					t.Fatal(err)
				}
			}
		}
		laterCollision := filepath.Join(claimDestination(destinations[1], "dva-ci"), "foreign-marker")
		if err := os.MkdirAll(filepath.Dir(laterCollision), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(laterCollision, []byte("foreign"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Install(options); err == nil || !strings.Contains(err.Error(), "newly bundled DVA skill") {
			t.Fatalf("multi-destination legacy expansion ignored later collision: %v", err)
		}
		if _, err := os.Stat(claimDestination(destinations[0], "dva-ci")); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("earlier legacy destination migrated before later collision: %v", err)
		}
	})
}

func TestInvalidReceiptReportedAndRefused(t *testing.T) {
	t.Parallel()
	options := testOptions(t, ScopeUser, RuntimeOpenCode)
	_, destinations, err := resolve(options)
	if err != nil {
		t.Fatal(err)
	}
	path := receiptPath(options.StateRoot, destinations[0].path)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(options); err == nil {
		t.Fatal("install with invalid receipt succeeded")
	}
	status, err := Status(options)
	if err != nil {
		t.Fatal(err)
	}
	if status.Destinations[0].Status != "invalid-receipt" {
		t.Fatalf("status = %s", status.Destinations[0].Status)
	}
}

func TestStatusIdentifiesForeignCollisionAndReceiptPaths(t *testing.T) {
	t.Parallel()
	options := testOptions(t, ScopeProject, RuntimeClaudeCode)
	_, destinations, err := resolve(options)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(destinations[0].path, "dva"), 0o755); err != nil {
		t.Fatal(err)
	}
	status, err := Status(options)
	if err != nil {
		t.Fatal(err)
	}
	if status.Destinations[0].Status != "foreign-conflict" {
		t.Fatalf("status = %s", status.Destinations[0].Status)
	}
	for _, value := range []string{"dva/SKILL.md", "dva-config/references/a.md", "dva-ci/references/execution.md", "dva-ci.md"} {
		if !validReceiptPath(value) {
			t.Fatalf("valid receipt path rejected: %q", value)
		}
	}
	for _, value := range []string{"../dva/SKILL.md", "dva/../x", "dva-config/../../x", "/dva/SKILL.md", "other/SKILL.md", "dva-ci/../x", "other.md"} {
		if validReceiptPath(value) {
			t.Fatalf("invalid receipt path accepted: %q", value)
		}
	}
}

func TestSymlinkCollisionAndLegacyConfigAreNeverReplaced(t *testing.T) {
	t.Parallel()
	options := testOptions(t, ScopeUser, RuntimeCodex)
	_, destinations, err := resolve(options)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(destinations[0].path, 0o755); err != nil {
		t.Fatal(err)
	}
	legacy := filepath.Join(destinations[0].path, "config")
	if err := os.MkdirAll(legacy, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "SKILL.md"), []byte("legacy"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(options); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(legacy, "SKILL.md")); err != nil {
		t.Fatalf("legacy config skill was removed: %v", err)
	}

	symlinkOptions := testOptions(t, ScopeUser, RuntimeGrok)
	_, symlinkDestinations, err := resolve(symlinkOptions)
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "target")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(symlinkDestinations[0].path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, symlinkDestinations[0].path); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := Install(symlinkOptions); err == nil {
		t.Fatal("symlink destination install succeeded")
	}
}
