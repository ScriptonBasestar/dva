package skillinstall

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
)

func verifyTakeoverBackups(stateRoot string, record receipt) (string, string) {
	if len(record.Takeovers) == 0 {
		return "", ""
	}
	first := ""
	groups := takeoverBackupGroups(record)
	ids := make([]string, 0, len(groups))
	for backupID := range groups {
		ids = append(ids, backupID)
	}
	sort.Strings(ids)
	for _, backupID := range ids {
		if first == "" {
			first = filepath.Join(takeoverDestinationRoot(stateRoot, record.Destination), backupID, groups[backupID][0])
		}
		if err := verifyTakeoverBackupGroup(stateRoot, record, backupID, groups[backupID]); err != nil {
			return "corrupt", first
		}
	}
	return "available", first
}

func verifyTakeoverBackupInventory(stateRoot string, record receipt) error {
	for backupID, skills := range takeoverBackupGroups(record) {
		if err := verifyTakeoverBackupGroup(stateRoot, record, backupID, skills); err != nil {
			return err
		}
	}
	return nil
}

func takeoverBackupGroups(record receipt) map[string][]string {
	groups := map[string][]string{}
	for _, takeover := range record.Takeovers {
		groups[takeover.BackupID] = append(groups[takeover.BackupID], takeover.Skill)
	}
	for _, skills := range groups {
		sort.Strings(skills)
	}
	return groups
}

// verifyTakeoverBackupGroup validates exactly one backup ID. The receipt may
// legally retain multiple IDs, so a corrupt group must not make a distinct,
// verified group unsafe to list.
func verifyTakeoverBackupGroup(stateRoot string, record receipt, backupID string, skills []string) error {
	if !validBackupID(backupID) {
		return errors.New("takeover receipt contains an invalid backup identity")
	}
	base := takeoverDestinationRoot(stateRoot, record.Destination)
	if err := requireRegularDirectory(base); err != nil {
		return fmt.Errorf("takeover backup destination %s: %w", base, err)
	}
	root := filepath.Join(base, backupID)
	if err := requireRegularDirectory(root); err != nil {
		return fmt.Errorf("takeover backup %s: %w", root, err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	got := make([]string, len(entries))
	for index, entry := range entries {
		got[index] = entry.Name()
	}
	sort.Strings(got)
	want := append([]string(nil), skills...)
	sort.Strings(want)
	if !slices.Equal(got, want) {
		return fmt.Errorf("takeover backup %s inventory differs from receipt: got %v, want %v", root, got, want)
	}
	for _, takeover := range record.Takeovers {
		if takeover.BackupID != backupID {
			continue
		}
		path, err := takeoverBackupPath(stateRoot, record.Destination, takeover)
		if err != nil {
			return err
		}
		kind, entries, err := inspectBackupTree(path)
		if err != nil || kind != takeover.Kind || backupManifestDigest(entries) != takeover.ManifestDigest || !equalBackupEntries(entries, takeover.Entries) {
			if err == nil {
				err = errors.New("backup differs from receipt")
			}
			return err
		}
	}
	return nil
}

func requireRegularDirectory(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return errors.New("is not a regular directory")
	}
	return nil
}

func inspectBackupTree(root string) (string, []backupEntry, error) {
	info, err := os.Lstat(root)
	if err != nil {
		return "", nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || (!info.IsDir() && !info.Mode().IsRegular()) {
		return "", nil, errors.New("backup is not regular")
	}
	if info.Mode().IsRegular() {
		contents, err := os.ReadFile(root)
		if err != nil {
			return "", nil, err
		}
		digest := sha256.Sum256(contents)
		return backupKindFile, []backupEntry{{Path: ".", Kind: backupKindFile, Mode: uint32(info.Mode().Perm()), SHA: hex.EncodeToString(digest[:])}}, nil
	}
	var entries []backupEntry
	err = filepath.WalkDir(root, func(path string, item fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := item.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || (!info.IsDir() && !info.Mode().IsRegular()) {
			return errors.New("backup contains a non-regular entry")
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		entry := backupEntry{Path: relative, Mode: uint32(info.Mode().Perm())}
		if info.IsDir() {
			entry.Kind = backupKindDirectory
		} else {
			entry.Kind = backupKindFile
			contents, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			digest := sha256.Sum256(contents)
			entry.SHA = hex.EncodeToString(digest[:])
		}
		entries = append(entries, entry)
		return nil
	})
	if err != nil {
		return "", nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return backupKindDirectory, entries, nil
}

func replaceWithTakeoverBackups(stateRoot string, record receipt) (func() error, func() error, error) {
	if err := verifyTakeoverBackupInventory(stateRoot, record); err != nil {
		return nil, nil, err
	}
	stage, err := os.MkdirTemp(record.Destination, ".dva-takeover-restore-")
	if err != nil {
		return nil, nil, err
	}
	backups := make(map[string]takeoverBackup, len(record.Takeovers))
	for _, takeover := range record.Takeovers {
		path, err := takeoverBackupPath(stateRoot, record.Destination, takeover)
		if err != nil {
			_ = os.RemoveAll(stage)
			return nil, nil, err
		}
		kind, entries, err := copyBackupTree(path, filepath.Join(stage, takeover.Skill+".original"))
		if err != nil || kind != takeover.Kind || !equalBackupEntries(entries, takeover.Entries) {
			_ = os.RemoveAll(stage)
			if err == nil {
				err = errors.New("staged takeover backup differs from receipt")
			}
			return nil, nil, err
		}
		backups[takeover.Skill] = takeover
	}

	type move struct {
		final    string
		dva      string
		restored bool
	}
	var moves []move
	rollback := func() error {
		var first error
		for _, move := range slices.Backward(moves) {
			if move.restored {
				if err := os.RemoveAll(move.final); err != nil && first == nil {
					first = err
				}
			}
			if move.dva != "" {
				if err := os.Rename(move.dva, move.final); err != nil && first == nil {
					first = err
				}
			}
		}
		if first == nil {
			if err := os.RemoveAll(stage); err != nil {
				first = err
			}
		}
		if err := syncDirectory(record.Destination); err != nil && first == nil {
			first = err
		}
		return first
	}
	fail := func(cause error) (func() error, func() error, error) {
		if rollbackErr := rollback(); rollbackErr != nil {
			return nil, nil, fmt.Errorf("%w (restore rollback also failed: %v; recovery stage: %s)", cause, rollbackErr, stage)
		}
		return nil, nil, cause
	}
	for _, name := range skillNames(skillBundle{files: record.Files}) {
		final := filepath.Join(record.Destination, name)
		dva := filepath.Join(stage, name+".dva")
		if _, err := os.Lstat(final); err == nil {
			if err := os.Rename(final, dva); err != nil {
				return fail(err)
			}
			moves = append(moves, move{final: final, dva: dva})
		} else if !errors.Is(err, os.ErrNotExist) {
			return fail(err)
		} else {
			moves = append(moves, move{final: final})
		}
		if _, ok := backups[name]; ok {
			if err := os.Rename(filepath.Join(stage, name+".original"), final); err != nil {
				return fail(err)
			}
			moves[len(moves)-1].restored = true
		}
	}
	if err := syncDirectory(record.Destination); err != nil {
		return fail(err)
	}
	finalize := func() error {
		if err := os.RemoveAll(stage); err != nil {
			return err
		}
		for _, takeover := range record.Takeovers {
			path, err := takeoverBackupPath(stateRoot, record.Destination, takeover)
			if err != nil {
				return err
			}
			kind, entries, err := inspectBackupTree(path)
			if err != nil || kind != takeover.Kind || !equalBackupEntries(entries, takeover.Entries) {
				if err == nil {
					err = errors.New("takeover backup changed before cleanup")
				}
				return err
			}
		}
		for _, takeover := range record.Takeovers {
			path, err := takeoverBackupPath(stateRoot, record.Destination, takeover)
			if err != nil {
				return err
			}
			if err := os.RemoveAll(path); err != nil {
				return err
			}
		}
		seen := map[string]bool{}
		for _, takeover := range record.Takeovers {
			if seen[takeover.BackupID] {
				continue
			}
			seen[takeover.BackupID] = true
			root := filepath.Join(takeoverDestinationRoot(stateRoot, record.Destination), takeover.BackupID)
			if err := os.Remove(root); err != nil {
				return err
			}
		}
		return syncDirectory(takeoverDestinationRoot(stateRoot, record.Destination))
	}
	return rollback, finalize, nil
}

func equalBackupEntries(left, right []backupEntry) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
