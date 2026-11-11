//go:build darwin || linux || freebsd || netbsd || openbsd

package compare

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

func lockFile(f *os.File) error {
	err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if errors.Is(err, syscall.EWOULDBLOCK) {
		return fmt.Errorf("%w (%s)", ErrLocked, f.Name())
	}
	if err != nil {
		return fmt.Errorf("lock compare: %w", err)
	}
	return nil
}

func unlockFile(f *os.File) { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN) }
