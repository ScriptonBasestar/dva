//go:build darwin || linux || freebsd || netbsd || openbsd

package cirun

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

const compositionTestRootEnv = "DVA_COMPOSITION_TEST_ROOT"
const compositionTestStateEnv = "DVA_COMPOSITION_TEST_STATE"

// Re-exec the real Run implementation in a distinct process, with real OS locks
// and process groups. Test-only lock storage is never exposed by production CLI.
func TestCompositionRunProcessHelper(t *testing.T) {
	root := os.Getenv(compositionTestRootEnv)
	if root == "" {
		return
	}
	if err := os.WriteFile(filepath.Join(root, "supervisor.pid"), []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil {
		os.Exit(2)
	}
	data, err := os.ReadFile(filepath.Join(root, "profile.json"))
	if err != nil {
		os.Exit(2)
	}
	var profile config.CIProfile
	if json.Unmarshal(data, &profile) != nil {
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	state := os.Getenv(compositionTestStateEnv)
	_, err = Run(ctx, Options{Root: root, ProfileName: "full", Profile: profile, Env: os.Environ(), StateDir: state, lockDirectory: state, Output: os.Stdout})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(17)
	}
	os.Exit(0)
}

func waitCompositionFile(t *testing.T, path string) string {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(path)
		if err == nil && len(data) > 0 {
			return strings.TrimSpace(string(data))
		}
		if err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("readiness file not produced: %s", path)
	return ""
}

func assertCompositionProcessStopped(t *testing.T, pid string) {
	t.Helper()
	n, err := strconv.Atoi(pid)
	if err != nil || n < 1 {
		t.Fatalf("invalid process identity: %q", pid)
	}
	deadline := time.Now().Add(time.Second)
	for {
		data, err := exec.Command("ps", "-o", "stat=", "-p", pid).Output()
		if err != nil {
			if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 1 {
				t.Fatalf("cannot inspect process %s: %v", pid, err)
			}
			return
		}
		if strings.TrimSpace(string(data)) == "" || strings.HasPrefix(strings.TrimSpace(string(data)), "Z") {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("descendant %s still running: %s", pid, data)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestCompositionRunProcessesSignals(t *testing.T) {
	for _, sig := range []os.Signal{os.Interrupt, syscall.SIGTERM} {
		t.Run(sig.String(), func(t *testing.T) {
			state := compositionTestDir(t)
			leafProfile := compositionProcessProfile("sleep 20 & echo $! > workload.pid; wait", "")
			leafProfile.Timeout = "20s"
			leaf := compositionProcessRoot(t, leafProfile)
			middleProfile := compositionProcessProfile(compositionProcessCommand(leaf), config.CIChildIndependentRoot)
			middleProfile.Timeout = "20s"
			middle := compositionProcessRoot(t, middleProfile)
			parentProfile := compositionProcessProfile(compositionProcessCommand(middle), config.CIChildIndependentRoot)
			parentProfile.Timeout = "20s"
			parent := compositionProcessRoot(t, parentProfile)
			cmd := exec.Command(os.Args[0], "-test.run=^TestCompositionRunProcessHelper$")
			cmd.Env = append(compositionEnvironment(os.Environ(), ""), compositionTestRootEnv+"="+parent, compositionTestStateEnv+"="+state)
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			done := make(chan struct{})
			var waitErr error
			go func() { waitErr = cmd.Wait(); close(done) }()
			t.Cleanup(func() {
				select {
				case <-done:
					return
				default:
				}
				_ = cmd.Process.Signal(syscall.SIGTERM)
				select {
				case <-done:
				case <-time.After(20 * time.Second):
					_ = cmd.Process.Kill()
					<-done
				}
			})
			pid := waitCompositionFile(t, filepath.Join(leaf, "workload.pid"))
			if err := cmd.Process.Signal(sig); err != nil {
				t.Fatal(err)
			}
			select {
			case <-done:
			case <-time.After(18 * time.Second):
				t.Fatal("signal cleanup did not terminate parent")
			}
			if waitErr == nil {
				t.Fatal("signal cancellation reported success")
			}
			assertCompositionProcessStopped(t, pid)
			for _, root := range []string{middle, leaf} {
				assertCompositionProcessStopped(t, waitCompositionFile(t, filepath.Join(root, "supervisor.pid")))
				owner, err := lockOwner(state, "root", root)
				if err != nil || owner != "" {
					t.Fatalf("live lock after signal: root=%s owner=%s err=%v", root, owner, err)
				}
			}
			reports, err := Status(state)
			if err != nil || len(reports) != 3 {
				t.Fatalf("terminal receipts missing: %+v err=%v", reports, err)
			}
			for _, report := range reports {
				if report.Status != "canceled" {
					t.Fatalf("unexpected signal outcome: %+v", report)
				}
			}
		})
	}
}

func compositionProcessRoot(t *testing.T, profile config.CIProfile) string {
	t.Helper()
	root := compositionTestDir(t)
	writeCompositionProfile(t, root, profile)
	return root
}

func writeCompositionProfile(t *testing.T, root string, profile config.CIProfile) {
	t.Helper()
	data, err := json.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "profile.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func compositionProcessCommand(root string) string {
	return compositionTestRootEnv + "=" + shellQuote(root) + " " + shellQuote(os.Args[0]) + " -test.run='^TestCompositionRunProcessHelper$'"
}

func compositionProcessProfile(run, mode string) config.CIProfile {
	return config.CIProfile{Timeout: "5s", MaxParallel: 1, Steps: []config.CIStep{{Name: "check", Run: run, ChildCI: mode}}}
}

func runCompositionProcessParent(t *testing.T, root, state string, profile config.CIProfile) (Report, error) {
	t.Helper()
	return Run(context.Background(), compositionParentOptions(root, state, profile))
}

func compositionParentOptions(root, state string, profile config.CIProfile) Options {
	env := compositionEnvironment(os.Environ(), "")
	// This fixture tests omitted worker declarations, independently of the
	// operator's ambient Cargo/Rust overrides. Explicit cases declare them below.
	filtered := env[:0]
	for _, value := range env {
		if !strings.HasPrefix(value, "CARGO_BUILD_JOBS=") && !strings.HasPrefix(value, "RUST_TEST_THREADS=") {
			filtered = append(filtered, value)
		}
	}
	env = filtered
	env = append(env, compositionTestStateEnv+"="+state)
	return Options{Root: root, ProfileName: "full", Profile: profile, Env: env, StateDir: state, lockDirectory: state}
}

func TestCompositionRunProcessesSuccessAndDefaults(t *testing.T) {
	for _, mode := range []string{"", config.CIChildIndependentRoot} {
		t.Run("mode="+mode, func(t *testing.T) {
			state := compositionTestDir(t)
			child := compositionProcessRoot(t, compositionProcessProfile("printf child", ""))
			parent := compositionTestDir(t)
			r, err := runCompositionProcessParent(t, parent, state, compositionProcessProfile(compositionProcessCommand(child), mode))
			if mode == "" {
				if err == nil {
					t.Fatal("legacy nested CI admitted")
				}
				return
			}
			if err != nil || r.Status != "succeeded" || r.Steps[0].CompositionBudget == nil {
				t.Fatalf("report=%+v err=%v", r, err)
			}
			reports, err := Status(state)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, childReport := range reports {
				if childReport.Root != child {
					continue
				}
				found = true
				if childReport.Status != "succeeded" || childReport.Parent == nil || childReport.Parent.RunID != r.ID || childReport.CompositionDepth != 1 {
					t.Fatalf("child edge=%+v", childReport)
				}
				if childReport.Steps[0].WorkerBudget.Source != "computed(cpu/max_parallel,composition)" {
					t.Fatalf("misattributed child budget: %+v", childReport.Steps[0].WorkerBudget)
				}
			}
			if !found {
				t.Fatal("missing child receipt")
			}
		})
	}
}

func TestCompositionRunProcessesGrandchildAndCycles(t *testing.T) {
	for _, cycle := range []bool{false, true} {
		t.Run(map[bool]string{false: "three roots", true: "indirect cycle"}[cycle], func(t *testing.T) {
			state := compositionTestDir(t)
			parent := compositionTestDir(t)
			leaf := compositionProcessRoot(t, compositionProcessProfile("true", ""))
			if cycle {
				leaf = parent
			}
			child := compositionProcessRoot(t, compositionProcessProfile(compositionProcessCommand(leaf), config.CIChildIndependentRoot))
			profile := compositionProcessProfile(compositionProcessCommand(child), config.CIChildIndependentRoot)
			writeCompositionProfile(t, parent, profile)
			r, err := runCompositionProcessParent(t, parent, state, profile)
			if cycle {
				if err == nil {
					t.Fatal("A-B-A cycle admitted")
				}
				data, readErr := os.ReadFile(r.LogPath)
				if readErr != nil || !strings.Contains(string(data), "tool_jobs") {
					t.Fatalf("missing execution evidence: %v", readErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("report=%+v err=%v", r, err)
			}
			reports, err := Status(state)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, result := range reports {
				if result.Root == leaf {
					found = result.CompositionDepth == 2 && result.Status == "succeeded"
				}
			}
			if !found {
				t.Fatal("missing terminal grandchild edge")
			}
		})
	}
}

func TestCompositionRunProcessesSwallowedFailure(t *testing.T) {
	state := compositionTestDir(t)
	child := compositionProcessRoot(t, compositionProcessProfile("exit 9", ""))
	parent := compositionTestDir(t)
	r, err := runCompositionProcessParent(t, parent, state, compositionProcessProfile(compositionProcessCommand(child)+" || :", config.CIChildIndependentRoot))
	if err == nil || r.Status == "succeeded" || !strings.Contains(err.Error(), "child") {
		t.Fatalf("child failure swallowed: report=%+v err=%v", r, err)
	}
}

func TestCompositionRunProcessesDeadline(t *testing.T) {
	state := compositionTestDir(t)
	child := compositionProcessRoot(t, compositionProcessProfile("sleep 20", ""))
	parent := compositionTestDir(t)
	profile := compositionProcessProfile(compositionProcessCommand(child), config.CIChildIndependentRoot)
	profile.Steps[0].Timeout = "300ms"
	started := time.Now()
	r, err := runCompositionProcessParent(t, parent, state, profile)
	if err == nil || r.Steps[0].Status != "timed_out" || time.Since(started) > 5*time.Second {
		t.Fatalf("deadline exceeded cleanup bound: %+v %v elapsed=%s", r, err, time.Since(started))
	}
	entries, readErr := Status(state)
	if readErr != nil {
		t.Fatal(readErr)
	}
	for _, result := range entries {
		if result.Root == child && (result.Status == "running" || result.Status == "stale") {
			t.Fatalf("child not finalized: %+v", result)
		}
		owner, lockErr := lockOwner(state, "root", result.Root)
		if lockErr != nil || owner != "" {
			t.Fatalf("unreleased child lock %s: owner=%s err=%v", result.Root, owner, lockErr)
		}
	}
}

func TestCompositionRunProcessesJobs(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		t.Run(map[bool]string{false: "defaults recomputed", true: "explicit excess refused"}[explicit], func(t *testing.T) {
			state := compositionTestDir(t)
			childProfile := compositionProcessProfile("true", "")
			childProfile.MaxParallel = 2
			childProfile.Steps = append(childProfile.Steps, config.CIStep{Name: "other", Run: "true"})
			if explicit {
				childProfile.Steps[0].Environment = map[string]string{"CARGO_BUILD_JOBS": "2"}
			}
			child := compositionProcessRoot(t, childProfile)
			parent := compositionTestDir(t)
			parentProfile := compositionProcessProfile(compositionProcessCommand(child), config.CIChildIndependentRoot)
			jobs := 2
			parentProfile.ToolJobs = &jobs
			r, err := runCompositionProcessParent(t, parent, state, parentProfile)
			if explicit {
				if err == nil {
					t.Fatal("explicit aggregate jobs excess admitted")
				}
				return
			}
			if err != nil {
				log, readErr := os.ReadFile(r.LogPath)
				t.Fatalf("omitted child jobs rejected: %+v err=%v log=%s readErr=%v", r, err, log, readErr)
			}
			reports, err := Status(state)
			if err != nil {
				t.Fatal(err)
			}
			for _, report := range reports {
				if report.Root == child {
					for _, step := range report.Steps {
						if step.WorkerBudget == nil || step.WorkerBudget.CargoBuildJobs != 1 || step.WorkerBudget.ToolJobs != 1 {
							t.Fatalf("child budget exceeds aggregate ceiling: %+v", step)
						}
					}
				}
			}
		})
	}
}

func TestCompositionRunProcessesSiblingFailure(t *testing.T) {
	state := compositionTestDir(t)
	leaf := compositionProcessRoot(t, compositionProcessProfile("sleep 20 & echo $! > workload.pid; wait", ""))
	middle := compositionProcessRoot(t, compositionProcessProfile(compositionProcessCommand(leaf), config.CIChildIndependentRoot))
	parent := compositionTestDir(t)
	profile := compositionProcessProfile(compositionProcessCommand(middle), config.CIChildIndependentRoot)
	profile.MaxParallel = 2
	profile.Steps = append(profile.Steps, config.CIStep{Name: "fail", Run: "while [ ! -s " + shellQuote(filepath.Join(leaf, "workload.pid")) + " ]; do sleep 0.01; done; exit 9"})
	r, err := runCompositionProcessParent(t, parent, state, profile)
	if err == nil || r.Status != "failed" {
		t.Fatalf("sibling failure lost: %+v %v", r, err)
	}
	assertCompositionProcessStopped(t, waitCompositionFile(t, filepath.Join(leaf, "workload.pid")))
	for _, root := range []string{middle, leaf} {
		assertCompositionProcessStopped(t, waitCompositionFile(t, filepath.Join(root, "supervisor.pid")))
		owner, lockErr := lockOwner(state, "root", root)
		if lockErr != nil || owner != "" {
			t.Fatalf("sibling cancellation lock retained: %s %s %v", root, owner, lockErr)
		}
	}
}

func TestCompositionRunProcessesBackgroundChildAndDescendant(t *testing.T) {
	state := compositionTestDir(t)
	// Redirect the intentional background workload so this success case does not
	// trigger the existing native WaitDelay pipe-retention failure contract.
	child := compositionProcessRoot(t, compositionProcessProfile("sleep 20 > /dev/null 2>&1 & echo $! > workload.pid; sleep 0.2", ""))
	parent := compositionTestDir(t)
	command := compositionProcessCommand(child) + " > /dev/null 2>&1 & while [ ! -s " + shellQuote(filepath.Join(child, "workload.pid")) + " ]; do sleep 0.01; done"
	r, err := runCompositionProcessParent(t, parent, state, compositionProcessProfile(command, config.CIChildIndependentRoot))
	if err != nil || r.Status != "succeeded" {
		t.Fatalf("parent killed a registered background child before its natural end: %+v %v", r, err)
	}
	assertCompositionProcessStopped(t, waitCompositionFile(t, filepath.Join(child, "workload.pid")))
	assertCompositionProcessStopped(t, waitCompositionFile(t, filepath.Join(child, "supervisor.pid")))
}

func TestCompositionCancellationStopsOuterWorkBeforeChildDrain(t *testing.T) {
	state := compositionTestDir(t)
	// This workload needs the existing 750ms TERM-to-KILL grace, exposing an
	// incorrect implementation which leaves the outer step working while draining.
	child := compositionProcessRoot(t, compositionProcessProfile("trap '' TERM; printf ready > ready; sleep 20", ""))
	parent := compositionTestDir(t)
	marker := filepath.Join(parent, "work-after-cancel")
	command := compositionProcessCommand(child) + " > /dev/null 2>&1 & while [ ! -s " + shellQuote(filepath.Join(child, "ready")) + " ]; do sleep 0.01; done; (sleep 0.3; touch " + shellQuote(marker) + ") & printf READY; wait"
	profile := compositionProcessProfile(command, config.CIChildIndependentRoot)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ready := &scopedReadyWriter{ready: make(chan struct{})}
	watchDone := make(chan struct{})
	go func() {
		defer close(watchDone)
		select {
		case <-ready.ready:
			cancel()
		case <-ctx.Done():
		}
	}()
	opts := compositionParentOptions(parent, state, profile)
	opts.Output = ready
	r, err := Run(ctx, opts)
	cancel()
	<-watchDone
	select {
	case <-ready.ready:
	default:
		t.Fatal("parent/child readiness handshake never completed")
	}
	if err == nil || r.Status != "canceled" {
		t.Fatalf("readiness-triggered cancellation not preserved: %+v %v", r, err)
	}
	if _, err := os.Stat(filepath.Join(child, "ready")); err != nil {
		t.Fatalf("TERM-ignoring child never reached readiness; cleanup regression was not exercised: %v", err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("outer workload continued during child cleanup: %v", err)
	}
}

func TestCompositionRunProcessesSharedResourceBusy(t *testing.T) {
	state := compositionTestDir(t)
	childProfile := compositionProcessProfile("touch forbidden", "")
	childProfile.Locks = []string{"composition.fixture.shared"}
	child := compositionProcessRoot(t, childProfile)
	parent := compositionTestDir(t)
	profile := compositionProcessProfile(compositionProcessCommand(child), config.CIChildIndependentRoot)
	profile.Locks = append([]string(nil), childProfile.Locks...)
	r, err := runCompositionProcessParent(t, parent, state, profile)
	if err == nil {
		t.Fatal("shared resource became reentrant")
	}
	log, readErr := os.ReadFile(r.LogPath)
	if readErr != nil || !strings.Contains(string(log), "resource") {
		t.Fatalf("resource conflict hidden: %s %v", log, readErr)
	}
	if _, err := os.Stat(filepath.Join(child, "forbidden")); !os.IsNotExist(err) {
		t.Fatalf("rejected child command ran: %v", err)
	}
	owner, lockErr := lockOwner(state, "root", child)
	if lockErr != nil || owner != "" {
		t.Fatalf("partial admission leaked child lock: %s %v", owner, lockErr)
	}
}

func TestCompositionRunProcessesDirectCycle(t *testing.T) {
	state, root := compositionTestDir(t), compositionTestDir(t)
	profile := compositionProcessProfile(compositionProcessCommand(root), config.CIChildIndependentRoot)
	writeCompositionProfile(t, root, profile)
	r, err := runCompositionProcessParent(t, root, state, profile)
	if err == nil {
		t.Fatal("A-A direct cycle admitted")
	}
	log, readErr := os.ReadFile(r.LogPath)
	if readErr != nil || !strings.Contains(string(log), "composition_cycle") {
		t.Fatalf("direct cycle was not classified before lock acquisition: %s %v", log, readErr)
	}
}

func TestCompositionRunProcessesConcurrentChildrenBusy(t *testing.T) {
	state := compositionTestDir(t)
	first := compositionProcessRoot(t, compositionProcessProfile("touch ready; sleep 20", ""))
	second := compositionProcessRoot(t, compositionProcessProfile("touch forbidden", ""))
	parent := compositionTestDir(t)
	secondLog := filepath.Join(parent, "second.log")
	command := compositionProcessCommand(first) + " > /dev/null 2>&1 & while [ ! -f " + shellQuote(filepath.Join(first, "ready")) + " ]; do sleep 0.01; done; " + compositionProcessCommand(second) + " > " + shellQuote(secondLog) + " 2>&1; wait"
	r, err := runCompositionProcessParent(t, parent, state, compositionProcessProfile(command, config.CIChildIndependentRoot))
	if err == nil || r.Status == "succeeded" {
		t.Fatal("concurrent child grants overcommitted worker ceiling")
	}
	log, readErr := os.ReadFile(secondLog)
	if readErr != nil || !strings.Contains(string(log), "composition_child") {
		t.Fatalf("concurrent rejection was not a verified busy result: %s %v", log, readErr)
	}
	if _, err := os.Stat(filepath.Join(second, "forbidden")); !os.IsNotExist(err) {
		t.Fatalf("rejected second command ran: %v", err)
	}
	for _, root := range []string{first, second} {
		owner, lockErr := lockOwner(state, "root", root)
		if lockErr != nil || owner != "" {
			t.Fatalf("child root lock retained: %s %s %v", root, owner, lockErr)
		}
	}
}
