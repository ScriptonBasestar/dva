//go:build darwin || linux || freebsd || netbsd || openbsd

package cimetrics

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
)

func ownedByUser(info os.FileInfo) bool {
	s, ok := info.Sys().(*syscall.Stat_t)
	return ok && s.Uid == uint32(os.Geteuid())
}

func openFlags(path string, flags int) (*os.File, error) {
	fd, err := syscall.Open(path, flags|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), path), nil
}
func openExclusive(path string) (*os.File, error) {
	return openFlags(path, syscall.O_WRONLY|syscall.O_CREAT|syscall.O_EXCL)
}
func openNoFollow(path string) (*os.File, error) {
	return openFlags(path, syscall.O_RDONLY|syscall.O_NONBLOCK)
}
func openNoFollowRW(path string) (*os.File, error) {
	return openFlags(path, syscall.O_RDWR|syscall.O_NONBLOCK)
}
func tryLockFile(f *os.File) (bool, error) {
	err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
		return false, nil
	}
	return err == nil, err
}
func unlockFile(f *os.File) { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN) }
func lockRetention(dir string) (*os.File, error) {
	path := filepath.Join(dir, ".retention.lock")
	f, err := openFlags(path, syscall.O_RDWR|syscall.O_CREAT)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 || !ownedByUser(info) {
		_ = f.Close()
		return nil, errors.New("unrecognized retention lock")
	}
	locked, err := tryLockFile(f)
	if err != nil || !locked {
		_ = f.Close()
		return nil, err
	}
	return f, nil
}
func unlockRetention(f *os.File) { unlockFile(f); _ = f.Close() }
