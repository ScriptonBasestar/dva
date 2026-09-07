package lifecycle

import (
	"context"
	"log/slog"
	"reflect"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// argvRecorder collects the argv every dry-run log line reports, so a test can assert
// on the exact `docker compose` invocation a plan produces without running docker.
type argvRecorder struct {
	calls [][]string
}

func (r *argvRecorder) Enabled(context.Context, slog.Level) bool { return true }
func (r *argvRecorder) WithAttrs([]slog.Attr) slog.Handler       { return r }
func (r *argvRecorder) WithGroup(string) slog.Handler            { return r }

func (r *argvRecorder) Handle(_ context.Context, rec slog.Record) error {
	if rec.Message != "dry-run" {
		return nil
	}
	var cmd string
	var args []string
	rec.Attrs(func(a slog.Attr) bool {
		switch a.Key {
		case "command":
			cmd, _ = a.Value.Any().(string)
		case "args":
			args, _ = a.Value.Any().([]string)
		}
		return true
	})
	r.calls = append(r.calls, append([]string{cmd}, args...))
	return nil
}

// planProfilesConfig builds the dns-bridge shape: one compose stack entry, one plan
// entry selecting it, with whatever profiles/services the caller wants to test.
func planProfilesConfig(profiles, services []string) *config.Config {
	return &config.Config{
		Stack: map[string]*config.LifecycleEntry{
			"core-compose": {
				Name:          "core-compose",
				DefaultRunner: "compose",
				Runners: map[string]any{
					"compose": &config.ComposePluginConfig{Files: []string{"compose.yaml"}},
				},
			},
		},
		Plans: map[string]*config.PlanConfig{
			"docker-dev": {Entries: []config.PlanEntry{{
				Name: "core-compose", Runner: "compose", Order: 10,
				Profiles: profiles, Services: services,
			}}},
		},
	}
}

// planUpArgv runs one dry-run `up` for the plan and returns the argv it would exec.
func planUpArgv(t *testing.T, cfg *config.Config) []string {
	t.Helper()
	plan, err := ResolvePlan(cfg, "docker-dev", nil)
	if err != nil {
		t.Fatalf("ResolvePlan: %v", err)
	}
	orch, err := NewPlanOrchestrator(cfg, config.NewEnvironment(nil, ".", "."), plan)
	if err != nil {
		t.Fatalf("NewPlanOrchestrator: %v", err)
	}
	rec := &argvRecorder{}
	orch.logger = slog.New(rec)
	if err := orch.Up(context.Background(), UpOptions{DryRun: true, Wait: true}); err != nil {
		t.Fatalf("dry-run up: %v", err)
	}
	if len(rec.calls) != 1 {
		t.Fatalf("dry-run logged %d commands, want 1: %v", len(rec.calls), rec.calls)
	}
	return rec.calls[0]
}

// TestPlanProfilesReachComposeArgv locks TASK-315's whole point: `profiles:` on a plan
// entry becomes --profile on the docker compose invocation, ahead of the subcommand
// (docker only accepts it as a top-level flag), in declaration order.
func TestPlanProfilesReachComposeArgv(t *testing.T) {
	got := planUpArgv(t, planProfilesConfig([]string{"rust", "monitoring"}, nil))
	want := []string{
		"docker", "compose", "-f", "compose.yaml",
		"--profile", "rust", "--profile", "monitoring",
		"up", "-d", "--wait",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("argv = %v, want %v", got, want)
	}
}

// TestPlanProfilesComposeWithServices locks the stated composition rule: profiles decide
// which services compose considers, `services:` then narrows what this entry starts. Both
// must appear, each in its own position — profiles before the subcommand, services after.
func TestPlanProfilesComposeWithServices(t *testing.T) {
	got := planUpArgv(t, planProfilesConfig([]string{"rust"}, []string{"postgres", "dns-bridge-api-rs"}))
	want := []string{
		"docker", "compose", "-f", "compose.yaml",
		"--profile", "rust",
		"up", "-d", "--wait", "postgres", "dns-bridge-api-rs",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("argv = %v, want %v", got, want)
	}
}

// TestPlanWithoutProfilesArgvUnchanged is the regression guard TASK-315 asks for: an
// absent or empty `profiles:` must produce the argv this repo produced before the field
// existed. Both spellings are checked because `profiles: []` is what a template renders
// for an empty list, and it must not be distinguishable from omitting the key.
func TestPlanWithoutProfilesArgvUnchanged(t *testing.T) {
	want := []string{
		"docker", "compose", "-f", "compose.yaml",
		"up", "-d", "--wait", "postgres",
	}
	for _, tc := range []struct {
		name     string
		profiles []string
	}{
		{"absent", nil},
		{"empty", []string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := planUpArgv(t, planProfilesConfig(tc.profiles, []string{"postgres"}))
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("argv = %v, want %v", got, want)
			}
		})
	}
}

// TestPlanProfilesRegisteredOnlyForComposeRunner locks the runner gate: --profile is a
// docker compose flag, so an entry a plan pointed at another runner must not collect one.
func TestPlanProfilesRegisteredOnlyForComposeRunner(t *testing.T) {
	cfg := planProfilesConfig([]string{"rust"}, nil)
	cfg.Stack["core-compose"].Runners["process"] = &config.ProcessPluginConfig{Command: "echo hi"}
	cfg.Plans["docker-dev"].Entries[0].Runner = "process"

	plan, err := ResolvePlan(cfg, "docker-dev", nil)
	if err != nil {
		t.Fatalf("ResolvePlan: %v", err)
	}
	orch, err := NewPlanOrchestrator(cfg, config.NewEnvironment(nil, ".", "."), plan)
	if err != nil {
		t.Fatalf("NewPlanOrchestrator: %v", err)
	}
	if got, ok := orch.composeProfiles["core-compose"]; ok {
		t.Fatalf("non-compose runner registered compose profiles: %v", got)
	}
}
