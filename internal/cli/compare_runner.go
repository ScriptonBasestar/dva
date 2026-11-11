package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ScriptonBasestar/dva/internal/config"
	"github.com/ScriptonBasestar/dva/internal/lifecycle"
)

// compareRunner runs a named subset of a plan's entries in one config directory. A non-empty
// instance runs them as that instance (config mode: separate pid files, logs and compose
// project in a directory another variant also uses); empty is the ordinary run. It is the
// seam between `dva compare` and the named lifecycle: compare decides which checkout, which
// entries and which vars, and the runner applies the same plan resolution `dva up <plan>` does.
// Tests replace compareHost with a fake; production uses lifecycleRunner.
type compareRunner interface {
	Up(ctx context.Context, dir, instance, plan string, vars map[string]string, names []string, wait bool) (*compareObserved, error)
	Down(ctx context.Context, dir, instance, plan string, vars map[string]string, names []string) error
	Status(ctx context.Context, dir, instance, plan string, vars map[string]string, names []string) (*compareObserved, error)
	// Endpoints returns the plan's endpoints in dir interpolated with vars, as observe would
	// report them for an entry run with those vars. It starts and reads nothing.
	Endpoints(ctx context.Context, dir, plan string, vars map[string]string) ([]planEndpointOutput, error)
}

var compareHost compareRunner = lifecycleRunner{}

type lifecycleRunner struct{}

// open resolves planName for the config in dir exactly as the plan verbs do: the plan's owner
// decides the environment, and a failed env_file input stops the call before any entry starts.
func (lifecycleRunner) open(dir, instance, planName string, vars map[string]string) (*planRuntime, *lifecycle.Orchestrator, error) {
	c, load, err := compareConfigAt(dir)
	if err != nil {
		return nil, nil, err
	}
	runtime, err := resolvePlanRuntime(c, load, planName, vars)
	if err != nil {
		return nil, nil, err
	}
	printPlanWarnings(runtime.plan)
	warnUnsealedEnv(runtime.config, runtime.report, "")
	if err := runtime.report.Err(); err != nil {
		return nil, nil, err
	}
	orch, err := lifecycle.NewPlanOrchestrator(runtime.config, runtime.env, runtime.plan)
	if err != nil {
		return nil, nil, err
	}
	orch.SetInstance(instance)
	return runtime, orch, nil
}

// compareConfigAt loads the dva.yml in dir with its own environment. The calling checkout
// reuses the config the command already loaded, so a shared entry sees exactly what `dva up`
// from there would; a variant checkout is loaded fresh from its own files.
// errVariantConfigUnreadable marks a checkout whose dva.yml cannot be loaded. A snapshot is a fixed
// commit, so a config that cannot load now could not have loaded when `up` ran: nothing was
// started there, and `down` has nothing to stop.
var errVariantConfigUnreadable = errors.New("variant config unreadable")

func compareConfigAt(dir string) (*config.Config, *envLoad, error) {
	if cfg != nil && filepath.Clean(cfg.FileDir()) == filepath.Clean(dir) {
		return cfg, rootEnvLoad(cfg), nil
	}
	c, err := config.Load(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("load %s: %w: %w", dir, errVariantConfigUnreadable, err)
	}
	return c, newConfigEnvironmentAt(c, c.FileDir()), nil
}

func (r lifecycleRunner) Up(ctx context.Context, dir, instance, plan string, vars map[string]string, names []string, wait bool) (*compareObserved, error) {
	runtime, orch, err := r.open(dir, instance, plan, vars)
	if err != nil {
		return nil, err
	}
	if err := orch.Up(ctx, lifecycle.UpOptions{Wait: wait, Names: names, Env: runtime.plan.EnvironmentName}); err != nil {
		return nil, err
	}
	return observe(ctx, runtime, orch, names)
}

func (r lifecycleRunner) Down(ctx context.Context, dir, instance, plan string, vars map[string]string, names []string) error {
	runtime, orch, err := r.open(dir, instance, plan, vars)
	if err != nil {
		return err
	}
	return orch.Down(ctx, lifecycle.DownOptions{Names: names, Env: runtime.plan.EnvironmentName})
}

func (r lifecycleRunner) Status(ctx context.Context, dir, instance, plan string, vars map[string]string, names []string) (*compareObserved, error) {
	runtime, orch, err := r.open(dir, instance, plan, vars)
	if err != nil {
		return nil, err
	}
	return observe(ctx, runtime, orch, names)
}

func (lifecycleRunner) Endpoints(_ context.Context, dir, plan string, vars map[string]string) ([]planEndpointOutput, error) {
	c, load, err := compareConfigAt(dir)
	if err != nil {
		return nil, err
	}
	// resolvePlanRuntime merges the vars into the environment it is given. The calling
	// checkout's environment is the command's own, which the shared entries' calls read, so
	// one variant's vars are merged into a copy and never into it.
	load = &envLoad{env: load.env.Clone(), report: load.report}
	runtime, err := resolvePlanRuntime(c, load, plan, vars)
	if err != nil {
		return nil, err
	}
	return interpolatedEndpoints(runtime), nil
}

func interpolatedEndpoints(runtime *planRuntime) []planEndpointOutput {
	endpoints := planEndpointOutputs(filterEndpoints(runtime.config.Endpoints, runtime.plan.EndpointTags))
	for i := range endpoints {
		endpoints[i].URL = runtime.env.Interpolate(endpoints[i].URL)
	}
	return endpoints
}

// observe reads the status of names and the plan's endpoints. An endpoint URL is interpolated
// with the environment the entries ran in, so a variant's URL carries the port it was given.
// A status read that fails is an error, not an empty observation: compare reports facts and
// must not present "could not look" as "nothing there".
func observe(ctx context.Context, runtime *planRuntime, orch *lifecycle.Orchestrator, names []string) (*compareObserved, error) {
	status, err := orch.Status(ctx)
	if err != nil {
		return nil, fmt.Errorf("read status: %w", err)
	}
	return &compareObserved{Status: filterStatusByNames(status, names), Endpoints: interpolatedEndpoints(runtime)}, nil
}

// pathExists reports whether p exists. Used to skip stopping entries in a checkout that is
// already gone: there is nothing left there to stop or to read.
func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// annotateProcesses adds the pid and log file dva keeps for each native entry of the checkout
// in configDir. The lifecycle status carries neither, and the files are where `dva logs` and
// the stop path read them, so this reads the same files. Entries without a pid file (compose
// and the like) simply have nothing to add.
func annotateProcesses(row *compareVariantResult, configDir, instance string, entries []string) {
	pctx := &lifecycle.PluginContext{ConfigDir: configDir, Instance: instance}
	for _, e := range entries {
		if data, err := os.ReadFile(pctx.PidFile(e)); err == nil {
			if pid, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil && pid > 0 {
				if row.Pids == nil {
					row.Pids = map[string]int{}
				}
				row.Pids[e] = pid
			}
		}
		if log := pctx.LogFile(e); pathExists(log) {
			row.LogPaths = append(row.LogPaths, log)
		}
	}
}
