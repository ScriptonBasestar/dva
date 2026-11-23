package cirun

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
	"github.com/ScriptonBasestar/dva/internal/resourceadmission"
)

// noClaimDeadline bounds a Run that must finish without claiming the heavy slot
// the test holds. It is only a hang guard, not a speed budget: a claim is caught
// by the step's assertion (it waits on admission and reports timed_out, not
// reused or condition-false), so the deadline must leave room for git and sh
// children on a loaded host (ISSUE-627: a 1s deadline killed `git` under load).
const noClaimDeadline = 30 * time.Second

// noClaimContext returns a context for a Run that must not claim the held slot.
func noClaimContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), noClaimDeadline)
	t.Cleanup(cancel)
	return ctx
}

// failNoClaim reports a no-claim Run failure and names the likely cause.
func failNoClaim(t *testing.T, ctx context.Context, report any, err error) {
	t.Helper()
	if ctx.Err() != nil {
		t.Fatalf("run hit the %s hang guard while the heavy slot was held: report=%+v err=%v", noClaimDeadline, report, err)
	}
	t.Fatalf("run did not skip the held heavy slot (a timed_out status means it waited on admission): report=%+v err=%v", report, err)
}

func bridgeFixture(t *testing.T) (string, string, string, resourceadmission.Policy, string) {
	t.Helper()
	t.Setenv("DVA_RESOURCE_SUPERVISED", "")
	t.Setenv(parentRunEnv, "")
	root, state, resource := t.TempDir(), t.TempDir(), filepath.Join(t.TempDir(), "resource")
	policy := filepath.Join(root, "policy.json")
	data := `{"capacity":4,"max_running":3,"max_heavy":1,"weights":{"heavy":2,"light":1},"aging_interval":"5m","worker_limits":{"heavy":2,"light":1}}`
	if err := os.WriteFile(policy, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	p, digest, err := resourceadmission.ReadPolicy(policy)
	if err != nil {
		t.Fatal(err)
	}
	return root, state, resource, p, digest
}

func TestHostBridgePolicyAndNesting(t *testing.T) {
	root, state, resource, _, _ := bridgeFixture(t)
	policy := filepath.Join(root, "policy.json")
	step := config.CIStep{Name: "one", Run: "true", HostSlot: true}
	base := Options{Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, resourceDirectory: resource, Env: []string{"DVA_RESOURCE_POLICY=" + policy}, Profile: profile(step)}
	for _, tc := range []struct {
		name   string
		change func(*Options)
		want   string
	}{
		{"empty policy", func(o *Options) { o.Env = []string{"DVA_RESOURCE_POLICY="} }, "explicitly empty"},
		{"missing policy", func(o *Options) { o.Env = []string{"DVA_RESOURCE_POLICY=/not-found"} }, "no such file"},
		{"supervised", func(o *Options) { o.Env = append(o.Env, "DVA_RESOURCE_SUPERVISED=1") }, "nested"},
		{"parent marker", func(o *Options) { o.Env = append(o.Env, parentRunEnv+"=x") }, "nested"},
		{"child ci", func(o *Options) { o.Profile.Steps[0].ChildCI = config.CIChildIndependentRoot }, "child_ci"},
		{"profile and step", func(o *Options) {
			o.Profile.Admission = &config.CIAdmission{Class: "heavy", Priority: "normal", WaitTimeout: "1s"}
		}, "cannot be combined"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o := base
			o.Profile.Steps = append([]config.CIStep(nil), base.Profile.Steps...)
			tc.change(&o)
			_, err := Run(context.Background(), o)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v want %s", err, tc.want)
			}
		})
	}
	for _, n := range []int{0, 2} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			data := fmt.Sprintf(`{"capacity":4,"max_running":3,"max_heavy":%d,"weights":{"heavy":2,"light":1},"aging_interval":"5m"}`, n)
			if err := os.WriteFile(policy, []byte(data), 0600); err != nil {
				t.Fatal(err)
			}
			_, err := Run(context.Background(), base)
			if err == nil || !strings.Contains(err.Error(), "max_heavy=1") {
				t.Fatalf("got %v", err)
			}
		})
	}
}

func TestHostBridgeHeavyMutualExclusionAndLightBackfill(t *testing.T) {
	root, state, resource, p, digest := bridgeFixture(t)
	acquireCtx, cancelAcquire := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelAcquire()
	held, err := resourceadmission.Acquire(acquireCtx, resource, p, digest, "heavy", "normal")
	if err != nil {
		t.Fatal(err)
	}
	light, err := resourceadmission.Acquire(acquireCtx, resource, p, digest, "light", "normal")
	if err != nil {
		t.Fatalf("light backfill: %v", err)
	}
	if err = light.Complete(context.Background()); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	marker := filepath.Join(root, "started")
	go func() {
		_, e := Run(context.Background(), Options{Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, resourceDirectory: resource, Env: []string{"DVA_RESOURCE_POLICY=" + filepath.Join(root, "policy.json")}, Profile: profile(config.CIStep{Name: "one", Run: "touch " + marker, HostSlot: true})})
		done <- e
	}()
	time.Sleep(200 * time.Millisecond)
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("host slot ran concurrently with heavy provider")
	}
	if err = held.Complete(context.Background()); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("bridge failed to proceed")
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatal(err)
	}
}

func TestHostBridgeConditionFalseDoesNotClaim(t *testing.T) {
	root, state, resource, p, digest := bridgeFixture(t)
	acquireCtx, cancelAcquire := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelAcquire()
	held, err := resourceadmission.Acquire(acquireCtx, resource, p, digest, "heavy", "normal")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = held.Complete(context.Background()) }()
	ctx := noClaimContext(t)
	step := config.CIStep{Name: "one", Run: "exit 1", HostSlot: true, Condition: &config.CICondition{Run: `printf '{"selected":false}'`, Timeout: "1s"}}
	r, err := Run(ctx, Options{Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, resourceDirectory: resource, Env: []string{"DVA_RESOURCE_POLICY=" + filepath.Join(root, "policy.json")}, Profile: profile(step), Output: io.Discard})
	if err != nil || r.Steps[0].SkipReason != "condition-false" {
		failNoClaim(t, ctx, r, err)
	}
}

func TestHostBridgeWorkerCapAndQuarantine(t *testing.T) {
	root, state, resource, p, digest := bridgeFixture(t)
	step := config.CIStep{Name: "one", Run: `test "$CARGO_BUILD_JOBS" -eq 2 && test "$RUST_TEST_THREADS" -eq 2 && test "$DVA_CI_JOBS" -le 2`, HostSlot: true, Environment: map[string]string{"CARGO_BUILD_JOBS": "8", "RUST_TEST_THREADS": "8"}}
	opts := Options{Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, resourceDirectory: resource, Env: []string{"DVA_RESOURCE_POLICY=" + filepath.Join(root, "policy.json")}, Profile: profile(step)}
	r, err := Run(context.Background(), opts)
	if err != nil {
		t.Fatalf("report=%+v err=%v", r, err)
	}
	if r.Steps[0].WorkerBudget == nil || r.Steps[0].WorkerBudget.CargoBuildJobs != 2 || r.Steps[0].WorkerBudget.RustTestThreads != 2 {
		t.Fatalf("budget=%+v", r.Steps[0].WorkerBudget)
	}
	ctx := context.WithValue(context.Background(), resourceCleanupProbeKey{}, func(int) bool { return false })
	r, err = Run(ctx, opts)
	if err == nil || r.Status != "failed" {
		t.Fatalf("unsafe cleanup: report=%+v err=%v", r, err)
	}
	entries, err := resourceadmission.Status(context.Background(), resource, digest)
	if err != nil || len(entries) != 1 || entries[0].State != "quarantined" {
		t.Fatalf("entries=%+v err=%v policy=%+v", entries, err, p)
	}
}

func TestHostBridgeQueueWaitSeparateFromExecutionTimeout(t *testing.T) {
	root, state, resource, p, digest := bridgeFixture(t)
	acquireCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	held, err := resourceadmission.Acquire(acquireCtx, resource, p, digest, "heavy", "normal")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct {
		r   Report
		err error
	}, 1)
	go func() {
		r, e := Run(context.Background(), Options{Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, resourceDirectory: resource, Env: []string{"DVA_RESOURCE_POLICY=" + filepath.Join(root, "policy.json")}, Profile: profile(config.CIStep{Name: "one", Run: "true", Timeout: "100ms", HostSlot: true})})
		done <- struct {
			r   Report
			err error
		}{r, e}
	}()
	time.Sleep(500 * time.Millisecond)
	if err := held.Complete(context.Background()); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-done:
		if got.err != nil || got.r.Steps[0].HostSlotWait < 150*time.Millisecond {
			t.Fatalf("report=%+v err=%v", got.r, got.err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting")
	}
}

func TestHostBridgeReuseHitDoesNotClaim(t *testing.T) {
	root, state, resource, p, digest := bridgeFixture(t)
	if err := runGit(root, "init"); err != nil {
		t.Skipf("git unavailable: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "input"), []byte("one"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("count\npolicy.json\n"), 0600); err != nil {
		t.Fatal(err)
	}
	commitAll(t, root)
	step := config.CIStep{Name: "cached", Run: "printf x >> count", HostSlot: true, Reuse: &config.CIReuse{Inputs: []string{"input"}, Tools: []string{"sh"}, Outputs: []string{"count"}, MaxAge: "1h"}}
	opts := Options{Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, resourceDirectory: resource, Env: []string{"PATH=" + os.Getenv("PATH"), "DVA_RESOURCE_POLICY=" + filepath.Join(root, "policy.json")}, Profile: profile(step)}
	r, err := Run(context.Background(), opts)
	if err != nil || r.Steps[0].Reuse == nil || r.Steps[0].Reuse.Disposition != "executed" {
		t.Fatalf("first=%+v err=%v", r, err)
	}
	acquireCtx, cancelAcquire := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelAcquire()
	held, err := resourceadmission.Acquire(acquireCtx, resource, p, digest, "heavy", "normal")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = held.Complete(context.Background()) }()
	ctx := noClaimContext(t)
	r, err = Run(ctx, opts)
	if err != nil || r.Steps[0].Reuse == nil || r.Steps[0].Reuse.Disposition != "reused" {
		failNoClaim(t, ctx, r, err)
	}
}

func TestHostBridgeReuseTracksEffectiveWorkerCap(t *testing.T) {
	root, state, resource, _, _ := bridgeFixture(t)
	if err := runGit(root, "init"); err != nil {
		t.Skipf("git unavailable: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "input"), []byte("one"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("count\npolicy.json\n"), 0600); err != nil {
		t.Fatal(err)
	}
	commitAll(t, root)
	step := config.CIStep{Name: "cached", Run: `printf '%s\n' "$CARGO_BUILD_JOBS" >> count`, HostSlot: true,
		Environment: map[string]string{"CARGO_BUILD_JOBS": "8"},
		Condition:   &config.CICondition{Run: `test "$CARGO_BUILD_JOBS" -eq "$DVA_RESOURCE_WORKER_LIMIT"; printf '{"selected":true}'`, Timeout: "1s"},
		Reuse:       &config.CIReuse{Inputs: []string{"input"}, Tools: []string{"sh"}, Outputs: []string{"count"}, MaxAge: "1h"}}
	policy := filepath.Join(root, "policy.json")
	opts := Options{Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, resourceDirectory: resource, Env: []string{"PATH=" + os.Getenv("PATH"), "DVA_RESOURCE_POLICY=" + policy}, Profile: profile(step)}
	for i, limit := range []int{2, 4, 4} {
		data := fmt.Sprintf(`{"capacity":4,"max_running":3,"max_heavy":1,"weights":{"heavy":2,"light":1},"aging_interval":"5m","worker_limits":{"heavy":%d,"light":1}}`, limit)
		if err := os.WriteFile(policy, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		r, err := Run(context.Background(), opts)
		want := "executed"
		if i == 2 {
			want = "reused"
		}
		if err != nil || r.Steps[0].Reuse == nil || r.Steps[0].Reuse.Disposition != want {
			t.Fatalf("limit %d: report=%+v err=%v", limit, r, err)
		}
	}
	data, err := os.ReadFile(filepath.Join(root, "count"))
	if err != nil || string(data) != "2\n4\n" {
		t.Fatalf("child environments: %q, %v", data, err)
	}
}

func TestHostBridgeReuseTracksSupervisionMarker(t *testing.T) {
	root, state, resource, _, _ := bridgeFixture(t)
	if err := runGit(root, "init"); err != nil {
		t.Skipf("git unavailable: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "input"), []byte("one"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("count\npolicy.json\n"), 0600); err != nil {
		t.Fatal(err)
	}
	policy := filepath.Join(root, "policy.json")
	if err := os.WriteFile(policy, []byte(`{"capacity":4,"max_running":3,"max_heavy":1,"weights":{"heavy":2,"light":1},"aging_interval":"5m"}`), 0600); err != nil {
		t.Fatal(err)
	}
	commitAll(t, root)
	step := config.CIStep{Name: "cached", Run: `printf '%s\n' "${DVA_RESOURCE_SUPERVISED:-absent}" >> count`, HostSlot: true,
		Reuse: &config.CIReuse{Inputs: []string{"input"}, Tools: []string{"sh"}, Outputs: []string{"count"}, Environment: []string{"DVA_RESOURCE_SUPERVISED"}, MaxAge: "1h"}}
	opts := Options{Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, resourceDirectory: resource, Env: []string{"PATH=" + os.Getenv("PATH"), "DVA_RESOURCE_POLICY=" + policy}, Profile: profile(step)}
	for i := range 3 {
		if i == 1 {
			opts.Env = []string{"PATH=" + os.Getenv("PATH")}
		}
		r, err := Run(context.Background(), opts)
		want := "executed"
		if i == 2 {
			want = "reused"
		}
		if err != nil || r.Steps[0].Reuse == nil || r.Steps[0].Reuse.Disposition != want {
			t.Fatalf("run %d: report=%+v err=%v", i, r, err)
		}
	}
	data, err := os.ReadFile(filepath.Join(root, "count"))
	if err != nil || string(data) != "1\nabsent\n" {
		t.Fatalf("child supervision markers: %q, %v", data, err)
	}
}

func TestHostBridgeExternalSupervisorWithoutPolicyFails(t *testing.T) {
	root, state, resource, _, _ := bridgeFixture(t)
	t.Setenv("DVA_RESOURCE_SUPERVISED", "1")
	_, err := Run(context.Background(), Options{Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, resourceDirectory: resource, Env: []string{}, Profile: profile(config.CIStep{Name: "one", Run: "true", HostSlot: true})})
	if err == nil || !strings.Contains(err.Error(), "nested") {
		t.Fatalf("got %v", err)
	}
}

func TestHostBridgeExecutionTimeoutAndQueueDeadline(t *testing.T) {
	root, state, resource, p, digest := bridgeFixture(t)
	opts := Options{Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, resourceDirectory: resource, Env: []string{"DVA_RESOURCE_POLICY=" + filepath.Join(root, "policy.json")}, Profile: profile(config.CIStep{Name: "one", Run: "sleep 2", Timeout: "100ms", HostSlot: true})}
	r, err := Run(context.Background(), opts)
	if err == nil || r.Steps[0].Status != "timed_out" {
		t.Fatalf("execution report=%+v err=%v", r, err)
	}
	acquireCtx, cancelAcquire := context.WithTimeout(context.Background(), time.Second)
	defer cancelAcquire()
	held, err := resourceadmission.Acquire(acquireCtx, resource, p, digest, "heavy", "normal")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = held.Complete(context.Background()) }()
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()
	r, err = Run(ctx, opts)
	if err == nil || r.Steps[0].Status != "timed_out" {
		t.Fatalf("queue report=%+v err=%v", r, err)
	}
}
