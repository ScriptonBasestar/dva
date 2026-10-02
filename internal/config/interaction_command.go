package config

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// InteractionCommand defines a command in the interaction section.
type InteractionCommand struct {
	Description string                         `yaml:"description"`
	Service     string                         `yaml:"service"`
	Workdir     string                         `yaml:"workdir"`
	User        string                         `yaml:"user"`
	DefaultArgs string                         `yaml:"default_args"`
	Environment map[string]string              `yaml:"environment"`
	Compose     *ComposeOptions                `yaml:"compose"`
	Shell       *bool                          `yaml:"shell"`
	Entrypoint  string                         `yaml:"entrypoint"`
	Runner      string                         `yaml:"runner"`
	Pod         string                         `yaml:"pod"`
	Subcommands map[string]*InteractionCommand `yaml:"subcommands"`
	Tags        []string                       `yaml:"tags"`
	Destructive *bool                          `yaml:"destructive,omitempty"`

	// Command execution: one of the following should be set.
	// command: string or []string — single command or list executed sequentially
	Command string `yaml:"-"` // set by UnmarshalYAML from a scalar
	// CommandLines holds multiple commands when command: is specified as a list.
	CommandLines []string `yaml:"-"` // set by UnmarshalYAML from a sequence
	// script: inline shell script block (multi-line heredoc / block scalar)
	Script string `yaml:"script"`
	// script_file: path to an external shell script (relative to dva.yml)
	ScriptFile string `yaml:"script_file"`
	// steps: named steps executed sequentially (reuses ProvisionItem)
	Steps []ProvisionItem `yaml:"steps"`

	// Hook fields: extend or replace hookable built-in commands (up, down, build, etc.)
	Before  []ProvisionItem `yaml:"before"`
	Replace []ProvisionItem `yaml:"replace"`
	After   []ProvisionItem `yaml:"after"`

	SubprojectPath string `yaml:"-"`

	// SubprojectName is the subprojects: map key this command was imported from ("" for
	// a locally declared command). Unlike SubprojectPath, this is a logical name, not a
	// filesystem location, so it is safe to publish on ls/manifest output (TASK-333's
	// owner field) where SubprojectPath's absolute local path is not (see owner's
	// comment below).
	SubprojectName string `yaml:"-"`

	// CanonicalAddress is the "<subproject>/<key>" address subproject.go's import loop
	// always assigns first when this command is imported ("" for a locally declared
	// command). An import's optional `as:` alias points the SAME *InteractionCommand at
	// a second map key — the "one declaration exposed twice" identity
	// warnDuplicatePlanDeclarations already relies on for plans — and CanonicalAddress is
	// how a reader tells which of the two keys is the canonical one without re-deriving
	// subprojectName+"/"+name from context or comparing pointers itself (TASK-333).
	CanonicalAddress string `yaml:"-"`

	// owner is the fully loaded configuration that declared this command when it is
	// imported from a subproject, mirroring PlanConfig.owner (TASK-262/264). It has no
	// YAML representation on purpose: importing an interaction exposes a route in the
	// parent, not the child's declaration namespace, and the owner holds the child's
	// absolute local paths, which must never reach manifest, show or list output.
	owner *Config
}

// OwnerConfig returns the configuration whose declarations this command resolves
// against. Locally declared and manually constructed commands have no recorded owner,
// so fallback preserves their historical behavior.
func (c *InteractionCommand) OwnerConfig(fallback *Config) *Config {
	if c != nil && c.owner != nil {
		return c.owner
	}
	return fallback
}

// HasHooks reports whether the command defines any hook steps (before/replace/after).
func (c *InteractionCommand) HasHooks() bool {
	return len(c.Before) > 0 || len(c.Replace) > 0 || len(c.After) > 0
}

// HasSteps reports whether the command uses step-based execution.
func (c *InteractionCommand) HasSteps() bool {
	return len(c.Steps) > 0
}

// HasScript reports whether the command uses an inline script.
func (c *InteractionCommand) HasScript() bool {
	return c.Script != ""
}

// HasScriptFile reports whether the command references an external script file.
func (c *InteractionCommand) HasScriptFile() bool {
	return c.ScriptFile != ""
}

// HasMultiCommand reports whether the command was specified as a list.
func (c *InteractionCommand) HasMultiCommand() bool {
	return len(c.CommandLines) > 0
}

// EffectiveCommand was here, joining CommandLines with " && " "for display". It is deleted
// rather than wired up, and TASK-178 is where the reasoning lives: no runner gives a list those
// semantics. Local runs one subprocess per line, and compose and kubectl now run one exec per
// line, so `cd build` and `make` as two lines do not compose the way `cd build && make` does —
// a helper rendering them as if they did describes an execution dva does not perform. It had
// zero non-test callers for its whole life, which is much of how the gap it was written for
// stayed invisible: the handling existed on paper and nothing reached it.
//
// polymorphicCommand holds the polymorphic `command:` field — a scalar string or a sequence of
// strings — and is decoded by yaml.Decode rather than a hand-written node scan. Riding on Decode
// is what makes `command` honour merge keys (`<<:`) like every other InteractionCommand field
// (TASK-162).
type polymorphicCommand struct {
	scalar string
	lines  []string
}

// UnmarshalYAML accepts a scalar string or a sequence of strings, exposing both the single
// display form (scalar) and the list form (sequence).
func (p *polymorphicCommand) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		p.scalar = strings.TrimSpace(value.Value)
	case yaml.SequenceNode:
		var lines []string
		if err := value.Decode(&lines); err != nil {
			return fmt.Errorf("command: expected string or list of strings: %w", err)
		}
		for i, l := range lines {
			lines[i] = strings.TrimSpace(l)
		}
		p.lines = lines
		if len(lines) > 0 {
			// The first line also lands in the scalar, and the comment here used to call that
			// "display/backward-compat". Display is where it ended up: for every runner but
			// local the scalar *was* the execution, so a two-line list ran one line and the
			// plan printed one line (TASK-178). Both of those now read CommandLines. What is
			// left for the scalar is the reachability check in validate_warnings.go, which
			// asks whether an interaction declares any work at all.
			p.scalar = lines[0]
		}
	default:
		return fmt.Errorf("command: unsupported YAML type (expected string or sequence)")
	}
	return nil
}

// UnmarshalYAML implements custom unmarshaling for InteractionCommand. Every field is decoded
// normally via a plain type alias; the polymorphic `command` field goes through polymorphicCommand
// so it stays on the Decode path and honours merge keys like its neighbours (TASK-162).
func (c *InteractionCommand) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("interaction command: expected mapping node")
	}

	// Decode all fields using the tag-based alias. `command` is polymorphic (scalar or sequence),
	// so it goes through polymorphicCommand rather than a bare string — that keeps it on the Decode
	// path, which is what makes it honour merge keys like its neighbours (TASK-162). The alias has
	// no UnmarshalYAML of its own to avoid recursion.
	type plain struct {
		Description string                         `yaml:"description"`
		Service     string                         `yaml:"service"`
		Workdir     string                         `yaml:"workdir"`
		User        string                         `yaml:"user"`
		DefaultArgs string                         `yaml:"default_args"`
		Environment map[string]string              `yaml:"environment"`
		Compose     *ComposeOptions                `yaml:"compose"`
		Shell       *bool                          `yaml:"shell"`
		Entrypoint  string                         `yaml:"entrypoint"`
		Runner      string                         `yaml:"runner"`
		Pod         string                         `yaml:"pod"`
		Subcommands map[string]*InteractionCommand `yaml:"subcommands"`
		Tags        []string                       `yaml:"tags"`
		Destructive *bool                          `yaml:"destructive,omitempty"`
		Script      string                         `yaml:"script"`
		ScriptFile  string                         `yaml:"script_file"`
		Steps       []ProvisionItem                `yaml:"steps"`
		Before      []ProvisionItem                `yaml:"before"`
		Replace     []ProvisionItem                `yaml:"replace"`
		After       []ProvisionItem                `yaml:"after"`
		Command     polymorphicCommand             `yaml:"command"`
	}
	var p plain
	if err := node.Decode(&p); err != nil {
		return err
	}
	c.Description = p.Description
	c.Service = p.Service
	c.Workdir = p.Workdir
	c.User = p.User
	c.DefaultArgs = p.DefaultArgs
	c.Environment = p.Environment
	c.Compose = p.Compose
	c.Shell = p.Shell
	c.Entrypoint = p.Entrypoint
	c.Runner = p.Runner
	c.Pod = p.Pod
	c.Subcommands = p.Subcommands
	c.Tags = p.Tags
	c.Destructive = p.Destructive
	c.Script = p.Script
	c.ScriptFile = p.ScriptFile
	c.Steps = p.Steps
	c.Before = p.Before
	c.Replace = p.Replace
	c.After = p.After
	c.Command = p.Command.scalar
	c.CommandLines = p.Command.lines
	return nil
}

// ShellEnabled returns whether shell mode is enabled (default: true).
func (c *InteractionCommand) ShellEnabled() bool {
	if c.Shell == nil {
		return true
	}
	return *c.Shell
}

// IsDestructive reports whether this interaction command is marked destructive.
func (c *InteractionCommand) IsDestructive() bool {
	return c != nil && c.Destructive != nil && *c.Destructive
}
