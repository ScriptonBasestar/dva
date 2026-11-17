//go:build !darwin && !linux

package cirun

import "testing"

func TestSummaryUnsupportedPlatformFailsClosed(t *testing.T) {
	reports, excluded, err := ReadSummaryReceipts(t.TempDir())
	if err != nil || len(reports) != 0 || excluded["state_directory_unavailable"] != 1 {
		t.Fatalf("reports=%v excluded=%v err=%v", reports, excluded, err)
	}
}
