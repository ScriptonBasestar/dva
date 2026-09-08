package config

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
)

// SecretsConfig contains references and destinations, never plaintext values.
type SecretsConfig struct {
	Sources map[string]SecretSource `yaml:"sources" json:"sources"`
	Targets map[string]SecretTarget `yaml:"targets" json:"targets"`
}

type SecretSource struct {
	Sops string `yaml:"sops" json:"sops"`
}

type SecretTarget struct {
	Provider   string            `yaml:"provider" json:"provider"`
	Repository string            `yaml:"repository" json:"repository"`
	Source     string            `yaml:"source" json:"source"`
	Keys       map[string]string `yaml:"keys" json:"keys"`
}

// JobConfig describes a finite batch of repository-owned artifact jobs. It is
// independent of service plans and local CI profiles. Inputs are public data.
type JobConfig struct {
	Provider      string              `yaml:"provider" json:"provider"`
	Repository    string              `yaml:"repository" json:"repository"`
	Ref           string              `yaml:"ref" json:"ref"`
	Timeout       string              `yaml:"timeout" json:"timeout"`
	Inputs        map[string]JobInput `yaml:"inputs,omitempty" json:"inputs,omitempty"`
	SecretTargets []string            `yaml:"secret_targets,omitempty" json:"secret_targets,omitempty"`
	Runs          []JobRun            `yaml:"runs" json:"runs"`
}

type JobInput struct {
	Default  string   `yaml:"default,omitempty" json:"default,omitempty"`
	Required bool     `yaml:"required,omitempty" json:"required,omitempty"`
	Values   []string `yaml:"values,omitempty" json:"values,omitempty"`
}

type JobRun struct {
	Name           string            `yaml:"name" json:"name"`
	Workflow       string            `yaml:"workflow" json:"workflow"`
	Inputs         map[string]string `yaml:"inputs,omitempty" json:"inputs,omitempty"`
	Images         []JobImage        `yaml:"images,omitempty" json:"images,omitempty"`
	ResultArtifact string            `yaml:"result_artifact,omitempty" json:"result_artifact,omitempty"`
}

type JobImage struct {
	Reference string   `yaml:"reference" json:"reference"`
	Platforms []string `yaml:"platforms,omitempty" json:"platforms,omitempty"`
}

var remoteRepositoryPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*/[A-Za-z0-9][A-Za-z0-9_.-]*$`)
var secretKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
var jobWorkflowPattern = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]*\.ya?ml$`)
var jobTemplatePattern = regexp.MustCompile(`\{\{input\.([A-Za-z_][A-Za-z0-9_]*)\}\}`)

// ValidateRemoteDeclarations validates an effective config selected directly as
// a child as well as the ordinary finalized root load. It does not read secrets
// or contact providers.
func (c *Config) ValidateRemoteDeclarations() error {
	return c.validateRemoteDeclarations()
}

func (c *Config) validateRemoteDeclarations() error {
	if c.Secrets != nil {
		if err := validateSecrets(c.Secrets); err != nil {
			return err
		}
	}
	for _, name := range sortedKeysOf(c.Jobs) {
		job := c.Jobs[name]
		if !validCIName(name) {
			return fmt.Errorf("jobs: invalid name %q", name)
		}
		if err := validateJob(job); err != nil {
			return fmt.Errorf("job %q: %w", name, err)
		}
		seen := map[string]bool{}
		for _, target := range job.SecretTargets {
			if seen[target] {
				return fmt.Errorf("job %q: duplicate secret target %q", name, target)
			}
			seen[target] = true
			if c.Secrets == nil {
				return fmt.Errorf("job %q: secret target %q is undefined", name, target)
			}
			binding, ok := c.Secrets.Targets[target]
			if !ok || !strings.EqualFold(binding.Repository, job.Repository) {
				return fmt.Errorf("job %q: secret target %q must exist in the same repository", name, target)
			}
		}
	}
	return nil
}

func validateSecrets(c *SecretsConfig) error {
	for _, name := range sortedKeysOf(c.Sources) {
		source := c.Sources[name]
		clean := filepath.Clean(source.Sops)
		if !validCIName(name) || source.Sops == "" || filepath.IsAbs(source.Sops) || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || strings.ContainsAny(source.Sops, "\\\x00\r\n") {
			return fmt.Errorf("secrets source %q: sops must be a contained relative file", name)
		}
	}
	for _, name := range sortedKeysOf(c.Targets) {
		target := c.Targets[name]
		if !validCIName(name) || target.Provider != "github-actions" || !remoteRepositoryPattern.MatchString(target.Repository) {
			return fmt.Errorf("secrets target %q: expected github-actions and owner/repository", name)
		}
		if _, ok := c.Sources[target.Source]; !ok {
			return fmt.Errorf("secrets target %q: source %q is undefined", name, target.Source)
		}
		if len(target.Keys) == 0 || len(target.Keys) > 64 {
			return fmt.Errorf("secrets target %q: select between 1 and 64 keys", name)
		}
		seen := map[string]bool{}
		for _, key := range sortedKeysOf(target.Keys) {
			dest := target.Keys[key]
			if !secretKeyPattern.MatchString(key) || !secretKeyPattern.MatchString(dest) || len(key) > 256 || len(dest) > 256 || strings.HasPrefix(strings.ToUpper(dest), "GITHUB_") {
				return fmt.Errorf("secrets target %q: invalid key mapping", name)
			}
			if seen[strings.ToUpper(dest)] {
				return fmt.Errorf("secrets target %q: duplicate destination key", name)
			}
			seen[strings.ToUpper(dest)] = true
		}
	}
	return nil
}

func validateJob(job JobConfig) error {
	if job.Provider != "github-actions" || !remoteRepositoryPattern.MatchString(job.Repository) {
		return fmt.Errorf("expected github-actions and owner/repository")
	}
	if job.Ref == "" || len(job.Ref) > 255 || strings.HasPrefix(job.Ref, "-") || strings.ContainsAny(job.Ref, " \t\r\n\x00") {
		return fmt.Errorf("ref must be an explicit branch or tag")
	}
	timeout, err := time.ParseDuration(job.Timeout)
	if err != nil || timeout <= 0 || timeout > 24*time.Hour {
		return fmt.Errorf("timeout must be positive and at most 24h")
	}
	if len(job.Inputs) > 25 {
		return fmt.Errorf("at most 25 public inputs are supported")
	}
	for _, name := range sortedKeysOf(job.Inputs) {
		input := job.Inputs[name]
		if !secretKeyPattern.MatchString(name) || len(input.Default) > 4096 {
			return fmt.Errorf("invalid public input %q", name)
		}
		if input.Default != "" && len(input.Values) > 0 && !slices.Contains(input.Values, input.Default) {
			return fmt.Errorf("input %q default is outside values", name)
		}
	}
	if len(job.Runs) < 1 || len(job.Runs) > 16 {
		return fmt.Errorf("runs must contain between 1 and 16 workflows")
	}
	seen := map[string]bool{}
	for _, run := range job.Runs {
		if !validCIName(run.Name) || seen[run.Name] || !jobWorkflowPattern.MatchString(run.Workflow) {
			return fmt.Errorf("invalid or duplicate run %q, or invalid workflow filename", run.Name)
		}
		seen[run.Name] = true
		if len(run.Inputs) > 25 || len(run.Images) > 16 {
			return fmt.Errorf("run %q exceeds input or image limit", run.Name)
		}
		for _, key := range sortedKeysOf(run.Inputs) {
			if !secretKeyPattern.MatchString(key) {
				return fmt.Errorf("run %q: invalid input name", run.Name)
			}
			if err := validateJobTemplate(run.Inputs[key], job.Inputs); err != nil {
				return err
			}
		}
		if len(run.Images) > 0 && !validCIName(run.ResultArtifact) {
			return fmt.Errorf("run %q: images require a named result_artifact", run.Name)
		}
		refs := map[string]bool{}
		for _, image := range run.Images {
			if image.Reference == "" || refs[image.Reference] {
				return fmt.Errorf("run %q: empty or duplicate image reference", run.Name)
			}
			refs[image.Reference] = true
			if err := validateJobTemplate(image.Reference, job.Inputs); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateJobTemplate(value string, inputs map[string]JobInput) error {
	if len(value) > 4096 || strings.ContainsRune(value, 0) {
		return fmt.Errorf("job value exceeds limit or contains NUL")
	}
	for _, match := range jobTemplatePattern.FindAllStringSubmatch(value, -1) {
		if _, ok := inputs[match[1]]; !ok {
			return fmt.Errorf("job references undefined input %q", match[1])
		}
	}
	if strings.ContainsAny(jobTemplatePattern.ReplaceAllString(value, ""), "{}") {
		return fmt.Errorf("job uses unsupported interpolation; use {{input.NAME}}")
	}
	return nil
}
