package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestCIEventRoutingAndCheckoutIdentity(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var workflow struct {
		On map[string]struct {
			Branches []string `yaml:"branches"`
			Types    []string `yaml:"types"`
		} `yaml:"on"`
		Concurrency struct {
			Group  string `yaml:"group"`
			Cancel bool   `yaml:"cancel-in-progress"`
		} `yaml:"concurrency"`
		Jobs map[string]struct {
			If    string `yaml:"if"`
			Steps []struct {
				Name string         `yaml:"name"`
				Uses string         `yaml:"uses"`
				Run  string         `yaml:"run"`
				If   string         `yaml:"if"`
				With map[string]any `yaml:"with"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		t.Fatal(err)
	}
	if len(workflow.On) != 4 || !reflect.DeepEqual(workflow.On["push"].Branches, []string{"master", "dev/**"}) ||
		!reflect.DeepEqual(workflow.On["pull_request"].Branches, []string{"master"}) ||
		!reflect.DeepEqual(workflow.On["merge_group"].Types, []string{"checks_requested"}) {
		t.Fatalf("unexpected event triggers: %+v", workflow.On)
	}
	if _, ok := workflow.On["workflow_dispatch"]; !ok {
		t.Fatal("manual trigger missing")
	}
	if workflow.Concurrency.Group != "ci-${{ github.event_name }}-${{ github.ref }}" || !workflow.Concurrency.Cancel {
		t.Fatalf("unexpected concurrency: %+v", workflow.Concurrency)
	}
	want := []string{"test", "commit-subjects", "lint", "config-env-platform", "goreleaser-snapshot"}
	checks := map[string][]string{
		"test":                {"Documentation", "Build", "Skill Installer Dogfood", "Test", "Preserve test progress", "Integration Test", "Preserve integration progress", "Vet"},
		"commit-subjects":     {"Commit Subjects"},
		"lint":                {"Format", "Lint", "Install gopls", "gopls check"},
		"config-env-platform": {"Unit suite", "Install cosign", "Install verified sops and age", "Verify the pinned toolchain resolves", "Integration Test"},
		"goreleaser-snapshot": {"GoReleaser snapshot"},
	}
	if len(workflow.Jobs) != len(want) {
		t.Fatalf("unexpected jobs: %+v", workflow.Jobs)
	}
	for _, name := range want {
		job, ok := workflow.Jobs[name]
		if !ok || job.If != "" || len(job.Steps) < 2 {
			t.Fatalf("job %s absent, conditional or incomplete", name)
		}
		if job.Steps[0].Uses != "actions/checkout@v7" || job.Steps[0].If != "" {
			t.Fatalf("%s does not start with unconditional checkout", name)
		}
		if _, pinned := job.Steps[0].With["ref"]; pinned {
			t.Fatalf("%s overrides the event checkout", name)
		}
		if job.Steps[1].Run != "sh scripts/ci-input.sh" || job.Steps[1].If != "" {
			t.Fatalf("%s does not verify event SHA after checkout", name)
		}
		if name == "test" || name == "commit-subjects" {
			if job.Steps[0].With["fetch-depth"] != 0 {
				t.Fatalf("%s must fetch full history", name)
			}
		}
		var names []string
		for _, step := range job.Steps {
			if step.Name != "" {
				names = append(names, step.Name)
			}
			if step.If != "" && step.If != "always()" && step.If != "matrix.sops == 'latest'" {
				t.Fatalf("%s has unexpected step condition %q", name, step.If)
			}
		}
		if !reflect.DeepEqual(names, checks[name]) {
			t.Fatalf("%s named steps changed: %q", name, names)
		}
	}
}

func TestCIInputScriptChecksActualGitCommit(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(root, "scripts", "ci-input.sh")
	dir := t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %s: %v", args, out, err)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-q")
	git("-c", "user.name=fixture", "-c", "user.email=fixture@example.invalid", "commit", "--allow-empty", "-qm", "fixture")
	sha := git("rev-parse", "HEAD")
	run := func(extra ...string) ([]byte, error) {
		cmd := exec.Command("sh", script)
		cmd.Dir = dir
		cmd.Env = append([]string{"PATH=" + os.Getenv("PATH")}, extra...)
		return cmd.CombinedOutput()
	}
	valid := []string{"GITHUB_SHA=" + sha, "GITHUB_REF=refs/heads/merge-queue", "GITHUB_EVENT_NAME=merge_group"}
	out, err := run(valid...)
	if err != nil || !strings.Contains(string(out), sha) || !strings.Contains(string(out), "merge_group") {
		t.Fatalf("matching SHA: %s %v", out, err)
	}
	out, err = run("GITHUB_SHA=bad", valid[1], valid[2])
	if err == nil || !strings.Contains(string(out), "does not match") {
		t.Fatalf("mismatched SHA: %s %v", out, err)
	}
	for missing := range valid {
		var args []string
		for i, value := range valid {
			if i != missing {
				args = append(args, value)
			}
		}
		if out, err := run(args...); err == nil || !strings.Contains(string(out), "required") {
			t.Errorf("missing input %d: %s %v", missing, out, err)
		}
	}
}
