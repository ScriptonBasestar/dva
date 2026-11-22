package cirun

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/ScriptonBasestar/dva/internal/resourceadmission"
)

type resourceCleanupKey struct{}
type resourceCleanupProbeKey struct{}
type resourceCleanupTracker struct {
	mu        sync.Mutex
	uncertain bool
	// lease receives each started process group so a quarantined claim can
	// later be proven ended without an operator.
	lease *resourceadmission.Lease
}

// startResourceProcess starts cmd and records its process group on the
// lease. The launch is announced on the claim before Start, so a supervisor
// crash between Start and the record leaves the claim unprovable instead of
// looking ended. A failed record keeps that announcement and marks the
// tracker uncertain, so the lease is quarantined at the end.
func startResourceProcess(ctx context.Context, cmd *exec.Cmd) error {
	tracker, ok := ctx.Value(resourceCleanupKey{}).(*resourceCleanupTracker)
	if !ok {
		return startProcess(cmd)
	}
	lease := tracker.currentLease()
	if lease == nil {
		return startProcess(cmd)
	}
	recordCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
	defer cancel()
	if err := lease.BeginLaunch(recordCtx); err != nil {
		return fmt.Errorf("record process launch on resource lease: %w", err)
	}
	if err := startProcess(cmd); err != nil {
		endCtx, endCancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
		defer endCancel()
		return errors.Join(err, lease.EndLaunch(endCtx))
	}
	if err := lease.TrackProcessGroup(recordCtx, cmd.Process.Pid); err != nil {
		tracker.mu.Lock()
		tracker.uncertain = true
		tracker.mu.Unlock()
	}
	return nil
}

func observeResourceCleanup(ctx context.Context, pid int) {
	tracker, ok := ctx.Value(resourceCleanupKey{}).(*resourceCleanupTracker)
	if !ok {
		return
	}
	probe := resourceProcessGroupGone
	if testProbe, ok := ctx.Value(resourceCleanupProbeKey{}).(func(int) bool); ok {
		probe = testProbe
	}
	if !probe(pid) {
		tracker.mu.Lock()
		tracker.uncertain = true
		tracker.mu.Unlock()
	}
}

func (t *resourceCleanupTracker) isUncertain() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.uncertain
}

// currentLease and setLease guard the claim a profile run swaps when it yields.
func (t *resourceCleanupTracker) currentLease() *resourceadmission.Lease {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.lease
}

func (t *resourceCleanupTracker) setLease(lease *resourceadmission.Lease) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.lease = lease
}
