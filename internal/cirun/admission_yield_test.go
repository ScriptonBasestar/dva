package cirun

import (
	"context"
	"os"
	"path/filepath"
	"sync"
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

func fileExists(path string) bool {
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

// claimSampler records the set of running claim IDs over time.
type claimSampler struct {
	mu        sync.Mutex
	snapshots [][]string
	stop      chan struct{}
	done      chan struct{}
}

func sampleClaims(resource, digest string) *claimSampler {
	s := &claimSampler{stop: make(chan struct{}), done: make(chan struct{})}
	go func() {
		defer close(s.done)
		for {
			select {
			case <-s.stop:
				return
			default:
			}
			if entries, err := resourceadmission.Status(context.Background(), resource, digest); err == nil {
				var ids []string
				for _, e := range entries {
					if e.State == "running" {
						ids = append(ids, e.ID)
					}
				}
				s.mu.Lock()
				s.snapshots = append(s.snapshots, ids)
				s.mu.Unlock()
			}
			time.Sleep(5 * time.Millisecond)
		}
	}()
	return s
}

// maxOwned returns the most claims from owned seen running at one instant.
func (s *claimSampler) maxOwned(owned map[string]bool) int {
	close(s.stop)
	<-s.done
	peak := 0
	for _, snap := range s.snapshots {
		n := 0
		for _, id := range snap {
			if owned[id] {
				n++
			}
		}
		peak = max(peak, n)
	}
	return peak
}

func ownedClaims(r Report) map[string]bool {
	owned := map[string]bool{}
	if r.Admission != nil {
		owned[r.Admission.ClaimID] = true
		for _, y := range r.Admission.Yields {
			if y.ClaimID != "" {
				owned[y.ClaimID] = true
			}
		}
	}
	return owned
}

const holdUntilRelease = `touch "$FIRST"; i=0; while [ ! -e "$RELEASE" ] && [ $i -lt 1000 ]; do sleep 0.01; i=$((i+1)); done`

func TestAdmissionUncontendedRunKeepsParallelism(t *testing.T) {
	root, state, resource, _, digest := bridgeFixture(t)
	marks := t.TempDir()
	p := config.CIProfile{Timeout: "20s", MaxParallel: 2, Admission: &config.CIAdmission{Class: "heavy", Priority: "background", WaitTimeout: "2s"}, Steps: []config.CIStep{
		{Name: "a", Run: `touch "$M/a"; i=0; while [ ! -e "$M/b" ] && [ $i -lt 500 ]; do sleep 0.01; i=$((i+1)); done; test -e "$M/b"`},
		{Name: "b", Run: `touch "$M/b"; i=0; while [ ! -e "$M/a" ] && [ $i -lt 500 ]; do sleep 0.01; i=$((i+1)); done; test -e "$M/a"`},
	}}
	sampler := sampleClaims(resource, digest)
	r, err := Run(context.Background(), Options{Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, resourceDirectory: resource, Env: []string{"DVA_RESOURCE_POLICY=" + filepath.Join(root, "policy.json"), "M=" + marks}, Profile: p})
	if peak := sampler.maxOwned(ownedClaims(r)); peak > 1 {
		t.Fatalf("held %d claims at once", peak)
	}
	if err != nil || r.Status != "succeeded" {
		t.Fatalf("run=%s err=%v", r.Status, err)
	}
	a, b := r.Steps[0], r.Steps[1]
	if !a.StartedAt.Before(b.FinishedAt) || !b.StartedAt.Before(a.FinishedAt) {
		t.Fatalf("steps did not overlap: a=%s..%s b=%s..%s", a.StartedAt, a.FinishedAt, b.StartedAt, b.FinishedAt)
	}
	if len(r.Admission.Yields) != 0 || r.Admission.Outcome != "completed" {
		t.Fatalf("uncontended run yielded: %+v", r.Admission)
	}
	assertNoClaims(t, resource, digest)
}

// runContended starts a two-step background run, queues a waiter of the given
// priority while step one runs, then releases step one.
func runContended(t *testing.T, runPriority, waiterPriority string) (Report, *resourceadmission.Lease, bool, bool, map[string]bool) {
	t.Helper()
	root, state, resource, policy, digest := bridgeFixture(t)
	marks := t.TempDir()
	first, release, second := filepath.Join(marks, "first"), filepath.Join(marks, "release"), filepath.Join(marks, "second")
	p := config.CIProfile{Timeout: "20s", MaxParallel: 2, Admission: &config.CIAdmission{Class: "heavy", Priority: runPriority, WaitTimeout: "10s"}, Steps: []config.CIStep{
		{Name: "one", Run: holdUntilRelease},
		{Name: "two", Run: `touch "$SECOND"`, DependsOn: []string{"one"}},
	}}
	env := []string{"DVA_RESOURCE_POLICY=" + filepath.Join(root, "policy.json"), "FIRST=" + first, "RELEASE=" + release, "SECOND=" + second}
	sampler := sampleClaims(resource, digest)
	done := make(chan Report, 1)
	go func() {
		r, _ := Run(context.Background(), Options{Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, resourceDirectory: resource, Env: env, Profile: p})
		done <- r
	}()
	waitForFile(t, first, 5*time.Second)
	admitted := make(chan *resourceadmission.Lease, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		lease, err := resourceadmission.Acquire(ctx, resource, policy, digest, "heavy", waiterPriority)
		if err != nil {
			lease = nil
		}
		admitted <- lease
	}()
	time.Sleep(300 * time.Millisecond)
	if err := os.WriteFile(release, nil, 0600); err != nil {
		t.Fatal(err)
	}
	var lease *resourceadmission.Lease
	var r Report
	waiterFirst, secondDuringWaiter := false, false
	select {
	case lease = <-admitted:
		// The waiter got in first only if step two had not run yet.
		waiterFirst = !fileExists(second)
		time.Sleep(300 * time.Millisecond)
		secondDuringWaiter = fileExists(second)
		if lease != nil {
			if err := lease.Complete(context.Background()); err != nil {
				t.Fatal(err)
			}
		}
		r = <-done
	case r = <-done:
		lease = <-admitted
		if lease != nil {
			if err := lease.Complete(context.Background()); err != nil {
				t.Fatal(err)
			}
		}
	}
	owned := ownedClaims(r)
	if peak := sampler.maxOwned(owned); peak > 1 {
		t.Fatalf("held %d claims at once", peak)
	}
	assertNoClaims(t, resource, digest)
	return r, lease, waiterFirst, secondDuringWaiter, owned
}

func TestAdmissionContendedRunYieldsAtNextBoundary(t *testing.T) {
	r, lease, waiterFirst, secondDuringWaiter, _ := runContended(t, "background", "interactive")
	if lease == nil || !waiterFirst {
		t.Fatal("interactive waiter was not admitted before the run finished")
	}
	if secondDuringWaiter {
		t.Fatal("step two ran while the interactive waiter held the heavy slot")
	}
	if r.Status != "succeeded" || r.Admission == nil || r.Admission.Outcome != "completed" || len(r.Admission.Yields) != 1 {
		t.Fatalf("run=%s admission=%+v err=%s", r.Status, r.Admission, r.Error)
	}
	y := r.Admission.Yields[0]
	if y.Outcome != "acquired" || y.WaitDuration == nil || y.ClaimID == "" || y.ReleasedClaimID != r.Admission.ClaimID {
		t.Fatalf("yield=%+v", y)
	}
	if waited, _ := time.ParseDuration(*y.WaitDuration); waited < 250*time.Millisecond {
		t.Fatalf("re-acquire did not wait behind the waiter: %s", waited)
	}
}

// A lower-priority waiter does not make the run yield.
func TestAdmissionLowerPriorityWaiterDoesNotCauseYield(t *testing.T) {
	r, lease, waiterFirst, _, _ := runContended(t, "normal", "background")
	if waiterFirst || lease == nil {
		t.Fatalf("lower-priority waiter got in first (lease=%v)", lease != nil)
	}
	if r.Status != "succeeded" || len(r.Admission.Yields) != 0 {
		t.Fatalf("run=%s admission=%+v", r.Status, r.Admission)
	}
}

func TestAdmissionNoClaimLeakOnFailureCancelOrPanic(t *testing.T) {
	t.Run("failure", func(t *testing.T) {
		root, state, resource, _, digest := bridgeFixture(t)
		p := config.CIProfile{Timeout: "10s", MaxParallel: 2, Admission: &config.CIAdmission{Class: "heavy", Priority: "background", WaitTimeout: "2s"}, Steps: []config.CIStep{{Name: "fails", Run: "exit 3"}, {Name: "other", Run: "true"}}}
		r, err := Run(context.Background(), Options{Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, resourceDirectory: resource, Env: []string{"DVA_RESOURCE_POLICY=" + filepath.Join(root, "policy.json")}, Profile: p})
		if err == nil || r.Status != "failed" {
			t.Fatalf("failure accepted: %s %v", r.Status, err)
		}
		assertNoClaims(t, resource, digest)
	})
	t.Run("cancel", func(t *testing.T) {
		root, state, resource, _, digest := bridgeFixture(t)
		started := filepath.Join(t.TempDir(), "started")
		p := config.CIProfile{Timeout: "20s", MaxParallel: 2, Admission: &config.CIAdmission{Class: "heavy", Priority: "background", WaitTimeout: "2s"}, Steps: []config.CIStep{{Name: "long", Run: `touch "$STARTED"; sleep 15`}}}
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
	})
	t.Run("panic", func(t *testing.T) {
		root, state, resource, _, digest := bridgeFixture(t)
		p := config.CIProfile{Timeout: "10s", MaxParallel: 2, Admission: &config.CIAdmission{Class: "heavy", Priority: "background", WaitTimeout: "2s"}, Steps: []config.CIStep{{Name: "boom", Run: "true"}, {Name: "other", Run: "true"}}}
		ctx := context.WithValue(context.Background(), stepStartHookKey{}, func(name string) {
			if name == "boom" {
				panic("injected step panic")
			}
		})
		r, err := Run(ctx, Options{Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, resourceDirectory: resource, Env: []string{"DVA_RESOURCE_POLICY=" + filepath.Join(root, "policy.json")}, Profile: p})
		if err == nil || r.Status != "failed" || r.Steps[0].Status != "failed" {
			t.Fatalf("panic accepted: %s %v", r.Status, err)
		}
		// A panic may leave processes behind, so the claim is quarantined.
		if r.Admission == nil || r.Admission.Outcome != "quarantined" {
			t.Fatalf("admission=%+v", r.Admission)
		}
		entries, err := resourceadmission.Status(context.Background(), resource, digest)
		if err != nil || len(entries) != 1 || entries[0].State != "quarantined" {
			t.Fatalf("claims=%+v err=%v", entries, err)
		}
	})
}

// Two equal-priority runs that each do real work must not hand the claim back
// and forth: they run one after the other in about the sum of their work.
func TestAdmissionEqualPriorityRunsDoNotPingPong(t *testing.T) {
	root, state, resource, _, digest := bridgeFixture(t)
	steps := func(prefix string) []config.CIStep {
		var out []config.CIStep
		for i := 0; i < 5; i++ {
			s := config.CIStep{Name: prefix + string(rune('a'+i)), Run: "sleep 0.4"}
			if i > 0 {
				s.DependsOn = []string{out[i-1].Name}
			}
			out = append(out, s)
		}
		return out
	}
	start := time.Now()
	var wg sync.WaitGroup
	reports := make([]Report, 2)
	for i, name := range []string{"x", "y"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p := config.CIProfile{Timeout: "15s", MaxParallel: 2, Admission: &config.CIAdmission{Class: "heavy", Priority: "background", WaitTimeout: "15s"}, Steps: steps(name)}
			reports[i], _ = Run(context.Background(), Options{Root: root, ProfileName: "full", StateDir: filepath.Join(state, name), lockDirectory: filepath.Join(state, name), resourceDirectory: resource, Env: []string{"DVA_RESOURCE_POLICY=" + filepath.Join(root, "policy.json")}, Profile: p})
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)
	for i, r := range reports {
		if r.Status != "succeeded" || r.Admission == nil || len(r.Admission.Yields) != 0 {
			t.Fatalf("run %d: status=%s admission=%+v err=%s", i, r.Status, r.Admission, r.Error)
		}
	}
	// Work is 2 x 5 x 0.4s = 4s run back to back; allow scheduling slack.
	if elapsed > 9*time.Second {
		t.Fatalf("equal-priority runs took %s for 4s of work", elapsed)
	}
	assertNoClaims(t, resource, digest)
}
