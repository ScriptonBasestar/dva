package cirun

import (
	"context"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepositoryLocalIDAcrossWorktreesAndSeparateClones(t *testing.T) {
	base := t.TempDir()
	repo := filepath.Join(base, "one")
	other := filepath.Join(base, "two")
	linked := filepath.Join(base, "linked")
	run := func(args ...string) {
		t.Helper()
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git: %s %v", out, err)
		}
	}
	run("init", "-q", repo)
	run("-C", repo, "-c", "user.name=test", "-c", "user.email=test@example.invalid", "commit", "--allow-empty", "-qm", "initial")
	run("-C", repo, "worktree", "add", "--detach", linked, "HEAD")
	run("init", "-q", other)
	a, err := RepositoryLocalID(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	b, err := RepositoryLocalID(context.Background(), linked)
	if err != nil {
		t.Fatal(err)
	}
	c, err := RepositoryLocalID(context.Background(), other)
	if err != nil {
		t.Fatal(err)
	}
	if a != b || a == c {
		t.Fatalf("identities: %q %q %q", a, b, c)
	}
	if _, err = RepositoryLocalID(context.Background(), base); err == nil {
		t.Fatal("non-repository accepted")
	}
}

func TestAnalyzeRepositoryHistoryDoesNotAttributeUnassigned(t *testing.T) {
	r := []Report{{ID: "one", Root: "/tmp/one", Profile: "full", Observability: &Observability{RepositoryLocalID: "a"}}, {ID: "two", Root: "/tmp/two", Profile: "full", Observability: &Observability{RepositoryLocalID: "a"}}, {ID: "legacy", Profile: "full"}, {ID: "other", Profile: "full", Observability: &Observability{RepositoryLocalID: "b"}}}
	a := AnalyzeRepositoryHistory(r, "a", "full")
	if len(a.Runs) != 2 || a.ProfileUnassigned != 1 {
		t.Fatalf("unexpected result: %+v", a)
	}
}

func TestRepositoryHistoryObservationsAcrossRootsShareBudget(t *testing.T) {
	admission := &AdmissionObservation{Class: "test", Outcome: "acquired"}
	output := &OutputObservation{Scope: "native_leaf_child_output"}
	steps := make([]StepReport, maxHistoryStepObservations-1)
	steps[0] = StepReport{Name: "observed", Admission: admission, OutputObservation: output}
	reports := []Report{
		{ID: "first", Root: "/one", Profile: "full", Observability: &Observability{RepositoryLocalID: "a"}, Admission: admission, Steps: steps},
		{ID: "legacy", Root: "/two", Profile: "full", Observability: &Observability{RepositoryLocalID: "a"}, Steps: []StepReport{{Name: "legacy"}, {Name: "omitted"}}},
		{ID: "unassigned", Profile: "full", Steps: []StepReport{{Name: "excluded"}}},
		{ID: "other", Profile: "full", Observability: &Observability{RepositoryLocalID: "b"}},
	}
	a := AnalyzeRepositoryHistory(reports, "a", "full")
	if len(a.Runs) != 2 || a.Runs[0].Root != "/one" || a.Runs[1].Root != "/two" || a.ProfileUnassigned != 1 || len(a.Observations) != 2 {
		t.Fatalf("selection: %+v", a)
	}
	first, legacy := a.Observations[0], a.Observations[1]
	if first.RunID != "first" || first.Admission != admission || first.Steps[0].Admission != admission || first.Steps[0].OutputObservation != output || len(first.Steps) != maxHistoryStepObservations-1 {
		t.Fatal("first root observation was not preserved")
	}
	if legacy.RunID != "legacy" || legacy.Admission != nil || len(legacy.Steps) != 1 || legacy.Steps[0].Admission != nil || legacy.Steps[0].OutputObservation != nil || legacy.OmittedSteps != 1 {
		t.Fatalf("legacy observation or aggregate budget: %+v", legacy)
	}
	data, err := json.Marshal(legacy)
	if err != nil || !strings.Contains(string(data), `"admission":null`) || !strings.Contains(string(data), `"output_observation":null`) {
		t.Fatalf("legacy nulls: %s %v", data, err)
	}
}
