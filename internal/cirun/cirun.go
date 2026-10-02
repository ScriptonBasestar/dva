// Package cirun runs a CI profile without changing the caller's process state.
// It provides conservative Go and Rust worker defaults through environment
// variables. Tools which ignore these variables own their internal concurrency.
package cirun

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

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
