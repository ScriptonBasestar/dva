//go:build darwin || linux

package secretpush

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"syscall"
)

// lockTarget uses an advisory kernel lock. The inode is deliberately retained:
// a killed process releases its lock, while an O_EXCL marker would strand every
// later invocation behind a permanent false busy result.
func lockTarget(state, repository string) (func(), error) {
	digest := sha256.Sum256([]byte(repository))
	path := filepath.Join(state, ".lock-"+hex.EncodeToString(digest[:]))
	info, err := os.Lstat(path)
	if err == nil && (!info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o600) {
		return nil, codeError("state_directory_failed")
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, codeError("state_directory_failed")
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, codeError("state_directory_failed")
	}
	info, err = f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
		_ = f.Close()
		return nil, codeError("state_directory_failed")
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, codeError("secret_push_busy")
		}
		return nil, codeError("state_directory_failed")
	}
	return func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN); _ = f.Close() }, nil
}
