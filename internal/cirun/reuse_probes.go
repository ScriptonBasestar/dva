package cirun

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os/exec"
	"sync"
	"time"
)

// Probe output is fingerprint material, never a log or persisted plaintext.
// Probes are project-owned read-only commands, supervised like native checks.
func reuseProbeDigest(ctx context.Context, _, workdir, command string, env []string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return "", err
	}
	stdout := &probeOutput{hash: sha256.New()}
	cmd := exec.Command("/bin/sh", "-eu", "-c", command)
	cmd.Dir, cmd.Env = workdir, env
	cmd.Stdout, cmd.Stderr = stdout, io.Discard
	cmd.WaitDelay = time.Second
	if err := startProcess(cmd); err != nil {
		return "", errors.New("reuse fingerprint probe could not start")
	}
	pid := cmd.Process.Pid
	waited := make(chan error, 1)
	go func() { waited <- cmd.Wait() }()
	select {
	case err := <-waited:
		cleanupGroup(pid)
		if err != nil {
			return "", errors.New("reuse fingerprint probe failed (output withheld)")
		}
	case <-ctx.Done():
		terminateProcessGroup(pid, waited)
		return "", ctx.Err()
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	stdout.mu.Lock()
	defer stdout.mu.Unlock()
	if stdout.overflow {
		return "", errors.New("reuse fingerprint probe exceeded 1 MiB output limit")
	}
	return hex.EncodeToString(stdout.hash.Sum(nil)), nil
}

type probeOutput struct {
	mu   sync.Mutex
	hash interface {
		Write([]byte) (int, error)
		Sum([]byte) []byte
	}
	size     int
	overflow bool
}

func (w *probeOutput) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.size+len(data) > 1<<20 {
		w.overflow = true
		return 0, errors.New("reuse probe output limit exceeded")
	}
	w.size += len(data)
	return w.hash.Write(data)
}
