//go:build !darwin && !linux && !freebsd && !netbsd && !openbsd

package jobrun

import (
	"errors"
	"os"
)

func lockAndRun(*os.File, func() error) error {
	return errors.New("dva jobs locking is unsupported on this operating system")
}
func lockingSupported() bool { return false }
