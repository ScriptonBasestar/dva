package cirun

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func compositionGrantValue(env []string) (string, bool) {
	if value, present := os.LookupEnv(config.CICompositionEnv); present {
		return value, true
	}
	var value string
	var present bool
	for _, entry := range env {
		if found, ok := strings.CutPrefix(entry, config.CICompositionEnv+"="); ok {
			value, present = found, true
		}
	}
	return value, present
}

// PreviewWorkers resolves the same profile and cooperative limits as Run without
// acquiring child locks, registering an execution or starting any command.
// An inherited grant is a point-in-time observation; Run revalidates admission.
func PreviewWorkers(opts Options) ([]StepWorkers, error) {
	declared := &config.Config{CI: &config.CIConfig{Profiles: map[string]config.CIProfile{opts.ProfileName: opts.Profile}}}
	profile, err := declared.ResolveCIProfile(opts.ProfileName)
	if err != nil {
		return nil, err
	}
	value, present := compositionGrantValue(opts.Env)
	if !present {
		return WorkerPlan(profile, opts.Env), nil
	}
	state, err := stateDirectory("")
	if err != nil {
		return nil, err
	}
	if opts.lockDirectory != "" {
		state = opts.lockDirectory
	}
	state, err = filepath.Abs(state)
	if err != nil {
		return nil, err
	}
	state, err = filepath.EvalSymlinks(state)
	if err != nil {
		return nil, err
	}
	root, err := filepath.EvalSymlinks(opts.Root)
	if err != nil {
		return nil, err
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	parent, inherited := os.LookupEnv(parentRunEnv)
	if !inherited {
		parent = environmentValue(opts.Env, parentRunEnv)
	}
	grant, err := admitComposition(state, root, parent, value, time.Now(), nil)
	if err != nil {
		return nil, err
	}
	scope := scopeFor(state, grant.Ancestry[len(grant.Ancestry)-1])
	if err := scope.verifyGrant(*grant); err != nil {
		return nil, err
	}
	if err := scope.open(); err != nil {
		return nil, err
	}
	env := compositionWorkerEnvironment(opts.Env, *grant, opts.WorkerEnvironment)
	workers, err := constrainCompositionWorkers(profile, env, grant.JobsLimit, availableCPUs())
	if err != nil {
		return nil, err
	}
	return workerPlan(profile, env, workers.jobs, workers.source), nil
}
