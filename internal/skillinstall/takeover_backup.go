package skillinstall

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
)

const (
	backupKindDirectory = "directory"
	backupKindFile      = "file"
)

func createTakeoverBackups(stateRoot string, target destination, bundle skillBundle) ([]takeoverBackup, func() error, func() error, func() error, string, error) {
	var names []string
	for _, name := range skillNames(bundle) {
		if _, err := os.Lstat(claimDestination(target, name)); err == nil {
			names = append(names, name)
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, nil, nil, nil, "", err
		}
	}
	if len(names) == 0 {
		noop := func() error { return nil }
		return nil, noop, noop, noop, "", nil
	}

	// Rename each live foreign entry out of the destination before copying it. A
	// rename is the mutation boundary: the durable backup is therefore made from
	// the exact object removed from the runtime, not from a concurrently changing
	// live tree that would later be overwritten.
	captureStage, err := os.MkdirTemp(target.path, ".dva-takeover-capture-")
	if err != nil {
		return nil, nil, nil, nil, "", err
	}
	var captured []string
	rollbackOriginals := func() error {
		var first error
		for _, name := range slices.Backward(captured) {
			source := filepath.Join(captureStage, name)
			destination := claimDestination(target, name)
			if _, err := os.Lstat(destination); err == nil {
				if first == nil {
					first = fmt.Errorf("cannot restore captured takeover entry because %s now exists", destination)
				}
				continue
			} else if !errors.Is(err, os.ErrNotExist) {
				if first == nil {
					first = err
				}
				continue
			}
			if err := os.Rename(source, destination); err != nil && first == nil {
				first = err
			}
		}
		if err := syncDirectory(captureStage); err != nil && first == nil {
			first = err
		}
		if err := syncDirectory(target.path); err != nil && first == nil {
			first = err
		}
		if first == nil {
			if err := os.RemoveAll(captureStage); err != nil {
				first = err
			}
		}
		if err := syncDirectory(target.path); err != nil && first == nil {
			first = err
		}
		return first
	}
	for _, name := range names {
		if err := os.Rename(claimDestination(target, name), filepath.Join(captureStage, name)); err != nil {
			rollbackErr := rollbackOriginals()
			return nil, nil, nil, nil, captureStage, fmt.Errorf("capture takeover skill %s: %w (rollback: %v; recovery stage: %s)", name, err, rollbackErr, captureStage)
		}
		captured = append(captured, name)
	}
	if err := syncDirectory(captureStage); err != nil {
		rollbackErr := rollbackOriginals()
		return nil, nil, nil, nil, captureStage, fmt.Errorf("sync takeover capture stage: %w (rollback: %v; recovery stage: %s)", err, rollbackErr, captureStage)
	}
	if err := syncDirectory(target.path); err != nil {
		rollbackErr := rollbackOriginals()
		return nil, nil, nil, nil, captureStage, fmt.Errorf("sync captured takeover skills: %w (rollback: %v; recovery stage: %s)", err, rollbackErr, captureStage)
	}
	finalizeOriginals := func() error {
		if err := os.RemoveAll(captureStage); err != nil {
			return err
		}
		return syncDirectory(target.path)
	}

	backupID, err := newBackupID()
	if err != nil {
		rollbackErr := rollbackOriginals()
		return nil, nil, nil, nil, captureStage, fmt.Errorf("create takeover backup id: %w (rollback: %v; recovery stage: %s)", err, rollbackErr, captureStage)
	}
	base := takeoverDestinationRoot(stateRoot, target.path)
	if err := mkdirAllSynced(base, 0o700); err != nil {
		rollbackErr := rollbackOriginals()
		return nil, nil, nil, nil, captureStage, fmt.Errorf("create takeover backup root: %w (rollback: %v; recovery stage: %s)", err, rollbackErr, captureStage)
	}
	stage, err := os.MkdirTemp(base, ".stage-")
	if err != nil {
		rollbackErr := rollbackOriginals()
		return nil, nil, nil, nil, captureStage, fmt.Errorf("create takeover backup stage: %w (rollback: %v; recovery stage: %s)", err, rollbackErr, captureStage)
	}
	cleanupStage := true
	defer func() {
		if cleanupStage {
			_ = os.RemoveAll(stage)
		}
	}()

	records := make([]takeoverBackup, 0, len(names))
	for _, name := range names {
		kind, entries, err := copyBackupTree(filepath.Join(captureStage, name), filepath.Join(stage, name))
		if err != nil {
			rollbackErr := rollbackOriginals()
			return nil, nil, nil, nil, captureStage, fmt.Errorf("back up captured takeover skill %s: %w (rollback: %v; recovery stage: %s)", name, err, rollbackErr, captureStage)
		}
		records = append(records, takeoverBackup{
			Skill: name, BackupID: backupID, Kind: kind,
			ManifestDigest: backupManifestDigest(entries), Entries: entries,
		})
	}
	final := filepath.Join(base, backupID)
	if err := os.Rename(stage, final); err != nil {
		rollbackErr := rollbackOriginals()
		return nil, nil, nil, nil, captureStage, fmt.Errorf("publish takeover backup: %w (rollback: %v; recovery stage: %s)", err, rollbackErr, captureStage)
	}
	cleanupStage = false
	if err := syncDirectory(base); err != nil {
		rollbackErr := rollbackOriginals()
		if rollbackErr != nil {
			return nil, nil, nil, nil, final, fmt.Errorf("sync takeover backup: %w (original rollback: %v; recovery artifacts: %s, %s)", err, rollbackErr, captureStage, final)
		}
		cleanupErr := os.RemoveAll(final)
		if cleanupErr == nil {
			cleanupErr = syncDirectory(base)
		}
		return nil, nil, nil, nil, final, fmt.Errorf("sync takeover backup: %w (original rollback succeeded; backup cleanup: %v; recovery artifact if cleanup failed: %s)", err, cleanupErr, final)
	}
	cleanup := func() error {
		if err := os.RemoveAll(final); err != nil {
			return err
		}
		return syncDirectory(base)
	}
	return records, rollbackOriginals, finalizeOriginals, cleanup, final, nil
}

func newBackupID() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func takeoverDestinationRoot(stateRoot, destination string) string {
	digest := sha256.Sum256([]byte(destination))
	return filepath.Join(stateRoot, "skill-takeovers", hex.EncodeToString(digest[:]))
}

func takeoverBackupPath(stateRoot, destination string, record takeoverBackup) (string, error) {
	if !validBackupID(record.BackupID) || !validTakeoverSkill(record.Skill) {
		return "", errors.New("takeover receipt contains an invalid backup identity")
	}
	return filepath.Join(takeoverDestinationRoot(stateRoot, destination), record.BackupID, record.Skill), nil
}

func validBackupID(value string) bool {
	raw, err := hex.DecodeString(value)
	return err == nil && len(raw) == 16 && value == strings.ToLower(value)
}

func validTakeoverSkill(value string) bool {
	if base, ok := strings.CutSuffix(value, ".md"); ok {
		return isBundledSkillName(base)
	}
	return isBundledSkillName(value)
}

func copyBackupTree(source, destination string) (string, []backupEntry, error) {
	info, err := os.Lstat(source)
	if err != nil {
		return "", nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || (!info.Mode().IsRegular() && !info.IsDir()) {
		return "", nil, fmt.Errorf("%s is not a regular file or directory", source)
	}
	if info.Mode().IsRegular() {
		entry, err := copyBackupFile(source, destination, ".", info.Mode().Perm())
		return backupKindFile, []backupEntry{entry}, err
	}

	if err := os.MkdirAll(destination, 0o700); err != nil {
		return "", nil, err
	}
	var entries []backupEntry
	var directories []struct {
		path string
		mode fs.FileMode
	}
	err = filepath.WalkDir(source, func(path string, item fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := item.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || (!info.IsDir() && !info.Mode().IsRegular()) {
			return fmt.Errorf("%s is not a regular file or directory", path)
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		target := filepath.Join(destination, filepath.FromSlash(relative))
		if info.IsDir() {
			if err := os.MkdirAll(target, 0o700); err != nil {
				return err
			}
			entries = append(entries, backupEntry{Path: relative, Kind: backupKindDirectory, Mode: uint32(info.Mode().Perm())})
			directories = append(directories, struct {
				path string
				mode fs.FileMode
			}{target, info.Mode().Perm()})
			return nil
		}
		entry, err := copyBackupFile(path, target, relative, info.Mode().Perm())
		if err == nil {
			entries = append(entries, entry)
		}
		return err
	})
	if err != nil {
		return "", nil, err
	}
	for _, directory := range slices.Backward(directories) {
		if err := os.Chmod(directory.path, directory.mode); err != nil {
			return "", nil, err
		}
		if err := syncDirectory(directory.path); err != nil {
			return "", nil, err
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return backupKindDirectory, entries, nil
}

func copyBackupFile(source, destination, relative string, mode fs.FileMode) (backupEntry, error) {
	contents, err := os.ReadFile(source)
	if err != nil {
		return backupEntry{}, err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		return backupEntry{}, err
	}
	file, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return backupEntry{}, err
	}
	_, writeErr := file.Write(contents)
	chmodErr := file.Chmod(mode)
	syncErr := file.Sync()
	closeErr := file.Close()
	for _, candidate := range []error{writeErr, chmodErr, syncErr, closeErr} {
		if candidate != nil {
			return backupEntry{}, candidate
		}
	}
	digest := sha256.Sum256(contents)
	return backupEntry{Path: relative, Kind: backupKindFile, Mode: uint32(mode), SHA: hex.EncodeToString(digest[:])}, nil
}

func backupManifestDigest(entries []backupEntry) string {
	hash := sha256.New()
	for _, entry := range entries {
		_, _ = hash.Write([]byte(entry.Path))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write([]byte(entry.Kind))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write([]byte(strconv.FormatUint(uint64(entry.Mode), 8)))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write([]byte(entry.SHA))
		_, _ = hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))
}
