package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadOption configures optional behavior for Load.
type LoadOption func(*loadOptions)

type loadOptions struct {
	skipVersionCheck     bool
	ciProfileOnly        bool
	envInspectionOnly    bool
	collectEntryProblems bool
	explicitFile         string
	hasExplicitFile      bool
}

// CollectEntryProblems returns a LoadOption that records stack-entry shape errors
// (legacy compose declarations, unresolvable plugins, missing sources) on the loaded
// Config instead of failing the load. Only `dva validate` uses it (TASK-305): the
// diagnostics that follow the load are worth running even when one entry is malformed,
// and the recorded problems are reported as hard errors at the end. Lifecycle commands
// keep the strict load, so a malformed entry never reaches a runner. The option applies
// to the root file only; imported subprojects are still finalized strictly.
func CollectEntryProblems() LoadOption {
	return func(o *loadOptions) {
		o.collectEntryProblems = true
	}
}

// SkipVersionCheck returns a LoadOption that disables version compatibility checking.
// Use this for commands like "config improve" that need to load outdated configs to fix them.
func SkipVersionCheck() LoadOption {
	return func(o *loadOptions) {
		o.skipVersionCheck = true
	}
}

// ForCI loads the root config fields used by a CI profile: root variables,
// environment inputs, and CI declarations. It still parses and merges the root,
// modules, and override and validates environment inputs and CI profiles, while
// omitting imported children and unrelated runtime declarations. Ordinary Load
// remains fully strict.
func ForCI() LoadOption {
	return func(o *loadOptions) {
		o.ciProfileOnly = true
	}
}

// ForEnvInspection loads effective environment declarations without importing
// nested projects or validating unrelated execution/CI declarations. Modules,
// overrides, version checks and env-source validation retain their normal rules.
func ForEnvInspection() LoadOption {
	return func(o *loadOptions) { o.envInspectionOnly = true }
}

// FromFile makes Load use path as its root config instead of discovery or
// DVA_FILE. A relative path is resolved from workDir. The selected file follows
// the ordinary Load path, including modules, overrides, finalization, and version
// compatibility checks; a missing explicit file is an error and never falls back.
func FromFile(path string) LoadOption {
	return func(o *loadOptions) {
		o.explicitFile = path
		o.hasExplicitFile = true
	}
}

// Load discovers and loads the dva.yml configuration.
func Load(workDir string, opts ...LoadOption) (*Config, error) {
	var o loadOptions
	for _, opt := range opts {
		opt(&o)
	}
	if o.ciProfileOnly && o.envInspectionOnly {
		return nil, fmt.Errorf("CI loading and environment inspection are mutually exclusive")
	}

	filePath, err := loadFilePath(workDir, o)
	if err != nil {
		return nil, err
	}

	cfg, err := loadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("loading %s: %w", filePath, err)
	}
	cfg.filePath = filePath
	cfg.deferEntryProblems = o.collectEntryProblems
	cfg.setEnvFileOrigin(cfg.EnvFile, EnvOriginRoot, filePath)
	cfg.setEnvBridgeOrigin(cfg.EnvBridge, EnvBridgeOriginRoot, filePath)

	if !o.skipVersionCheck {
		if err := checkConfigVersion(cfg); err != nil {
			return nil, err
		}
	}

	// Load modules
	if len(cfg.Modules) > 0 {
		modulesDir := filepath.Join(filepath.Dir(filePath), DotDirName)
		for _, mod := range cfg.Modules {
			modFile := filepath.Join(modulesDir, mod+".yml")
			modCfg, err := loadFile(modFile)
			if err != nil {
				return nil, fmt.Errorf("loading module `%s`: %w", mod, err)
			}
			if !o.skipVersionCheck {
				if err := checkConfigVersion(modCfg); err != nil {
					return nil, fmt.Errorf("loading module `%s`: %w", mod, err)
				}
			}
			if len(modCfg.Modules) > 0 {
				return nil, fmt.Errorf("nested modules are not supported")
			}
			if err := cfg.mergeFrom(modCfg); err != nil {
				return nil, fmt.Errorf("merging module %q: %w", mod, err)
			}
			cfg.setEnvFileOrigin(modCfg.EnvFile, EnvOriginModule, modFile)
			cfg.setEnvBridgeOrigin(modCfg.EnvBridge, EnvBridgeOriginModule, modFile)
		}
	}

	// Load override (if exists)
	overrideFile := strings.TrimSuffix(filePath, ".yml") + OverrideExt
	if overCfg, err := loadFile(overrideFile); err == nil {
		if !o.skipVersionCheck {
			if err := checkConfigVersion(overCfg); err != nil {
				return nil, fmt.Errorf("loading override: %w", err)
			}
		}
		if err := cfg.mergeFrom(overCfg); err != nil {
			return nil, fmt.Errorf("merging override: %w", err)
		}
		cfg.setEnvFileOrigin(overCfg.EnvFile, EnvOriginOverride, overrideFile)
		cfg.setEnvBridgeOrigin(overCfg.EnvBridge, EnvBridgeOriginOverride, overrideFile)
	} else if !isAbsentOverrideFile(overrideFile, err) {
		return nil, fmt.Errorf("loading override: %w", err)
	}

	applyConfigDefaults(cfg)

	if len(cfg.Subprojects) > 0 && !o.ciProfileOnly && !o.envInspectionOnly {
		if err := resolveSubprojectImports(cfg, opts...); err != nil {
			return nil, fmt.Errorf("resolving subprojects: %w", err)
		}
	}

	var migrated []string
	if o.envInspectionOnly {
		applyConfigDefaults(cfg)
		err = validateEnvSourceDeclarations(cfg)
	} else if o.ciProfileOnly {
		err = finalizeCIConfig(cfg)
	} else {
		migrated, err = finalizeLoadedConfig(cfg)
	}
	if err != nil {
		return nil, err
	}
	if len(migrated) > 0 {
		fmt.Fprintf(os.Stderr, "⚠  'infra:' is deprecated (TASK-051): migrated %s into stack: as source-backed compose entries (tag: infra).\n   Declare them under stack.<name>.source instead; 'infra:' will be removed in a future release.\n", strings.Join(migrated, ", "))
	}

	// Reserved-command warnings belong to the root command surface. Imported
	// children are finalized for execution but do not independently expose their
	// whole interaction tree, so warning here avoids duplicate child warnings.
	if !o.ciProfileOnly && !o.envInspectionOnly {
		WarnReservedCommandConflicts(cfg.Interaction)
	}

	return cfg, nil
}

// isAbsentOverrideFile distinguishes a missing optional override from a
// directory entry that cannot be read. In particular, os.ReadFile reports a
// dangling symlink as fs.ErrNotExist even though silently skipping it would
// hide a configured override from the user.
func isAbsentOverrideFile(path string, loadErr error) bool {
	if !errors.Is(loadErr, fs.ErrNotExist) {
		return false
	}
	_, err := os.Lstat(path)
	return errors.Is(err, fs.ErrNotExist)
}

// loadFilePath resolves the root-file choice before Load begins its common
// processing path. Keeping explicit selection here prevents callers from
// temporarily mutating DVA_FILE, which is process-global and unsafe for parallel
// inspections.
func loadFilePath(workDir string, o loadOptions) (string, error) {
	if !o.hasExplicitFile {
		return findConfig(workDir)
	}
	path := o.explicitFile
	if !filepath.IsAbs(path) {
		path = filepath.Join(workDir, path)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolving explicit config file %q: %w", o.explicitFile, err)
	}
	return absPath, nil
}

// finalizeCIConfig validates only the configuration consumed by a root CI
// profile. Environment-source declarations must remain valid because CI loads
// them into its step environment; all CI profiles are checked before selection.
// Plans, stack entries, remote jobs/secrets, and endpoints belong to other
// execution surfaces and do not participate in CI profile loading.
func finalizeCIConfig(cfg *Config) error {
	applyConfigDefaults(cfg)
	if err := validateEnvSourceDeclarations(cfg); err != nil {
		return err
	}
	return cfg.validateCIProfiles()
}

// applyConfigDefaults initializes the maps that downstream config consumers may
// write or index. It is shared by root and imported-child loading so an imported
// plan has the same effective declaration shape as a directly loaded plan.
func applyConfigDefaults(cfg *Config) {
	if cfg.Environment == nil {
		cfg.Environment = make(map[string]string)
	}
	if cfg.Vars == nil {
		cfg.Vars = make(map[string]string)
	}
	if cfg.Interaction == nil {
		cfg.Interaction = make(map[string]*InteractionCommand)
	}
	if cfg.CI != nil && cfg.CI.Profiles == nil {
		cfg.CI.Profiles = make(map[string]CIProfile)
	}
	if cfg.Provision.Profiles == nil {
		cfg.Provision.Profiles = make(map[string][]ProvisionItem)
	}
	if cfg.Stack == nil {
		cfg.Stack = make(map[string]*LifecycleEntry)
	}
	if cfg.Plans == nil {
		cfg.Plans = make(map[string]*PlanConfig)
	}
	if cfg.Sites == nil {
		cfg.Sites = make(map[string]*SiteConfig)
	}
}

// finalizeLoadedConfig applies the post-merge work required before a config can
// supply executable declarations. It deliberately does not resolve subprojects:
// imported children are finalized as independent roots, never recursively
// imported into their parent.
func finalizeLoadedConfig(cfg *Config) ([]string, error) {
	applyConfigDefaults(cfg)

	migrated, err := cfg.migrateInfraToStack()
	if err != nil {
		return nil, err
	}

	// Sorted so deferred problems (and the first-error path) name entries in a stable
	// order across runs; cfg.Stack is a map.
	entryNames := make([]string, 0, len(cfg.Stack))
	for name := range cfg.Stack {
		entryNames = append(entryNames, name)
	}
	sort.Strings(entryNames)
	for _, name := range entryNames {
		entry := cfg.Stack[name]
		entry.Name = name
		if err := entry.ResolvePluginFromName(); err != nil {
			if !cfg.deferEntryProblems {
				return nil, err
			}
			cfg.loadProblems = append(cfg.loadProblems, err.Error())
			continue
		}
		if err := validateEntrySource(name, entry, cfg.FileDir()); err != nil {
			if !cfg.deferEntryProblems {
				return nil, err
			}
			cfg.loadProblems = append(cfg.loadProblems, err.Error())
		}
		if err := validateEntryTunnel(name, entry); err != nil {
			if !cfg.deferEntryProblems {
				return nil, err
			}
			cfg.loadProblems = append(cfg.loadProblems, err.Error())
		}
	}

	if err := validateEnvSourceDeclarations(cfg); err != nil {
		return nil, err
	}

	if err := cfg.validateRemoteDeclarations(); err != nil {
		return nil, err
	}

	if err := cfg.validateCIProfiles(); err != nil {
		return nil, err
	}

	// Runs after any subproject imports already resolved into cfg.Plans (root
	// path) or against cfg's own local plans only (subproject path — see
	// resolveSubprojectImports, which finalizes each subCfg before importing
	// from it and never recursively resolves a subCfg's own subprojects).
	if err := validateCompositionPlans(cfg); err != nil {
		return nil, err
	}
	if err := cfg.validateSuggestionAliases(); err != nil {
		return nil, err
	}

	cfg.ResolveEndpoints()
	return migrated, nil
}

// ErrConfigNotFound means automatic discovery found no project configuration.
// An explicitly selected DVA_FILE that cannot be loaded is a different error.
var ErrConfigNotFound = errors.New("could not find dva.yml")

var yamlDeprecationWarned bool

// findConfig walks up from workDir to find dva.yml.
func findConfig(workDir string) (string, error) {
	// Check DVA_FILE env var first
	if env := os.Getenv(EnvFileKey); env != "" {
		if _, err := os.Stat(env); err != nil {
			return "", fmt.Errorf("DVA_FILE=%s: %w", env, err)
		}
		return env, nil
	}

	dir, err := filepath.Abs(workDir)
	if err != nil {
		return "", err
	}

	for {
		if candidate, ok := ConfigFileInDir(dir); ok {
			if filepath.Base(candidate) == FileNameAlt && !yamlDeprecationWarned {
				// Fallback: accept dva.yaml with deprecation warning (once per process)
				yamlDeprecationWarned = true
				fmt.Fprintf(os.Stderr, "⚠  Found %s — consider renaming to dva.yml (canonical name)\n", candidate)
			}
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			absWork, _ := filepath.Abs(workDir)
			return "", fmt.Errorf("%w (searched from %s to /).\n  Hint: run 'dva config init' or set DVA_FILE=/path/to/dva.yml", ErrConfigNotFound, absWork)
		}
		dir = parent
	}
}

// decodeConfig turns raw dva.yml bytes into a Config.
//
// Every path that decodes user-supplied bytes into config types goes through here,
// because the anchor cycle scan it performs is not optional: the alternative to
// rejecting a cyclic document is a runtime stack overflow that ends the process
// (see checkAnchorCycles).
func decodeConfig(data []byte) (*Config, error) {
	// Parse to a node first: the scan must see the document before any config type
	// does, and decoding the node tree costs no second parse of the text.
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}
	if err := checkAnchorCycles(&doc); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}
	if err := rejectRemovedTopLevelKeys(&doc); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	cfg := &Config{}
	if doc.IsZero() {
		// Empty or comment-only input: nothing was parsed, so there is nothing to decode.
		return cfg, nil
	}
	if err := doc.Decode(cfg); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	return cfg, nil
}

// rejectRemovedTopLevelKeys rejects legacy root keys before yaml.Node.Decode can
// silently discard them after their Config fields are removed. YAML merge keys are
// part of a mapping's effective key set, so aliases that inject either key are
// rejected too; nested keys with the same spelling remain outside this root-only rule.
func rejectRemovedTopLevelKeys(doc *yaml.Node) error {
	if doc == nil || doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil
	}
	keys := make(map[string]bool)
	collectEffectiveMappingKeys(doc.Content[0], keys, make(map[*yaml.Node]bool))
	for _, key := range []string{"modes", "default_mode"} {
		if keys[key] {
			return fmt.Errorf("top-level %q was removed in v0.5.0; run 'dva config migrate'", key)
		}
	}
	return nil
}

func collectEffectiveMappingKeys(node *yaml.Node, keys map[string]bool, visited map[*yaml.Node]bool) {
	if node == nil || visited[node] {
		return
	}
	visited[node] = true
	if node.Kind == yaml.AliasNode {
		collectEffectiveMappingKeys(node.Alias, keys, visited)
		return
	}
	if node.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		key, value := node.Content[i], node.Content[i+1]
		if key.ShortTag() == "!!merge" {
			collectMergedMappingKeys(value, keys, visited)
			continue
		}
		var name string
		if err := key.Decode(&name); err != nil {
			// The normal YAML decoder reports malformed keys with the surrounding
			// document; do not replace that error with this presence-only guard.
			continue
		}
		keys[name] = true
	}
}

func collectMergedMappingKeys(node *yaml.Node, keys map[string]bool, visited map[*yaml.Node]bool) {
	if node == nil {
		return
	}
	if node.Kind == yaml.SequenceNode {
		for _, item := range node.Content {
			collectMergedMappingKeys(item, keys, visited)
		}
		return
	}
	collectEffectiveMappingKeys(node, keys, visited)
}

func loadFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return decodeConfig(data)
}
