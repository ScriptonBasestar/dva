package cirun

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/ScriptonBasestar/dva/internal/resourceadmission"
)

type AdmissionObservation struct {
	Class             string  `json:"class"`
	Priority          string  `json:"priority"`
	PolicyFingerprint string  `json:"policy_fingerprint,omitempty"`
	Outcome           string  `json:"outcome"`
	WaitDuration      *string `json:"wait_duration"`
	ExecutionDuration *string `json:"execution_duration"`
	ClaimID           string  `json:"claim_id,omitempty"`
	// Granularity is "step" when every step takes its own claim; ClaimID and
	// WaitDuration then describe the initial claim only.
	Granularity string `json:"granularity,omitempty"`
}

func admissionWait(ctx context.Context, dir string, p resourceadmission.Policy, digest, class, priority, label string, out io.Writer) (*resourceadmission.Lease, *AdmissionObservation, error) {
	obs := &AdmissionObservation{Class: class, Priority: priority, PolicyFingerprint: digest, Outcome: "waiting"}
	start := time.Now()
	_, _ = fmt.Fprintf(out, "ci admission %s waiting class=%s priority=%s\n", label, class, priority)
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				_, _ = fmt.Fprintf(out, "ci admission %s waiting elapsed=%s\n", label, time.Since(start).Round(time.Second))
			}
		}
	}()
	lease, err := resourceadmission.Acquire(ctx, dir, p, digest, class, priority)
	close(stop)
	<-done
	duration := time.Since(start).String()
	obs.WaitDuration = &duration
	if err != nil {
		var blocked *resourceadmission.QuarantineBlockedError
		switch {
		case errors.As(err, &blocked):
			obs.Outcome = "quarantine_blocked"
		case errors.Is(err, context.DeadlineExceeded):
			obs.Outcome = "timed_out"
		case errors.Is(err, context.Canceled):
			obs.Outcome = "canceled"
		default:
			obs.Outcome = "failed"
		}
		_, _ = fmt.Fprintf(out, "ci admission %s %s waited=%s\n", label, obs.Outcome, duration)
		return nil, obs, err
	}
	obs.Outcome = "acquired"
	obs.ClaimID = lease.ID()
	_, _ = fmt.Fprintf(out, "ci admission %s acquired waited=%s\n", label, duration)
	return lease, obs, nil
}
