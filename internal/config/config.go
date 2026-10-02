package config

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ModulesDirExt is the extension for module files.
// Config represents the parsed dva.yml configuration.
type Config struct {
	Version          string                         `yaml:"version"`
	Vars             map[string]string              `yaml:"vars"`
	Environment      map[string]string              `yaml:"environment"`
	EnvFile          any                            `yaml:"env_file"`
	EnvBridge        *EnvBridgeConfig               `yaml:"env_bridge"`
	Interaction      map[string]*InteractionCommand `yaml:"interaction"`
	Secrets          *SecretsConfig                 `yaml:"secrets,omitempty" json:"secrets,omitempty"`
	Jobs             map[string]JobConfig           `yaml:"jobs,omitempty" json:"jobs,omitempty"`
	CI               *CIConfig                      `yaml:"ci" json:"ci"`
	Provision        ProvisionConfig                `yaml:"provision"`
	Infra            map[string]InfraConfig         `yaml:"infra"`
	Modules          []string                       `yaml:"modules"`
	Devcontainer     map[string]any                 `yaml:"devcontainer"`
	Subprojects      map[string]SubprojectConfig    `yaml:"subprojects"`
	HealthChecks     map[string]HealthCheckConfig   `yaml:"health_checks"`
	Endpoints        map[string]EndpointConfig      `yaml:"endpoints"`
	DefaultMode      string                         `yaml:"default_mode"`
	SuggestionIgnore []string                       `yaml:"suggestion_ignore"`
	Suggestions      *SuggestionCategories          `yaml:"suggestions"`
	DriftIgnore      []string                       `yaml:"drift_ignore"`
	Modes            map[string]ModeConfig          `yaml:"modes"`
	Environments     map[string]EnvironmentProfile  `yaml:"environments"`
	Plans            map[string]*PlanConfig         `yaml:"plans"`
	DefaultPlanName  string                         `yaml:"default_plan"`
	Sites            map[string]*SiteConfig         `yaml:"sites"`
	Ssh              SshConfig                      `yaml:"ssh"`
	DoctorChecks     []DoctorCheck                  `yaml:"checks"`
	Stack            map[string]*LifecycleEntry     `yaml:"stack"`

	// Internal fields
	filePath string

	// deferEntryProblems and loadProblems back the CollectEntryProblems load option:
	// entry-shape errors that a strict load would return are recorded here instead.
	deferEntryProblems bool
	loadProblems       []string
	// envFileOrigin records which file's env_file declaration survived the merge.
	// Unexported like filePath so `config show` — which marshals this struct and
	// reads it back — gains no new key (TASK-245 §5-2).
	envFileOrigin EnvFileOrigin
	// envBridgeOrigin records which file most recently declared env_bridge, for
	// the TASK-281 §3-2 origin check. Unexported for the same reason as
	// envFileOrigin.
	envBridgeOrigin EnvBridgeOrigin
}

// DoctorCheck defines a single environment check for `dva doctor`.
type DoctorCheck struct {
	Name    string `yaml:"name"`     // human-readable check name
	Type    string `yaml:"type"`     // file_exists, command, docker_socket
	Path    string `yaml:"path"`     // for file_exists type
	Command string `yaml:"command"`  // for command type
	FixHint string `yaml:"fix_hint"` // suggestion shown when check fails
	Fix     string `yaml:"fix"`      // shell command to auto-fix (used by dva doctor --fix)
}

// SubprojectConfig defines a sub-project reference.
type SubprojectConfig struct {
	Path        string                  `yaml:"path"`
	ExcludeTags []string                `yaml:"exclude_tags"`
	Import      *SubprojectImportConfig `yaml:"import"`
}

// PlanConfig defines a named executable plan.
type PlanConfig struct {
	Description  string             `yaml:"description"`
	Environment  string             `yaml:"environment"`
	Site         string             `yaml:"site"`
	EndpointTags []string           `yaml:"endpoint_tags"`
	Vars         map[string]string  `yaml:"vars"`
	Entries      []PlanEntry        `yaml:"entries"`
	Composes     []CompositionEntry `yaml:"composes"`

	// Alias is an optional reference to another plan by name. When set, this plan
	// becomes an alias for the target plan. Alias is mutually exclusive with all
	// other plan fields except Description — it has no entries, vars, environment,
	// site, endpoint_tags, or composes of its own.
	Alias string `yaml:"alias"`

	// Extends is an optional reference to a single parent plan by name. When set,
	// this plan inherits from the parent and can override fields. Extends is mutually
	// exclusive with Alias and Composes. The parent must be a concrete plan (not an
	// alias or composition plan). Merge rules: scalar fields (Description, Environment,
	// Site, EndpointTags) are overridden by child; Vars is key-merged; Entries are
	// matched by Name — a child entry with the same Name replaces the parent entry
	// entirely (no services union); new entries are appended.
	Extends string `yaml:"extends"`

	SubprojectPath string `yaml:"-"`

	// SubprojectName is the subprojects: map key this plan was imported from ("" for
	// a locally declared plan). Unlike SubprojectPath, this is a logical name, not a
	// filesystem location, so it is safe to publish on ls/manifest output (TASK-366's
	// owner field).
	SubprojectName string `yaml:"-"`

	// CanonicalAddress is the "<subproject>/<key>" address subproject.go's import loop
	// always assigns first when this plan is imported ("" for a locally declared plan).
	// An import's optional `as:` alias points the SAME *PlanConfig at a second map key,
	// and CanonicalAddress is how a reader tells which key is canonical without comparing
	// pointers itself (TASK-366).
	CanonicalAddress string `yaml:"-"`

	// owner is the fully loaded configuration that declared this plan when it is
	// imported from a subproject. It intentionally has no YAML representation:
	// importing a plan exposes a route in the parent, not the child's complete
	// declaration namespace.
	owner *Config
}

// OwnerConfig returns the configuration whose declarations a plan resolves
// against. Locally declared and manually constructed plans have no recorded
// owner, so fallback preserves their historical behavior.
func (p *PlanConfig) OwnerConfig(fallback *Config) *Config {
	if p != nil && p.owner != nil {
		return p.owner
	}
	return fallback
}

// PlanEntry is a single entry in a plan, referencing a stack declaration.
//
// Profiles and Services are both compose-only selections and they compose in one
// direction: profiles decide which services compose *considers* at all (a service
// behind `profiles:` in the compose file is invisible until one of its profiles is
// active), and Services then narrows that set to the ones this entry starts. Naming
// a gated service in Services also activates it — that is docker's own rule — so the
// two are not alternatives: profiles are how a plan turns on a group without having
// to track its membership, which is what stack.<entry>.runners.compose declares.
type PlanEntry struct {
	Name      string            `yaml:"name"`
	Runner    string            `yaml:"runner"`
	Order     int               `yaml:"order"`
	DependsOn []string          `yaml:"depends_on"`
	Profiles  []string          `yaml:"profiles"`
	Services  []string          `yaml:"services"`
	Vars      map[string]string `yaml:"vars"`
}

// SiteConfig defines host-based execution conditions.
type SiteConfig struct {
	Description    string                        `yaml:"description"`
	Vars           map[string]string             `yaml:"vars"`
	EntryOverrides map[string]*SiteEntryOverride `yaml:"entry_overrides"`
}

// SiteEntryOverride defines site-specific overrides for a stack entry.
type SiteEntryOverride struct {
	Runner string            `yaml:"runner"`
	Vars   map[string]string `yaml:"vars"`
}

// EnvironmentV2 is the simplified environment profile (vars-only).
type EnvironmentV2 struct {
	Description string            `yaml:"description"`
	Vars        map[string]string `yaml:"vars"`
}

// SubprojectImportConfig defines what to import from a subproject.
type SubprojectImportConfig struct {
	Plans        []SubprojectImportEntry `yaml:"plans"`
	Interactions []SubprojectImportEntry `yaml:"interactions"`
	Provision    []SubprojectImportEntry `yaml:"provision"`
}

// SubprojectImportEntry represents a single import item, optionally with alias.
type SubprojectImportEntry struct {
	Name string `yaml:"name"`
	As   string `yaml:"as"`
}

// UnmarshalYAML supports both string shorthand and object forms.
func (e *SubprojectImportEntry) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode {
		e.Name = strings.TrimSpace(node.Value)
		e.As = ""
		if e.Name == "" {
			return fmt.Errorf("subproject import entry cannot be empty")
		}
		return nil
	}

	type plain SubprojectImportEntry
	if err := node.Decode((*plain)(e)); err != nil {
		return err
	}
	if strings.TrimSpace(e.Name) == "" {
		return fmt.Errorf("subproject import entry name is required")
	}
	return nil
}

// ModeConfig defines a named operational mode for dva up (--mode/-M flag).
type ModeConfig struct {
	Description     string            `yaml:"description"`
	ComposeProfiles []string          `yaml:"compose_profiles"`
	ComposeServices *[]string         `yaml:"compose_services"` // nil=all, empty=none, items=only those
	HealthChecks    []string          `yaml:"health_checks"`
	EndpointTags    []string          `yaml:"endpoint_tags"` // filter endpoints by tags (empty=show all)
	Environment     map[string]string `yaml:"environment"`
	Provision       string            `yaml:"provision"` // provision profile to suggest on first run
	Stack           []string          `yaml:"stack"`     // stack entry names to include (empty=all)
	Build           string            `yaml:"build"`     // build strategy: "docker" (compose build), "native" (run command), or custom shell command
	Run             string            `yaml:"run"`       // run strategy: "docker" (compose up), "native" (process via health_checks.start), or custom shell command
}

// StackEntries returns the stack entry names for mode filtering.
func (m *ModeConfig) StackEntries() []string {
	return m.Stack
}

// EnvironmentProfile defines a named environment configuration for --env flag.
type EnvironmentProfile struct {
	Description    string                     `yaml:"description"`
	Environment    map[string]string          `yaml:"environment"`
	Stack          []string                   `yaml:"stack"` // stack entry names to include (empty=all)
	StackOverrides map[string]*LifecycleEntry `yaml:"stack_overrides"`
}

// StackEntries returns the stack entry names for environment filtering.
func (ep *EnvironmentProfile) StackEntries() []string {
	return ep.Stack
}

// SshConfig holds SSH agent configuration.
type SshConfig struct {
	AgentImage string `yaml:"agent_image"`
}

// EndpointConfig defines a user-facing endpoint URL for the project.
type EndpointConfig struct {
	URL    string            `yaml:"url"`
	Label  string            `yaml:"label"`
	Tags   []string          `yaml:"tags"`
	Paths  map[string]string `yaml:"paths"`  // sub-path -> description
	Source string            `yaml:"source"` // compose "service:host_port" reference (URL auto-resolved)
}

// HealthCheckConfig defines a health check for a non-compose service.
type HealthCheckConfig struct {
	Type         string `yaml:"type"`          // http, tcp, command
	URL          string `yaml:"url"`           // for http type
	Address      string `yaml:"address"`       // for tcp type
	Command      string `yaml:"command"`       // for command type
	Start        string `yaml:"start"`         // command to auto-start (background)
	StartHint    string `yaml:"start_hint"`    // human-readable start instructions
	Timeout      int    `yaml:"timeout"`       // health check timeout in seconds (default: 2)
	ReadyTimeout int    `yaml:"ready_timeout"` // max wait after start in seconds (default: 30)
	// `required` (TASK-118 opt-in strict readiness) is gone with the field. Its only
	// schema home was applications.<app>.health, and its only reader was
	// AppManager.startApp; docs/43 removed both. Top-level health_checks has always
	// rejected the key (schema.json health_checks, additionalProperties:false), so
	// keeping the struct field would leave a strictness knob nothing can turn.
	// The capability itself did not move to the plan path — see docs/43.
}

// ServiceTagConfig defines per-service tag configuration.
type ServiceTagConfig struct {
	Tags []string `yaml:"tags"`
}

// SuggestionCategories turns a whole suggestion source off (docs/56 §6-2). The keys are
// source kinds — where the candidate name was read from — and never target groups like
// `docker-*`: those are the suggestion rules' own business, and promoting them to a
// user-facing category would make the author re-declare a judgement the rules already
// make, with the two drifting apart the moment the rules change.
//
// Every field is a pointer because omission has to mean "on". A plain bool would make an
// undeclared `suggestions:` block read as false and silence every suggestion in every
// pre-TASK-309 config.
type SuggestionCategories struct {
	Makefile    *bool `yaml:"makefile"`
	PackageJSON *bool `yaml:"package_json"`
}

// Suggestion source kinds. These are the `suggestions:` keys and the only values
// SuggestionCategories.Enabled answers about.
const (
	SuggestionSourceMakefile    = "makefile"
	SuggestionSourcePackageJSON = "package_json"
)

// Enabled reports whether suggestions from the named source kind should be produced. A
// nil receiver, an undeclared field, and an unknown key all answer true: this gate can
// only ever suppress on an explicit `false`.
func (s *SuggestionCategories) Enabled(source string) bool {
	if s == nil {
		return true
	}
	switch source {
	case SuggestionSourceMakefile:
		return s.Makefile == nil || *s.Makefile
	case SuggestionSourcePackageJSON:
		return s.PackageJSON == nil || *s.PackageJSON
	}
	return true
}

// InfraConfig holds infrastructure service configuration.
type InfraConfig struct {
	Git  string `yaml:"git"`
	Ref  string `yaml:"ref"`
	Path string `yaml:"path"`
}

// FilePath returns the path to the loaded config file.
func (c *Config) FilePath() string {
	return c.filePath
}

// LoadProblems returns the stack-entry shape errors a load made with
// CollectEntryProblems recorded instead of failing. Empty for a strict load.
func (c *Config) LoadProblems() []string { return c.loadProblems }

// FileDir returns the directory containing the config file.
func (c *Config) FileDir() string {
	return filepath.Dir(c.filePath)
}

// HasPlans reports whether any plans are configured.
func (c *Config) HasPlans() bool {
	return len(c.Plans) > 0
}

// DefaultPlan returns the only plan name when exactly one plan exists.
func (c *Config) DefaultPlan() string {
	// Explicit default_plan wins when it references a defined plan. A missing
	// reference falls through to "" (Validate reports it as a hard error).
	if c.DefaultPlanName != "" {
		if _, ok := c.Plans[c.DefaultPlanName]; ok {
			return c.DefaultPlanName
		}
		return ""
	}
	// Otherwise a lone plan is the implicit default.
	if len(c.Plans) != 1 {
		return ""
	}
	for name := range c.Plans {
		return name
	}
	return ""
}

// DefaultPlanSource reports why DefaultPlan selected its effective value.
//
// The value is intentionally about the resolved lifecycle behavior, not merely
// whether default_plan was declared. An invalid explicit name therefore reports
// "none": validation rejects the declaration and bare lifecycle commands have
// no effective default to select.
func (c *Config) DefaultPlanSource() string {
	if c.DefaultPlan() == "" {
		return "none"
	}
	if c.DefaultPlanName != "" {
		return "explicit"
	}
	return "implicit-single"
}

func copyStringMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	result := make(map[string]string, len(m))
	maps.Copy(result, m)
	return result
}

// nonHTTPServices are compose service name prefixes that resolve to plain host:port
// instead of http://localhost:port. Users needing other protocols should use url: directly.
var nonHTTPServices = map[string]bool{
	// Databases
	"postgres": true, "postgresql": true, "pg": true,
	"mysql": true, "mariadb": true,
	"mssql": true, "sqlserver": true,
	"mongo": true, "mongodb": true,
	"cassandra": true, "scylla": true,
	"db": true, "database": true,
	// Caches
	"redis": true, "valkey": true,
	"memcached": true,
	"cache":     true,
	// Messaging
	"kafka": true, "zookeeper": true,
	"rabbitmq": true, "nats": true,
	"mq": true, "queue": true, "broker": true,
	// Other
	"ssh": true,
}

// ResolveEndpoints auto-fills URL for endpoints that have source but no url.
// Source format: "service:host_port" → resolves to http://localhost:{port}
// or plain localhost:{port} for known non-HTTP infrastructure services.
func (c *Config) ResolveEndpoints() {
	if c.Endpoints == nil {
		return
	}

	for name, ep := range c.Endpoints {
		if ep.Source == "" || ep.URL != "" {
			continue
		}

		parts := strings.SplitN(ep.Source, ":", 2)
		if len(parts) != 2 || parts[1] == "" {
			continue
		}

		svc := parts[0]
		port := parts[1]

		if nonHTTPServices[strings.ToLower(svc)] {
			ep.URL = "localhost:" + port
		} else {
			ep.URL = "http://localhost:" + port
		}
		c.Endpoints[name] = ep
	}
}

// ConfigFileInDir returns the config file in dir, preferring the canonical dva.yml and
// accepting dva.yaml as the legacy alternative. Every command that looks for a project's
// config in one directory — the loader, `config migrate`, `config docs` — must share this
// rule; `config migrate` once checked only dva.yml and so refused exactly the legacy
// projects that needed it most (TASK-304).
func ConfigFileInDir(dir string) (string, bool) {
	for _, name := range []string{FileName, FileNameAlt} {
		candidate := filepath.Join(dir, name)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, true
		}
	}
	return "", false
}
