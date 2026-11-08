package config

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- Unit tests: merge helpers ---

func TestMergeStringMap(t *testing.T) {
	t.Run("nil dst", func(t *testing.T) {
		got := mergeStringMap(nil, map[string]string{"a": "1"})
		if got["a"] != "1" {
			t.Errorf("got %v, want map with a=1", got)
		}
	})
	t.Run("nil src", func(t *testing.T) {
		dst := map[string]string{"a": "1"}
		got := mergeStringMap(dst, nil)
		if got["a"] != "1" {
			t.Errorf("got %v, want original dst", got)
		}
	})
	t.Run("overlap and new keys", func(t *testing.T) {
		dst := map[string]string{"a": "1", "b": "2"}
		src := map[string]string{"b": "X", "c": "3"}
		got := mergeStringMap(dst, src)
		if got["a"] != "1" || got["b"] != "X" || got["c"] != "3" {
			t.Errorf("got %v, want a=1 b=X c=3", got)
		}
	})
}

func TestMergeLifecycleEntryPartialOverride(t *testing.T) {
	base := &LifecycleEntry{
		Name:   "compose",
		Plugin: "compose",
		Order:  10,
		Tags:   []string{"core"},
		Exports: map[string]string{
			"DB_HOST": "postgres",
		},
		Compose: &ComposePluginConfig{
			Files:       []string{"docker-compose.yml"},
			ProjectName: "myapp",
		},
	}
	other := &LifecycleEntry{
		Plugin: "compose", // same plugin — allowed
		Compose: &ComposePluginConfig{
			Files: []string{"docker-compose.yml", "docker-compose.dev.yml"},
		},
		Exports: map[string]string{
			"REDIS_HOST": "redis",
		},
	}

	merged, err := MergeLifecycleEntry(base, other)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Order preserved from base (other has zero-value)
	if merged.Order != 10 {
		t.Errorf("order = %d, want 10 (preserved)", merged.Order)
	}
	// Tags preserved from base (other has nil)
	if len(merged.Tags) != 1 || merged.Tags[0] != "core" {
		t.Errorf("tags = %v, want [core] (preserved)", merged.Tags)
	}
	// Files replaced by other
	if len(merged.Compose.Files) != 2 {
		t.Errorf("compose.files = %v, want 2 files (replaced)", merged.Compose.Files)
	}
	// ProjectName preserved from base (other has zero-value)
	if merged.Compose.ProjectName != "myapp" {
		t.Errorf("compose.project_name = %q, want myapp (preserved)", merged.Compose.ProjectName)
	}
	// Exports merged (both keys present)
	if merged.Exports["DB_HOST"] != "postgres" {
		t.Errorf("exports.DB_HOST = %q, want postgres", merged.Exports["DB_HOST"])
	}
	if merged.Exports["REDIS_HOST"] != "redis" {
		t.Errorf("exports.REDIS_HOST = %q, want redis", merged.Exports["REDIS_HOST"])
	}
}

func TestMergeLifecycleEntryRestrictedPlugin(t *testing.T) {
	base := &LifecycleEntry{
		Name:   "web",
		Plugin: "compose",
	}
	other := &LifecycleEntry{
		Plugin: "helm",
	}

	_, err := MergeLifecycleEntry(base, other)
	if err == nil {
		t.Fatal("expected error for restricted plugin change, got nil")
	}
	if !strings.Contains(err.Error(), "restricted field") {
		t.Errorf("error = %q, want mention of restricted field", err.Error())
	}
}

func TestMergeLifecycleEntryNilBase(t *testing.T) {
	other := &LifecycleEntry{Plugin: "compose", Order: 5}
	got, err := MergeLifecycleEntry(nil, other)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != other {
		t.Error("nil base should return other directly")
	}
}

func TestMergeComposeConfig(t *testing.T) {
	base := &ComposePluginConfig{
		Files:       []string{"docker-compose.yml"},
		ProjectName: "myapp",
		Services: map[string]ServiceTagConfig{
			"web": {Tags: []string{"app"}},
		},
	}
	other := &ComposePluginConfig{
		Files: []string{"docker-compose.yml", "override.yml"},
		Services: map[string]ServiceTagConfig{
			"db": {Tags: []string{"infra"}},
		},
	}

	mergeComposeConfig(base, other)

	// Files: list replace
	if len(base.Files) != 2 || base.Files[1] != "override.yml" {
		t.Errorf("files = %v, want replaced list", base.Files)
	}
	// ProjectName: preserved (other is zero-value)
	if base.ProjectName != "myapp" {
		t.Errorf("project_name = %q, want myapp", base.ProjectName)
	}
	// Services: map merge (both keys)
	if _, ok := base.Services["web"]; !ok {
		t.Error("services.web should be preserved")
	}
	if _, ok := base.Services["db"]; !ok {
		t.Error("services.db should be added")
	}
}

func TestMergeInteractionCommandPartialOverride(t *testing.T) {
	base := &InteractionCommand{
		Description: "Open shell",
		Service:     "app",
		Command:     "/bin/bash",
		Environment: map[string]string{"TERM": "xterm"},
	}
	other := &InteractionCommand{
		Command: "/bin/zsh",
		Environment: map[string]string{
			"SHELL": "/bin/zsh",
		},
	}

	merged, err := mergeInteractionCommand(base, other)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if merged.Description != "Open shell" {
		t.Errorf("description = %q, want preserved", merged.Description)
	}
	if merged.Service != "app" {
		t.Errorf("service = %q, want preserved", merged.Service)
	}
	if merged.Command != "/bin/zsh" {
		t.Errorf("command = %q, want /bin/zsh (replaced)", merged.Command)
	}
	if merged.Environment["TERM"] != "xterm" {
		t.Errorf("env.TERM = %q, want xterm (preserved)", merged.Environment["TERM"])
	}
	if merged.Environment["SHELL"] != "/bin/zsh" {
		t.Errorf("env.SHELL = %q, want /bin/zsh (added)", merged.Environment["SHELL"])
	}
}

func TestMergeInteractionCommandRestrictedRunner(t *testing.T) {
	base := &InteractionCommand{Runner: "docker"}
	other := &InteractionCommand{Runner: "kubectl"}

	_, err := mergeInteractionCommand(base, other)
	if err == nil {
		t.Fatal("expected error for restricted runner change, got nil")
	}
	if !strings.Contains(err.Error(), "restricted field") {
		t.Errorf("error = %q, want mention of restricted field", err.Error())
	}
}

func TestMergeEnvironmentProfile(t *testing.T) {
	base := EnvironmentProfile{
		Description: "staging",
		Stack:       []string{"compose", "kubectl"},
		Environment: map[string]string{"ENV": "stg"},
	}
	other := EnvironmentProfile{
		Stack:       []string{"compose"},
		Environment: map[string]string{"REGION": "us-east-1"},
	}

	merged := mergeEnvironmentProfile(base, other)

	if merged.Description != "staging" {
		t.Errorf("description = %q, want preserved", merged.Description)
	}
	// List replace
	if len(merged.Stack) != 1 || merged.Stack[0] != "compose" {
		t.Errorf("stack = %v, want replaced to [compose]", merged.Stack)
	}
	// Map merge
	if merged.Environment["ENV"] != "stg" {
		t.Errorf("env.ENV = %q, want preserved", merged.Environment["ENV"])
	}
	if merged.Environment["REGION"] != "us-east-1" {
		t.Errorf("env.REGION = %q, want added", merged.Environment["REGION"])
	}
}

// --- Integration tests: YAML file → Load() ---

func TestLoadDeepMergeOverride(t *testing.T) {
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, FileName), []byte(`
stack:
  compose:
    default_runner: compose
    order: 10
    tags: [core]
    exports:
      DB_HOST: postgres
    runners:
      compose:
        files:
          - docker-compose.yml
        project_name: myapp

interaction:
  shell:
    description: "Open shell"
    service: app
    command: /bin/bash
    environment:
      TERM: xterm

`), 0644)

	os.WriteFile(filepath.Join(tmpDir, "dva.override.yml"), []byte(`
stack:
  compose:
    exports:
      REDIS_HOST: redis
    runners:
      compose:
        files:
          - docker-compose.yml
          - docker-compose.dev.yml

interaction:
  shell:
    command: /bin/zsh
    environment:
      SHELL: /bin/zsh

`), 0644)

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	// Stack: compose entry deep merged
	entry := cfg.Stack["compose"]
	if entry == nil {
		t.Fatal("compose stack entry not found")
	}
	if entry.Order != 10 {
		t.Errorf("compose entry order = %d, want 10 (preserved)", entry.Order)
	}
	composeCfg := entry.ComposeConfig()
	if composeCfg == nil {
		t.Fatal("compose runner config not found")
	}
	if composeCfg.ProjectName != "myapp" {
		t.Errorf("compose runner project_name = %q, want myapp (preserved)", composeCfg.ProjectName)
	}
	if len(composeCfg.Files) != 2 {
		t.Errorf("compose runner files = %v, want 2 files", composeCfg.Files)
	}
	if entry.Exports["DB_HOST"] != "postgres" {
		t.Errorf("exports.DB_HOST = %q, want postgres", entry.Exports["DB_HOST"])
	}
	if entry.Exports["REDIS_HOST"] != "redis" {
		t.Errorf("exports.REDIS_HOST = %q, want redis", entry.Exports["REDIS_HOST"])
	}
	if len(entry.Tags) != 1 || entry.Tags[0] != "core" {
		t.Errorf("tags = %v, want [core] (preserved)", entry.Tags)
	}

	// Interaction: shell deep merged
	shell := cfg.Interaction["shell"]
	if shell == nil {
		t.Fatal("interaction.shell not found")
	}
	if shell.Description != "Open shell" {
		t.Errorf("shell.description = %q, want preserved", shell.Description)
	}
	if shell.Service != "app" {
		t.Errorf("shell.service = %q, want preserved", shell.Service)
	}
	if shell.Command != "/bin/zsh" {
		t.Errorf("shell.command = %q, want /bin/zsh (replaced)", shell.Command)
	}
	if shell.Environment["TERM"] != "xterm" {
		t.Errorf("shell.env.TERM = %q, want preserved", shell.Environment["TERM"])
	}
	if shell.Environment["SHELL"] != "/bin/zsh" {
		t.Errorf("shell.env.SHELL = %q, want added", shell.Environment["SHELL"])
	}

}

func TestLoadRestrictedFieldError(t *testing.T) {
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, FileName), []byte(`
stack:
  web:
    plugin: docker
    order: 10
    image: nginx:latest
`), 0644)

	os.WriteFile(filepath.Join(tmpDir, "dva.override.yml"), []byte(`
stack:
  web:
    plugin: helm
`), 0644)

	_, err := Load(tmpDir)
	if err == nil {
		t.Fatal("expected error for restricted plugin change, got nil")
	}
	if !strings.Contains(err.Error(), "restricted field") {
		t.Errorf("error = %q, want mention of restricted field", err.Error())
	}
}

// captureStd runs fn with os.Stdout and os.Stderr replaced by pipes and returns
// what each received. Both are restored before returning.
//
// Both pipes are drained while fn runs. Reading only after fn returns relies on the
// output fitting the pipe buffer, which is not a fixed size: Darwin hands out 512-byte
// buffers once kernel pipe memory runs short, and then the writer blocks for good.
func captureStd(t *testing.T, fn func()) (stdout, stderr string) {
	t.Helper()

	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}

	origOut, origErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = outW, errW
	defer func() {
		os.Stdout, os.Stderr = origOut, origErr
	}()

	type drained struct {
		data []byte
		err  error
	}
	drain := func(r *os.File) <-chan drained {
		done := make(chan drained, 1)
		go func() { data, err := io.ReadAll(r); done <- drained{data, err} }()
		return done
	}
	outDone, errDone := drain(outR), drain(errR)

	fn()

	if err := outW.Close(); err != nil {
		t.Fatalf("close stdout writer: %v", err)
	}
	if err := errW.Close(); err != nil {
		t.Fatalf("close stderr writer: %v", err)
	}

	out, errOut := <-outDone, <-errDone
	if out.err != nil {
		t.Fatalf("read stdout: %v", out.err)
	}
	if errOut.err != nil {
		t.Fatalf("read stderr: %v", errOut.err)
	}
	return string(out.data), string(errOut.data)
}

// TestStackOverrideConflictWarnsOnStderrNotStdout pins TASK-116.
//
// mergeEnvironmentProfile cannot return an error, so a plugin-type conflict in
// stack_overrides can only be reported by printing. It printed to stdout — the
// one production [warn] in the repo that did — and it fires during config load,
// before any command has produced output. Under --json that put a non-JSON line
// ahead of the document, so `dva … | jq` failed on line 1 while the command
// still exited 0.
//
// Asserting stdout is empty is the half that would regress silently; asserting
// the warning still reaches stderr is what stops the "fix" from being a deletion.
func TestStackOverrideConflictWarnsOnStderrNotStdout(t *testing.T) {
	base := EnvironmentProfile{
		StackOverrides: map[string]*LifecycleEntry{
			"api": {Name: "api", Plugin: "compose"},
		},
	}
	other := EnvironmentProfile{
		StackOverrides: map[string]*LifecycleEntry{
			"api": {Name: "api", Plugin: "helm"},
		},
	}

	stdout, stderr := captureStd(t, func() {
		mergeEnvironmentProfile(base, other)
	})

	if stdout != "" {
		t.Errorf("config merge wrote %d bytes to stdout, which corrupts --json output:\n%s",
			len(stdout), stdout)
	}
	if !strings.Contains(stderr, "[warn] stack_override") {
		t.Errorf("the warning did not reach stderr; got %q — a conflict must still be reported", stderr)
	}
	if !strings.Contains(stderr, "api") {
		t.Errorf("stderr %q does not name the offending key", stderr)
	}
	// TASK-157: the outer key is the identity; do not also print `for stack entry ""`.
	if strings.Contains(stderr, `stack entry ""`) || strings.Contains(stderr, `entry ""`) {
		t.Errorf("override warning still names an empty entry: %q", stderr)
	}
	if !strings.Contains(stderr, "cannot override plugin type") {
		t.Errorf("stderr %q missing the plugin conflict", stderr)
	}
	t.Logf("stdout=%d bytes, stderr=%d bytes; message=%q", len(stdout), len(stderr), stderr)
}
