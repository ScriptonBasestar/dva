//go:build !darwin && !linux && !freebsd && !netbsd && !openbsd

package cirun

import (
	"errors"
	"os"
	"os/exec"
)

func tryLock(*os.File) error {
	return errors.New("dva ci locking is unsupported on this operating system")
}
func unlock(*os.File) error    { return nil }
func lockContended(error) bool { return false }
func startProcess(*exec.Cmd) error {
	return errors.New("dva ci process supervision is unsupported on this operating system")
}
func cleanupGroup(int)                        {}
func terminateProcessGroup(int, <-chan error) {}
