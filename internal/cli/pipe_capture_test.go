package cli

import (
	"bytes"
	"os"
	"testing"
)

// startCapture points every target (for example &os.Stderr, or &os.Stdout and &os.Stderr
// together) at one pipe and starts draining it at once. The returned finish restores the
// targets and returns everything written.
//
// The drain must run while the code under test writes. A pipe's buffer is not a fixed size:
// Darwin hands out 512-byte buffers once kernel pipe memory runs short, so a helper that reads
// only after fn returns blocks the first write past that size and the test hangs until the
// package timeout (observed 2026-11-08: TestCompositionWarnsEachOwner writes 580 bytes).
func startCapture(t *testing.T, targets ...**os.File) (finish func() string) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	olds := make([]*os.File, len(targets))
	for i, target := range targets {
		olds[i] = *target
		*target = w
	}
	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)
		_ = r.Close()
		done <- buf.String()
	}()
	finished := false
	restore := func() {
		if finished {
			return
		}
		finished = true
		_ = w.Close()
		for i, target := range targets {
			*target = olds[i]
		}
	}
	// A t.Fatal inside the captured function must not leave os.Stderr pointing at a closed
	// pipe for the tests that follow.
	t.Cleanup(restore)
	return func() string {
		restore()
		return <-done
	}
}

// TestStartCaptureDrainsWhileWriting writes far more than any pipe buffer holds (64KiB at
// most) before finish is called, so a capture that reads only afterwards blocks here on every
// host instead of only on one whose pipe memory is short.
func TestStartCaptureDrainsWhileWriting(t *testing.T) {
	payload := bytes.Repeat([]byte("x"), 1<<20)
	finish := startCapture(t, &os.Stdout, &os.Stderr)
	if _, err := os.Stderr.Write(payload); err != nil {
		t.Fatalf("write stderr: %v", err)
	}
	if _, err := os.Stdout.Write([]byte("y")); err != nil {
		t.Fatalf("write stdout: %v", err)
	}
	if got := finish(); len(got) != len(payload)+1 {
		t.Fatalf("captured %d bytes, want %d", len(got), len(payload)+1)
	}
}
