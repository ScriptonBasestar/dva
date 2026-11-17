//go:build !darwin && !linux && !freebsd && !netbsd && !openbsd

package resourceadmission

import (
	"errors"
	"os"
)

func tryLock(*os.File) error { return errors.New("resource admission unsupported on this platform") }
func unlock(*os.File) error  { return nil }
func busy(error) bool        { return false }
func owned(os.FileInfo) bool { return false }

func openPolicy(string) (*os.File, error) {
	return nil, errors.New("resource admission unsupported on this platform")
}
func openPrivateRegular(string, int) (*os.File, error) {
	return nil, errors.New("resource admission unsupported on this platform")
}
