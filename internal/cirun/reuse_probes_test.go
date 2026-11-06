package cirun

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"testing"
	"time"
)

func TestReuseProbeDigestAndFailure(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	got, err := reuseProbeDigest(ctx, dir, dir, "printf '%s' \"$PROBE\"", []string{"PROBE=private"})
	want := sha256.Sum256([]byte("private"))
	if err != nil || got != hex.EncodeToString(want[:]) {
		t.Fatalf("digest %s %v", got, err)
	}
	if _, err := reuseProbeDigest(ctx, dir, dir, "echo secret >&2; exit 1", os.Environ()); err == nil {
		t.Fatal("failed probe passed")
	}
	if _, err := reuseProbeDigest(ctx, dir, dir, "head -c 1048577 /dev/zero", os.Environ()); err == nil {
		t.Fatal("unbounded probe passed")
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := reuseProbeDigest(cancelled, dir, dir, "true", nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel: %v", err)
	}
	timed, stop := context.WithTimeout(ctx, 40*time.Millisecond)
	defer stop()
	if _, err := reuseProbeDigest(timed, dir, dir, "sleep 30 & wait", os.Environ()); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("timeout: %v", err)
	}
}
