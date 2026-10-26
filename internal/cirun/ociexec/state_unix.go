//go:build darwin || linux

package ociexec

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// secureStateDirectory creates a private state directory without relaxing an
// existing directory's permissions.  Callers allocate children only after it
// succeeds.
func secureStateDirectory(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("OCI state directory is empty")
	}
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("OCI state directory must be absolute")
	}
	if err := os.Mkdir(path, 0o700); err != nil && !os.IsExist(err) {
		return "", err
	}
	canonical, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	for current := canonical; ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("OCI state ancestor is a symlink: %s", current)
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			return "", fmt.Errorf("OCI state ownership unavailable")
		}
		mode := info.Mode().Perm()
		sticky := info.Mode()&os.ModeSticky != 0
		if stat.Uid != uint32(os.Getuid()) && stat.Uid != 0 {
			return "", fmt.Errorf("OCI state ancestor is not owned by current user or root: %s", current)
		}
		if mode&0o022 != 0 && (!sticky || (stat.Uid != 0 && stat.Uid != uint32(os.Getuid()))) {
			return "", fmt.Errorf("OCI state ancestor is writable by group/world: %s", current)
		}
		if current == "/" {
			break
		}
	}
	info, err := os.Stat(canonical)
	if err != nil {
		return "", err
	}
	stat := info.Sys().(*syscall.Stat_t)
	if stat.Uid != uint32(os.Getuid()) || info.Mode().Perm() != 0o700 {
		return "", fmt.Errorf("OCI state directory must be current-user owned and mode 0700")
	}
	return canonical, nil
}
