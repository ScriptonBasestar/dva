// Package statedir resolves DVA's per-user state root under the ScriptonBasestar
// group directory (<state>/sb/dva) and migrates the legacy <state>/dva layout.
package statedir

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
)

const (
	// Group is the shared ScriptonBasestar tool group directory name.
	Group = "sb"
	// Tool is DVA's directory name inside the group.
	Tool = "dva"
)

// XDGRoot returns $XDG_STATE_HOME/sb/dva, or ~/.local/state/sb/dva when
// XDG_STATE_HOME is unset, migrating a legacy <state>/dva directory first.
func XDGRoot() (string, error) {
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".local", "state")
	}
	return RootUnder(base)
}

// RootUnderHome is XDGRoot with an explicit home directory used when
// XDG_STATE_HOME is unset, for callers that accept a home override.
func RootUnderHome(home string) (string, error) {
	if base := os.Getenv("XDG_STATE_HOME"); base != "" {
		return RootUnder(base)
	}
	return RootUnder(filepath.Join(home, ".local", "state"))
}

// HostRoot returns <uid home>/.local/state/sb/dva. It ignores HOME and
// XDG_STATE_HOME so every session of one UID shares the same host-wide locks.
func HostRoot() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return RootUnder(filepath.Join(u.HomeDir, ".local", "state"))
}

// RootUnder returns <base>/sb/dva after migrating a legacy <base>/dva.
func RootUnder(base string) (string, error) {
	return GroupDir(base, Tool)
}

// GroupDir returns <base>/sb/<name> after migrating a legacy <base>/<name>.
func GroupDir(base, name string) (string, error) {
	base, err := filepath.Abs(base)
	if err != nil {
		return "", err
	}
	if err := migrateLegacy(base, name); err != nil {
		return "", err
	}
	return filepath.Join(base, Group, name), nil
}

// migrateLegacy moves <base>/<name> to <base>/sb/<name> once and leaves a
// relative symlink at the legacy path so older binaries keep sharing the same
// data. An existing target is never merged or overwritten.
func migrateLegacy(base, name string) error {
	legacy := filepath.Join(base, name)
	target := filepath.Join(base, Group, name)
	info, err := os.Lstat(legacy)
	if errors.Is(err, fs.ErrNotExist) || (err == nil && info.Mode()&fs.ModeSymlink != 0) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect legacy state %s: %w", legacy, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("legacy state %s is not a directory", legacy)
	}
	if _, err := os.Lstat(target); err == nil {
		return fmt.Errorf("state migration blocked: both %s and %s exist; merge them manually", legacy, target)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("inspect state %s: %w", target, err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return fmt.Errorf("create state group directory: %w", err)
	}
	if err := os.Rename(legacy, target); err != nil {
		// A concurrent dva process may have migrated first.
		if _, statErr := os.Lstat(target); statErr != nil {
			return fmt.Errorf("migrate state %s -> %s: %w", legacy, target, err)
		}
	}
	if err := os.Symlink(filepath.Join(Group, name), legacy); err != nil && !errors.Is(err, fs.ErrExist) {
		return fmt.Errorf("link legacy state path %s: %w", legacy, err)
	}
	return nil
}
