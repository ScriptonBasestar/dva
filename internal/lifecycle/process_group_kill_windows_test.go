//go:build windows

package lifecycle

import "testing"

// killProcessGroupInCleanup is a no-op: the process-group tests skip on Windows before they
// start anything to clean up.
func killProcessGroupInCleanup(t *testing.T, _ int) { t.Helper() }
