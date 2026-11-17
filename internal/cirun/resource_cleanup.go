package cirun

import (
	"context"
	"sync"
)

type resourceCleanupKey struct{}
type resourceCleanupProbeKey struct{}
type resourceCleanupTracker struct {
	mu        sync.Mutex
	uncertain bool
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
