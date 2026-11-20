//go:build freebsd || netbsd || openbsd

package resourceadmission

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// bootID identifies the current boot; it is empty when unavailable. The
// boottime: prefix marks it as unstable (clock steps move it), so it never
// proves a reboot on its own.
func bootID() string {
	tv, err := unix.SysctlTimeval("kern.boottime")
	if err != nil {
		return ""
	}
	return fmt.Sprintf("boottime:%d.%06d", tv.Sec, tv.Usec)
}
