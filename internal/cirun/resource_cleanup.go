package cirun

import (
	"context"
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

// observeResourceStart records a just-started process group on the lease.
// A failed record leaves the claim unable to prove that group ended, so the
// tracker is marked uncertain and the lease is quarantined at the end.
func observeResourceStart(ctx context.Context, pid int) {
	tracker, ok := ctx.Value(resourceCleanupKey{}).(*resourceCleanupTracker)
	if !ok || tracker.lease == nil {
		return
	}
	recordCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
	defer cancel()
	if err := tracker.lease.TrackProcessGroup(recordCtx, pid); err != nil {
		tracker.mu.Lock()
		tracker.uncertain = true
		tracker.mu.Unlock()
	}
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
