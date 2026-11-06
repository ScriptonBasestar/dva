package cirun

// The cache is deliberately opt-in per step.  A profile describes the command,
// but only the owner of that command knows its complete source and output set.

import (
	"bytes"
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
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

const reuseSchema = 1

type reuseRuntimeKey struct{}

type reuseRuntime struct {
	runID string
	fresh bool
}

type reuseRecord struct {
	Schema      int               `json:"schema"`
	Key         string            `json:"key"`
	RunID       string            `json:"run_id"`
	FinishedAt  time.Time         `json:"finished_at"`
	OutputState map[string]string `json:"output_state,omitempty"`
}

// reusePlan is deliberately computed before and after a command.  The second
// measurement prevents a successful command which changed its own declared
// inputs from becoming a false cache hit for a later run.
type reusePlan struct {
	key     string
	cache   string
	outputs []string
}

// A cache integrity failure means the command's own declared contract could
// not be proven after it ran. Continuing independent steps would produce a
// misleading partial CI result even for profiles that opt out of ordinary
// fail-fast cancellation.
type reuseIntegrityError struct{ err error }

func (e *reuseIntegrityError) Error() string { return e.err.Error() }
func (e *reuseIntegrityError) Unwrap() error { return e.err }

func reuseFor(ctx context.Context, root string, step config.CIStep, env []string) (reusePlan, error) {
	if step.Reuse == nil {
		return reusePlan{}, nil
	}
	if step.OCI != nil || step.ChildCI != "" {
		return reusePlan{}, fmt.Errorf("ci step %q: reuse is only supported by native run steps", step.Name)
	}
	if strings.TrimSpace(step.Reuse.MaxAge) == "" {
		return reusePlan{}, fmt.Errorf("ci step %q: reuse.max_age is required", step.Name)
	}
	if d, err := time.ParseDuration(step.Reuse.MaxAge); err != nil || d <= 0 {
		return reusePlan{}, fmt.Errorf("ci step %q: reuse.max_age must be a positive duration", step.Name)
	}
	if len(step.Reuse.Inputs) == 0 {
		return reusePlan{}, fmt.Errorf("ci step %q: reuse.inputs must not be empty", step.Name)
	}
	h := sha256.New()
	writeReuseField(h, "schema", fmt.Sprint(reuseSchema))
	writeReuseField(h, "step", step.Name)
	writeReuseField(h, "run", step.Run)
	writeReuseField(h, "workdir", step.Workdir)
	writeReuseField(h, "goos", runtime.GOOS)
	writeReuseField(h, "goarch", runtime.GOARCH)
	writeReuseField(h, "dva_version", config.Version)
	writeReuseField(h, "dva_commit", config.ResolvedCommit().String())
	engine, err := os.Executable()
	if err != nil {
		return reusePlan{}, fmt.Errorf("locate DVA executable: %w", err)
	}
	engine, err = filepath.EvalSymlinks(engine)
	if err != nil {
		return reusePlan{}, fmt.Errorf("canonicalize DVA executable: %w", err)
	}
	engineHash, err := hashPath(engine)
	if err != nil {
		return reusePlan{}, fmt.Errorf("hash DVA executable: %w", err)
	}
	writeReuseField(h, "dva_engine", engineHash)
	shellHash, err := hashPath("/bin/sh")
	if err != nil {
		return reusePlan{}, fmt.Errorf("hash shell: %w", err)
	}
	writeReuseField(h, "shell", shellHash)
	inputs, err := reuseInputs(ctx, root, step.Reuse.Inputs)
	if err != nil {
		return reusePlan{}, fmt.Errorf("ci step %q reuse inputs: %w", step.Name, err)
	}
	for _, input := range inputs {
		writeReuseField(h, "input", input)
	}
	workdir := root
	if step.Workdir != "" {
		workdir = filepath.Join(root, step.Workdir)
		if filepath.IsAbs(step.Workdir) {
			workdir = step.Workdir
		}
	}
	workdir, err = filepath.EvalSymlinks(workdir)
	if err != nil {
		return reusePlan{}, fmt.Errorf("canonicalize step workdir: %w", err)
	}
	rel, err := filepath.Rel(root, workdir)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return reusePlan{}, errors.New("step workdir escapes root")
	}
	for _, name := range step.Reuse.Tools {
		resolved, err := reuseTool(ctx, workdir, name, env)
		if err != nil {
			return reusePlan{}, fmt.Errorf("ci step %q reuse tool %q: %w", step.Name, name, err)
		}
		writeReuseField(h, "tool", name+"\x00"+resolved)
	}
	values := environmentMap(env)
	keys := []string{"PATH", "LANG", "LC_ALL", "LC_CTYPE", "TZ", "GOFLAGS", "GOTOOLCHAIN", "CGO_ENABLED", "GOMAXPROCS", "CARGO_BUILD_JOBS", "RUST_TEST_THREADS", "DVA_CI_JOBS", "PYTHONPATH", "PYTHONHOME", "PYTHONSAFEPATH", "MAKEFLAGS", "MFLAGS", config.CISubprojectEnv}
	keys = append(keys, step.Reuse.Environment...)
	keys = append(keys, step.Reuse.UnlessEnvironment...)
	for key := range step.Environment {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	keys = compactStrings(keys)
	for _, key := range keys {
		if key == "" || strings.Contains(key, "=") {
			return reusePlan{}, fmt.Errorf("invalid reuse environment name %q", key)
		}
		// The key is hashed with the value and only the digest is persisted.
		value, present := values[key]
		presence := "absent"
		if present {
			presence = "present"
		}
		writeReuseField(h, "env", key+"\x00"+presence+"\x00"+value)
	}
	for _, command := range step.Reuse.Probes {
		digest, err := reuseProbeDigest(ctx, root, workdir, command, env)
		if err != nil {
			return reusePlan{}, fmt.Errorf("reuse probe failed: %w", err)
		}
		writeReuseField(h, "probe", command+"\x00"+digest)
	}
	outputs, err := canonicalReusePaths(step.Reuse.Outputs)
	if err != nil {
		return reusePlan{}, fmt.Errorf("ci step %q reuse outputs: %w", step.Name, err)
	}
	common, err := gitStdout(ctx, root, "rev-parse", "--git-common-dir")
	if err != nil {
		return reusePlan{}, fmt.Errorf("locate git common directory: %w", gitError(err))
	}
	commonDir := strings.TrimSpace(string(common))
	if !filepath.IsAbs(commonDir) {
		commonDir = filepath.Join(root, commonDir)
	}
	commonDir, err = filepath.EvalSymlinks(commonDir)
	if err != nil {
		return reusePlan{}, fmt.Errorf("canonicalize git common directory: %w", err)
	}
	rootDigest := sha256.Sum256([]byte(root))
	key := hex.EncodeToString(h.Sum(nil))
	return reusePlan{key: key, cache: filepath.Join(commonDir, "dva-ci-cache", "v1", hex.EncodeToString(rootDigest[:]), key+".json"), outputs: outputs}, nil
}

func writeReuseField(w io.Writer, key, value string) {
	_, _ = fmt.Fprintf(w, "%d:%s=%d:%s\x00", len(key), key, len(value), value)
}

func environmentMap(env []string) map[string]string {
	values := map[string]string{}
	for _, value := range env {
		if key, val, ok := strings.Cut(value, "="); ok {
			values[key] = val
		}
	}
	return values
}

func compactStrings(values []string) []string {
	if len(values) == 0 {
		return values
	}
	out := values[:1]
	for _, value := range values[1:] {
		if value != out[len(out)-1] {
			out = append(out, value)
		}
	}
	return out
}

func reuseInputs(ctx context.Context, root string, patterns []string) ([]string, error) {
	for _, pattern := range patterns {
		if pattern == "" || filepath.IsAbs(pattern) || strings.HasPrefix(filepath.Clean(pattern), ".."+string(os.PathSeparator)) || filepath.Clean(pattern) == ".." {
			return nil, fmt.Errorf("pathspec escapes root: %q", pattern)
		}
	}
	data, err := gitStdout(ctx, root, append([]string{"ls-files", "-z", "--cached", "--others", "--exclude-standard", "--"}, patterns...)...)
	if err != nil {
		return nil, gitError(err)
	}
	paths := []string{}
	for path := range strings.SplitSeq(string(data), "\x00") {
		if path != "" {
			paths = append(paths, path)
		}
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("declared inputs matched no files: %s", strings.Join(patterns, " "))
	}
	sort.Strings(paths)
	entries := make([]string, 0, len(paths))
	for _, name := range paths {
		path := filepath.Join(root, filepath.FromSlash(name))
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			// Git still lists a tracked deletion.  Its explicit marker makes the
			// manifest differ from the prior file without treating deletion as a
			// path escape or an unreadable cache input.
			entries = append(entries, name+"\x00missing")
			continue
		}
		if err != nil {
			return nil, err
		}
		resolved, err := filepath.EvalSymlinks(filepath.Dir(path))
		if err != nil {
			return nil, err
		}
		rel, err := filepath.Rel(root, resolved)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return nil, fmt.Errorf("input escapes root: %s", name)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return nil, err
			}
			resolved, err := filepath.EvalSymlinks(path)
			if err != nil {
				return nil, err
			}
			rel, err := filepath.Rel(root, resolved)
			if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
				return nil, fmt.Errorf("input symlink escapes root: %s", name)
			}
			resolvedInfo, err := os.Stat(resolved)
			if err != nil {
				return nil, err
			}
			if !resolvedInfo.Mode().IsRegular() {
				return nil, fmt.Errorf("input symlink does not resolve to a regular file: %s", name)
			}
			digest, err := hashPath(resolved)
			if err != nil {
				return nil, err
			}
			entries = append(entries, name+"\x00link\x00"+target+"\x00"+resolvedInfo.Mode().String()+"\x00"+digest)
			continue
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("input is not a regular file: %s", name)
		}
		digest, err := hashPath(path)
		if err != nil {
			return nil, err
		}
		entries = append(entries, name+"\x00"+info.Mode().String()+"\x00"+digest)
	}
	return entries, nil
}

func hashPath(path string) (digest string, retErr error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { retErr = errors.Join(retErr, f.Close()) }()
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func reuseTool(ctx context.Context, workdir, name string, env []string) (string, error) {
	if name == "" || strings.ContainsRune(name, os.PathSeparator) {
		return "", errors.New("tool must be a PATH command name")
	}
	path, err := lookupReuseTool(name, environmentMap(env)["PATH"], workdir)
	if err != nil {
		return "", err
	}
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	// Mise exposes a single shim whose bytes do not identify the selected tool.
	// Ask that shim for its selected executable under precisely the step's PATH.
	if filepath.Base(resolvedPath) == "mise" || filepath.Base(path) == "mise" {
		out, err := runMiseWhich(ctx, workdir, resolvedPath, name, env)
		if err != nil {
			return "", fmt.Errorf("mise which %s: %w", name, err)
		}
		selected := strings.TrimSpace(string(out))
		if selected == "" || !filepath.IsAbs(selected) {
			return "", errors.New("mise which returned no absolute executable")
		}
		resolvedPath, err = filepath.EvalSymlinks(selected)
		if err != nil {
			return "", err
		}
	}
	info, err := os.Stat(resolvedPath)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("resolved tool is not a regular file")
	}
	digest, err := hashPath(resolvedPath)
	if err != nil {
		return "", err
	}
	return resolvedPath + "\x00" + digest, nil
}

func lookupReuseTool(name, searchPath, workdir string) (string, error) {
	if searchPath == "" {
		return "", errors.New("step PATH is empty")
	}
	for _, dir := range filepath.SplitList(searchPath) {
		if dir == "" {
			dir = workdir
		} else if !filepath.IsAbs(dir) {
			dir = filepath.Join(workdir, dir)
		}
		candidate := filepath.Join(dir, name)
		info, err := os.Stat(candidate)
		if err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0o111 != 0 {
			return candidate, nil
		}
		if err != nil && !os.IsNotExist(err) {
			return "", err
		}
	}
	return "", exec.ErrNotFound
}

const miseWhichOutputLimit = 64 << 10

type limitedReuseOutput struct {
	bytes.Buffer
	overflow bool
}

func (w *limitedReuseOutput) Write(data []byte) (int, error) {
	if w.Len()+len(data) > miseWhichOutputLimit {
		w.overflow = true
		return 0, errors.New("mise which output exceeded limit")
	}
	return w.Buffer.Write(data)
}

func runMiseWhich(parent context.Context, workdir, mise, name string, env []string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := &limitedReuseOutput{}
	cmd := exec.CommandContext(ctx, mise, "which", name)
	cmd.Dir, cmd.Env = workdir, env
	cmd.Stdout, cmd.Stderr = out, io.Discard
	cmd.WaitDelay = time.Second
	if err := startProcess(cmd); err != nil {
		return nil, fmt.Errorf("start mise which: %w", err)
	}
	pid := cmd.Process.Pid
	waited := make(chan error, 1)
	go func() { waited <- cmd.Wait() }()
	select {
	case err := <-waited:
		cleanupGroup(pid)
		if out.overflow {
			return nil, errors.New("mise which output exceeded 64 KiB")
		}
		if err != nil {
			return nil, fmt.Errorf("mise which %s: %w", name, err)
		}
	case <-ctx.Done():
		terminateProcessGroup(pid, waited)
		return nil, ctx.Err()
	}
	return out.Bytes(), nil
}

func canonicalReusePaths(paths []string) ([]string, error) {
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		if path == "" || filepath.IsAbs(path) || filepath.Clean(path) == ".." || strings.HasPrefix(filepath.Clean(path), ".."+string(os.PathSeparator)) {
			return nil, fmt.Errorf("path escapes root: %q", path)
		}
		out = append(out, filepath.Clean(path))
	}
	sort.Strings(out)
	return compactStrings(out), nil
}

func reuseOutputState(root string, paths []string) (map[string]string, error) {
	state := map[string]string{}
	for _, name := range paths {
		path := filepath.Join(root, name)
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("declared output is missing: %s", name)
		}
		if err != nil {
			return nil, err
		}
		parent, err := filepath.EvalSymlinks(filepath.Dir(path))
		if err != nil {
			return nil, err
		}
		rel, err := filepath.Rel(root, parent)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return nil, fmt.Errorf("declared output parent escapes root: %s", name)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("declared output is a symlink: %s", name)
		}
		if info.Mode().IsRegular() {
			digest, err := hashPath(path)
			if err != nil {
				return nil, err
			}
			state[name] = info.Mode().String() + "\x00" + digest
			continue
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("declared output has unsupported type: %s", name)
		}
		err = filepath.WalkDir(path, func(current string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("declared output contains symlink: %s", current)
			}
			entryInfo, err := entry.Info()
			if err != nil {
				return err
			}
			if entry.IsDir() {
				rel, err := filepath.Rel(root, current)
				if err != nil {
					return err
				}
				state[filepath.ToSlash(rel)] = "dir\x00" + entryInfo.Mode().String()
				return nil
			}
			if !entryInfo.Mode().IsRegular() {
				return fmt.Errorf("declared output contains unsupported type: %s", current)
			}
			rel, err := filepath.Rel(root, current)
			if err != nil {
				return err
			}
			digest, err := hashPath(current)
			if err != nil {
				return err
			}
			state[filepath.ToSlash(rel)] = entryInfo.Mode().String() + "\x00" + digest
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return state, nil
}

func loadReuse(plan reusePlan, maxAge time.Duration, now time.Time, root string) (*reuseRecord, string) {
	b, err := os.ReadFile(plan.cache)
	if os.IsNotExist(err) {
		return nil, "miss"
	}
	if err != nil {
		return nil, "unreadable cache"
	}
	var record reuseRecord
	if json.Unmarshal(b, &record) != nil || record.Schema != reuseSchema || record.Key != plan.key || record.RunID == "" || record.FinishedAt.IsZero() {
		return nil, "malformed cache"
	}
	if record.FinishedAt.After(now) {
		return nil, "future cache timestamp"
	}
	if now.Sub(record.FinishedAt) > maxAge {
		return nil, "expired cache"
	}
	if len(plan.outputs) > 0 {
		outputs, err := reuseOutputState(root, plan.outputs)
		if err != nil || !sameReuseState(record.OutputState, outputs) {
			return nil, "output mismatch"
		}
	}
	return &record, "reused"
}

func sameReuseState(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for key, value := range a {
		if b[key] != value {
			return false
		}
	}
	return true
}

func saveReuse(plan reusePlan, root, runID string, now time.Time) (retErr error) {
	outputs, err := reuseOutputState(root, plan.outputs)
	if err != nil {
		return err
	}
	record := reuseRecord{Schema: reuseSchema, Key: plan.key, RunID: runID, FinishedAt: now, OutputState: outputs}
	b, err := json.Marshal(record)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(plan.cache), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(plan.cache), ".entry-")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer func() {
		if err := os.Remove(name); err != nil && !os.IsNotExist(err) {
			retErr = errors.Join(retErr, err)
		}
	}()
	if _, err = tmp.Write(b); err == nil {
		err = tmp.Chmod(0o600)
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(name, plan.cache)
}
