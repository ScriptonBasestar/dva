// Package jobrun runs declared GitHub Actions jobs and keeps resumable receipts.
package jobrun

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/ScriptonBasestar/dva/internal/ociverify"
	"github.com/ScriptonBasestar/dva/internal/remotetarget"
)

const ProviderGitHub = "github-actions"

type Definition struct {
	Provider      string           `json:"provider" yaml:"provider"`
	Repository    string           `json:"repository" yaml:"repository"`
	Ref           string           `json:"ref" yaml:"ref"`
	Timeout       string           `json:"timeout" yaml:"timeout"`
	Inputs        map[string]Input `json:"inputs" yaml:"inputs"`
	SecretTargets []string         `json:"secret_targets" yaml:"secret_targets"`
	Runs          []RunDefinition  `json:"runs" yaml:"runs"`
}

type Input struct {
	Default  string   `json:"default" yaml:"default"`
	Required bool     `json:"required" yaml:"required"`
	Values   []string `json:"values" yaml:"values"`
}

type RunDefinition struct {
	Name           string            `json:"name" yaml:"name"`
	Workflow       string            `json:"workflow" yaml:"workflow"`
	Inputs         map[string]string `json:"inputs" yaml:"inputs"`
	Images         []Image           `json:"images" yaml:"images"`
	ResultArtifact string            `json:"result_artifact" yaml:"result_artifact"`
}

type Image struct {
	Reference string   `json:"reference" yaml:"reference"`
	Platforms []string `json:"platforms" yaml:"platforms"`
}

type Options struct {
	Root           string
	Name           string
	StateDir       string
	Definition     Definition
	Inputs         map[string]string
	Wait           bool
	Verify         bool
	DryRun         bool
	BeforeDispatch func(context.Context) error
	Provider       Provider
}

type Report struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Repository string      `json:"repository"`
	Ref        string      `json:"ref"`
	HeadSHA    string      `json:"head_sha"`
	Runs       []RunReport `json:"runs"`
}

type RunReport struct {
	Name          string             `json:"name"`
	Workflow      string             `json:"workflow"`
	RunID         int64              `json:"run_id"`
	URL           string             `json:"url"`
	Status        string             `json:"status"`
	Conclusion    string             `json:"conclusion"`
	JobSucceeded  bool               `json:"job_succeeded"`
	Verified      bool               `json:"verified"`
	DispatchState string             `json:"dispatch_state"`
	Images        []ociverify.Result `json:"images,omitempty"`
}

type Provider interface {
	HeadSHA(context.Context, string, string) (string, error)
	Dispatch(context.Context, DispatchRequest) (DispatchResponse, error)
	ResolveWorkflow(context.Context, string, string) (string, error)
	GetRun(context.Context, string, int64) (RemoteRun, error)
	ListArtifacts(context.Context, string, int64) ([]Artifact, error)
	DownloadArtifact(context.Context, string, int64) ([]byte, error)
}

type DispatchRequest struct {
	Repository string            `json:"repository"`
	Workflow   string            `json:"workflow"`
	Ref        string            `json:"ref"`
	Inputs     map[string]string `json:"inputs"`
}
type DispatchResponse struct {
	RunID   int64  `json:"workflow_run_id"`
	RunURL  string `json:"run_url"`
	HTMLURL string `json:"html_url"`
}
type RemoteRun struct {
	ID         int64
	Repository string
	Workflow   string
	Event      string
	Ref        string
	HeadSHA    string
	Status     string
	Conclusion string
	HTMLURL    string
	Attempt    int
}
type Artifact struct {
	ID      int64
	Name    string
	Expired bool
	Size    int64
}

var inputRef = regexp.MustCompile(`\{\{input\.([A-Za-z_][A-Za-z0-9_]*)\}\}`)

// Resolve validates public inputs and returns a deep copy with only run inputs and
// image references interpolated. No other field is a template language.
func Resolve(def Definition, supplied map[string]string) (Definition, error) {
	if def.Provider != ProviderGitHub {
		return Definition{}, fmt.Errorf("jobs provider must be %q", ProviderGitHub)
	}
	if !validRepository(def.Repository) {
		return Definition{}, fmt.Errorf("invalid GitHub repository %q", def.Repository)
	}
	if strings.TrimSpace(def.Ref) == "" {
		return Definition{}, fmt.Errorf("jobs ref is required")
	}
	if len(def.Ref) > 255 || strings.HasPrefix(def.Ref, "-") || strings.ContainsAny(def.Ref, " \t\r\n\x00") {
		return Definition{}, fmt.Errorf("jobs ref must be an explicit branch or tag")
	}
	timeout, err := time.ParseDuration(def.Timeout)
	if err != nil || timeout <= 0 || timeout > 24*time.Hour {
		return Definition{}, fmt.Errorf("jobs timeout must be positive and at most 24h")
	}
	if len(def.Inputs) > 25 {
		return Definition{}, fmt.Errorf("at most 25 public inputs are supported")
	}
	if len(def.Runs) == 0 || len(def.Runs) > 16 {
		return Definition{}, fmt.Errorf("runs must contain between 1 and 16 workflows")
	}
	values := make(map[string]string, len(def.Inputs))
	for name, spec := range def.Inputs {
		if !validInputName(name) {
			return Definition{}, fmt.Errorf("invalid input name %q", name)
		}
		if len(spec.Default) > 4096 {
			return Definition{}, fmt.Errorf("input %q default exceeds limit", name)
		}
		if spec.Default != "" && len(spec.Values) > 0 && !contains(spec.Values, spec.Default) {
			return Definition{}, fmt.Errorf("input %q default is outside values", name)
		}
		v, present := supplied[name]
		if !present {
			v = spec.Default
		}
		if len(v) > 4096 || strings.ContainsRune(v, 0) {
			return Definition{}, fmt.Errorf("input %q exceeds limit or contains NUL", name)
		}
		if spec.Required && v == "" {
			return Definition{}, fmt.Errorf("required input %q is missing", name)
		}
		if len(spec.Values) > 0 && !contains(spec.Values, v) {
			return Definition{}, fmt.Errorf("input %q has invalid value", name)
		}
		values[name] = v
	}
	for name := range supplied {
		if _, ok := def.Inputs[name]; !ok {
			return Definition{}, fmt.Errorf("unknown input %q", name)
		}
	}
	out := def
	out.Inputs = cloneInputs(def.Inputs)
	out.SecretTargets = append([]string(nil), def.SecretTargets...)
	out.Runs = make([]RunDefinition, len(def.Runs))
	names := map[string]bool{}
	for i, run := range def.Runs {
		if run.Name == "" || names[run.Name] {
			return Definition{}, fmt.Errorf("job run name must be unique and non-empty")
		}
		names[run.Name] = true
		if run.Workflow == "" || !regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]*\.ya?ml$`).MatchString(run.Workflow) {
			return Definition{}, fmt.Errorf("job run %q requires workflow", run.Name)
		}
		if len(run.Inputs) > 25 || len(run.Images) > 16 {
			return Definition{}, fmt.Errorf("job run %q exceeds input or image limit", run.Name)
		}
		if len(run.Images) > 0 && run.ResultArtifact == "" {
			return Definition{}, fmt.Errorf("job run %q with images requires result_artifact", run.Name)
		}
		out.Runs[i] = run
		out.Runs[i].Inputs = map[string]string{}
		for k, v := range run.Inputs {
			if !validInputName(k) || len(v) > 4096 || strings.ContainsRune(v, 0) || strings.ContainsAny(inputRef.ReplaceAllString(v, ""), "{}") {
				return Definition{}, fmt.Errorf("run %q has invalid input %q", run.Name, k)
			}
			resolved, err := expand(v, values)
			if err != nil {
				return Definition{}, fmt.Errorf("run %q input %q: %w", run.Name, k, err)
			}
			if len(resolved) > 4096 || strings.ContainsRune(resolved, 0) {
				return Definition{}, fmt.Errorf("run %q input %q exceeds limit", run.Name, k)
			}
			out.Runs[i].Inputs[k] = resolved
		}
		out.Runs[i].Images = append([]Image(nil), run.Images...)
		for j := range out.Runs[i].Images {
			if len(run.Images[j].Reference) > 4096 || strings.ContainsRune(run.Images[j].Reference, 0) || strings.ContainsAny(inputRef.ReplaceAllString(run.Images[j].Reference, ""), "{}") {
				return Definition{}, fmt.Errorf("run %q image has unsupported interpolation", run.Name)
			}
			v, err := expand(run.Images[j].Reference, values)
			if err != nil {
				return Definition{}, fmt.Errorf("run %q image: %w", run.Name, err)
			}
			if v == "" {
				return Definition{}, fmt.Errorf("run %q image reference is required", run.Name)
			}
			if len(v) > 4096 || strings.ContainsRune(v, 0) {
				return Definition{}, fmt.Errorf("run %q image exceeds limit", run.Name)
			}
			if err := ociverify.ValidateReference(v, out.Runs[i].Images[j].Platforms); err != nil {
				return Definition{}, fmt.Errorf("run %q image: %w", run.Name, err)
			}
			out.Runs[i].Images[j].Reference = v
			out.Runs[i].Images[j].Platforms = append([]string(nil), run.Images[j].Platforms...)
		}
	}
	return out, nil
}
func expand(s string, values map[string]string) (string, error) {
	var err error
	out := inputRef.ReplaceAllStringFunc(s, func(m string) string {
		name := inputRef.FindStringSubmatch(m)[1]
		v, ok := values[name]
		if !ok {
			err = fmt.Errorf("unknown input %q", name)
			return m
		}
		return v
	})
	return out, err
}
func validRepository(s string) bool {
	return remotetarget.ValidRepository(s)
}
func validInputName(s string) bool {
	return regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`).MatchString(s)
}
func contains(v []string, want string) bool {
	return slices.Contains(v, want)
}
func cloneInputs(in map[string]Input) map[string]Input {
	out := map[string]Input{}
	for k, v := range in {
		v.Values = append([]string(nil), v.Values...)
		out[k] = v
	}
	return out
}
