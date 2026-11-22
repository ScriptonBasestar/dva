package lifecycle

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// livenessPollInterval matches the health-check poll cadence (health.go): a native
// run command that died right after spawn is caught within one interval.
const livenessPollInterval = 2 * time.Second

// defaultReadyTimeout is the readiness ceiling a health check gets when it declares
// no ready_timeout.
const defaultReadyTimeout = 30 * time.Second

// waitEntryReady gates one entry. Two bounds close the forever-polling window
// TASK-402 measured: the wait carries a deadline derived from the entry's health
// checks' ready_timeout, and while the checks are pending the entry's pidfile is
// rechecked, so a native run command that exited after spawn fails `up` within one
// poll interval — with its log path — instead of hanging until the deadline.
func (o *Orchestrator) waitEntryReady(ctx context.Context, entry config.LifecycleEntry, entryEnv *config.Environment) error {
	timeout := entryReadyTimeout(entry)
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resultsCh := make(chan []HealthCheckResult, 1)
	go func() {
		resultsCh <- o.hc.WaitUntilReadyWithContext(waitCtx, entry.HealthChecks, entryEnv.WorkDir(), entryEnv)
	}()

	ticker := time.NewTicker(livenessPollInterval)
	defer ticker.Stop()
	for {
		select {
		case results := <-resultsCh:
			// A completed probe can race cancellation or the readiness deadline.
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("entry %q readiness wait cancelled: %w", entry.Name, err)
			}
			if waitCtx.Err() == context.DeadlineExceeded {
				return fmt.Errorf("entry %q not ready within %s (ready_timeout)", entry.Name, timeout)
			}
			for _, result := range results {
				if !result.Ready {
					return fmt.Errorf("entry %q health check %q not ready", entry.Name, result.Name)
				}
			}
			// Ready, but served by whom? A server that daemonized answers the probe after
			// the tracked pid already exited (TASK-629).
			if pid, known := o.entryPid(entry); known {
				if !IsProcessRunning(pid) {
					if err := escapedCheck(ctx, entry, entryEnv, pid); err != nil {
						return err
					}
					return nil
				}
				// Alive, but does the probed socket belong to it (TASK-631)?
				report := checkHealthPortOwners(entry.HealthChecks, entryEnv, pid)
				if len(report.Mismatched) > 0 {
					return portOwnerError(entry.Name, pid, report.Mismatched)
				}
				if len(report.Unverified) > 0 {
					slog.Default().Warn("health port owner unverified", "entry", entry.Name,
						"addresses", strings.Join(report.Unverified, ", "))
				}
			}
			return nil
		case <-waitCtx.Done():
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("entry %q readiness wait cancelled: %w", entry.Name, err)
			}
			return fmt.Errorf("entry %q not ready within %s (ready_timeout)", entry.Name, timeout)
		case <-ticker.C:
			if pid, known := o.entryPid(entry); known && !IsProcessRunning(pid) {
				if err := escapedCheck(ctx, entry, entryEnv, pid); err != nil {
					return err
				}
				return fmt.Errorf("entry %q native process (pid %d) exited before becoming ready — see %s",
					entry.Name, pid, o.entryLogPath(entry.Name))
			}
		}
	}
}

// escapedCheck returns escapedError when the entry's health addresses still answer after its
// tracked pid exited, and nil otherwise.
func escapedCheck(ctx context.Context, entry config.LifecycleEntry, env *config.Environment, pid int) error {
	if held := heldHealthEndpoints(ctx, entry.HealthChecks, env); len(held) > 0 {
		return escapedError(entry.Name, pid, held)
	}
	return nil
}

// entryReadyTimeout returns the largest ready_timeout declared across the entry's
// health checks, or defaultReadyTimeout when none declares one — the same per-check
// semantics the single-plan wait path applies to its single check.
func entryReadyTimeout(entry config.LifecycleEntry) time.Duration {
	timeout := time.Duration(0)
	for _, check := range entry.HealthChecks {
		if d := time.Duration(check.ReadyTimeout) * time.Second; d > timeout {
			timeout = d
		}
	}
	if timeout == 0 {
		return defaultReadyTimeout
	}
	return timeout
}

// entryPid reads the native pidfile for the entry, reporting known=false when the
// entry is not a process/native entry or no readable pidfile exists — those wait
// on their health checks alone.
func (o *Orchestrator) entryPid(entry config.LifecycleEntry) (int, bool) {
	if entry.Process == nil {
		return 0, false
	}
	data, err := os.ReadFile(o.entryPidPath(entry.Name))
	if err != nil {
		return 0, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, false
	}
	return pid, true
}

func (o *Orchestrator) entryPidPath(name string) string {
	return (&PluginContext{ConfigDir: o.cfg.FileDir(), Instance: o.instance}).PidFile(name)
}

func (o *Orchestrator) entryLogPath(name string) string {
	return (&PluginContext{ConfigDir: o.cfg.FileDir(), Instance: o.instance}).LogFile(name)
}
