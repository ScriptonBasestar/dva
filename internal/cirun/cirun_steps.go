package cirun

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
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
		if s.Name == "" || s.Run == "" {
			return reports, errors.New("ci steps require name and run")
		}
		if _, ok := byName[s.Name]; ok {
			return reports, fmt.Errorf("duplicate ci step %q", s.Name)
		}
		byName[s.Name] = i
		reports[i] = StepReport{Name: s.Name, Status: "pending"}
	}
	deps := make([][]int, n)
	children := make([][]int, n)
	left := make([]int, n)
	for i, s := range p.Steps {
		for _, name := range s.DependsOn {
			j, ok := byName[name]
			if !ok {
				return reports, fmt.Errorf("step %q depends on unknown step %q", s.Name, name)
			}
			deps[i] = append(deps[i], j)
			children[j] = append(children[j], i)
		}
		left[i] = len(deps[i])
	}
	max := p.MaxParallel
	if max < 1 {
		max = 1
	}
	if max > n {
		max = n
	}
	workers, source := toolWorkerBudget(p, availableCPUs())
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	type result struct {
		i   int
		r   StepReport
		err error
	}
	done := make(chan result, n)
	running := 0
	complete := 0
	failed := false
	var stepErr error
	start := func(i int) error {
		reports[i].Status = "running"
		reports[i].StartedAt = time.Now()
		reports[i].WorkerBudget = workerBudget(ciEnv(inherited, p.Steps[i].Environment, workers), p.Steps[i].Environment, inherited, workers, source)
		if err := checkpoint(reports); err != nil {
			reports[i].Status = "pending"
			reports[i].StartedAt = time.Time{}
			reports[i].WorkerBudget = nil
			return fmt.Errorf("checkpoint started step %q: %w", p.Steps[i].Name, err)
		}
		running++
		startedAt := reports[i].StartedAt
		go func() {
			r, e := runStep(ctx, root, p.Steps[i], inherited, workers, source, out, startedAt)
			done <- result{i, r, e}
		}()
		return nil
	}
	for complete < n {
		if ctx.Err() != nil {
			failed = true
		}
		for !failed && running < max {
			next := -1
			for i := range p.Steps {
				if reports[i].Status == "pending" && left[i] == 0 {
					next = i
					break
				}
			}
			if next < 0 {
				break
			}
			if err := start(next); err != nil {
				stepErr = errors.Join(stepErr, err)
				failed = true
				cancel()
				break
			}
		}
		if running == 0 {
			if !failed {
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
		r := <-done
		running--
		complete++
		reports[r.i] = r.r
		if err := checkpoint(reports); err != nil {
			stepErr = errors.Join(stepErr, fmt.Errorf("checkpoint completed step %q: %w", p.Steps[r.i].Name, err))
			failed = true
			cancel()
		}
		if r.err != nil {
			stepErr = errors.Join(stepErr, fmt.Errorf("CI step %q: %w", p.Steps[r.i].Name, r.err))
		}
		if r.err != nil && !failed {
			failed = true
			cancel()
		}
		for _, child := range children[r.i] {
			left[child]--
		}
	}
	if failed || ctx.Err() != nil {
		for i := range reports {
			if reports[i].Status == "pending" {
				reports[i].Status = "skipped"
			}
		}
		if ctx.Err() != nil {
			return reports, errors.Join(stepErr, ctx.Err())
		}
		return reports, errors.New("ci step failed")
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
	cmd := exec.Command("/bin/sh", "-eu", "-c", step.Run)
	cmd.Dir = workdir
	cmd.Env = ciEnv(inherited, step.Environment, workers)
	r.WorkerBudget = workerBudget(cmd.Env, step.Environment, inherited, workers, source)
	if r.WorkerBudget != nil {
		_, _ = fmt.Fprintf(out, "ci step %s tool_jobs=%d source=%s cargo_build_jobs=%d source=%s rust_test_threads=%d source=%s dva_ci_jobs=%d\n", step.Name, r.WorkerBudget.ToolJobs, r.WorkerBudget.Source, r.WorkerBudget.CargoBuildJobs, r.WorkerBudget.CargoBuildJobsSource, r.WorkerBudget.RustTestThreads, r.WorkerBudget.RustTestThreadsSource, r.WorkerBudget.DVAJobs)
	}
	cmd.Stdout = out
	cmd.Stderr = out
	cmd.Stdin = nil
	// A background descendant retaining the output pipes must not keep Wait
	// blocked after its parent exited. The group is cleaned on every wait path.
	cmd.WaitDelay = time.Second
	if err := startProcess(cmd); err != nil {
		return r, err
	}
	pid := cmd.Process.Pid
	waited := make(chan error, 1)
	go func() { waited <- cmd.Wait() }()
	select {
	case err := <-waited:
		cleanupGroup(pid)
		return r, err
	case <-stepCtx.Done():
		terminateProcessGroup(pid, waited)
		return r, stepCtx.Err()
	}
}

func ciEnv(base []string, add map[string]string, workers int) []string {
	m := map[string]string{}
	for _, e := range base {
		if i := strings.IndexByte(e, '='); i > 0 {
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
		if k == "DVA_CI_JOBS" || k == parentRunEnv {
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
	return out
}

func runID() (string, error) {
	b := make([]byte, 16)
	if _, e := rand.Read(b); e != nil {
		return "", e
	}
	return hex.EncodeToString(b), nil
}
