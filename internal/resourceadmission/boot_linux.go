//go:build linux

package resourceadmission

import (
	"os"
	"strings"
)

// bootID identifies the current boot; it is empty when unavailable.
func bootID() string {
	b, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil {
		return ""
	}
	if id := strings.TrimSpace(string(b)); id != "" {
		return "boot_id:" + id
	}
	return ""
}
