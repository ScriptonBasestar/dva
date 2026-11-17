package cirun

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestReuseProbeExitCodeWithholdsBothOutputs(t *testing.T) {
	dir := t.TempDir()
	_, err := reuseProbeDigest(context.Background(), dir, dir,
		`printf stdout-canary; printf stderr-canary >&2; exit 20`, os.Environ())
	if err == nil || err.Error() != "reuse fingerprint probe failed: exit_code=20 (output withheld)" {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(err.Error(), "stdout-canary") || strings.Contains(err.Error(), "stderr-canary") {
		t.Fatalf("probe output leaked: %v", err)
	}
}

func TestReuseProbeOverflowIsNotReportedAsExitFailure(t *testing.T) {
	dir := t.TempDir()
	_, err := reuseProbeDigest(context.Background(), dir, dir,
		`head -c 1048577 /dev/zero; printf stderr-canary >&2`, os.Environ())
	if err == nil || err.Error() != "reuse fingerprint probe exceeded 1 MiB output limit" {
		t.Fatalf("unexpected error: %v", err)
	}
}
