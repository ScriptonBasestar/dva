package cirun

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
	"github.com/ScriptonBasestar/dva/internal/resourceadmission"
)

func outputOrDiscard(w io.Writer) io.Writer {
	if w == nil {
		return io.Discard
	}
	return w
}
func warn(ctx context.Context, w io.Writer, d time.Duration, id string) {
	select {
	case <-time.After(d):
		_, _ = fmt.Fprintf(w, "dva ci %s still running after %s\n", id, d)
	case <-ctx.Done():
	}
}

type lockedWriter struct {
	mu  sync.Mutex
	w   io.Writer
	err error
}

type progressIntervalKey struct{}

func (w *lockedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	n, err := w.w.Write(p)
	w.err = errors.Join(w.err, err)
	return n, err
}

func execute(ctx context.Context, root string, p config.CIProfile, inherited []string, out io.Writer, checkpoint func([]StepReport) error) ([]StepReport, error) {
	n := len(p.Steps)
	reports := make([]StepReport, n)
	byName := map[string]int{}
	for i, s := range p.Steps {
		if s.Name == "" || (s.Run == "" && s.OCI == nil) {
			return reports, errors.New("ci steps require name and run")
		}
		if _, ok := byName[s.Name]; ok {
			return reports, fmt.Errorf("duplicate ci step %q", s.Name)
		}
		byName[s.Name] = i
		reports[i] = StepReport{Name: s.Name, Status: "pending"}
	}
	deps := make([][]int, n)
	for i, s := range p.Steps {
		for _, name := range s.DependsOn {
			j, ok := byName[name]
			if !ok {
				return reports, fmt.Errorf("step %q depends on unknown step %q", s.Name, name)
			}
			deps[i] = append(deps[i], j)
		}
	}
	max := p.MaxParallel
	if max < 1 {
		max = 1
	}
	if max > n {
		max = n
	}
	workers, source := toolWorkerBudget(p, availableCPUs())
	if limit, err := strconv.Atoi(environmentValue(inherited, "DVA_RESOURCE_WORKER_LIMIT")); err == nil && limit > 0 && workers > limit {
		workers, source = limit, "admission"
	}
	if scope, ok := ctx.Value(compositionRuntimeKey{}).(compositionRuntime); ok && scope.workers != nil {
		workers, source = scope.workers.jobs, scope.workers.source
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	type result struct {
		i   int
		r   StepReport
		err error
	}
	done := make(chan result, n)
	interval := 30 * time.Second
	if v, ok := ctx.Value(progressIntervalKey{}).(time.Duration); ok && v > 0 {
		interval = v
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	started := time.Now()
	running := 0
	complete := 0
	failed := false
	fatal := false
	var stepErr error
	dependencyState := func(i int) (ready, blocked bool) {
		ready = true
		for _, dependency := range deps[i] {
			switch reports[dependency].Status {
			case "succeeded":
				continue
			case "failed", "timed_out", "canceled", "skipped":
				return false, true
			default:
				ready = false
			}
		}
		return ready, false
	}
	markSkipped := func(i int) error {
		reports[i].Status = "skipped"
		reports[i].SkipReason = "dependency"
		complete++
		if err := checkpoint(reports); err != nil {
			return fmt.Errorf("checkpoint skipped step %q: %w", p.Steps[i].Name, err)
		}
		return nil
	}
	start := func(i int) error {
		stepEnv, stepWorkers, stepSource := inherited, workers, source
		if bridge, ok := ctx.Value(hostSlotBridgeKey{}).(*hostSlotBridge); ok && p.Steps[i].HostSlot {
			stepEnv = setCIEnvironment(stepEnv, "DVA_RESOURCE_SUPERVISED", "1")
			if limit := bridge.policy.WorkerLimits["heavy"]; limit > 0 {
				stepWorkers = min(stepWorkers, limit)
				stepSource = "admission"
				stepEnv = setCIEnvironment(stepEnv, "DVA_RESOURCE_WORKER_LIMIT", strconv.Itoa(limit))
			}
		}
		reports[i].Status = "running"
		reports[i].StartedAt = time.Now()
		if p.Steps[i].OCI == nil {
			reports[i].WorkerBudget = workerBudget(ciEnv(stepEnv, p.Steps[i].Environment, stepWorkers), p.Steps[i].Environment, stepEnv, stepWorkers, stepSource)
		}
		if err := checkpoint(reports); err != nil {
			reports[i].Status = "pending"
			reports[i].StartedAt = time.Time{}
			reports[i].WorkerBudget = nil
			return fmt.Errorf("checkpoint started step %q: %w", p.Steps[i].Name, err)
		}
		running++
		_, _ = fmt.Fprintf(out, "ci dispatch step=%s completed=%d/%d\n", p.Steps[i].Name, complete, n)
		startedAt := reports[i].StartedAt
		go func() {
			r, e := runConditionedStep(ctx, root, p.Steps[i], stepEnv, stepWorkers, stepSource, out, startedAt)
			done <- result{i, r, e}
		}()
		return nil
	}
	for complete < n {
		if ctx.Err() != nil {
			fatal = true
		}
		// A failed prerequisite is terminal for every descendant, but it must not
		// stop independent work when fail_fast is explicitly disabled. Repeating
		// the scan propagates skipped state even when a descendant appears first in
		// declaration order.
		for !fatal {
			changed := false
			for i := range p.Steps {
				if reports[i].Status != "pending" {
					continue
				}
				_, blocked := dependencyState(i)
				if !blocked {
					continue
				}
				if err := markSkipped(i); err != nil {
					stepErr = errors.Join(stepErr, err)
					fatal = true
					cancel()
					break
				}
				changed = true
			}
			if !changed {
				break
			}
		}
		for !fatal && running < max {
			next := -1
			for i := range p.Steps {
				ready, blocked := dependencyState(i)
				if reports[i].Status == "pending" && ready && !blocked {
					next = i
					break
				}
			}
			if next < 0 {
				break
			}
			if err := start(next); err != nil {
				stepErr = errors.Join(stepErr, err)
				fatal = true
				cancel()
				break
			}
		}
		if running == 0 {
			if !fatal && complete < n {
				return reports, errors.New("ci step dependency cycle")
			}
			for i := range reports {
				if reports[i].Status == "pending" {
					reports[i].Status = "skipped"
					complete++
				}
			}
			break
		}
		var r result
		select {
		case r = <-done:
		case <-ticker.C:
			var active []string
			for _, report := range reports {
				if report.Status == "running" {
					active = append(active, report.Name)
				}
			}
			_, _ = fmt.Fprintf(out, "ci progress completed=%d/%d running=%d pending=%d elapsed=%s active=%s\n", complete, n, running, n-complete-running, time.Since(started).Round(time.Second), strings.Join(active, ","))
			continue
		}
		running--
		complete++
		reports[r.i] = r.r
		_, _ = fmt.Fprintf(out, "ci completed step=%s status=%s elapsed=%s completed=%d/%d\n", r.r.Name, r.r.Status, r.r.Duration.Round(time.Millisecond), complete, n)
		if err := checkpoint(reports); err != nil {
			stepErr = errors.Join(stepErr, fmt.Errorf("checkpoint completed step %q: %w", p.Steps[r.i].Name, err))
			fatal = true
			cancel()
		}
		if r.err != nil {
			stepErr = errors.Join(stepErr, fmt.Errorf("CI step %q: %w", p.Steps[r.i].Name, r.err))
		}
		if r.err != nil {
			failed = true
			var integrity *reuseIntegrityError
			if p.FailFastEnabled() || errors.As(r.err, &integrity) {
				fatal = true
				cancel()
			}
		}
	}
	if failed || fatal || ctx.Err() != nil {
		for i := range reports {
			if reports[i].Status == "pending" {
				reports[i].Status = "skipped"
			}
		}
		if ctx.Err() != nil {
			return reports, errors.Join(stepErr, ctx.Err())
		}
		return reports, errors.Join(stepErr, errors.New("ci step failed"))
	}
	return reports, nil
}

func runStep(ctx context.Context, root string, step config.CIStep, inherited []string, workers int, source string, out io.Writer, startedAt time.Time) (r StepReport, ret error) {
	r.Name = step.Name
	r.Status = "running"
	r.StartedAt = startedAt
	defer func() {
		r.FinishedAt = time.Now()
		r.Duration = r.FinishedAt.Sub(r.StartedAt)
		if ret == nil {
			r.Status = "succeeded"
		} else if errors.Is(ret, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			r.Status = "timed_out"
		} else if errors.Is(ret, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
			r.Status = "canceled"
		} else {
			r.Status = "failed"
		}
		if ret != nil {
			r.Error = ret.Error()
		}
	}()
	workdir := root
	if step.Workdir != "" {
		workdir = filepath.Join(root, step.Workdir)
		if filepath.IsAbs(step.Workdir) {
			workdir = step.Workdir
		}
		rel, e := filepath.Rel(root, workdir)
		if e != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return r, errors.New("step workdir escapes root")
		}
	}
	canonicalWorkdir, err := filepath.EvalSymlinks(workdir)
	if err != nil {
		return r, fmt.Errorf("canonicalize step workdir: %w", err)
	}
	rel, err := filepath.Rel(root, canonicalWorkdir)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return r, errors.New("step workdir escapes root")
	}
	workdir = canonicalWorkdir
	if step.HostSlot {
		// Take the host-wide slot before the step budget is armed and measure
		// the wait against the pre-budget context, so queueing for the slot
		// never consumes the declared step timeout. The profile deadline and
		// external cancellation still bound the wait itself.
		origin, _ := ctx.Value(hostSlotOriginKey{}).(context.Context)
		if origin == nil {
			origin = ctx
		}
		if bridge, ok := origin.Value(hostSlotBridgeKey{}).(*hostSlotBridge); ok {
			lease, observation, err := bridge.acquire(origin, step.Name, out)
			r.Admission = observation
			if observation.WaitDuration != nil {
				r.HostSlotWait, _ = time.ParseDuration(*observation.WaitDuration)
			}
			if err != nil {
				return r, err
			}
			tracker := &resourceCleanupTracker{}
			startedExecution := time.Now()
			defer func() {
				finishErr := finishHostSlotBridge(lease, tracker)
				duration := time.Since(startedExecution).String()
				observation.ExecutionDuration = &duration
				if finishErr != nil {
					observation.Outcome = "quarantined"
				} else {
					observation.Outcome = "completed"
				}
				ret = errors.Join(ret, finishErr)
			}()
			origin = context.WithValue(origin, resourceCleanupKey{}, tracker)
			_, _ = fmt.Fprintf(out, "ci step %s host_slot=acquired provider=resource waited=%s\n", step.Name, r.HostSlotWait.Round(time.Millisecond))
		} else {
			runtime, _ := origin.Value(compositionRuntimeKey{}).(compositionRuntime)
			release, waited, err := acquireHostSlot(origin, hostSlotState(origin), hostSlotMetadata{
				RunID: runtime.id, PID: os.Getpid(), Root: root, Profile: runtime.profile, Step: step.Name,
			}, step.Name, out, runtime.ancestry)
			r.HostSlotWait = waited
			if err != nil {
				return r, err
			}
			defer release()
		}
		ctx = origin
	}
	stepCtx := ctx
	var cancel context.CancelFunc
	if step.Timeout != "" {
		d, e := time.ParseDuration(step.Timeout)
		if e != nil {
			return r, e
		}
		stepCtx, cancel = context.WithTimeout(ctx, d)
		defer cancel()
	}
	if step.OCI != nil {
		r.OCI, ret = runOCIStep(stepCtx, root, step, out)
		return r, ret
	}
	r.Workdir = workdir
	cmd := exec.Command("/bin/sh", "-eu", "-c", step.Run)
	cmd.Dir = workdir
	cmd.Env = ciEnv(inherited, step.Environment, workers)
	r.WorkerBudget = workerBudget(cmd.Env, step.Environment, inherited, workers, source)
	var childScope *compositionScope
	var lockState string
	if step.ChildCI != "" {
		runtime, ok := ctx.Value(compositionRuntimeKey{}).(compositionRuntime)
		if !ok {
			return r, compositionFailure("scope", "opt-in step requires its owning CI execution")
		}
		deadline, bounded := stepCtx.Deadline()
		if !bounded || !time.Now().Before(deadline) {
			return r, compositionFailure("deadline", "opt-in step requires a live finite deadline")
		}
		if r.WorkerBudget == nil {
			return r, compositionFailure("jobs", "opt-in step requires valid numeric worker limits")
		}
		jobs := max(r.WorkerBudget.ToolJobs, r.WorkerBudget.CargoBuildJobs, r.WorkerBudget.RustTestThreads)
		ancestry := append(append([]compositionFrame(nil), runtime.ancestry...), compositionFrame{RunID: runtime.id, Root: root, Step: step.Name})
		if len(ancestry) > maxCompositionDepth {
			return r, compositionFailure("depth", "maximum composition depth reached")
		}
		grant := compositionGrant{Version: 1, Ancestry: ancestry, Deadline: deadline.UTC().Format(time.RFC3339Nano), JobsLimit: jobs, LockState: runtime.state, Fresh: runtime.fresh}
		grant.WorkerDefaults = map[string]string{}
		for key, source := range map[string]string{"CARGO_BUILD_JOBS": r.WorkerBudget.CargoBuildJobsSource, "RUST_TEST_THREADS": r.WorkerBudget.RustTestThreadsSource} {
			if source != "step.environment" && source != "inherited.environment" {
				grant.WorkerDefaults[key] = environmentValue(cmd.Env, key)
			}
		}
		scope := scopeFor(runtime.state, ancestry[len(ancestry)-1])
		if err := scope.createGrant(grant); err != nil {
			return r, err
		}
		childScope, lockState = &scope, runtime.state
		encoded, err := json.Marshal(grant)
		if err != nil {
			return r, err
		}
		cmd.Env = compositionEnvironment(cmd.Env, string(encoded))
		r.CompositionBudget = &CompositionBudget{Deadline: grant.Deadline, JobsLimit: jobs}
	}
	if r.WorkerBudget != nil {
		_, _ = fmt.Fprintf(out, "ci step %s tool_jobs=%d source=%s cargo_build_jobs=%d source=%s rust_test_threads=%d source=%s dva_ci_jobs=%d\n", step.Name, r.WorkerBudget.ToolJobs, r.WorkerBudget.Source, r.WorkerBudget.CargoBuildJobs, r.WorkerBudget.CargoBuildJobsSource, r.WorkerBudget.RustTestThreads, r.WorkerBudget.RustTestThreadsSource, r.WorkerBudget.DVAJobs)
	}
	var watch *outputWatch
	if step.ChildCI == "" {
		watch = newOutputWatch(out, step.Name, 3*time.Minute, 30*time.Second)
		cmd.Stdout, cmd.Stderr = watch, watch
	} else {
		cmd.Stdout, cmd.Stderr = out, out
	}
	cmd.Stdin = nil
	// A background descendant retaining the output pipes must not keep Wait
	// blocked after its parent exited. The group is cleaned on every wait path.
	cmd.WaitDelay = time.Second
	if err := stepCtx.Err(); err != nil {
		if childScope != nil {
			err = errors.Join(err, childScope.drain(stepCtx, lockState, true, nil))
		}
		return r, err
	}
	if err := startProcess(cmd); err != nil {
		if childScope != nil {
			err = errors.Join(err, childScope.drain(stepCtx, lockState, true, nil))
		}
		return r, err
	}
	if watch != nil {
		watch.start()
		defer func() { r.OutputObservation = watch.stop() }()
	}
	pid := cmd.Process.Pid
	defer observeResourceCleanup(stepCtx, pid)
	var signaledAt time.Time
	signalWork := func() {
		if signaledAt.IsZero() {
			signaledAt = time.Now()
			signalCompositionGroup(pid)
		}
	}
	waited := make(chan error, 1)
	go func() { waited <- cmd.Wait() }()
	select {
	case err := <-waited:
		if childScope != nil {
			err = errors.Join(err, childScope.drain(stepCtx, lockState, err != nil || stepCtx.Err() != nil, signalWork))
		}
		cleanupGroup(pid)
		return r, err
	case <-stepCtx.Done():
		var cleanupErr error
		if childScope != nil {
			cleanupErr = childScope.drain(stepCtx, lockState, true, signalWork)
			finishCompositionGroup(pid, waited, signaledAt)
		} else {
			terminateProcessGroup(pid, waited)
		}
		return r, errors.Join(context.Cause(stepCtx), cleanupErr)
	}
}

// subprojectEnv drops any inherited config.CISubprojectEnv and exports the
// --subproject value only when one was given, so an ambient or env_file value
// can never stand in for the flag. The name is the only thing DVA adds: the
// root's own steps decide what scoping it means, so DVA neither selects task
// cards nor knows the board tool that does.
func subprojectEnv(env []string, subproject string) []string {
	prefix := config.CISubprojectEnv + "="
	out := make([]string, 0, len(env)+1)
	for _, e := range env {
		if !strings.HasPrefix(e, prefix) {
			out = append(out, e)
		}
	}
	if subproject != "" {
		out = append(out, prefix+subproject)
	}
	return out
}

func ciEnv(base []string, add map[string]string, workers int) []string {
	m := map[string]string{}
	for _, e := range base {
		if i := strings.IndexByte(e, '='); i > 0 {
			if e[:i] == config.CICompositionEnv {
				continue
			}
			m[e[:i]] = e[i+1:]
		}
	}
	if workers < 1 {
		workers = 1
	}
	m["DVA_CI_JOBS"] = fmt.Sprint(workers)
	for _, key := range []string{"GOMAXPROCS", "CARGO_BUILD_JOBS", "RUST_TEST_THREADS"} {
		if m[key] == "" {
			if key == "CARGO_BUILD_JOBS" {
				m[key] = fmt.Sprint(workers)
			} else {
				m[key] = "1"
			}
		}
	}
	if !strings.Contains(" "+m["GOFLAGS"]+" ", " -p=") {
		m["GOFLAGS"] = strings.TrimSpace(m["GOFLAGS"] + " -p=1")
	}
	for k, v := range add {
		if k == "DVA_RESOURCE_WORKER_LIMIT" || k == "DVA_RESOURCE_SUPERVISED" || k == "DVA_CI_RUN_ID" || k == "DVA_CI_EVENTS_DIR" || k == "DVA_CI_JOBS" || k == parentRunEnv || k == config.CISubprojectEnv || k == config.CICompositionEnv || k == config.CIFreshEnv {
			continue
		}
		m[k] = v
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, k+"="+m[k])
	}
	if limit, err := strconv.Atoi(m["DVA_RESOURCE_WORKER_LIMIT"]); err == nil && limit > 0 {
		out = resourceadmission.CapWorkerEnvironment(out, limit)
	}
	return out
}

func setCIEnvironment(env []string, key, value string) []string {
	prefix := key + "="
	out := make([]string, 0, len(env)+1)
	for _, entry := range env {
		if !strings.HasPrefix(entry, prefix) {
			out = append(out, entry)
		}
	}
	return append(out, prefix+value)
}

func runID() (string, error) {
	b := make([]byte, 16)
	if _, e := rand.Read(b); e != nil {
		return "", e
	}
	return hex.EncodeToString(b), nil
}
