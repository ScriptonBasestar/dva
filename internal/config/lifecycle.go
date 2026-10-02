package config

import (
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// knownPluginNames maps entry/YAML key names to canonical plugin type strings.
// Keep in sync with lifecycle.AllPluginTypes() in internal/lifecycle/plugin_type.go.
var knownPluginNames = map[string]string{
	"compose":        "compose",
	"process":        "process",
	"script":         "script",
	"docker":         "docker",
	"kubectl":        "kubectl",
	"helm":           "helm",
	"kustomize":      "kustomize",
	"tilt":           "tilt",
	"skaffold":       "skaffold",
	"podman_compose": "podman-compose",
	"podman-compose": "podman-compose",
	"vagrant":        "vagrant",
	"sam":            "sam",
	"serverless":     "serverless",
	"multipass":      "multipass",
}

// LifecycleEntry defines a single entry in the stack pipeline.
type LifecycleEntry struct {
	Name          string                       `yaml:"-"` // populated from map key
	Plugin        string                       `yaml:"plugin,omitempty"`
	Order         int                          `yaml:"order"`
	Description   string                       `yaml:"description"`
	Tags          []string                     `yaml:"tags"`
	Vars          map[string]string            `yaml:"vars"`
	Exports       map[string]string            `yaml:"exports"`
	HealthChecks  map[string]HealthCheckConfig `yaml:"health_checks"`
	DefaultRunner string                       `yaml:"default_runner"`
	Runners       map[string]any               `yaml:"runners"`

	// Source declares an externally-owned stack fetched (git) or referenced
	// (path) before the entry's runner executes. When set, runner file paths
	// and working directory resolve against the sourced directory. (TASK-051)
	Source *SourceConfig `yaml:"source,omitempty"`

	// Tunnel declares an access prerequisite for a remote kubectl/helm entry
	// (docs/68): DVA opens the tunnel before the entry runs, waits until it is
	// authenticated and forwarding, and closes only the process it started.
	// Only cloudflared is supported in v1; validation restricts the field to
	// kubectl and helm entries. (TASK-459)
	Tunnel *TunnelConfig `yaml:"tunnel,omitempty"`

	// Optional marks this entry as optional — if the directory it declares does not
	// exist, the entry is dropped from the plan instead of failing the plan. The
	// directory consulted is the one belonging to the runner the plan selected, so an
	// entry declaring several is judged on the one that would actually have run.
	//
	// A skip prints as a warning on every execution path and is also recorded in the
	// resolution trace that --dry-run shows in full.
	//
	// Optional tolerates a missing directory, not a broken declaration. An optional
	// entry naming an undeclared or unresolvable runner fails the plan with that
	// error rather than being dropped: a directory that is not checked out is the
	// case this field exists for, while a declaration that does not resolve is a
	// typo, and silently dropping it would hide the typo forever. (TASK-319, TASK-374)
	Optional bool `yaml:"optional,omitempty"`

	// Primary marks this compose entry as the primary compose entry. When set,
	// PrimaryComposeEntry() returns this entry instead of inferring from order.
	// Only one entry should have Primary=true. (TASK-319)
	Primary bool `yaml:"primary,omitempty"`

	// --- Tier 1: Core ---
	Compose *ComposePluginConfig `yaml:"compose,omitempty"`
	Process *ProcessPluginConfig `yaml:"process,omitempty"`
	Script  *ScriptPluginConfig  `yaml:"script,omitempty"`
	Docker  *DockerPluginConfig  `yaml:"docker,omitempty"`
	Kubectl *KubectlPluginConfig `yaml:"kubectl,omitempty"`
	Helm    *HelmPluginConfig    `yaml:"helm,omitempty"`

	// --- Tier 2: Extended ---
	Kustomize     *KustomizePluginConfig     `yaml:"kustomize,omitempty"`
	Tilt          *TiltPluginConfig          `yaml:"tilt,omitempty"`
	Skaffold      *SkaffoldPluginConfig      `yaml:"skaffold,omitempty"`
	PodmanCompose *PodmanComposePluginConfig `yaml:"podman_compose,omitempty"`
	Vagrant       *VagrantPluginConfig       `yaml:"vagrant,omitempty"`

	// --- Tier 3: Niche ---
	SAM        *SAMPluginConfig        `yaml:"sam,omitempty"`
	Serverless *ServerlessPluginConfig `yaml:"serverless,omitempty"`
	Multipass  *MultipassPluginConfig  `yaml:"multipass,omitempty"`

	// rawNode stores the YAML node for deferred plugin resolution
	// when plugin type is inferred from the entry name.
	rawNode *yaml.Node `yaml:"-"`
}

// SourceConfig declares where an externally-owned stack is obtained from.
// Exactly one of Git or Path must be set. Git repositories are cloned into a
// cache directory; Path references a local directory in place. (TASK-051)
type SourceConfig struct {
	Git  string `yaml:"git"`
	Ref  string `yaml:"ref"`
	Path string `yaml:"path"`
}

// IsGit reports whether this source is fetched from a git repository.
func (s *SourceConfig) IsGit() bool { return s != nil && strings.TrimSpace(s.Git) != "" }

// Validate ensures exactly one of git/path is set. ref is only meaningful with git.
func (s *SourceConfig) Validate() error {
	if s == nil {
		return nil
	}
	git := strings.TrimSpace(s.Git)
	path := strings.TrimSpace(s.Path)
	ref := strings.TrimSpace(s.Ref)
	switch {
	case git == "" && path == "":
		return fmt.Errorf("source: requires either 'git' or 'path'")
	case git != "" && path != "":
		return fmt.Errorf("source: 'git' and 'path' are mutually exclusive")
	case path != "" && ref != "":
		return fmt.Errorf("source: 'ref' is only valid with 'git'")
	}
	return nil
}

type NativeRunnerConfig struct {
	Dir       string            `yaml:"dir"`
	Build     string            `yaml:"build"`
	PostBuild string            `yaml:"post_build"`
	Run       string            `yaml:"run"`
	Env       map[string]string `yaml:"env"`
}

type DockerRunnerConfig struct {
	Image   string            `yaml:"image"`
	Run     string            `yaml:"run"`
	Build   string            `yaml:"build"`
	Command string            `yaml:"command"`
	Ports   []string          `yaml:"ports"`
	Volumes []string          `yaml:"volumes"`
	Env     map[string]string `yaml:"env"`
	Options []string          `yaml:"options"`
}

func (e *LifecycleEntry) GetRunnerConfig(runnerName string) (any, error) {
	selected := normalizeRunnerName(runnerName)
	if selected == "" {
		selected = normalizeRunnerName(e.DefaultRunner)
	}

	if len(e.Runners) > 0 {
		if selected == "" && len(e.Runners) == 1 {
			for k := range e.Runners {
				selected = normalizeRunnerName(k)
				break
			}
		}

		if selected != "" {
			if cfg, ok := e.Runners[selected]; ok {
				return cfg, nil
			}
			if selected == "podman-compose" {
				if cfg, ok := e.Runners["podman_compose"]; ok {
					return cfg, nil
				}
			}
			if selected == "podman_compose" {
				if cfg, ok := e.Runners["podman-compose"]; ok {
					return cfg, nil
				}
			}
			return nil, fmt.Errorf("runner %q is not declared in entry %q", selected, e.Name)
		}
	}

	if selected == "" {
		selected = normalizeRunnerName(e.DetectPlugin())
	}

	switch selected {
	case "compose":
		if e.Compose == nil {
			return nil, fmt.Errorf("runner %q is not configured in entry %q", selected, e.Name)
		}
		return e.Compose, nil
	case "process":
		if e.Process == nil {
			return nil, fmt.Errorf("runner %q is not configured in entry %q", selected, e.Name)
		}
		return e.Process, nil
	case "script":
		if e.Script == nil {
			return nil, fmt.Errorf("runner %q is not configured in entry %q", selected, e.Name)
		}
		return e.Script, nil
	case "docker":
		if e.Docker == nil {
			return nil, fmt.Errorf("runner %q is not configured in entry %q", selected, e.Name)
		}
		return e.Docker, nil
	case "kubectl":
		if e.Kubectl == nil {
			return nil, fmt.Errorf("runner %q is not configured in entry %q", selected, e.Name)
		}
		return e.Kubectl, nil
	case "helm":
		if e.Helm == nil {
			return nil, fmt.Errorf("runner %q is not configured in entry %q", selected, e.Name)
		}
		return e.Helm, nil
	case "kustomize":
		if e.Kustomize == nil {
			return nil, fmt.Errorf("runner %q is not configured in entry %q", selected, e.Name)
		}
		return e.Kustomize, nil
	case "tilt":
		if e.Tilt == nil {
			return nil, fmt.Errorf("runner %q is not configured in entry %q", selected, e.Name)
		}
		return e.Tilt, nil
	case "skaffold":
		if e.Skaffold == nil {
			return nil, fmt.Errorf("runner %q is not configured in entry %q", selected, e.Name)
		}
		return e.Skaffold, nil
	case "podman-compose":
		if e.PodmanCompose == nil {
			return nil, fmt.Errorf("runner %q is not configured in entry %q", selected, e.Name)
		}
		return e.PodmanCompose, nil
	case "vagrant":
		if e.Vagrant == nil {
			return nil, fmt.Errorf("runner %q is not configured in entry %q", selected, e.Name)
		}
		return e.Vagrant, nil
	case "sam":
		if e.SAM == nil {
			return nil, fmt.Errorf("runner %q is not configured in entry %q", selected, e.Name)
		}
		return e.SAM, nil
	case "serverless":
		if e.Serverless == nil {
			return nil, fmt.Errorf("runner %q is not configured in entry %q", selected, e.Name)
		}
		return e.Serverless, nil
	case "multipass":
		if e.Multipass == nil {
			return nil, fmt.Errorf("runner %q is not configured in entry %q", selected, e.Name)
		}
		return e.Multipass, nil
	}

	return nil, fmt.Errorf("runner %q is not configured in entry %q", selected, e.Name)
}

// DefaultRunnerName returns default_runner canonicalized the same way RunnerNames canonicalizes
// the runners map, so the two are comparable. Every other reader of DefaultRunner already
// normalizes it (runnerPluginName, resolveRunner, source.go, lifecycle/resolver.go); this exports
// that step for callers outside the package, which otherwise compare a raw `podman_compose`
// against a normalized `podman-compose` and conclude the default names an undeclared runner.
func (e *LifecycleEntry) DefaultRunnerName() string {
	return normalizeRunnerName(e.DefaultRunner)
}

func (e *LifecycleEntry) RunnerNames() []string {
	names := make([]string, 0, len(e.Runners)+1)
	if len(e.Runners) > 0 {
		for name := range e.Runners {
			names = append(names, normalizeRunnerName(name))
		}
	}
	if len(names) == 0 {
		if detected := normalizeRunnerName(e.DetectPlugin()); detected != "" {
			names = append(names, detected)
		}
	}
	sort.Strings(names)
	if len(names) < 2 {
		return names
	}
	uniq := names[:1]
	for i := 1; i < len(names); i++ {
		if names[i] != names[i-1] {
			uniq = append(uniq, names[i])
		}
	}
	return uniq
}

// ResolvePluginFromName infers the plugin type from the entry name when neither
// plugin: field nor nested config is present, then enforces the compose contract.
// Called after Name is set from the map key in Config.Load().
//
// It is the only hook both load paths (Config.Load and Config.Merge) run per
// entry once Name is known, which is why the compose check lives here: the
// legacy shapes it rejects are only identifiable after name inference has run.
func (e *LifecycleEntry) ResolvePluginFromName() error {
	if err := e.resolvePluginFromName(); err != nil {
		return err
	}
	return e.rejectLegacyComposeShape()
}

// rejectLegacyComposeShape refuses every compose declaration schema.json refuses,
// so Load() and Validate() answer the same question the same way.
//
// e.Compose is written by exactly three paths, all legacy: entry-name inference
// (stack.compose carrying flat compose keys), an explicit plugin: compose, and a
// nested compose: sub-key. The supported shape decodes into e.Runners and never
// touches e.Compose, so a non-nil e.Compose is precisely the disagreement schema
// validation reports — unhelpfully — as "Must not validate the schema (not)".
func (e *LifecycleEntry) rejectLegacyComposeShape() error {
	if e.Compose == nil {
		return nil
	}
	return fmt.Errorf("entry %q: compose must be declared under runners.compose, not on the entry itself\n"+
		"  rewrite it as:\n"+
		"    %s:\n"+
		"      default_runner: compose\n"+
		"      runners:\n"+
		"        compose:\n"+
		"          files: [...]   # move this entry's existing compose keys here",
		e.Name, e.Name)
}

func (e *LifecycleEntry) resolvePluginFromName() error {
	// Checked before the guard below, not after it: DetectPlugin now resolves the runners
	// shape too (TASK-102), so a runners entry would exit at that guard and leave rawNode
	// holding its parsed YAML for the life of the config. Nothing reads rawNode outside
	// this function, so this only keeps it from being retained — the return is the same.
	if len(e.Runners) > 0 {
		e.rawNode = nil
		return nil
	}
	if e.Plugin != "" || e.DetectPlugin() != "" || e.rawNode == nil {
		return nil
	}
	if pt, ok := knownPluginNames[e.Name]; ok {
		e.Plugin = pt
		if err := e.resolvePluginConfig(e.rawNode); err != nil {
			return fmt.Errorf("entry %q: %w", e.Name, err)
		}
	}
	e.rawNode = nil
	return nil
}

// DetectPlugin returns the plugin type string.
// Uses Plugin field if set, otherwise inspects nested config pointers.
func (e *LifecycleEntry) DetectPlugin() string {
	if e.Plugin != "" {
		return e.Plugin
	}
	switch {
	case e.Compose != nil:
		return "compose"
	case e.Process != nil:
		return "process"
	case e.Script != nil:
		return "script"
	case e.Docker != nil:
		return "docker"
	case e.Kubectl != nil:
		return "kubectl"
	case e.Helm != nil:
		return "helm"
	case e.Kustomize != nil:
		return "kustomize"
	case e.Tilt != nil:
		return "tilt"
	case e.Skaffold != nil:
		return "skaffold"
	case e.PodmanCompose != nil:
		return "podman-compose"
	case e.Vagrant != nil:
		return "vagrant"
	case e.SAM != nil:
		return "sam"
	case e.Serverless != nil:
		return "serverless"
	case e.Multipass != nil:
		return "multipass"
	}

	// The modern default_runner:/runners: shape writes none of the fields above — only
	// resolveRunnerPlugin backfills them, and it runs solely on the copies SortedStack
	// returns. Entries reached by name (FindStackEntry is a bare c.Stack read) are still
	// raw, so without this they detect as "" and every caller falls through to its default
	// branch. Resolved read-only rather than by backfilling, because that pointer is shared
	// through c.Stack and because a non-nil e.Compose is what rejectLegacyComposeShape uses
	// to identify the legacy shape. TASK-102.
	if name := e.runnerPluginName(); name != "" {
		if name == "native" {
			// runners.native is an alias for the process plugin, as applyRunnerConfig maps it.
			return "process"
		}
		return name
	}
	return ""
}

// runnerPluginName returns the runner selected by DefaultRunner, or the sole
// declared runner when DefaultRunner is empty. It reports the declared name
// only; resolveRunnerPlugin decides whether that runner can be served.
func (e *LifecycleEntry) runnerPluginName() string {
	if name := normalizeRunnerName(e.DefaultRunner); name != "" {
		return name
	}
	if len(e.Runners) == 1 {
		for name := range e.Runners {
			return normalizeRunnerName(name)
		}
	}
	return ""
}
