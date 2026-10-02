package skillinstall

import (
	"errors"
	"fmt"
	bundled "github.com/ScriptonBasestar/dva/skills"
	"os"
	"path/filepath"
	"slices"
	"sort"
)

func containsRuntimes(have, wanted []Runtime) bool {
	set := make(map[Runtime]bool, len(have))
	for _, runtime := range have {
		set[runtime] = true
	}
	for _, runtime := range wanted {
		if !set[runtime] {
			return false
		}
	}
	return true
}

func unionRuntimes(left, right []Runtime) []Runtime {
	set := make(map[Runtime]bool, len(left)+len(right))
	for _, runtime := range append(append([]Runtime(nil), left...), right...) {
		set[runtime] = true
	}
	result := make([]Runtime, 0, len(set))
	for runtime := range set {
		result = append(result, runtime)
	}
	slices.Sort(result)
	return result
}

func removeRuntimes(have, removed []Runtime) []Runtime {
	remove := make(map[Runtime]bool, len(removed))
	for _, runtime := range removed {
		remove[runtime] = true
	}
	var result []Runtime
	for _, runtime := range have {
		if !remove[runtime] {
			result = append(result, runtime)
		}
	}
	slices.Sort(result)
	return result
}

func intersectRuntimes(left, right []Runtime) []Runtime {
	wanted := make(map[Runtime]bool, len(right))
	for _, runtime := range right {
		wanted[runtime] = true
	}
	var result []Runtime
	for _, runtime := range left {
		if wanted[runtime] {
			result = append(result, runtime)
		}
	}
	slices.Sort(result)
	return result
}
func resolve(options Options) (Options, []destination, error) {
	if options.Scope != ScopeUser && options.Scope != ScopeProject {

		return Options{}, nil, fmt.Errorf("skill install scope must be %q or %q", ScopeUser, ScopeProject)
	}
	if (options.Takeover || options.RestoreTakeoverBackup) && len(options.Runtimes) == 0 {
		return Options{}, nil, errors.New("--takeover and --restore-takeover-backup require at least one explicit --runtime")
	}
	if options.HomeDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return Options{}, nil, fmt.Errorf("resolve home directory: %w", err)
		}
		options.HomeDir = home
	}
	home, err := filepath.Abs(options.HomeDir)
	if err != nil {
		return Options{}, nil, err
	}
	options.HomeDir = home
	if options.ProjectRoot == "" {
		project, err := os.Getwd()
		if err != nil {
			return Options{}, nil, err
		}
		options.ProjectRoot = project
	}
	project, err := filepath.Abs(options.ProjectRoot)
	if err != nil {
		return Options{}, nil, err
	}
	options.ProjectRoot = project
	if options.StateRoot == "" {
		if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
			options.StateRoot = filepath.Join(xdg, "dva")
		} else {
			options.StateRoot = filepath.Join(home, ".local", "state", "dva")
		}
	}
	state, err := filepath.Abs(options.StateRoot)
	if err != nil {
		return Options{}, nil, err
	}
	options.StateRoot = state
	if options.ClaimRoot == "" {
		options.ClaimRoot = filepath.Dir(state)
	}
	claimRoot, err := filepath.Abs(options.ClaimRoot)
	if err != nil {
		return Options{}, nil, err
	}
	options.ClaimRoot = claimRoot
	if options.Version == "" {
		options.Version = "unknown"
	}
	if len(options.Runtimes) == 0 {
		options.Runtimes = DefaultRuntimes()
	}
	seen := map[Runtime]bool{}
	groups := map[string][]Runtime{}
	for _, runtime := range options.Runtimes {
		if seen[runtime] {
			continue
		}
		seen[runtime] = true
		path, err := runtimePath(runtime, options.Scope, home, project)
		if err != nil {
			return Options{}, nil, err
		}
		groups[path] = append(groups[path], runtime)
	}
	paths := make([]string, 0, len(groups))
	for path := range groups {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	destinations := make([]destination, 0, len(paths))
	for _, path := range paths {
		runtimes := groups[path]
		slices.Sort(runtimes)
		destinations = append(destinations, destination{path: path, runtimes: runtimes})
	}
	return options, destinations, nil
}

func runtimePath(runtime Runtime, scope Scope, home, project string) (string, error) {
	var relative string
	switch scope {
	case ScopeUser:
		switch runtime {
		case RuntimeClaudeCode:
			relative = ".claude/skills"
		case RuntimeCodex:
			relative = ".agents/skills"
		case RuntimeOpenCode:
			relative = ".config/opencode/skills"
		case RuntimeGrok:
			relative = ".grok/skills"
		case RuntimeAntigravity:
			relative = ".gemini/config/skills"
		case RuntimeAgentMesh:
			relative = ".config/agent-mesh/skills/dva"
		default:
			return "", fmt.Errorf("unsupported skill runtime %q", runtime)
		}
		return filepath.Join(home, relative), nil
	case ScopeProject:
		switch runtime {
		case RuntimeClaudeCode:
			relative = ".claude/skills"
		case RuntimeCodex, RuntimeAntigravity:
			relative = ".agents/skills"
		case RuntimeOpenCode:
			relative = ".opencode/skills"
		case RuntimeGrok:
			relative = ".grok/skills"
		case RuntimeAgentMesh:
			relative = ".agent-mesh/skills/dva"
		default:
			return "", fmt.Errorf("unsupported skill runtime %q", runtime)
		}
		return filepath.Join(project, relative), nil
	default:
		return "", fmt.Errorf("unsupported skill scope %q", scope)
	}
}

func resultEntry(target destination, version, bundleSHA string) DestinationResult {
	return DestinationResult{
		Destination: target.path, Runtimes: append([]Runtime(nil), target.runtimes...),
		Skills: append([]string(nil), bundled.Names...), SourceVersion: version, SourceBundleSHA: bundleSHA,
		RuntimeStatuses: make([]RuntimeStatus, 0, len(target.runtimes)),
	}
}

func setAllRuntimeStatuses(entry *DestinationResult, status string) {
	entry.RuntimeStatuses = entry.RuntimeStatuses[:0]
	for _, runtime := range entry.Runtimes {
		entry.RuntimeStatuses = append(entry.RuntimeStatuses, RuntimeStatus{Runtime: runtime, Status: status})
	}
}

func setMembershipStatuses(entry *DestinationResult, present []Runtime, presentStatus, absentStatus string) string {
	set := make(map[Runtime]bool, len(present))
	for _, runtime := range present {
		set[runtime] = true
	}
	entry.RuntimeStatuses = entry.RuntimeStatuses[:0]
	allPresent, allAbsent := true, true
	for _, runtime := range entry.Runtimes {
		status := absentStatus
		if set[runtime] {
			status = presentStatus
			allAbsent = false
		} else {
			allPresent = false
		}
		entry.RuntimeStatuses = append(entry.RuntimeStatuses, RuntimeStatus{Runtime: runtime, Status: status})
	}
	if allPresent {
		return presentStatus
	}
	if allAbsent {
		return absentStatus
	}
	return "partial"
}
