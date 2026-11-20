//go:build darwin || freebsd || netbsd || openbsd

package resourceadmission

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// bootID identifies the current boot; it is empty when unavailable.
func bootID() string {
	tv, err := unix.SysctlTimeval("kern.boottime")
	if err != nil {
		return ""
	}
	return fmt.Sprintf("boottime:%d.%06d", tv.Sec, tv.Usec)
}
