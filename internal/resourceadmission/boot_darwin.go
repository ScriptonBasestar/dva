//go:build darwin

package resourceadmission

import "golang.org/x/sys/unix"

// bootID identifies the current boot; it is empty when unavailable.
// kern.bootsessionuuid is fixed for one boot, unlike kern.boottime, which
// moves when the wall clock is stepped.
func bootID() string {
	id, err := unix.Sysctl("kern.bootsessionuuid")
	if err != nil || id == "" {
		return ""
	}
	return "bootsession:" + id
}
