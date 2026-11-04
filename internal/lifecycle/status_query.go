package lifecycle

import (
	"bytes"
	"context"
	"os/exec"
	"time"
)

const statusQueryGrace = 250 * time.Millisecond

func runStatusQuery(ctx context.Context, cmd *exec.Cmd) ([]byte, error) {
	if ctx.Done() == nil {
		return cmd.Output()
	}
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err := runOwnedQuery(ctx, cmd)
	return stdout.Bytes(), err
}

// runOwnedQuery runs a cancellable, DVA-owned query without changing callers'
// configured stdout/stderr. Background callers retain exec.Cmd.Run semantics.
func runOwnedQuery(ctx context.Context, cmd *exec.Cmd) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if ctx.Done() == nil {
		return cmd.Run()
	}
	if err := configureProcessGroup(cmd); err != nil && processGroupsSupported() {
		return err
	}
	cmd.WaitDelay = time.Second
	if err := cmd.Start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil && processGroupsSupported() {
			// WaitDelay can close inherited pipes after the parent exits;
			// also reclaim descendants of this failed, owned query.
			_ = killProcessGroup(cmd.Process.Pid)
		}
		return err
	case <-ctx.Done():
		if processGroupsSupported() {
			_ = terminateProcessGroup(cmd.Process.Pid)
		} else {
			_ = cmd.Process.Kill()
		}
		select {
		case _ = <-done:
		case <-time.After(statusQueryGrace):
			if processGroupsSupported() {
				_ = killProcessGroup(cmd.Process.Pid)
			} else {
				_ = cmd.Process.Kill()
			}
			_ = <-done
		}
		return ctx.Err()
	}
}
