package cirun

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/ScriptonBasestar/dva/internal/resourceadmission"
)

// AdmissionYield records one time a profile run gave its claim back at a step
// boundary and waited to be admitted again.
type AdmissionYield struct {
	ReleasedClaimID string  `json:"released_claim_id"`
	ClaimID         string  `json:"claim_id,omitempty"`
	Outcome         string  `json:"outcome"`
	WaitDuration    *string `json:"wait_duration"`
}

type profileAdmissionKey struct{}

// profileAdmission is the single claim a profile run holds. The run keeps its
// declared parallelism under that claim. At a dispatch boundary, when another
// request that the ordering ranks at least as high would be admitted by the
// release, the run stops dispatching, waits for its running steps, releases
// the claim and queues again under the same class and priority. The run never
// holds two claims: it re-queues only after the release completed.
type profileAdmission struct {
	dir, digest, class, priority string
	policy                       resourceadmission.Policy
	wait                         time.Duration
	tracker                      *resourceCleanupTracker
	observation                  *AdmissionObservation
	out                          io.Writer
}

// contended reports whether the run should yield at this boundary. A claim
// whose cleanup is already unconfirmed is never released early, and a failed
// query keeps the claim: the yield is cooperative, not a correctness gate.
func (a *profileAdmission) contended(ctx context.Context) bool {
	lease := a.tracker.currentLease()
	if lease == nil || a.tracker.isUncertain() {
		return false
	}
	checkCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
	defer cancel()
	wanted, err := resourceadmission.YieldWanted(checkCtx, a.dir, a.policy, a.digest, lease.ID(), a.priority)
	return err == nil && wanted
}

// yield releases the held claim and waits for a new one. It runs only when no
// step of this run is executing. On a failed wait the run holds no claim.
func (a *profileAdmission) yield(ctx context.Context) error {
	old := a.tracker.currentLease()
	if old == nil {
		return errors.New("resource admission: no claim to yield")
	}
	if err := finishHostSlotBridge(old, a.tracker); err != nil {
		a.tracker.setLease(nil)
		return err
	}
	a.tracker.setLease(nil)
	waitCtx, cancel := context.WithTimeout(ctx, a.wait)
	lease, obs, err := admissionWait(waitCtx, a.dir, a.policy, a.digest, a.class, a.priority, "profile yield", a.out)
	cancel()
	record := AdmissionYield{ReleasedClaimID: old.ID(), Outcome: obs.Outcome, WaitDuration: obs.WaitDuration}
	if err == nil {
		record.ClaimID = lease.ID()
		a.tracker.setLease(lease)
	}
	a.observation.Yields = append(a.observation.Yields, record)
	return err
}
