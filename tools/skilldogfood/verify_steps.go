package main

import (
	"errors"
	"fmt"
	bundled "github.com/ScriptonBasestar/dva/skills"
	"os"
	"path/filepath"
	"reflect"
)

func verifyFlowDryRun(inv invocation, flowRoot string) (err error) {
	before, err := snapshotGitTreeState(flowRoot)
	if err != nil {
		return err
	}
	beforeSkills, err := snapshotRuntimePaths(flowRoot)
	if err != nil {
		return fmt.Errorf("snapshot runtime paths before dry-run: %w", err)
	}
	stateDir, err := os.MkdirTemp("", "dva-skill-dogfood-env-")
	if err != nil {
		return fmt.Errorf("create dry-run state directory: %w", err)
	}
	defer func() { err = errors.Join(err, removeAll("clean dry-run environment "+stateDir, stateDir)) }()
	dryRoots := map[string]string{
		"HOME": filepath.Join(stateDir, "home"), "XDG_CONFIG_HOME": filepath.Join(stateDir, "config"),
		"XDG_DATA_HOME": filepath.Join(stateDir, "data"), "XDG_CACHE_HOME": filepath.Join(stateDir, "cache"), "XDG_STATE_HOME": filepath.Join(stateDir, "state"),
	}
	for _, root := range dryRoots {
		if err := os.MkdirAll(root, 0o755); err != nil {
			return err
		}
	}
	dryRun := invocation{binary: inv.binary, env: withEnvironment(inv.env, dryRoots)}
	result, err := dryRun.json(flowRoot, "skill", "install", "--scope", "project", "--runtime", allSkillRuntimes, "--dry-run")
	if err != nil {
		return fmt.Errorf("run project-scope dry-run: %w", err)
	}
	if result.Operation != "install" || !result.DryRun || result.Scope != "project" {
		return fmt.Errorf("unexpected dry-run response: operation=%q dry_run=%t scope=%q", result.Operation, result.DryRun, result.Scope)
	}
	if err := requireEnvelope(result, "install", true); err != nil {
		return fmt.Errorf("project-scope dry-run envelope: %w", err)
	}
	if err := requireDestinations(flowRoot, result, "would-install"); err != nil {
		return fmt.Errorf("project-scope dry-run result: %w", err)
	}
	after, err := snapshotGitTreeState(flowRoot)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(before, after) {
		return fmt.Errorf("project-scope dry-run changed Git-visible state:\nbefore:\n%safter:\n%s", formatGitTreeState(before), formatGitTreeState(after))
	}
	afterSkills, err := snapshotRuntimePaths(flowRoot)
	if err != nil {
		return fmt.Errorf("snapshot runtime paths after dry-run: %w", err)
	}
	if !reflect.DeepEqual(beforeSkills, afterSkills) {
		return fmt.Errorf("project-scope dry-run changed runtime paths:\nbefore:\n%safter:\n%s", formatSnapshot(beforeSkills), formatSnapshot(afterSkills))
	}
	for name, root := range dryRoots {
		if err := requireEmptyDirectory(root); err != nil {
			return fmt.Errorf("project-scope dry-run wrote %s: %w", name, err)
		}
	}
	return nil
}

func verifyFixtureRoundTrip(inv invocation, project, stateRoot string) error {
	installed, err := inv.json(project, "skill", "install", "--scope", "project", "--runtime", allSkillRuntimes)
	if err != nil {
		return fmt.Errorf("install isolated fixture: %w", err)
	}
	if err := requireEnvelope(installed, "install", false); err != nil {
		return fmt.Errorf("install envelope: %w", err)
	}
	if err := requireDestinations(project, installed, "installed"); err != nil {
		return fmt.Errorf("install result: %w", err)
	}

	status, err := inv.json(project, "skill", "status", "--scope", "project", "--runtime", allSkillRuntimes)
	if err != nil {
		return fmt.Errorf("check installed fixture status: %w", err)
	}
	if err := requireEnvelope(status, "status", false); err != nil {
		return fmt.Errorf("status envelope: %w", err)
	}
	if err := requireDestinations(project, status, "installed"); err != nil {
		return fmt.Errorf("installed status: %w", err)
	}
	if err := verifyOwnedArtifacts(project, stateRoot); err != nil {
		return fmt.Errorf("installed artifacts: %w", err)
	}
	sharedBefore, sharedFilesBefore, err := sharedReceiptSnapshot(project, stateRoot)
	if err != nil {
		return fmt.Errorf("snapshot shared receipt before Codex unlink: %w", err)
	}

	unlinked, err := inv.json(project, "skill", "uninstall", "--scope", "project", "--runtime", "codex")
	if err != nil {
		return fmt.Errorf("uninstall Codex from shared destination: %w", err)
	}
	if err := requireEnvelope(unlinked, "uninstall", false); err != nil {
		return fmt.Errorf("codex uninstall envelope: %w", err)
	}
	if err := requireOnlyDestination(project, unlinked, ".agents/skills", "unlinked"); err != nil {
		return fmt.Errorf("codex-only uninstall result: %w", err)
	}
	if err := requireRuntimeStatuses(unlinked.Results[0], map[string]string{"codex": "unlinked"}); err != nil {
		return fmt.Errorf("codex-only uninstall membership: %w", err)
	}

	partial, err := inv.json(project, "skill", "status", "--scope", "project", "--runtime", "codex,antigravity")
	if err != nil {
		return fmt.Errorf("check shared destination after Codex uninstall: %w", err)
	}
	if err := requireEnvelope(partial, "status", false); err != nil {
		return fmt.Errorf("shared status envelope: %w", err)
	}
	if err := requireOnlyDestination(project, partial, ".agents/skills", "partial"); err != nil {
		return fmt.Errorf("shared destination status: %w", err)
	}
	if err := requireRuntimeStatuses(partial.Results[0], map[string]string{"codex": "absent", "antigravity": "installed"}); err != nil {
		return fmt.Errorf("shared destination membership: %w", err)
	}
	if err := verifySharedUnlink(project, stateRoot, sharedBefore, sharedFilesBefore); err != nil {
		return fmt.Errorf("shared runtime unlink artifacts: %w", err)
	}

	removed, err := inv.json(project, "skill", "uninstall", "--scope", "project", "--runtime", allSkillRuntimes)
	if err != nil {
		return fmt.Errorf("uninstall remaining fixture skills: %w", err)
	}
	if err := requireEnvelope(removed, "uninstall", false); err != nil {
		return fmt.Errorf("remaining uninstall envelope: %w", err)
	}
	if err := requireDestinationStatusSet(project, removed, map[string]string{
		".agent-mesh/skills/dva": "uninstalled", ".agents/skills": "uninstalled", ".claude/skills": "uninstalled", ".grok/skills": "uninstalled", ".opencode/skills": "uninstalled",
	}); err != nil {
		return fmt.Errorf("remaining uninstall result: %w", err)
	}

	absent, err := inv.json(project, "skill", "status", "--scope", "project", "--runtime", allSkillRuntimes)
	if err != nil {
		return fmt.Errorf("check removed fixture status: %w", err)
	}
	if err := requireEnvelope(absent, "status", false); err != nil {
		return fmt.Errorf("absent status envelope: %w", err)
	}
	if err := requireDestinations(project, absent, "absent"); err != nil {
		return fmt.Errorf("final status: %w", err)
	}
	if err := verifyArtifactsAbsent(project, stateRoot); err != nil {
		return fmt.Errorf("final artifacts: %w", err)
	}
	return nil
}

func verifyTakeoverLifecycle(inv invocation, project string) error {
	if err := os.MkdirAll(project, 0o755); err != nil {
		return err
	}
	if _, err := inv.json(project, "skill", "install", "--scope", "project", "--takeover"); err == nil {
		return errors.New("takeover without an explicit runtime unexpectedly succeeded")
	}
	writeForeign := func() error {
		for _, name := range bundled.Names {
			root := filepath.Join(project, ".grok", "skills", name)
			if err := os.MkdirAll(filepath.Join(root, "empty"), 0o750); err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(root, "original.txt"), []byte("foreign-"+name), 0o640); err != nil {
				return err
			}
		}
		return nil
	}
	verifyForeign := func() error {
		for _, name := range bundled.Names {
			root := filepath.Join(project, ".grok", "skills", name)
			contents, err := os.ReadFile(filepath.Join(root, "original.txt"))
			if err != nil || string(contents) != "foreign-"+name {
				return fmt.Errorf("restored %s bytes = %q: %w", name, contents, err)
			}
			info, err := os.Stat(filepath.Join(root, "empty"))
			if err != nil || !info.IsDir() {
				return fmt.Errorf("restored %s empty directory: %w", name, err)
			}
		}
		return nil
	}
	installTakeover := func() (destinationResult, error) {
		result, err := inv.json(project, "skill", "install", "--scope", "project", "--runtime", "grok", "--takeover")
		if err != nil {
			return destinationResult{}, err
		}
		if err := requireOnlyDestination(project, result, ".grok/skills", "installed"); err != nil {
			return destinationResult{}, err
		}
		if result.Results[0].BackupStatus != "available" || result.Results[0].TakeoverBackup == "" {
			return destinationResult{}, fmt.Errorf("takeover did not report an available backup: %#v", result.Results[0])
		}
		return result.Results[0], nil
	}

	if err := writeForeign(); err != nil {
		return err
	}
	if _, err := installTakeover(); err != nil {
		return fmt.Errorf("takeover for tombstone restore: %w", err)
	}
	removed, err := inv.json(project, "skill", "uninstall", "--scope", "project", "--runtime", "grok")
	if err != nil || len(removed.Results) != 1 || removed.Results[0].Status != "uninstalled" || removed.Results[0].BackupStatus != "available" {
		return fmt.Errorf("ordinary takeover uninstall = %#v: %w", removed, err)
	}
	tombstone, err := inv.json(project, "skill", "status", "--scope", "project", "--runtime", "grok")
	if err != nil || len(tombstone.Results) != 1 || tombstone.Results[0].Status != "backup-only" {
		return fmt.Errorf("backup-only status = %#v: %w", tombstone, err)
	}
	restored, err := inv.json(project, "skill", "uninstall", "--scope", "project", "--runtime", "grok", "--restore-takeover-backup")
	if err != nil || len(restored.Results) != 1 || restored.Results[0].Status != "restored-takeover" {
		return fmt.Errorf("tombstone restore = %#v: %w", restored, err)
	}
	if err := verifyForeign(); err != nil {
		return err
	}

	if _, err := installTakeover(); err != nil {
		return fmt.Errorf("takeover for active restore: %w", err)
	}
	restored, err = inv.json(project, "skill", "uninstall", "--scope", "project", "--runtime", "grok", "--restore-takeover-backup")
	if err != nil || len(restored.Results) != 1 || restored.Results[0].Status != "restored-takeover" {
		return fmt.Errorf("active restore = %#v: %w", restored, err)
	}
	if err := verifyForeign(); err != nil {
		return err
	}

	installed, err := installTakeover()
	if err != nil {
		return fmt.Errorf("takeover for corrupt-backup refusal: %w", err)
	}
	if err := os.WriteFile(filepath.Join(installed.TakeoverBackup, "original.txt"), []byte("corrupt"), 0o640); err != nil {
		return err
	}
	if _, err := inv.json(project, "skill", "uninstall", "--scope", "project", "--runtime", "grok", "--restore-takeover-backup"); err == nil {
		return errors.New("corrupt takeover backup unexpectedly restored")
	}
	status, err := inv.json(project, "skill", "status", "--scope", "project", "--runtime", "grok")
	if err != nil || len(status.Results) != 1 || status.Results[0].Status != "installed" || status.Results[0].BackupStatus != "corrupt" {
		return fmt.Errorf("corrupt takeover status = %#v: %w", status, err)
	}
	return nil
}
