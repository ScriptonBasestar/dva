package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// UpOptions configures orchestrator Up behavior.
type UpOptions struct {
	DryRun      bool
	Force       bool
	Wait        bool
	Names       []string // specific stack entry names (empty = all)
	IncludeTags []string
	ExcludeTags []string
	Mode        string
	Env         string
}

// DownOptions configures orchestrator Down behavior.
type DownOptions struct {
	DryRun       bool
	Volumes      bool     // also remove named volumes
	RemoveImages bool     // also remove locally built images
	Purge        bool     // whole compose project, ignoring the plan's service selection (TASK-311)
	Names        []string // specific stack entry names (empty = all)
	IncludeTags  []string
	ExcludeTags  []string
	Mode         string
	Env          string
}

// StopOptions configures orchestrator Stop behavior.
type StopOptions struct {
	DryRun      bool
	Names       []string // specific stack entry names (empty = all)
	IncludeTags []string
	ExcludeTags []string
	Mode        string
	Env         string
}

// Orchestrator coordinates lifecycle plugin execution in order.
type Orchestrator struct {
	entries         []config.LifecycleEntry
	composeProfiles map[string][]string
	composeServices map[string][]string
	cfg             *config.Config
	env             *config.Environment
	logger          *slog.Logger
	hc              *HealthChecker
}

// NewOrchestrator creates a new orchestrator from config.
func NewOrchestrator(cfg *config.Config, env *config.Environment) *Orchestrator {
	entries := cfg.SortedStack()

	return &Orchestrator{
		entries: entries,
		cfg:     cfg,
		env:     env,
		logger:  slog.Default(),
		hc:      &HealthChecker{},
	}
}

// Up starts all matching lifecycle entries in order.
func (o *Orchestrator) Up(ctx context.Context, opts UpOptions) error {
	filtered, err := o.filterEntries(opts.Names, opts.IncludeTags, opts.ExcludeTags, opts.Mode, opts.Env)
	if err != nil {
		return err
	}
	if len(filtered) == 0 {
		fmt.Fprintln(os.Stderr, "[warn] no lifecycle entries matched filters")
		return nil
	}

	// Tunnel declarations are access prerequisites (docs/68): open once per
	// unique declaration before the entries run, close at command end. Dry-run
	// executes nothing, so it opens nothing.
	tunnels := newTunnelManager(o.logger)
	if !opts.DryRun {
		defer tunnels.Close()
		for i := range filtered {
			if err := tunnels.acquire(&filtered[i], o.env); err != nil {
				return fmt.Errorf("entry %q tunnel: %w", filtered[i].Name, err)
			}
		}
	}

	// Clone env so exports accumulate without mutating the original
	envClone := o.env.Clone()

	// Resolve mode-derived compose hints
	var modeProfiles []string
	var modeServices *[]string
	if opts.Mode != "" {
		if m, ok := o.cfg.Modes[opts.Mode]; ok {
			modeProfiles = m.ComposeProfiles
			modeServices = m.ComposeServices
		}
	}

	for _, entry := range filtered {
		pluginType := entry.DetectPlugin()
		plugin, err := NewPlugin(pluginType)
		if err != nil {
			return fmt.Errorf("entry %q: %w", entry.Name, err)
		}

		entryComposeServices := modeServices
		if services, ok := o.composeServices[entry.Name]; ok {
			selected := append([]string(nil), services...)
			entryComposeServices = &selected
		}

		// A plan entry's profiles: replaces the mode-derived list rather than adding to
		// it. --mode and plans are two generations of the same selection mechanism (see
		// migrate_report.go), never combined in one config that a migration produced, and
		// a union would make the effective profile set depend on a flag the plan cannot
		// see. Replacement keeps the plan's declaration readable as written.
		entryComposeProfiles := modeProfiles
		if profiles, ok := o.composeProfiles[entry.Name]; ok {
			entryComposeProfiles = append([]string(nil), profiles...)
		}

		entryEnv := envClone.Clone()
		entryEnv.MergeVars(entry.Vars)
		pctx := &PluginContext{
			Entry:           &entry,
			Env:             entryEnv,
			ConfigDir:       o.cfg.FileDir(),
			DryRun:          opts.DryRun,
			Force:           opts.Force,
			Wait:            opts.Wait,
			ComposeProfiles: entryComposeProfiles,
			ComposeServices: entryComposeServices,
			Logger:          o.logger.With("entry", entry.Name, "plugin", pluginType),
		}

		fmt.Fprintf(os.Stderr, "[lifecycle] %s (%s)\n", entry.Name, pluginType)

		if err := ensureSource(&entry, o.cfg.FileDir(), opts.DryRun, o.logger); err != nil {
			return fmt.Errorf("entry %q source: %w", entry.Name, err)
		}

		result, err := plugin.Up(ctx, pctx)
		if err != nil {
			return fmt.Errorf("entry %q up failed: %w", entry.Name, err)
		}

		// Merge dynamic exports from plugin result
		if result != nil && len(result.Exports) > 0 {
			envClone.MergeVars(result.Exports)
		}

		// Merge static exports from entry config (with interpolation)
		if len(entry.Exports) > 0 {
			envClone.MergeVars(entry.Exports)
		}

		// Run health checks for this entry and wait if needed
		if len(entry.HealthChecks) > 0 && opts.Wait && opts.DryRun {
			// Nothing was started, so nothing can become ready: report the wait instead of
			// polling until ctx cancels (TASK-312).
			fmt.Fprintf(os.Stderr, "[health] (dry-run) would wait for entry %q: %s\n",
				entry.Name, describeHealthChecks(entry.HealthChecks))
		} else if len(entry.HealthChecks) > 0 && opts.Wait {
			results := o.hc.WaitUntilReadyWithContext(ctx, entry.HealthChecks, entryEnv.WorkDir(), entryEnv)
			allReady := true
			for _, r := range results {
				if !r.Ready {
					allReady = false
					break
				}
			}
			if !allReady {
				fmt.Fprintf(os.Stderr, "[warn] some health checks not ready for entry %q\n", entry.Name)
			}
		}
	}

	// Start native processes defined in mode health_checks with start commands
	if err := o.startModeProcesses(ctx, opts, envClone); err != nil {
		return err
	}

	return nil
}

// Down stops all matching lifecycle entries in reverse order.
func (o *Orchestrator) Down(ctx context.Context, opts DownOptions) error {
	filtered, err := o.filterEntries(opts.Names, opts.IncludeTags, opts.ExcludeTags, opts.Mode, opts.Env)
	if err != nil {
		return err
	}
	if err := o.stopModeProcesses(opts.Mode, opts.DryRun); err != nil {
		return err
	}

	// Tunnel access prerequisites, same contract as Up: open once per unique
	// declaration, close at command end, open nothing on dry-run.
	tunnels := newTunnelManager(o.logger)
	if !opts.DryRun {
		defer tunnels.Close()
		for i := range filtered {
			if err := tunnels.acquire(&filtered[i], o.env); err != nil {
				return fmt.Errorf("entry %q tunnel: %w", filtered[i].Name, err)
			}
		}
	}

	// Reverse order for teardown
	for i, j := 0, len(filtered)-1; i < j; i, j = i+1, j-1 {
		filtered[i], filtered[j] = filtered[j], filtered[i]
	}

	var downErrs []error
	for _, entry := range filtered {
		pluginType := entry.DetectPlugin()
		plugin, err := NewPlugin(pluginType)
		if err != nil {
			return fmt.Errorf("entry %q: %w", entry.Name, err)
		}

		var entryComposeServices *[]string
		if services, ok := o.composeServices[entry.Name]; ok {
			selected := append([]string(nil), services...)
			entryComposeServices = &selected
		}

		entryEnv := o.env.Clone()
		entryEnv.MergeVars(entry.Vars)
		pctx := &PluginContext{
			Entry:           &entry,
			Env:             entryEnv,
			ConfigDir:       o.cfg.FileDir(),
			DryRun:          opts.DryRun,
			Volumes:         opts.Volumes,
			RemoveImages:    opts.RemoveImages,
			Purge:           opts.Purge,
			ComposeServices: entryComposeServices,
			Logger:          o.logger.With("entry", entry.Name, "plugin", pluginType),
		}

		fmt.Fprintf(os.Stderr, "[lifecycle] stopping %s (%s)\n", entry.Name, pluginType)
		if err := requireSource(&entry, o.cfg.FileDir()); err != nil {
			fmt.Fprintf(os.Stderr, "[warn] entry %q source unavailable for down: %v\n", entry.Name, err)
			continue
		}

		if err := plugin.Down(ctx, pctx); err != nil {
			fmt.Fprintf(os.Stderr, "[warn] entry %q down failed: %v\n", entry.Name, err)
			// Continue with other entries — don't abort on single failure during teardown —
			// but still report the failure to the caller instead of swallowing it.
			downErrs = append(downErrs, fmt.Errorf("entry %q down failed: %w", entry.Name, err))
		}
	}

	return errors.Join(downErrs...)
}

// Stop stops all matching lifecycle entries in reverse order without removing resources.
func (o *Orchestrator) Stop(ctx context.Context, opts StopOptions) error {
	filtered, err := o.filterEntries(opts.Names, opts.IncludeTags, opts.ExcludeTags, opts.Mode, opts.Env)
	if err != nil {
		return err
	}
	if err := o.haltModeProcesses(opts.Mode, opts.DryRun); err != nil {
		return err
	}

	// Tunnel access prerequisites, same contract as Up: open once per unique
	// declaration, close at command end, open nothing on dry-run.
	tunnels := newTunnelManager(o.logger)
	if !opts.DryRun {
		defer tunnels.Close()
		for i := range filtered {
			if err := tunnels.acquire(&filtered[i], o.env); err != nil {
				return fmt.Errorf("entry %q tunnel: %w", filtered[i].Name, err)
			}
		}
	}

	// Reverse order
	for i, j := 0, len(filtered)-1; i < j; i, j = i+1, j-1 {
		filtered[i], filtered[j] = filtered[j], filtered[i]
	}

	var stopErrs []error
	for _, entry := range filtered {
		pluginType := entry.DetectPlugin()
		plugin, err := NewPlugin(pluginType)
		if err != nil {
			return fmt.Errorf("entry %q: %w", entry.Name, err)
		}

		var entryComposeServices *[]string
		if services, ok := o.composeServices[entry.Name]; ok {
			selected := append([]string(nil), services...)
			entryComposeServices = &selected
		}

		entryEnv := o.env.Clone()
		entryEnv.MergeVars(entry.Vars)
		pctx := &PluginContext{
			Entry:           &entry,
			Env:             entryEnv,
			ConfigDir:       o.cfg.FileDir(),
			DryRun:          opts.DryRun,
			ComposeServices: entryComposeServices,
			Logger:          o.logger.With("entry", entry.Name, "plugin", pluginType),
		}

		fmt.Fprintf(os.Stderr, "[lifecycle] stopping %s (%s)\n", entry.Name, pluginType)
		if err := requireSource(&entry, o.cfg.FileDir()); err != nil {
			fmt.Fprintf(os.Stderr, "[warn] entry %q source unavailable for stop: %v\n", entry.Name, err)
			continue
		}

		if err := plugin.Stop(ctx, pctx); err != nil {
			fmt.Fprintf(os.Stderr, "[warn] entry %q stop failed: %v\n", entry.Name, err)
			// Continue with other entries — don't abort on single failure — but still
			// report the failure to the caller instead of swallowing it.
			stopErrs = append(stopErrs, fmt.Errorf("entry %q stop failed: %w", entry.Name, err))
		}
	}

	return errors.Join(stopErrs...)
}

// Restart stops then starts all matching entries.
func (o *Orchestrator) Restart(ctx context.Context, opts UpOptions) error {
	stopOpts := StopOptions{
		DryRun:      opts.DryRun,
		Names:       opts.Names,
		IncludeTags: opts.IncludeTags,
		ExcludeTags: opts.ExcludeTags,
		Mode:        opts.Mode,
		Env:         opts.Env,
	}
	if err := o.Stop(ctx, stopOpts); err != nil {
		return err
	}
	return o.Up(ctx, opts)
}

// Status queries the status of all lifecycle entries.
//
// When the orchestrator was built for a plan (or otherwise carries per-entry
// compose service selection), each entry's Services list is restricted to that
// selection and OutOfPlan lists other project services that are still running.
// Whole-workspace status (no selection) keeps the full compose project list.
func (o *Orchestrator) Status(ctx context.Context) (*AggregatedStatus, error) {
	status := &AggregatedStatus{}

	tunnels := newTunnelManager(o.logger)
	defer tunnels.Close()
	for i := range o.entries {
		if err := tunnels.acquire(&o.entries[i], o.env); err != nil {
			return nil, fmt.Errorf("entry %q tunnel: %w", o.entries[i].Name, err)
		}
	}

	for _, entry := range o.entries {
		pluginType := entry.DetectPlugin()
		plugin, err := NewPlugin(pluginType)
		if err != nil {
			// Report the entry as broken rather than dropping it: `up`/`down`/`stop`
			// fail fast on this same entry, so status must not read as a clean stack.
			o.logger.Warn("plugin could not be constructed", "entry", entry.Name, "plugin", pluginType, "error", err)
			status.Entries = append(status.Entries, EntryStatus{
				Name:   entry.Name,
				Plugin: pluginType,
				Error:  err.Error(),
			})
			continue
		}

		entryEnv := o.env.Clone()
		entryEnv.MergeVars(entry.Vars)
		// Always query the full project surface; partition below when a plan
		// selected a service subset. Filtering at the compose CLI would hide
		// out-of-plan running services that still occupy ports.
		pctx := &PluginContext{
			Entry:     &entry,
			Env:       entryEnv,
			ConfigDir: o.cfg.FileDir(),
			Logger:    o.logger.With("entry", entry.Name, "plugin", pluginType),
		}

		services, err := plugin.Status(ctx, pctx)
		if err != nil {
			o.logger.Warn("plugin status query failed", "entry", entry.Name, "plugin", pluginType, "error", err)
		}

		var healthResults []HealthCheckResult
		if len(entry.HealthChecks) > 0 {
			healthResults = o.hc.CheckWithContext(entry.HealthChecks, entryEnv.WorkDir(), entryEnv)
		}

		es := EntryStatus{
			Name:     entry.Name,
			Plugin:   pluginType,
			Services: services,
			Health:   healthResults,
		}
		if selected, ok := o.composeServices[entry.Name]; ok && len(selected) > 0 {
			es.Services, es.OutOfPlan = partitionPlanServices(services, selected)
		}

		status.Entries = append(status.Entries, es)
	}

	return status, nil
}
