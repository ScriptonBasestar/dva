package cirun

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func TestWorkerBudgetLimitsAndExplicitValue(t *testing.T) {
	for _, tc := range []struct{ cpu, parallel, steps, want int }{
		{1, 1, 1, 1}, {2, 1, 1, 2}, {8, 1, 1, 4},
		{8, 2, 3, 2}, {8, 8, 2, 2}, {2, 4, 4, 1}, {8, 3, 3, 1},
	} {
		p := config.CIProfile{MaxParallel: tc.parallel, Steps: make([]config.CIStep, tc.steps)}
		got, _ := toolWorkerBudget(p, tc.cpu)
		if got != tc.want {
			t.Fatalf("%+v: got %d", tc, got)
		}
		jobs := 12
		p.ToolJobs = &jobs
		if got, source := toolWorkerBudget(p, tc.cpu); got != 12 || source != "profile.tool_jobs" {
			t.Fatalf("explicit got %d %s", got, source)
		}
	}
}

func TestWorkersChildPreviewLogAndReceiptAgree(t *testing.T) {
	for _, tc := range []struct {
		name                    string
		base                    []string
		step                    map[string]string
		cargo, rust             int
		cargoSource, rustSource string
	}{
		{"defaults", nil, nil, 4, 1, "profile.tool_jobs", "default"},
		{"inherited", []string{"CARGO_BUILD_JOBS=9", "RUST_TEST_THREADS=6", "DVA_CI_JOBS=99"}, nil, 9, 6, "inherited.environment", "inherited.environment"},
		{"step", []string{"CARGO_BUILD_JOBS=9", "RUST_TEST_THREADS=6"}, map[string]string{"CARGO_BUILD_JOBS": "7", "RUST_TEST_THREADS": "3", "DVA_CI_JOBS": "99"}, 7, 3, "step.environment", "step.environment"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			jobs := 4
			p := profile(config.CIStep{Name: "workers", Run: `printf '%s/%s/%s/%s/%s' "$CARGO_BUILD_JOBS" "$RUST_TEST_THREADS" "$DVA_CI_JOBS" "$GOMAXPROCS" "$GOFLAGS" > observed`, Environment: tc.step})
			p.ToolJobs = &jobs
			preview := WorkerPlan(p, tc.base)[0].WorkerBudget
			if preview == nil || preview.CargoBuildJobs != tc.cargo || preview.RustTestThreads != tc.rust || preview.CargoBuildJobsSource != tc.cargoSource || preview.RustTestThreadsSource != tc.rustSource {
				t.Fatalf("preview %+v", preview)
			}
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			var log bytes.Buffer
			var checkpoints []StepReport
			reports, err := execute(context.Background(), root, p, tc.base, &log, func(steps []StepReport) error {
				if len(steps) != 1 || !reflect.DeepEqual(steps[0].WorkerBudget, preview) {
					return fmt.Errorf("checkpoint worker budget %#v, want %#v", steps, preview)
				}
				checkpoints = append(checkpoints, steps[0])
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(checkpoints) != 2 || checkpoints[0].Status != "running" || checkpoints[1].Status != "succeeded" {
				t.Fatalf("checkpoints %#v", checkpoints)
			}
			if !reflect.DeepEqual(reports[0].WorkerBudget, preview) {
				t.Fatalf("receipt %+v preview %+v", reports[0].WorkerBudget, preview)
			}
			actual, err := os.ReadFile(filepath.Join(root, "observed"))
			if err != nil {
				t.Fatal(err)
			}
			if want := strconv.Itoa(tc.cargo) + "/" + strconv.Itoa(tc.rust) + "/4/1/-p=1"; string(actual) != want {
				t.Fatalf("child %q want %q", actual, want)
			}
			for _, fragment := range []string{"tool_jobs=4", "cargo_build_jobs=" + strconv.Itoa(tc.cargo), "rust_test_threads=" + strconv.Itoa(tc.rust), "dva_ci_jobs=4", "source=" + tc.cargoSource} {
				if !strings.Contains(log.String(), fragment) {
					t.Fatalf("log missing %q: %s", fragment, log.String())
				}
			}
			state := t.TempDir()
			if err := writeReport(state, Report{ID: "workers", Status: "succeeded", Steps: reports}); err != nil {
				t.Fatal(err)
			}
			stored, err := Status(state)
			if err != nil || len(stored) != 1 || !reflect.DeepEqual(stored[0].Steps[0].WorkerBudget, preview) {
				t.Fatalf("stored %+v error %v", stored, err)
			}
		})
	}
}

func TestLegacyReceiptWithoutWorkerBudget(t *testing.T) {
	var r Report
	if err := json.Unmarshal([]byte(`{"id":"old","status":"succeeded","duration":"0s","steps":[{"name":"test","status":"succeeded"}]}`), &r); err != nil {
		t.Fatal(err)
	}
	if r.Steps[0].WorkerBudget != nil {
		t.Fatal("legacy receipt gained worker evidence")
	}
}

func TestLegacyNumericWorkerBudgetDoesNotGainNewFields(t *testing.T) {
	var r Report
	if err := json.Unmarshal([]byte(`{"id":"old","status":"succeeded","duration":"0s","steps":[{"name":"test","status":"succeeded","worker_budget":{"dva_ci_jobs":1,"cargo_build_jobs":1,"rust_test_threads":1}}]}`), &r); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"tool_jobs", "cargo_build_jobs_source", "rust_test_threads_source"} {
		if strings.Contains(string(encoded), key) {
			t.Fatalf("legacy receipt gained %s: %s", key, encoded)
		}
	}
}
