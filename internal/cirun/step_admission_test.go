package cirun

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
	"github.com/ScriptonBasestar/dva/internal/resourceadmission"
)

func waitForFile(t *testing.T, path string, limit time.Duration) {
	t.Helper()
	deadline := time.Now().Add(limit)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("%s did not appear", path)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func assertNoClaims(t *testing.T, resource, digest string) {
	t.Helper()
	entries, err := resourceadmission.Status(context.Background(), resource, digest)
	if err != nil || len(entries) != 0 {
		t.Fatalf("claims leaked: %+v %v", entries, err)
	}
}

// A background multi-step run must yield the heavy slot to a waiting
// interactive claim at its next step boundary, not at the end of the run.
func TestStepAdmissionYieldsToInteractiveWaiterAtStepBoundary(t *testing.T) {
	root, state, resource, policy, digest := bridgeFixture(t)
	marks := t.TempDir()
	first, release, second := filepath.Join(marks, "first"), filepath.Join(marks, "release"), filepath.Join(marks, "second")
	p := config.CIProfile{Timeout: "20s", MaxParallel: 1, Admission: &config.CIAdmission{Class: "heavy", Priority: "background", WaitTimeout: "10s"}, Steps: []config.CIStep{
		{Name: "one", Run: `touch "$FIRST"; i=0; while [ ! -e "$RELEASE" ] && [ $i -lt 1000 ]; do sleep 0.01; i=$((i+1)); done`},
		{Name: "two", Run: `touch "$SECOND"`, DependsOn: []string{"one"}},
	}}
	env := []string{"DVA_RESOURCE_POLICY=" + filepath.Join(root, "policy.json"), "FIRST=" + first, "RELEASE=" + release, "SECOND=" + second}
	done := make(chan Report, 1)
	go func() {
		r, _ := Run(context.Background(), Options{Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, resourceDirectory: resource, Env: env, Profile: p})
		done <- r
	}()
	waitForFile(t, first, 5*time.Second)

	admitted := make(chan *resourceadmission.Lease, 1)
	waitErr := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		lease, err := resourceadmission.Acquire(ctx, resource, policy, digest, "heavy", "interactive")
		if err != nil {
			waitErr <- err
			return
		}
		admitted <- lease
	}()
	select {
	case <-admitted:
		t.Fatal("interactive claim admitted while the background step held the heavy slot")
	case err := <-waitErr:
		t.Fatal(err)
	case <-time.After(300 * time.Millisecond):
	}
	if err := os.WriteFile(release, nil, 0600); err != nil {
		t.Fatal(err)
	}
	var lease *resourceadmission.Lease
	select {
	case lease = <-admitted:
	case err := <-waitErr:
		t.Fatal(err)
	case got := <-done:
		t.Fatalf("background run finished before the interactive claim was admitted: %+v", got.Steps)
	case <-time.After(5 * time.Second):
		t.Fatal("interactive claim was not admitted at the step boundary")
	}
	time.Sleep(300 * time.Millisecond)
	if exists(second) {
		t.Fatal("background step two ran while the interactive claim held the heavy slot")
	}
	if err := lease.Complete(context.Background()); err != nil {
		t.Fatal(err)
	}
	got := <-done
	if got.Status != "succeeded" || !exists(second) {
		t.Fatalf("run=%s err=%s", got.Status, got.Error)
	}
	two := got.Steps[1]
	if two.Admission == nil || two.Admission.Outcome != "completed" || two.Admission.WaitDuration == nil {
		t.Fatalf("step two admission=%+v", two.Admission)
	}
	if waited, _ := time.ParseDuration(*two.Admission.WaitDuration); waited < 250*time.Millisecond {
		t.Fatalf("step two did not wait behind the interactive claim: %s", waited)
	}
	if got.Admission == nil || got.Admission.Granularity != "step" || got.Admission.Outcome != "completed" {
		t.Fatalf("profile admission=%+v", got.Admission)
	}
	assertNoClaims(t, resource, digest)
}

func TestStepAdmissionReleasesClaimOnStepFailure(t *testing.T) {
	root, state, resource, _, digest := bridgeFixture(t)
	noFailFast := false
	for _, failFast := range []*bool{nil, &noFailFast} {
		p := config.CIProfile{Timeout: "10s", MaxParallel: 1, FailFast: failFast, Admission: &config.CIAdmission{Class: "heavy", Priority: "background", WaitTimeout: "2s"}, Steps: []config.CIStep{
			{Name: "fails", Run: "exit 3"}, {Name: "independent", Run: "true"},
		}}
		r, err := Run(context.Background(), Options{Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, resourceDirectory: resource, Env: []string{"DVA_RESOURCE_POLICY=" + filepath.Join(root, "policy.json")}, Profile: p})
		if err == nil || r.Status != "failed" {
			t.Fatalf("failure accepted: %s %v", r.Status, err)
		}
		if r.Steps[0].Admission == nil || r.Steps[0].Admission.Outcome != "completed" {
			t.Fatalf("failed step claim=%+v", r.Steps[0].Admission)
		}
		assertNoClaims(t, resource, digest)
	}
}

func TestStepAdmissionReleasesClaimOnCancel(t *testing.T) {
	root, state, resource, _, digest := bridgeFixture(t)
	marks := t.TempDir()
	started := filepath.Join(marks, "started")
	p := config.CIProfile{Timeout: "20s", MaxParallel: 1, Admission: &config.CIAdmission{Class: "heavy", Priority: "background", WaitTimeout: "2s"}, Steps: []config.CIStep{
		{Name: "long", Run: `touch "$STARTED"; sleep 15`}, {Name: "after", Run: "true", DependsOn: []string{"long"}},
	}}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan Report, 1)
	go func() {
		r, _ := Run(ctx, Options{Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, resourceDirectory: resource, Env: []string{"DVA_RESOURCE_POLICY=" + filepath.Join(root, "policy.json"), "STARTED=" + started}, Profile: p})
		done <- r
	}()
	waitForFile(t, started, 5*time.Second)
	cancel()
	select {
	case r := <-done:
		if r.Status != "canceled" {
			t.Fatalf("status=%s err=%s", r.Status, r.Error)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("cancel blocked")
	}
	assertNoClaims(t, resource, digest)
}

// A step whose admission wait times out records the outcome and leaves no claim.
func TestStepAdmissionWaitTimeoutLeavesNoClaim(t *testing.T) {
	root, state, resource, policy, digest := bridgeFixture(t)
	marks := t.TempDir()
	first, release := filepath.Join(marks, "first"), filepath.Join(marks, "release")
	p := config.CIProfile{Timeout: "20s", MaxParallel: 1, Admission: &config.CIAdmission{Class: "heavy", Priority: "background", WaitTimeout: "300ms"}, Steps: []config.CIStep{
		{Name: "one", Run: `touch "$FIRST"; i=0; while [ ! -e "$RELEASE" ] && [ $i -lt 1000 ]; do sleep 0.01; i=$((i+1)); done`},
		{Name: "two", Run: "true", DependsOn: []string{"one"}},
	}}
	done := make(chan Report, 1)
	go func() {
		r, _ := Run(context.Background(), Options{Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, resourceDirectory: resource, Env: []string{"DVA_RESOURCE_POLICY=" + filepath.Join(root, "policy.json"), "FIRST=" + first, "RELEASE=" + release}, Profile: p})
		done <- r
	}()
	waitForFile(t, first, 5*time.Second)
	admitted := make(chan *resourceadmission.Lease, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		lease, err := resourceadmission.Acquire(ctx, resource, policy, digest, "heavy", "interactive")
		if err == nil {
			admitted <- lease
		}
		close(admitted)
	}()
	time.Sleep(100 * time.Millisecond)
	if err := os.WriteFile(release, nil, 0600); err != nil {
		t.Fatal(err)
	}
	lease := <-admitted
	if lease == nil {
		t.Fatal("interactive claim not admitted")
	}
	r := <-done
	if err := lease.Complete(context.Background()); err != nil {
		t.Fatal(err)
	}
	if r.Status != "failed" || r.Steps[1].Status != "timed_out" || r.Steps[1].Admission == nil || r.Steps[1].Admission.Outcome != "timed_out" || r.Admission.Outcome != "timed_out" {
		t.Fatalf("run=%s step=%+v profile=%+v", r.Status, r.Steps[1], r.Admission)
	}
	assertNoClaims(t, resource, digest)
}

// The supervisor never holds two claims, even when the profile declares
// parallel steps and the class would fit twice.
func TestStepAdmissionNeverHoldsTwoClaims(t *testing.T) {
	root, state, resource, _, digest := bridgeFixture(t)
	p := config.CIProfile{Timeout: "20s", MaxParallel: 3, Admission: &config.CIAdmission{Class: "light", Priority: "background", WaitTimeout: "5s"}, Steps: []config.CIStep{
		{Name: "a", Run: "sleep 0.3"}, {Name: "b", Run: "sleep 0.3"}, {Name: "c", Run: "sleep 0.3"},
	}}
	var peak atomic.Int64
	stop := make(chan struct{})
	polled := make(chan struct{})
	go func() {
		defer close(polled)
		for {
			select {
			case <-stop:
				return
			default:
			}
			if entries, err := resourceadmission.Status(context.Background(), resource, digest); err == nil {
				running := int64(0)
				for _, e := range entries {
					if e.State == "running" {
						running++
					}
				}
				if running > peak.Load() {
					peak.Store(running)
				}
			}
			time.Sleep(5 * time.Millisecond)
		}
	}()
	r, err := Run(context.Background(), Options{Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, resourceDirectory: resource, Env: []string{"DVA_RESOURCE_POLICY=" + filepath.Join(root, "policy.json")}, Profile: p})
	close(stop)
	<-polled
	if err != nil || r.Status != "succeeded" {
		t.Fatalf("run=%s err=%v", r.Status, err)
	}
	if got := peak.Load(); got != 1 {
		t.Fatalf("peak concurrent claims=%d, want 1", got)
	}
	assertNoClaims(t, resource, digest)
}

func TestStepAdmissionRefusesSecondConcurrentClaim(t *testing.T) {
	_, _, resource, policy, digest := bridgeFixture(t)
	s := &stepAdmission{dir: resource, digest: digest, class: "heavy", priority: "normal", policy: policy, wait: time.Second}
	lease, _, err := s.acquire(context.Background(), "one", io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.acquire(context.Background(), "two", io.Discard); !errors.Is(err, errClaimAlreadyHeld) {
		t.Fatalf("second claim: %v", err)
	}
	if err := s.release(lease, &resourceCleanupTracker{lease: lease}); err != nil {
		t.Fatal(err)
	}
	lease, _, err = s.acquire(context.Background(), "three", io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.release(lease, &resourceCleanupTracker{lease: lease}); err != nil {
		t.Fatal(err)
	}
	if err := s.close(); err != nil {
		t.Fatal(err)
	}
	assertNoClaims(t, resource, digest)
}

// An initial claim that no step consumed is released when the run ends early.
func TestStepAdmissionReleasesUnconsumedInitialClaim(t *testing.T) {
	_, _, resource, policy, digest := bridgeFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	lease, err := resourceadmission.Acquire(ctx, resource, policy, digest, "heavy", "normal")
	if err != nil {
		t.Fatal(err)
	}
	s := &stepAdmission{dir: resource, digest: digest, class: "heavy", priority: "normal", policy: policy, wait: time.Second, pending: lease, pendingObs: &AdmissionObservation{}}
	if err := s.close(); err != nil {
		t.Fatal(err)
	}
	assertNoClaims(t, resource, digest)
}
