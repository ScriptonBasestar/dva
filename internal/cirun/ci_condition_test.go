package cirun

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func TestConditionSelectionStrict(t *testing.T) {
	for _, tc := range []struct {
		input           string
		valid, selected bool
	}{
		{`{"selected":true}`, true, true}, {` { "selected": false } `, true, false},
		{`{}`, false, false}, {`{"selected":null}`, false, false},
		{`{"selected":true,"selected":false}`, false, false},
		{`{"selected":true,"extra":1}`, false, false},
		{`{"selected":true} {}`, false, false}, {`true`, false, false},
	} {
		selected, err := conditionSelection([]byte(tc.input))
		if (err == nil) != tc.valid || (tc.valid && selected != tc.selected) {
			t.Fatalf("%q: %v %v", tc.input, selected, err)
		}
	}
}

func TestConditionSkipBlocksDependentsAndReportsNoApplicableSteps(t *testing.T) {
	root, step := reuseFixture(t)
	step.Condition = &config.CICondition{Run: `printf '{"selected":false}'`, Timeout: "1s"}
	step.Reuse.Probes = []string{"exit 44"}
	dependent := config.CIStep{Name: "dependent", Run: "exit 77", DependsOn: []string{"unit"}}
	state := t.TempDir()
	r, err := Run(context.Background(), Options{Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, Env: os.Environ(), Profile: profile(step, dependent)})
	if err != nil || r.Status != "skipped" || r.SkipReason != "no-applicable-steps" {
		t.Fatalf("report=%+v err=%v", r, err)
	}
	if r.Steps[0].SkipReason != "condition-false" || r.Steps[1].SkipReason != "dependency" || r.Steps[0].Reuse != nil {
		t.Fatalf("steps=%+v", r.Steps)
	}
	if _, err := os.Stat(filepath.Join(root, "artifact")); !os.IsNotExist(err) {
		t.Fatalf("unexpected body: %v", err)
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Report
	if err := json.Unmarshal(data, &decoded); err != nil || decoded.SkipReason != r.SkipReason || decoded.Steps[0].Condition == nil || *decoded.Steps[0].Condition.Selected {
		t.Fatalf("receipt=%s err=%v", data, err)
	}
}

func TestConditionTrueUsesReuseAndFreshStillEvaluates(t *testing.T) {
	root, step := reuseFixture(t)
	step.Condition = &config.CICondition{Run: `printf '{"selected":true}'`, Timeout: "1s"}
	state := t.TempDir()
	opts := Options{Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, Env: os.Environ(), Profile: profile(step)}
	for _, disposition := range []string{"executed", "reused", "bypassed"} {
		opts.Fresh = disposition == "bypassed"
		r, err := Run(context.Background(), opts)
		if err != nil || r.Status != "succeeded" || r.Steps[0].Reuse == nil || r.Steps[0].Reuse.Disposition != disposition || r.Steps[0].Condition == nil || !*r.Steps[0].Condition.Selected {
			t.Fatalf("%s report=%+v err=%v", disposition, r, err)
		}
	}
}

func TestConditionFalseOverridesLiveReuse(t *testing.T) {
	root, step := reuseFixture(t)
	probeFile := filepath.Join(t.TempDir(), "probe")
	step.Run = "printf x >> artifact"
	step.Reuse.Probes = []string{"printf x >> " + shellQuote(probeFile)}
	step.Condition = &config.CICondition{Run: `printf '{"selected":true}'`, Timeout: "1s"}
	state := t.TempDir()
	opts := Options{Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, Env: os.Environ(), Profile: profile(step)}
	first, err := Run(context.Background(), opts)
	if err != nil || first.Steps[0].Reuse == nil || first.Steps[0].Reuse.Disposition != "executed" {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	warm, err := Run(context.Background(), opts)
	if err != nil || warm.Steps[0].Reuse == nil || warm.Steps[0].Reuse.Disposition != "reused" {
		t.Fatalf("warm=%+v err=%v", warm, err)
	}
	if err := os.Remove(probeFile); err != nil {
		t.Fatal(err)
	}
	step.Condition = &config.CICondition{Run: `printf '{"selected":false}'`, Timeout: "1s"}
	opts.Profile = profile(step)
	second, err := Run(context.Background(), opts)
	if err != nil || second.Status != "skipped" || second.Steps[0].Reuse != nil {
		t.Fatalf("second=%+v err=%v", second, err)
	}
	if _, err := os.Stat(probeFile); !os.IsNotExist(err) {
		t.Fatalf("reuse probe executed: %v", err)
	}
	artifact, err := os.ReadFile(filepath.Join(root, "artifact"))
	if err != nil || string(artifact) != "x" {
		t.Fatalf("body executed: %q %v", artifact, err)
	}
}

func TestConditionDoesNotRunAfterFailedPrerequisite(t *testing.T) {
	root, step := reuseFixture(t)
	step.Condition = &config.CICondition{Run: `touch condition-ran; printf '{"selected":true}'`, Timeout: "1s"}
	step.DependsOn = []string{"prerequisite"}
	no := false
	p := profile(config.CIStep{Name: "prerequisite", Run: "exit 7"}, step)
	p.FailFast = &no
	state := t.TempDir()
	r, err := Run(context.Background(), Options{Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, Env: os.Environ(), Profile: p})
	if err == nil || r.Steps[1].Status != "skipped" || r.Steps[1].SkipReason != "dependency" || r.Steps[1].Condition != nil {
		t.Fatalf("report=%+v err=%v", r, err)
	}
	if _, err := os.Stat(filepath.Join(root, "condition-ran")); !os.IsNotExist(err) {
		t.Fatalf("condition executed: %v", err)
	}
}

func TestConditionOutputNeverAppearsInPersistentLogOrReceipt(t *testing.T) {
	root, _ := reuseFixture(t)
	state := t.TempDir()
	step := config.CIStep{Name: "unit", Run: "exit 77", Condition: &config.CICondition{
		Run: `printf '{"selected":false,"extra":"stdout-canary"}'; printf stderr-canary >&2`, Timeout: "1s"}}
	r, err := Run(context.Background(), Options{Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, Env: os.Environ(), Profile: profile(step)})
	if err == nil {
		t.Fatalf("malformed condition accepted: %+v", r)
	}
	for _, path := range []string{r.LogPath, filepath.Join(state, r.ID+".json")} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "stdout-canary") || strings.Contains(string(data), "stderr-canary") {
			t.Fatalf("condition output leaked to %s", path)
		}
	}
}

func TestConditionErrorsAreFailClosedAndOutputWithheld(t *testing.T) {
	for _, command := range []string{`printf 'secret'; exit 1`, `printf '{"selected":true,"extra":"secret"}'`, `head -c 4097 /dev/zero`, `sleep 2`} {
		t.Run(command, func(t *testing.T) {
			root := t.TempDir()
			step := config.CIStep{Name: "unit", Run: "exit 77", Timeout: "1s", Condition: &config.CICondition{Run: command, Timeout: "50ms"}}
			r, err := runConditionedStep(context.Background(), root, step, nil, 1, "test", nil, time.Now())
			if err == nil || (r.Status != "failed" && r.Status != "timed_out") || strings.Contains(r.Error, "secret") {
				t.Fatalf("report=%+v err=%v", r, err)
			}
		})
	}
}

func TestConditionUsesStepEnvironmentAndWorkdir(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "sub"), 0o700); err != nil {
		t.Fatal(err)
	}
	step := config.CIStep{Name: "unit", Workdir: "sub", Environment: map[string]string{"SELECT_VALUE": "yes"}, Condition: &config.CICondition{Run: `test "$SELECT_VALUE" = yes && test "${PWD##*/}" = sub && printf '{"selected":true}'`, Timeout: "1s"}}
	selected, err := runCondition(context.Background(), root, step, os.Environ(), 1)
	if err != nil || !selected {
		t.Fatalf("selected=%v err=%v", selected, err)
	}
}

func TestConditionCanonicalizesRootAndRejectsEscapingWorkdir(t *testing.T) {
	root := t.TempDir()
	alias := filepath.Join(t.TempDir(), "root-alias")
	if err := os.Symlink(root, alias); err != nil {
		t.Fatal(err)
	}
	step := config.CIStep{Name: "unit", Run: "true", Condition: &config.CICondition{Run: `printf '{"selected":true}'`, Timeout: "1s"}}
	selected, err := runCondition(context.Background(), alias, step, os.Environ(), 1)
	if err != nil || !selected {
		t.Fatalf("canonical root: selected=%v err=%v", selected, err)
	}
	report, err := runConditionedStep(context.Background(), alias, step, os.Environ(), 1, "test", io.Discard, time.Now())
	if err != nil || report.Status != "succeeded" {
		t.Fatalf("canonical root body: report=%+v err=%v", report, err)
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(root, "outside")); err != nil {
		t.Fatal(err)
	}
	step.Workdir = "outside"
	if _, err := runCondition(context.Background(), alias, step, os.Environ(), 1); err == nil {
		t.Fatal("accepted workdir outside canonical root")
	}
}

func TestConditionConsumesStepDeadline(t *testing.T) {
	root := t.TempDir()
	step := config.CIStep{Name: "unit", Run: "sleep 2", Timeout: "150ms", Condition: &config.CICondition{Run: `sleep 0.08; printf '{"selected":true}'`, Timeout: "120ms"}}
	started := time.Now()
	r, err := runConditionedStep(context.Background(), root, step, os.Environ(), 1, "test", io.Discard, started)
	if err == nil || r.Status != "timed_out" || time.Since(started) > time.Second {
		t.Fatalf("report=%+v err=%v", r, err)
	}
}
