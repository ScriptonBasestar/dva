package skillinstall

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/skillclaim"
	bundled "github.com/ScriptonBasestar/dva/skills"
)

func TestLegacyAbsentRuntimeUninstallDoesNotCreateClaims(t *testing.T) {
	t.Parallel()
	options := testOptions(t, ScopeProject, RuntimeCodex)
	installed, err := Install(options)
	if err != nil {
		t.Fatal(err)
	}
	target := destination{path: installed.Destinations[0].Destination, runtimes: options.Runtimes}
	receiptFile := receiptPath(options.StateRoot, target.path)
	record, found, err := readReceipt(receiptFile)
	if err != nil || !found {
		t.Fatalf("read current receipt = (%v, %t, %v)", record, found, err)
	}
	record.Schema, record.Format = 1, ""
	if err := writeReceipt(receiptFile, record); err != nil {
		t.Fatal(err)
	}
	claims, err := projectedClaims(target, options.Scope, record.Runtimes, skillBundle{files: record.Files}, skillclaim.StateActive, "legacy-absent")
	if err != nil {
		t.Fatal(err)
	}
	for _, claim := range claims {
		if err := os.Remove(skillclaim.Path(filepath.Dir(options.StateRoot), claim.Destination)); err != nil {
			t.Fatal(err)
		}
	}
	absent := options
	absent.Runtimes = []Runtime{RuntimeAntigravity}
	result, err := Uninstall(absent)
	if err != nil {
		t.Fatal(err)
	}
	if result.Destinations[0].Status != "not-installed" {
		t.Fatalf("absent runtime uninstall = %s", result.Destinations[0].Status)
	}
	for _, claim := range claims {
		if _, found, err := skillclaim.Read(filepath.Dir(options.StateRoot), claim.Destination); err != nil || found {
			t.Fatalf("absent runtime created claim = (%t, %v)", found, err)
		}
	}
	status, err := Status(options)
	if err != nil || status.Destinations[0].Status != "legacy-unclaimed" {
		t.Fatalf("legacy owner status = %#v, %v", status, err)
	}
}

func TestUpToDateInstallStillAcquiresClaimLocks(t *testing.T) {
	t.Parallel()
	options := testOptions(t, ScopeProject, RuntimeCodex)
	installed, err := Install(options)
	if err != nil {
		t.Fatal(err)
	}
	target := destination{path: installed.Destinations[0].Destination, runtimes: options.Runtimes}
	bundle, err := bundleFor(target)
	if err != nil {
		t.Fatal(err)
	}
	paths, err := claimDestinations(target, bundle)
	if err != nil {
		t.Fatal(err)
	}
	locks, err := skillclaim.AcquireLocks(filepath.Dir(options.StateRoot), paths)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = locks.Release() }()
	if _, err := Install(options); err == nil || !strings.Contains(err.Error(), "claim mutation lock exists") {
		t.Fatalf("up-to-date install bypassed claim locks: %v", err)
	}
}

func TestAgentMeshKeepsForeignNamespaceFilesAndDetectsDVAFileCollision(t *testing.T) {
	t.Parallel()
	options := testOptions(t, ScopeUser, RuntimeAgentMesh)
	_, destinations, err := resolve(options)
	if err != nil {
		t.Fatal(err)
	}
	destination := destinations[0].path
	if err := os.MkdirAll(destination, 0o755); err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(destination, "other.md")
	if err := os.WriteFile(foreign, []byte("foreign"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(options); err != nil {
		t.Fatal(err)
	}
	if _, err := Uninstall(options); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(foreign); err != nil {
		t.Fatalf("uninstall removed foreign Agent Mesh file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(destination, "dva.md"), []byte("foreign"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(options); err == nil {
		t.Fatal("Agent Mesh DVA filename collision was overwritten")
	}
}

func TestMixedAgentMeshInstallPreflightsBeforeMutation(t *testing.T) {
	t.Parallel()
	options := testOptions(t, ScopeProject, RuntimeAgentMesh, RuntimeClaudeCode)
	_, destinations, err := resolve(options)
	if err != nil {
		t.Fatal(err)
	}
	last := destinations[len(destinations)-1]
	if err := os.MkdirAll(filepath.Join(last.path, "dva"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(options); err == nil {
		t.Fatal("mixed install ignored a later collision")
	}
	for _, target := range destinations[:len(destinations)-1] {
		if _, err := os.Stat(filepath.Join(target.path, "dva.md")); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("earlier destination mutated before collision: %v", err)
		}
	}
}

func TestProjectDeduplicatesSharedAgentsDestination(t *testing.T) {
	t.Parallel()
	options := testOptions(t, ScopeProject, RuntimeCodex, RuntimeAntigravity)
	_, destinations, err := resolve(options)
	if err != nil {
		t.Fatal(err)
	}
	if len(destinations) != 1 {
		t.Fatalf("destinations = %d, want 1", len(destinations))
	}
	if got, want := destinations[0].runtimes, []Runtime{RuntimeAntigravity, RuntimeCodex}; !sameRuntimes(got, want) {
		t.Fatalf("runtimes = %v, want %v", got, want)
	}
}

func TestRiskySkillOperationsRequireExplicitRuntime(t *testing.T) {
	t.Parallel()
	for _, options := range []Options{
		{Scope: ScopeUser, HomeDir: t.TempDir(), ProjectRoot: t.TempDir(), StateRoot: filepath.Join(t.TempDir(), "dva"), Takeover: true},
		{Scope: ScopeUser, HomeDir: t.TempDir(), ProjectRoot: t.TempDir(), StateRoot: filepath.Join(t.TempDir(), "dva"), RestoreTakeoverBackup: true},
	} {
		if _, _, err := resolve(options); err == nil {
			t.Fatal("risky skill operation accepted implicit all-runtime selection")
		}
	}
}

func TestTakeoverDryRunDoesNotCreateClaimReceiptOrBackup(t *testing.T) {
	t.Parallel()
	options := testOptions(t, ScopeUser, RuntimeCodex)
	_, destinations, err := resolve(options)
	if err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(destinations[0].path, "dva")
	if err := os.MkdirAll(foreign, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(foreign, "marker")
	if err := os.WriteFile(marker, []byte("foreign"), 0o640); err != nil {
		t.Fatal(err)
	}
	options.Takeover, options.DryRun = true, true
	result, err := Install(options)
	if err != nil {
		t.Fatal(err)
	}
	if result.Destinations[0].BackupStatus != "would-backup" {
		t.Fatalf("dry-run backup status = %q", result.Destinations[0].BackupStatus)
	}
	contents, err := os.ReadFile(marker)
	if err != nil || string(contents) != "foreign" {
		t.Fatalf("dry-run changed foreign marker: %q, %v", contents, err)
	}
	if _, err := os.Stat(options.StateRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("dry-run created DVA state: %v", err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(options.StateRoot), "agent-skills")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("dry-run created neutral claims: %v", err)
	}
}

func TestEmbeddedSkillsContainCanonicalFiles(t *testing.T) {
	t.Parallel()
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test source")
	}
	repoSkills := filepath.Join(filepath.Dir(sourceFile), "..", "..", "skills")
	var embeddedPaths []string
	if err := fs.WalkDir(bundled.Files, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || path == "embed.go" {
			return err
		}
		embeddedPaths = append(embeddedPaths, path)
		embedded, err := fs.ReadFile(bundled.Files, path)
		if err != nil {
			return err
		}
		disk, err := os.ReadFile(filepath.Join(repoSkills, filepath.FromSlash(path)))
		if err != nil {
			return err
		}
		if !bytes.Equal(embedded, disk) {
			t.Errorf("embedded %s differs from canonical disk source", path)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	var diskPaths []string
	for _, name := range bundled.Names {
		if err := filepath.WalkDir(filepath.Join(repoSkills, name), func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return err
			}
			relative, err := filepath.Rel(repoSkills, path)
			if err != nil {
				return err
			}
			diskPaths = append(diskPaths, filepath.ToSlash(relative))
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	sort.Strings(embeddedPaths)
	sort.Strings(diskPaths)
	if !sameStrings(embeddedPaths, diskPaths) {
		t.Fatalf("embedded file inventory differs\nembedded: %v\ndisk: %v", embeddedPaths, diskPaths)
	}
}

func TestInstallStatusAndUninstallSharedDestination(t *testing.T) {
	t.Parallel()
	options := testOptions(t, ScopeProject, RuntimeCodex, RuntimeAntigravity)
	result, err := Install(options)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Destinations) != 1 || result.Destinations[0].Status != "installed" {
		t.Fatalf("install result = %#v", result)
	}
	destination := result.Destinations[0].Destination
	if _, err := os.Stat(filepath.Join(destination, "dva", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(destination, "dva-config", "references", "diagnosis.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(destination, "dva-ci", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	result, err = Install(options)
	if err != nil {
		t.Fatal(err)
	}
	if result.Destinations[0].Status != "up-to-date" {
		t.Fatalf("second install = %s", result.Destinations[0].Status)
	}
	result, err = Status(options)
	if err != nil {
		t.Fatal(err)
	}
	if result.Destinations[0].Status != "installed" {
		t.Fatalf("status = %s", result.Destinations[0].Status)
	}
	assertClaimConsumers(t, options, destination, []string{"antigravity", "codex"})

	codexOnly := options
	codexOnly.Runtimes = []Runtime{RuntimeCodex}
	result, err = Uninstall(codexOnly)
	if err != nil {
		t.Fatal(err)
	}
	if result.Destinations[0].Status != "unlinked" {
		t.Fatalf("partial uninstall = %s", result.Destinations[0].Status)
	}
	if _, err := os.Stat(filepath.Join(destination, "dva")); err != nil {
		t.Fatalf("shared skill was removed: %v", err)
	}
	assertClaimConsumers(t, options, destination, []string{"antigravity"})
	antigravityOnly := options
	antigravityOnly.Runtimes = []Runtime{RuntimeAntigravity}
	result, err = Uninstall(antigravityOnly)
	if err != nil {
		t.Fatal(err)
	}
	if result.Destinations[0].Status != "uninstalled" {
		t.Fatalf("last uninstall = %s", result.Destinations[0].Status)
	}
	if _, err := os.Stat(filepath.Join(destination, "dva")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("skill remains after uninstall: %v", err)
	}
	if _, err := os.Stat(filepath.Join(destination, "dva-ci")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("CI skill remains after uninstall: %v", err)
	}
}

func TestSharedDestinationReportsEachRuntimeState(t *testing.T) {
	t.Parallel()
	options := testOptions(t, ScopeProject, RuntimeCodex)
	if _, err := Install(options); err != nil {
		t.Fatal(err)
	}

	both := options
	both.Runtimes = []Runtime{RuntimeCodex, RuntimeAntigravity}
	status, err := Status(both)
	if err != nil {
		t.Fatal(err)
	}
	entry := status.Destinations[0]
	if entry.Status != "partial" {
		t.Fatalf("aggregate status = %s, want partial", entry.Status)
	}
	assertRuntimeStatuses(t, entry.RuntimeStatuses, map[Runtime]string{
		RuntimeCodex: "installed", RuntimeAntigravity: "absent",
	})

	antigravityOnly := options
	antigravityOnly.Runtimes = []Runtime{RuntimeAntigravity}
	result, err := Uninstall(antigravityOnly)
	if err != nil {
		t.Fatal(err)
	}
	if result.Destinations[0].Status != "not-installed" {
		t.Fatalf("absent runtime uninstall = %s", result.Destinations[0].Status)
	}
	status, err = Status(options)
	if err != nil {
		t.Fatal(err)
	}
	if status.Destinations[0].Status != "installed" {
		t.Fatalf("codex changed by absent runtime uninstall: %s", status.Destinations[0].Status)
	}
}
