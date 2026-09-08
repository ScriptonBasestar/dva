//go:build darwin || linux || freebsd || netbsd || openbsd

package jobrun

import (
	"fmt"
	"os"
	"syscall"
)

func lockAndRun(f *os.File, fn func() error) error {
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return fmt.Errorf("job receipt lock unavailable: %w", err)
	}
	defer func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN) }()
	return fn()
}
func lockingSupported() bool { return true }
