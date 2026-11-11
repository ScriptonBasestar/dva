//go:build !darwin && !linux && !freebsd && !netbsd && !openbsd

package compare

import (
	"errors"
	"os"
)

func lockFile(*os.File) error {
	return errors.New("compare locking is unsupported on this operating system")
}

func unlockFile(*os.File) {}
