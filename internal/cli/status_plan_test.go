package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestStatusRejectsUnknownPlanName(t *testing.T) {
	useConfig(t, planStackConfig)

	err := statusCmd.RunE(statusCmd, []string{"p1-typo"})
	if err == nil {
		t.Fatal("'dva status p1-typo' returned nil; an unknown plan name must not silently report on the whole workspace")
	}
	if !strings.Contains(err.Error(), "p1-typo") {
		t.Errorf("error must name the unknown argument, got: %v", err)
	}
	if !strings.Contains(err.Error(), "p1") {
		t.Errorf("error should list the available plans, got: %v", err)
	}
}

func TestStatusAcceptsKnownPlanName(t *testing.T) {
	useConfig(t, planStackConfig)

	if err := statusCmd.RunE(statusCmd, []string{"p1"}); err != nil {
		t.Fatalf("'dva status p1' must report the real plan: %v", err)
	}
}

// The guard keys off plans being configured. Without plans a positional argument
// was never a plan name, so the whole-workspace path must stay reachable.
func TestStatusWithoutPlansKeepsWorkspacePath(t *testing.T) {
	useConfig(t, noPlanStackConfig)

	if err := statusCmd.RunE(statusCmd, []string{}); err != nil {
		t.Fatalf("'dva status' with no plans must report the workspace: %v", err)
	}
	if err := statusCmd.RunE(statusCmd, []string{"s1"}); err != nil {
		t.Fatalf("'dva status s1' with no plans defined must keep its previous behavior: %v", err)
	}
}

// status is deliberately failure-tolerant: it reports "Config: not found" and
// succeeds with no dva.yml at all. The guard sits behind the config check and
// must not disturb that.
func TestStatusWithoutConfigStaysTolerant(t *testing.T) {
	t.Chdir(t.TempDir())
	cfg = nil
	t.Cleanup(func() { cfg = nil })

	if err := statusCmd.RunE(statusCmd, []string{}); err != nil {
		t.Fatalf("'dva status' with no config must still succeed: %v", err)
	}
}

// No args means detectPlanRoute was never given a plan name slot to read, so the
// guard must stay silent even when plans exist.
func TestStatusWithPlansAndNoArgs(t *testing.T) {
	useConfig(t, planStackConfig)

	if err := statusCmd.RunE(statusCmd, []string{}); err != nil {
		t.Fatalf("'dva status' with plans but no args must not error: %v", err)
	}
}

func TestStatusWithAmbiguousPlansFallsBackToWorkspace(t *testing.T) {
	useConfig(t, planStackConfig+`  p2:
    entries:
      - name: s2
`)

	out := captureStdout(t, func() {
		if err := statusCmd.RunE(statusCmd, []string{}); err != nil {
			t.Fatalf("'dva status' with ambiguous plans must report the workspace: %v", err)
		}
	})
	for _, entry := range []string{"[s1]", "[s2]"} {
		if !strings.Contains(out, entry) {
			t.Errorf("'dva status' output does not contain workspace entry %s:\n%s", entry, out)
		}
	}
}

const defaultPlanStatusConfig = `version: "0.1.44"
default_plan: p1
stack:
  s1:
    default_runner: script
    runners:
      script:
        up: "true"
  s2:
    default_runner: script
    runners:
      script:
        up: "true"
plans:
  p1:
    entries:
      - name: s1
`

func setStatusAll(t *testing.T) {
	t.Helper()
	statusAll = true
	t.Cleanup(func() { statusAll = false })
}

func setJSONOutput(t *testing.T) {
	t.Helper()
	jsonOutput = true
	t.Cleanup(func() { jsonOutput = false })
}

func TestStatusPlanPathEmitsJSON(t *testing.T) {
	useConfig(t, defaultPlanStatusConfig)
	setJSONOutput(t)

	var err error
	out := captureStdout(t, func() { err = statusCmd.RunE(statusCmd, []string{"p1"}) })
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Action string `json:"action"`
		Plan   string `json:"plan"`
		Status struct {
			Entries []struct {
				Name string `json:"name"`
			} `json:"entries"`
		} `json:"status"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, out)
	}
	if doc.Action != "status" || doc.Plan != "p1" || len(doc.Status.Entries) != 1 || doc.Status.Entries[0].Name != "s1" {
		t.Errorf("unexpected document: %+v", doc)
	}
}

func TestStatusAllReachesWholeStackPastDefaultPlan(t *testing.T) {
	useConfig(t, defaultPlanStatusConfig)
	setJSONOutput(t)
	setStatusAll(t)

	var err error
	out := captureStdout(t, func() { err = statusCmd.RunE(statusCmd, nil) })
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Stack []struct {
			Name string `json:"name"`
		} `json:"stack"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, out)
	}
	if len(doc.Stack) != 2 {
		t.Errorf("--all must report both entries, got: %s", out)
	}
}

func TestStatusAllRejectsName(t *testing.T) {
	useConfig(t, defaultPlanStatusConfig)
	setStatusAll(t)

	if err := statusCmd.RunE(statusCmd, []string{"p1"}); err == nil {
		t.Fatal("--all with a NAME must fail, not silently drop one of them")
	}
}
