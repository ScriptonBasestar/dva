//go:build !darwin && !linux && !freebsd && !netbsd && !openbsd

package cli

import "testing"

type ciRootLockHolder struct{ id string }

func (h *ciRootLockHolder) release() {}

func holdCIRootLock(t *testing.T, _ string) *ciRootLockHolder {
	t.Helper()
	t.Skip("CI root locking is unsupported on this operating system")
	return nil
}
