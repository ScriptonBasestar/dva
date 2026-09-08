// Package cirun runs a CI profile without changing the caller's process state.
// It provides conservative Go and Rust worker defaults through environment
// variables. Tools which ignore these variables own their internal concurrency.
package cirun

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

var ErrBusy = errors.New("a conflicting dva ci run is active")

// Conflict identifies the lock preventing admission. RunID is omitted when unknown.
type Conflict struct {
	Kind  string `json:"kind"`
	Key   string `json:"key"`
	RunID string `json:"run_id,omitempty"`
}

type BusyError struct {
	ActiveRunID string
	Conflict    Conflict
}

func (e *BusyError) Error() string {
	message := fmt.Sprintf("%s (%s %q)", ErrBusy, e.Conflict.Kind, e.Conflict.Key)
	if e.ActiveRunID != "" {
		message += fmt.Sprintf(" (active run %s)", e.ActiveRunID)
	}
	return message
}
func (e *BusyError) Unwrap() error { return ErrBusy }

type Options struct {
	Root, ProfileName, StateDir string
	Profile                     config.CIProfile
	Env                         []string
	Output                      io.Writer
	lockDirectory               string // test seam; production callers cannot set it.
}

type StepReport struct {
	Name       string        `json:"name"`
	Status     string        `json:"status"`
	Error      string        `json:"error,omitempty"`
	StartedAt  time.Time     `json:"started_at"`
	FinishedAt time.Time     `json:"finished_at"`
	Duration   time.Duration `json:"duration"`
}

type Attestation struct {
	Available bool   `json:"available"`
	Before    string `json:"before,omitempty"`
	After     string `json:"after,omitempty"`
}

type Report struct {
	ID, Root, Profile, Status, LogPath string
	Error                              string
	StartedAt, FinishedAt              time.Time
	Duration                           time.Duration
	Steps                              []StepReport
	Attestation                        Attestation
	Conflict                           *Conflict
}

func (r Report) MarshalJSON() ([]byte, error) {
	type wire struct {
		ID          string       `json:"id"`
		Root        string       `json:"root"`
		Profile     string       `json:"profile"`
		Status      string       `json:"status"`
		LogPath     string       `json:"log_path"`
		Error       string       `json:"error,omitempty"`
		StartedAt   time.Time    `json:"started_at"`
		FinishedAt  time.Time    `json:"finished_at"`
		Duration    string       `json:"duration"`
		Steps       []StepReport `json:"steps"`
		Attestation Attestation  `json:"attestation"`
		Conflict    *Conflict    `json:"conflict,omitempty"`
	}
	return json.Marshal(wire{r.ID, r.Root, r.Profile, r.Status, r.LogPath, r.Error, r.StartedAt, r.FinishedAt, r.Duration.String(), r.Steps, r.Attestation, r.Conflict})
}

func (r *Report) UnmarshalJSON(data []byte) error {
	type wire struct {
		ID          string       `json:"id"`
		Root        string       `json:"root"`
		Profile     string       `json:"profile"`
		Status      string       `json:"status"`
		LogPath     string       `json:"log_path"`
		Error       string       `json:"error,omitempty"`
		StartedAt   time.Time    `json:"started_at"`
		FinishedAt  time.Time    `json:"finished_at"`
		Duration    string       `json:"duration"`
		Steps       []StepReport `json:"steps"`
		Attestation Attestation  `json:"attestation"`
		Conflict    *Conflict    `json:"conflict,omitempty"`
	}
	var in wire
	if err := json.Unmarshal(data, &in); err != nil {
		return err
	}
	d, err := time.ParseDuration(in.Duration)
	if err != nil {
		return err
	}
	*r = Report{ID: in.ID, Root: in.Root, Profile: in.Profile, Status: in.Status, LogPath: in.LogPath, Error: in.Error, StartedAt: in.StartedAt, FinishedAt: in.FinishedAt, Duration: d, Steps: in.Steps, Attestation: in.Attestation, Conflict: in.Conflict}
	return nil
}

func stateDirectory(dir string) (string, error) {
	if dir != "" {
		return dir, nil
	}
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	// Do not use HOME/XDG_CACHE_HOME: sessions must share UID-owned resource locks.
	return filepath.Join(u.HomeDir, ".local", "state", "dva", "ci"), nil
}

func Run(parent context.Context, opts Options) (report Report, retErr error) {
	defer func() {
		if retErr != nil {
			if report.Status == "" || report.Status == "running" || report.Status == "succeeded" {
				report.Status = "failed"
			}
			report.Error = retErr.Error()
		}
	}()
	if opts.ProfileName == "" {
		return report, errors.New("ci profile name is required")
	}
	declared := &config.Config{CI: &config.CIConfig{Profiles: map[string]config.CIProfile{opts.ProfileName: opts.Profile}}}
	resolved, resolveErr := declared.ResolveCIProfile(opts.ProfileName)
	if resolveErr != nil {
		return report, resolveErr
	}
	opts.Profile = resolved
	ctx := parent
	if opts.Profile.Timeout != "" {
		d, err := time.ParseDuration(opts.Profile.Timeout)
		if err != nil {
			return report, fmt.Errorf("profile timeout: %w", err)
		}
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(parent, d)
		defer cancel()
	}
	if opts.Root == "" {
		return report, errors.New("ci root is required")
	}
	root, err := filepath.EvalSymlinks(opts.Root)
	if err != nil {
		return report, fmt.Errorf("canonicalize ci root: %w", err)
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return report, err
	}
	if len(opts.Profile.Steps) == 0 {
		return report, errors.New("ci profile has no steps")
	}
	state, err := stateDirectory(opts.StateDir)
	if err != nil {
		return report, err
	}
	if err := os.MkdirAll(state, 0o700); err != nil {
		return report, err
	}
	id, err := runID()
	if err != nil {
		return report, err
	}
	report = Report{ID: id, Root: root, Profile: opts.ProfileName, Status: "running", StartedAt: time.Now(), LogPath: filepath.Join(state, "logs", id+".log")}
	if err := os.MkdirAll(filepath.Dir(report.LogPath), 0o700); err != nil {
		return report, err
	}
	// Lock state is shared even when receipts use a custom directory.
	lockState, err := stateDirectory("")
	if err != nil {
		return report, err
	}
	if opts.lockDirectory != "" {
		lockState = opts.lockDirectory
	}
	parentValue, present := os.LookupEnv(parentRunEnv)
	if !present {
		parentValue = environmentValue(opts.Env, parentRunEnv)
	}
	err = checkParent(lockState, parentValue)
	var locks heldLocks
	if err == nil {
		locks, err = acquire(lockState, root, id, opts.Profile.Locks...)
	}
	if err != nil {
		if busy, ok := errors.AsType[*BusyError](err); ok {
			// This identifies an existing owner, not a new execution of this profile.
			report = Report{ID: busy.ActiveRunID, Status: "busy", Error: err.Error(), Conflict: &busy.Conflict}
		}
		return report, err
	}
	defer locks.release()
	log, err := os.OpenFile(report.LogPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return report, err
	}
	closed := false
	defer func() {
		if !closed {
			retErr = errors.Join(retErr, log.Close())
		}
	}()
	writer := &lockedWriter{w: io.MultiWriter(log, outputOrDiscard(opts.Output))}
	if err := writeReport(state, report); err != nil {
		return report, fmt.Errorf("record running ci receipt: %w", err)
	}
	warningCtx, stopWarnings := context.WithCancel(ctx)
	defer stopWarnings()
	warnDone := make(chan struct{})
	if opts.Profile.WarnAfter != "" {
		if d, e := time.ParseDuration(opts.Profile.WarnAfter); e == nil {
			go func() { defer close(warnDone); warn(warningCtx, writer, d, id) }()
		} else {
			return report, fmt.Errorf("profile warn_after: %w", e)
		}
	} else {
		close(warnDone)
	}
	report.Attestation, retErr = fingerprint(ctx, root)
	if retErr == nil {
		parentInfo, marshalErr := json.Marshal(parentRun{ID: id, Root: root})
		if marshalErr != nil {
			return report, marshalErr
		}
		env := append(append([]string(nil), opts.Env...), parentRunEnv+"="+string(parentInfo))
		report.Steps, retErr = execute(ctx, root, opts.Profile, env, writer)
	}
	after, attestErr := fingerprint(ctx, root)
	retErr = errors.Join(retErr, attestErr)
	if report.Attestation.Available {
		report.Attestation.After = after.Before
		if report.Attestation.Before != after.Before && retErr == nil {
			report.Status = "stale"
			retErr = errors.New("ci inputs changed during run")
		}
	}
	report.FinishedAt = time.Now()
	report.Duration = report.FinishedAt.Sub(report.StartedAt)
	stopWarnings()
	<-warnDone // no warning writer may outlive the log file.
	retErr = errors.Join(retErr, writer.err)
	if report.Status == "running" {
		if retErr == nil {
			report.Status = "succeeded"
		} else if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			report.Status = "timed_out"
		} else if errors.Is(ctx.Err(), context.Canceled) {
			report.Status = "canceled"
		} else {
			report.Status = "failed"
		}
	}
	if e := log.Close(); e != nil {
		retErr = errors.Join(retErr, fmt.Errorf("close ci log: %w", e))
		report.Status = "failed"
	}
	closed = true
	if retErr != nil {
		report.Error = retErr.Error()
	}
	if e := writeReport(state, report); e != nil {
		if report.Status == "succeeded" {
			report.Status = "failed"
		}
		report.Error = errors.Join(retErr, fmt.Errorf("record ci report: %w", e)).Error()
		retErr = errors.Join(retErr, fmt.Errorf("record ci report: %w", e))
	}
	return report, retErr
}

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

func execute(ctx context.Context, root string, p config.CIProfile, inherited []string, out io.Writer) ([]StepReport, error) {
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
	start := func(i int) {
		running++
		reports[i].Status = "running"
		go func() { r, e := runStep(ctx, root, p.Steps[i], inherited, out); done <- result{i, r, e} }()
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
			start(next)
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
		if r.err != nil && stepErr == nil {
			stepErr = fmt.Errorf("CI step %q: %w", p.Steps[r.i].Name, r.err)
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

func runStep(ctx context.Context, root string, step config.CIStep, inherited []string, out io.Writer) (r StepReport, ret error) {
	r.Name = step.Name
	r.Status = "running"
	r.StartedAt = time.Now()
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
	cmd.Env = ciEnv(inherited, step.Environment)
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

func ciEnv(base []string, add map[string]string) []string {
	m := map[string]string{}
	for _, e := range base {
		if i := strings.IndexByte(e, '='); i > 0 {
			m[e[:i]] = e[i+1:]
		}
	}
	m["DVA_CI_JOBS"] = "1" // each concurrently scheduled step gets one tool budget.
	for _, key := range []string{"GOMAXPROCS", "CARGO_BUILD_JOBS", "RUST_TEST_THREADS"} {
		if m[key] == "" {
			m[key] = "1"
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
func writeReport(state string, r Report) error {
	b, e := json.MarshalIndent(r, "", "  ")
	if e != nil {
		return e
	}
	tmp, e := os.CreateTemp(state, ".report-*")
	if e != nil {
		return e
	}
	name := tmp.Name()
	defer func() { _ = os.Remove(name) }()
	if _, e = tmp.Write(b); e == nil {
		e = tmp.Chmod(0o600)
	}
	if closeErr := tmp.Close(); e == nil {
		e = closeErr
	}
	if e != nil {
		return e
	}
	return os.Rename(name, filepath.Join(state, r.ID+".json"))
}

func Status(dir string) ([]Report, error) {
	state, e := stateDirectory(dir)
	if e != nil {
		return nil, e
	}
	entries, e := os.ReadDir(state)
	if os.IsNotExist(e) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	out := []Report{}
	for _, x := range entries {
		if x.IsDir() || !strings.HasSuffix(x.Name(), ".json") {
			continue
		}
		b, e := os.ReadFile(filepath.Join(state, x.Name()))
		if e != nil {
			return nil, e
		}
		var r Report
		if e = json.Unmarshal(b, &r); e != nil {
			return nil, e
		}
		if r.Status == "running" {
			active, err := receiptActive(state, r.Root, r.ID)
			if err != nil {
				return nil, fmt.Errorf("inspect CI receipt %s: %w", r.ID, err)
			}
			if !active {
				r.Status = "stale"
				r.Error = "running receipt has no active lock"
			}
		}
		if r.Status == "running" {
			r.Duration = time.Since(r.StartedAt)
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartedAt.After(out[j].StartedAt) })
	return out, nil
}
func receiptActive(state, root, id string) (bool, error) {
	dirs := []string{state}
	shared, err := stateDirectory("")
	if err != nil {
		return false, err
	}
	if shared != state {
		dirs = append(dirs, shared)
	}
	for _, dir := range dirs {
		active, err := lockOwner(dir, "root", root)
		if err != nil {
			return false, err
		}
		if active != "" && active == id {
			return true, nil
		}
	}
	return false, nil
}
func ReadLog(dir, id string) ([]byte, error) {
	if !safeID(id) {
		return nil, errors.New("invalid ci run id")
	}
	state, e := stateDirectory(dir)
	if e != nil {
		return nil, e
	}
	return os.ReadFile(filepath.Join(state, "logs", id+".log"))
}
func safeID(id string) bool {
	if len(id) != 32 {
		return false
	}
	for _, c := range id {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

func fingerprint(ctx context.Context, root string) (Attestation, error) {
	probe, e := gitOutput(ctx, root, "rev-parse", "--is-inside-work-tree")
	if e != nil {
		if ctx.Err() != nil {
			return Attestation{}, ctx.Err()
		}
		if strings.Contains(string(probe), "not a git repository") {
			return Attestation{}, nil
		}
		return Attestation{}, fmt.Errorf("probe git inputs: %w: %s", e, probe)
	}
	if strings.TrimSpace(string(probe)) != "true" {
		return Attestation{}, errors.New("CI requires a Git working tree, not a bare repository")
	}
	b, e := gitOutput(ctx, root, "ls-files", "-z", "--cached", "--others", "--exclude-standard")
	if e != nil {
		return Attestation{}, fmt.Errorf("list git inputs: %w", e)
	}
	index, e := gitIndexEntries(ctx, root)
	if e != nil {
		return Attestation{}, e
	}
	h := sha256.New()
	for name := range strings.SplitSeq(string(b), "\x00") {
		if name == "" {
			continue
		}
		if e := ctx.Err(); e != nil {
			return Attestation{}, e
		}
		_, _ = fmt.Fprintf(h, "%d:%s\x00", len(name), name)
		entry, tracked := index[name]
		if tracked {
			_, _ = fmt.Fprintf(h, "index:%s:%s\x00", entry.mode, entry.object)
		}
		if tracked && entry.mode == "160000" {
			if e := fingerprintGitlink(ctx, root, name, entry.object, h); e != nil {
				return Attestation{}, e
			}
			continue
		}
		path := filepath.Join(root, filepath.FromSlash(name))
		info, e := os.Lstat(path)
		if os.IsNotExist(e) {
			_, _ = io.WriteString(h, "missing\x00")
			continue
		}
		if e != nil {
			return Attestation{}, e
		}
		_, _ = fmt.Fprintf(h, "%s\x00", info.Mode().String())
		if info.Mode()&os.ModeSymlink != 0 {
			target, e := os.Readlink(path)
			if e != nil {
				return Attestation{}, e
			}
			_, _ = fmt.Fprintf(h, "link:%d:%s\x00", len(target), target)
			continue
		}
		if !info.Mode().IsRegular() {
			return Attestation{}, fmt.Errorf("unsupported git input type %s", name)
		}
		file, e := os.Open(path)
		if e != nil {
			return Attestation{}, e
		}
		_, _ = fmt.Fprintf(h, "data:%d:\x00", info.Size())
		_, readErr := io.CopyBuffer(h, &contextReader{ctx: ctx, reader: file}, make([]byte, 64*1024))
		closeErr := file.Close()
		if e := errors.Join(readErr, closeErr); e != nil {
			return Attestation{}, e
		}

	}
	return Attestation{Available: true, Before: hex.EncodeToString(h.Sum(nil))}, nil
}

type gitIndexEntry struct {
	mode, object string
}

// gitIndexEntries returns stage-zero index entries. An unmerged entry has no
// single revision to attest, so CI refuses to run until it is resolved.
func gitIndexEntries(ctx context.Context, root string) (map[string]gitIndexEntry, error) {
	b, err := gitOutput(ctx, root, "ls-files", "--cached", "--stage", "-z")
	if err != nil {
		return nil, fmt.Errorf("list git index inputs: %w", err)
	}
	entries := make(map[string]gitIndexEntry)
	for record := range strings.SplitSeq(string(b), "\x00") {
		if record == "" {
			continue
		}
		header, name, ok := strings.Cut(record, "\t")
		fields := strings.Fields(header)
		if !ok || len(fields) != 3 {
			return nil, fmt.Errorf("parse git index input %q", record)
		}
		if fields[2] != "0" {
			return nil, fmt.Errorf("cannot attest unmerged git input %s", name)
		}
		if _, exists := entries[name]; exists {
			return nil, fmt.Errorf("duplicate git index input %s", name)
		}
		entries[name] = gitIndexEntry{mode: fields[0], object: fields[1]}
	}
	return entries, nil
}

// fingerprintGitlink records the superproject's indexed submodule revision.
// If the submodule is initialized, it also recursively attests its checked-out
// HEAD and working tree. A non-empty non-repository directory is ambiguous and
// therefore fails closed rather than being silently omitted from CI inputs.
func fingerprintGitlink(ctx context.Context, root, name, object string, h io.Writer) error {
	path := filepath.Join(root, filepath.FromSlash(name))
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		_, _ = fmt.Fprintf(h, "gitlink:%s:uninitialized\x00", object)
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("gitlink %s is not a directory", name)
	}
	top, probeErr := gitOutput(ctx, path, "rev-parse", "--show-toplevel")
	initialized := false
	if probeErr == nil {
		resolvedPath, pathErr := filepath.EvalSymlinks(path)
		resolvedTop, topErr := filepath.EvalSymlinks(strings.TrimSpace(string(top)))
		initialized = pathErr == nil && topErr == nil && resolvedPath == resolvedTop
	}
	if !initialized {
		contents, readErr := os.ReadDir(path)
		if readErr != nil {
			return readErr
		}
		if len(contents) == 0 {
			_, _ = fmt.Fprintf(h, "gitlink:%s:uninitialized\x00", object)
			return nil
		}
		if probeErr != nil {
			return fmt.Errorf("inspect gitlink %s: %w", name, probeErr)
		}
		return fmt.Errorf("gitlink %s is not an initialized Git working tree", name)
	}
	head, err := gitOutput(ctx, path, "rev-parse", "HEAD")
	if err != nil {
		return fmt.Errorf("read gitlink %s HEAD: %w", name, err)
	}
	nested, err := fingerprint(ctx, path)
	if err != nil {
		return fmt.Errorf("fingerprint gitlink %s: %w", name, err)
	}
	if !nested.Available {
		return fmt.Errorf("gitlink %s has no Git attestation", name)
	}
	_, _ = fmt.Fprintf(h, "gitlink:%s:head:%s:worktree:%s\x00", object, strings.TrimSpace(string(head)), nested.Before)
	return nil
}

// Git sees the owning working tree, not an index/worktree override inherited from a caller.
func gitOutput(ctx context.Context, root string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", root}, args...)...)
	cmd.WaitDelay = time.Second
	for _, e := range os.Environ() {
		key, _, _ := strings.Cut(e, "=")
		if key != "GIT_DIR" && key != "GIT_WORK_TREE" && key != "GIT_INDEX_FILE" && key != "LC_ALL" {
			cmd.Env = append(cmd.Env, e)
		}
	}
	cmd.Env = append(cmd.Env, "LC_ALL=C")
	return cmd.CombinedOutput()
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}
