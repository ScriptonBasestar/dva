package cirun

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
	"github.com/ScriptonBasestar/dva/internal/resourceadmission"
)

// stepAdmissionKey carries the profile's step-granular admission into execute.
type stepAdmissionKey struct{}

// stepAdmission owns every resource claim of one admitted CI run. Each step
// takes its own claim and returns it when the step ends, so another session's
// waiting claim is ordered against this run at every step boundary instead of
// only once at the end of the run. The supervisor owns at most one claim at a
// time: acquire refuses while a claim is held or being requested, and the run
// executes steps one at a time while admission is active.
//
// The first claim is acquired before the run starts (the profile wait that the
// waiting receipt records) and handed to the first dispatched step, so the run
// never starts without a claim and never holds a second one for that step.
type stepAdmission struct {
	dir, digest, class, priority string
	policy                       resourceadmission.Policy
	wait                         time.Duration

	mu         sync.Mutex
	busy       bool // a claim is held or being requested
	held       *resourceadmission.Lease
	pending    *resourceadmission.Lease
	pendingObs *AdmissionObservation
	// failure is the first step admission outcome that was not an acquisition;
	// quarantined records a claim retained because cleanup was unconfirmed.
	failure     string
	quarantined bool
}

var errClaimAlreadyHeld = errors.New("resource admission: supervisor already holds a claim; refusing a second concurrent claim")

func (s *stepAdmission) acquire(ctx context.Context, label string, out io.Writer) (*resourceadmission.Lease, *AdmissionObservation, error) {
	s.mu.Lock()
	if s.busy {
		s.mu.Unlock()
		return nil, &AdmissionObservation{Class: s.class, Priority: s.priority, PolicyFingerprint: s.digest, Outcome: "failed"}, errClaimAlreadyHeld
	}
	s.busy = true
	if s.pending != nil {
		lease, obs := s.pending, *s.pendingObs
		s.pending, s.pendingObs, s.held = nil, nil, lease
		s.mu.Unlock()
		_, _ = fmt.Fprintf(out, "ci admission step %s uses the initial claim\n", label)
		return lease, &obs, nil
	}
	s.mu.Unlock()
	waitCtx, cancel := context.WithTimeout(ctx, s.wait)
	lease, obs, err := admissionWait(waitCtx, s.dir, s.policy, s.digest, s.class, s.priority, "step "+label, out)
	cancel()
	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		s.busy = false
		if s.failure == "" {
			s.failure = obs.Outcome
		}
		return nil, obs, err
	}
	s.held = lease
	return lease, obs, nil
}

// release ends the claim that acquire returned. A claim whose process group
// cleanup is unconfirmed is quarantined instead of released.
func (s *stepAdmission) release(lease *resourceadmission.Lease, tracker *resourceCleanupTracker) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.held != lease || lease == nil {
		return errors.New("resource admission: released claim is not the held claim")
	}
	s.held, s.busy = nil, false
	if tracker.isUncertain() {
		s.quarantined = true
	}
	return finishHostSlotBridge(lease, tracker)
}

// close returns an initial claim that no step consumed. A claim still held at
// close means a step did not release it; that is reported and the claim is
// quarantined, because its processes were not observed to end.
func (s *stepAdmission) close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var err error
	if s.pending != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		err = s.pending.Complete(ctx)
		cancel()
		s.pending, s.pendingObs = nil, nil
	}
	if s.held != nil {
		s.quarantined = true
		err = errors.Join(err, fmt.Errorf("resource lease %s still held at run end", s.held.ID()), s.held.Quarantine())
		s.held = nil
	}
	s.busy = false
	return err
}

// runAdmittedStep runs one step inside its own claim. The wait is measured
// before the step starts, so it never consumes the declared step timeout.
func runAdmittedStep(ctx context.Context, root string, step config.CIStep, inherited []string, workers int, source string, out io.Writer, startedAt time.Time) (StepReport, error) {
	admission, ok := ctx.Value(stepAdmissionKey{}).(*stepAdmission)
	if !ok {
		return runConditionedStep(ctx, root, step, inherited, workers, source, out, startedAt)
	}
	lease, obs, err := admission.acquire(ctx, step.Name, out)
	if err != nil {
		r := StepReport{Name: step.Name, StartedAt: startedAt, FinishedAt: time.Now(), Admission: obs, Error: err.Error()}
		r.Duration = r.FinishedAt.Sub(startedAt)
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			r.Status = "timed_out"
		case errors.Is(err, context.Canceled):
			r.Status = "canceled"
		default:
			r.Status = "failed"
		}
		return r, fmt.Errorf("step admission: %w", err)
	}
	tracker := &resourceCleanupTracker{lease: lease}
	executionStart := time.Now()
	r, err := runConditionedStep(context.WithValue(ctx, resourceCleanupKey{}, tracker), root, step, inherited, workers, source, out, executionStart)
	finishErr := admission.release(lease, tracker)
	duration := time.Since(executionStart).String()
	obs.ExecutionDuration = &duration
	if finishErr != nil {
		obs.Outcome = "quarantined"
		if r.Status == "succeeded" || r.Status == "skipped" {
			r.Status = "failed"
		}
		err = errors.Join(err, finishErr)
		r.Error = err.Error()
	} else {
		obs.Outcome = "completed"
	}
	r.Admission = obs
	return r, err
}
