package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// startModeProcesses launches native processes for mode health_checks that have a start command.
// This bridges the gap between compose-managed infra and natively-run app services.
func (o *Orchestrator) startModeProcesses(ctx context.Context, opts UpOptions, env *config.Environment) error {
	if opts.Mode == "" {
		return nil
	}
	mode, ok := o.cfg.Modes[opts.Mode]
	if !ok || len(mode.HealthChecks) == 0 {
		return nil
	}

	type nativeProc struct {
		name string
		hc   config.HealthCheckConfig
	}

	// Collect startable processes
	var procs []nativeProc
	for _, hcName := range mode.HealthChecks {
		hc, ok := o.cfg.HealthChecks[hcName]
		if !ok || hc.Start == "" {
			continue
		}

		if opts.DryRun {
			fmt.Fprintf(os.Stderr, "[native] (dry-run) would start %s: %s\n", hcName, hc.Start)
			continue
		}

		// Check if already running
		pidFile := filepath.Join(o.cfg.FileDir(), config.DotDirName, config.PidsDirName, hcName+".pid")
		if data, err := os.ReadFile(pidFile); err == nil {
			pidStr := strings.TrimSpace(string(data))
			if pid := 0; true {
				_, _ = fmt.Sscanf(pidStr, "%d", &pid)
				if pid > 0 && IsProcessRunning(pid) {
					fmt.Fprintf(os.Stderr, "[native] %s already running (pid %d)\n", hcName, pid)
					continue
				}
			}
		}

		procs = append(procs, nativeProc{name: hcName, hc: hc})
	}

	// Start all native processes concurrently
	var mu sync.Mutex
	var firstErr error
	var wg sync.WaitGroup

	for _, p := range procs {
		wg.Add(1)
		go func(p nativeProc) {
			defer wg.Done()

			dir := o.cfg.FileDir()
			pctx := &PluginContext{
				Entry: &config.LifecycleEntry{
					Name: p.name,
				},
				Env:       env,
				ConfigDir: o.cfg.FileDir(),
				DryRun:    opts.DryRun,
				Logger:    o.logger.With("native", p.name),
			}

			fmt.Fprintf(os.Stderr, "[native] starting %s\n", p.name)
			if err := startLocalProcess(p.name, p.hc.Start, dir, pctx); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("native start %s: %w", p.name, err)
				}
				mu.Unlock()
				return
			}
			fmt.Fprintf(os.Stderr, "[+] started %s\n", p.name)

			// Wait for health check readiness if --wait
			if opts.Wait {
				checks := map[string]config.HealthCheckConfig{p.name: p.hc}
				readyTimeout := time.Duration(p.hc.ReadyTimeout) * time.Second
				if readyTimeout == 0 {
					readyTimeout = 30 * time.Second
				}
				waitCtx, cancel := context.WithTimeout(ctx, readyTimeout)
				results := o.hc.WaitUntilReadyWithContext(waitCtx, checks, env.WorkDir(), env)
				cancel()
				for _, r := range results {
					if !r.Ready {
						fmt.Fprintf(os.Stderr, "[warn] %s not ready after %s\n", p.name, readyTimeout)
					}
				}
			}
		}(p)
	}

	wg.Wait()
	return firstErr
}

// haltModeProcesses sends SIGTERM to mode health_check processes but preserves
// PID files so they can be restarted by the next `up` call (halt semantics).
func (o *Orchestrator) haltModeProcesses(mode string, dryRun bool) error {
	return o.signalModeProcesses(mode, false, dryRun)
}

// stopModeProcesses sends SIGTERM to mode health_check processes and removes
// PID files (destroy semantics).
func (o *Orchestrator) stopModeProcesses(mode string, dryRun bool) error {
	return o.signalModeProcesses(mode, true, dryRun)
}

// signalModeProcesses terminates health_check native processes. When removePID
// is true, the PID files are deleted after signalling (down semantics).
//
// dryRun is threaded down rather than checked at the two callers because they are the
// first statement of Down and Stop, before the entry filtering that could exit early — a
// guard there would have to be repeated and would drift. startModeProcesses, the `up`
// half of this pair, has checked opts.DryRun since it was written (see its "would start"
// branch); this half never did, so `dva stop --mode dev --dry-run` sent a real SIGTERM
// and printed "stopped worker" — output indistinguishable from the run without the flag.
// Measured, then fixed under TASK-166.
func (o *Orchestrator) signalModeProcesses(mode string, removePID, dryRun bool) error {
	if mode == "" {
		return nil
	}
	m, ok := o.cfg.Modes[mode]
	if !ok || len(m.HealthChecks) == 0 {
		return nil
	}

	for _, hcName := range m.HealthChecks {
		hc, ok := o.cfg.HealthChecks[hcName]
		if !ok || hc.Start == "" {
			continue
		}

		pidFile := filepath.Join(o.cfg.FileDir(), config.DotDirName, config.PidsDirName, hcName+".pid")
		data, err := os.ReadFile(pidFile)
		if err != nil {
			continue
		}

		var pid int
		_, _ = fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &pid)
		if signalableProcessGroupPID(pid) {
			if err := requireProcessGroupPID(pid); err != nil {
				return fmt.Errorf("stop health check %s: %w", hcName, err)
			}
			if dryRun {
				// The signal line is conditional on the process existing because the real
				// path's is: it prints only when Kill returns nil. The delete line is not,
				// because the real path's os.Remove runs for any pid > 0 whether the kill
				// succeeded or not — a preview that hid it would understate the loss.
				if IsProcessRunning(pid) {
					fmt.Fprintf(os.Stderr, "[native] (dry-run) would stop %s (pid %d)\n", hcName, pid)
				}
				if removePID {
					fmt.Fprintf(os.Stderr, "[native] (dry-run) would delete %s\n", pidFile)
				}
				continue
			}
			if err := terminateProcessGroup(pid); err == nil {
				fmt.Fprintf(os.Stderr, "[-] stopped %s (pid %d)\n", hcName, pid)
			} else if errors.Is(err, errProcessGroupsUnsupported) {
				return fmt.Errorf("stop health check %s: %w", hcName, err)
			}
			if removePID {
				_ = os.Remove(pidFile)
			}
		}
	}
	return nil
}
