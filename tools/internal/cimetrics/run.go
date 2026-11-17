// Package cimetrics observes this project's checks without deciding their result.
package cimetrics

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func validCheck(check string) bool {
	switch check {
	case "commit-test", "dev-test", "unit", "integration", "skill-dogfood", "lint-preflight", "lint-golangci", "lint-gopls":
		return true
	default:
		return false
	}
}

func isTestCheck(check string) bool {
	switch check {
	case "commit-test", "dev-test", "unit", "integration", "skill-dogfood":
		return true
	default:
		return false
	}
}

// Run preserves the caller's stdin; the CI commit checker uses RunWithStdin
// with nil instead, matching its existing exec.Cmd contract.
func Run(ctx context.Context, checkID string, argv []string, stdout, stderr io.Writer) error {
	return RunWithStdin(ctx, checkID, argv, os.Stdin, stdout, stderr)
}

// RunWithStdin forwards non-JSON output and renders Go JSON output without
// exposing passing-test logs. Collection cannot decide the check's result.
func RunWithStdin(ctx context.Context, checkID string, argv []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if !validCheck(checkID) {
		return errors.New("ci-metrics: unsupported check ID")
	}
	if len(argv) == 0 || argv[0] == "" {
		return errors.New("ci-metrics: missing command")
	}
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}
	root, rootOK := findProjectRoot()
	jsonEvents := hasJSONFlag(argv)
	allowed := map[string]map[string]bool(nil)
	allowOK := !jsonEvents
	if rootOK && jsonEvents {
		ch := make(chan map[string]map[string]bool, 1)
		go func() {
			v, e := LoadAllowed(root)
			if e != nil {
				ch <- nil
			} else {
				ch <- v
			}
		}()
		select {
		case allowed = <-ch:
			allowOK = allowed != nil
		case <-time.After(100 * time.Millisecond):
		case <-ctx.Done():
		}
	}
	var c *collector
	var parser *Parser
	var renderer *outputRenderer
	started := time.Now()
	id, idErr := newID()
	if idErr == nil && rootOK && groupSupported() {
		c = newCollector(root, checkID, id, started)
		_, _ = fmt.Fprintf(stderr, "ci-metrics: measurement=%s\n", id)
	} else {
		_, _ = fmt.Fprintln(stderr, "ci-metrics: collection degraded (identity or platform unavailable)")
	}
	if jsonEvents {
		var emit func(Progress)
		if c != nil {
			emit = c.emit
		}
		parser = NewParser(allowed, emit)
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdin, cmd.Stderr = stdin, stderr
	if checkID == "commit-test" {
		cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")
	}
	if parser != nil {
		renderer = newOutputRenderer(parser, stdout, explicitVerbose(argv))
		cmd.Stdout = io.MultiWriter(parser, renderer)
	} else {
		cmd.Stdout = stdout
	}
	// Limit Go's pipe wait once the child exits; outer DVA cancellation remains
	// authoritative and no collector goroutine is joined without a bound.
	cmd.WaitDelay = 100 * time.Millisecond
	configureGroup(cmd)
	var runErr error
	var processDone bool
	var cancellation error
	if err := ctx.Err(); err != nil {
		runErr, cancellation = err, err
	} else if err := cmd.Start(); err != nil {
		runErr = err
	} else {
		pid := cmd.Process.Pid
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		select {
		case runErr = <-done:
			processDone = true
		case <-ctx.Done():
			select {
			case runErr = <-done:
				processDone = true
			default:
				cancellation = ctx.Err()
				_ = stopGroup(pid)
				select {
				case runErr = <-done:
					processDone = true
				case <-time.After(200 * time.Millisecond):
					_ = killGroup(pid)
					select {
					case runErr = <-done:
						processDone = true
					case <-time.After(100 * time.Millisecond):
						runErr = ctx.Err()
					}
				}
			}
		}
		// The group is owned by this invocation; a descendant must not outlive
		// the wrapper even if it keeps a pipe open after the direct child exits.
		if groupSupported() {
			_ = killGroup(pid)
		}
	}
	if cancellation != nil && runErr == nil {
		runErr = cancellation
	}
	finished := time.Now()
	outputDegraded := false
	if renderer != nil {
		renderer.Finish()
		outputDegraded = renderer.Degraded()
		if outputDegraded {
			_, _ = fmt.Fprintln(stderr, "ci-metrics: local test output incomplete; diagnostics may be truncated")
		}
	}
	var events *Snapshot
	if parser != nil {
		parser.Finish()
		s := parser.Snapshot()
		events = &s
	}
	if c != nil {
		wall := finished.Sub(started).Milliseconds()
		outcome := "succeeded"
		if runErr != nil {
			outcome = "failed"
		}
		if errors.Is(cancellation, context.DeadlineExceeded) {
			outcome = "timed_out"
		} else if cancellation != nil {
			outcome = "canceled"
		}
		r := Record{Schema: recordSchema, ID: id, Check: checkID, StartedAt: started.UTC(), FinishedAt: finished.UTC(), WallMS: &wall, Status: "complete", Outcome: outcome, Events: events, Options: safeOptions(argv, allowed)}
		if isTestCheck(checkID) && !jsonEvents {
			r.Status, r.Reason = "partial", "structured test events unavailable"
		}
		if outputDegraded {
			r.Status, r.Reason, r.OutputDegraded = "partial", "local test output incomplete", true
		}
		if !allowOK {
			r.Status, r.Reason = "partial", "test allowlist unavailable"
		}
		if !processDone && runErr != nil {
			r.Status, r.Reason = "partial", "process completion not observed"
		}
		if processDone && cmd.ProcessState != nil {
			code := cmd.ProcessState.ExitCode()
			r.ExitCode = &code
			r.Signal = signalName(cmd.ProcessState)
		}
		if !c.finish(r) {
			_, _ = fmt.Fprintf(stderr, "ci-metrics: collection partial (measurement=%s)\n", id)
		}
	}
	return runErr
}

func hasJSONFlag(argv []string) bool {
	enabled := false
	for _, v := range argv {
		if v == "-args" || v == "--" {
			break
		}
		if v == "-json" {
			enabled = true
			continue
		}
		if raw, ok := strings.CutPrefix(v, "-json="); ok {
			if value, err := strconv.ParseBool(raw); err == nil {
				enabled = value
			}
		}
	}
	return enabled
}

func findProjectRoot() (string, bool) {
	dir, err := os.Getwd()
	if err != nil {
		return "", false
	}
	for {
		if info, e := os.Stat(filepath.Join(dir, ".git")); e == nil && (info.IsDir() || info.Mode().IsRegular()) {
			b, e := readSmall(filepath.Join(dir, "go.mod"), 2*1024*1024)
			if e == nil && strings.HasPrefix(string(b), "module "+module+"\n") {
				return dir, true
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}
