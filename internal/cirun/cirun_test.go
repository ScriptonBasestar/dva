package cirun

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func profile(steps ...config.CIStep) config.CIProfile {
	return config.CIProfile{Timeout: "5s", MaxParallel: 2, Steps: steps}
}

func TestRustWorkerEnvironment(t *testing.T) {
	for _, test := range []struct {
		name string
		base []string
		step map[string]string
		want string
	}{
		{name: "defaults", want: "1"},
		{name: "inherited", base: []string{"CARGO_BUILD_JOBS=2", "RUST_TEST_THREADS=2"}, want: "2"},
		{name: "step override", base: []string{"CARGO_BUILD_JOBS=2", "RUST_TEST_THREADS=2"}, step: map[string]string{"CARGO_BUILD_JOBS": "3", "RUST_TEST_THREADS": "3"}, want: "3"},
	} {
		t.Run(test.name, func(t *testing.T) {
			env := "\n" + strings.Join(ciEnv(test.base, test.step, 1), "\n") + "\n"
			for _, key := range []string{"CARGO_BUILD_JOBS", "RUST_TEST_THREADS"} {
				if !strings.Contains(env, "\n"+key+"="+test.want+"\n") {
					t.Fatalf("missing %s=%s in %s", key, test.want, env)
				}
			}
		})
	}
}

func TestStepWorkerBudgetRecordsOnlyEffectiveNumericLimits(t *testing.T) {
	for _, test := range []struct {
		name   string
		base   []string
		step   map[string]string
		want   *WorkerBudget
		secret string
	}{
		{name: "defaults", want: &WorkerBudget{ToolJobs: 1, Source: "profile.tool_jobs", DVAJobs: 1, CargoBuildJobs: 1, CargoBuildJobsSource: "profile.tool_jobs", RustTestThreads: 1, RustTestThreadsSource: "default"}},
		{name: "inherited", base: []string{"CARGO_BUILD_JOBS=2", "RUST_TEST_THREADS=3"}, want: &WorkerBudget{ToolJobs: 1, Source: "profile.tool_jobs", DVAJobs: 1, CargoBuildJobs: 2, CargoBuildJobsSource: "inherited.environment", RustTestThreads: 3, RustTestThreadsSource: "inherited.environment"}},
		{name: "step override", base: []string{"CARGO_BUILD_JOBS=2", "RUST_TEST_THREADS=3"}, step: map[string]string{"CARGO_BUILD_JOBS": "4", "RUST_TEST_THREADS": "5"}, want: &WorkerBudget{ToolJobs: 1, Source: "profile.tool_jobs", DVAJobs: 1, CargoBuildJobs: 4, CargoBuildJobsSource: "step.environment", RustTestThreads: 5, RustTestThreadsSource: "step.environment"}},
		{name: "invalid omits budget and arbitrary value", step: map[string]string{"CARGO_BUILD_JOBS": "not-a-number", "UNRELATED_SECRET": "do-not-record"}, secret: "do-not-record"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root, state := t.TempDir(), t.TempDir()
			jobs := 1
			p := profile(config.CIStep{Name: "budget", Run: "exit 0", Environment: test.step})
			p.ToolJobs = &jobs
			var output bytes.Buffer
			report, err := Run(context.Background(), Options{
				Root: root, ProfileName: "budget", StateDir: state, lockDirectory: state, Env: test.base,
				Profile: p, Output: &output,
			})
			if err != nil || len(report.Steps) != 1 {
				t.Fatalf("report=%#v err=%v", report, err)
			}
			if got := report.Steps[0].WorkerBudget; test.want != nil && (got == nil || got.DVAJobs != test.want.DVAJobs || got.CargoBuildJobs != test.want.CargoBuildJobs || got.RustTestThreads != test.want.RustTestThreads || got.CargoBuildJobsSource != test.want.CargoBuildJobsSource || got.RustTestThreadsSource != test.want.RustTestThreadsSource) {
				t.Fatalf("worker budget=%#v want=%#v", got, test.want)
			}
			encoded, err := json.Marshal(report)
			if err != nil {
				t.Fatal(err)
			}
			if test.want == nil && strings.Contains(string(encoded), "worker_budget") {
				t.Fatalf("invalid budget was serialized: %s", encoded)
			}
			if test.secret != "" && strings.Contains(string(encoded), test.secret) {
				t.Fatalf("arbitrary environment value leaked: %s", encoded)
			}
			if test.secret != "" && strings.Contains(output.String(), test.secret) {
				t.Fatalf("arbitrary environment value leaked to log: %s", output.String())
			}
		})
	}
}

func TestToolWorkerBudgetCPUAndParallelism(t *testing.T) {
	for _, tc := range []struct{ cpu, parallel, steps, want int }{
		{1, 1, 3, 1}, {2, 2, 3, 1}, {8, 1, 3, 4}, {8, 2, 3, 2},
	} {
		p := config.CIProfile{MaxParallel: tc.parallel}
		for i := 0; i < tc.steps; i++ {
			p.Steps = append(p.Steps, config.CIStep{Name: string(rune('a' + i)), Run: "true"})
		}
		got, _ := toolWorkerBudget(p, tc.cpu)
		if got != tc.want {
			t.Errorf("cpu=%d parallel=%d: got %d, want %d", tc.cpu, tc.parallel, got, tc.want)
		}
	}
}

func TestCanceledPreparationHasTruthfulStatus(t *testing.T) {
	for _, deadline := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		if deadline {
			cancel()
			ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		} else {
			cancel()
		}
		root, state := t.TempDir(), t.TempDir()
		r, err := Run(ctx, Options{Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, Profile: profile(config.CIStep{Name: "never", Run: "touch executed"})})
		cancel()
		want := "canceled"
		if deadline {
			want = "timed_out"
		}
		if err == nil || r.Status != want {
			t.Fatalf("status=%s want=%s error=%v", r.Status, want, err)
		}
		if _, err := os.Stat(filepath.Join(root, "executed")); !os.IsNotExist(err) {
			t.Fatal("canceled preparation ran a step")
		}
	}
}

func TestOldReceiptDoesNotBorrowNewRunLock(t *testing.T) {
	root, state := t.TempDir(), t.TempDir()
	oldID := strings.Repeat("a", 32)
	newID := strings.Repeat("b", 32)
	if err := writeReport(state, Report{ID: oldID, Root: root, Status: "running", StartedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	h, err := acquire(state, root, newID)
	if err != nil {
		t.Fatal(err)
	}
	defer h.release()
	reports, err := Status(state)
	if err != nil || len(reports) != 1 || reports[0].Status != "stale" {
		t.Fatalf("reports=%#v err=%v", reports, err)
	}
}

func TestLockErrorsAreNotContention(t *testing.T) {
	if lockContended(errors.New("unsupported platform")) {
		t.Fatal("ordinary error classified as busy")
	}
}

func TestRunSchedulesDependenciesAndStatusReadsReport(t *testing.T) {
	root := t.TempDir()
	state := t.TempDir()
	r, err := Run(context.Background(), Options{Root: root, ProfileName: "fast", StateDir: state, lockDirectory: state, Env: os.Environ(), Profile: profile(
		config.CIStep{Name: "first", Run: "printf first"},
		config.CIStep{Name: "second", Run: "printf second", DependsOn: []string{"first"}},
	)})
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != "succeeded" {
		t.Fatalf("status %q", r.Status)
	}
	if r.Steps[0].Status != "succeeded" || r.Steps[1].Status != "succeeded" {
		t.Fatalf("steps %#v", r.Steps)
	}
	got, err := Status(state)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != r.ID {
		t.Fatalf("status %#v", got)
	}
	log, err := ReadLog(state, r.ID)
	if err != nil || !strings.Contains(string(log), "first") || !strings.Contains(string(log), "second") || !strings.Contains(string(log), "tool_jobs=") {
		t.Fatalf("log=%q err=%v", log, err)
	}
	if _, err := ReadLog(state, "../bad"); err == nil {
		t.Fatal("unsafe id accepted")
	}
}

func TestRunFailsFastAndCancelsSibling(t *testing.T) {
	root, state := t.TempDir(), t.TempDir()
	r, err := Run(context.Background(), Options{Root: root, ProfileName: "fail", StateDir: state, lockDirectory: state, Env: os.Environ(), Profile: profile(
		config.CIStep{Name: "bad", Run: "exit 4"},
		config.CIStep{Name: "slow", Run: "sleep 5"},
		config.CIStep{Name: "after", Run: "exit 0", DependsOn: []string{"bad"}},
	)})
	if err == nil || r.Status != "failed" {
		t.Fatalf("report=%#v err=%v", r, err)
	}
	if r.Steps[2].Status != "skipped" {
		t.Fatalf("dependent status %q", r.Steps[2].Status)
	}
}

func TestStepTimeoutIsReported(t *testing.T) {
	root, state := t.TempDir(), t.TempDir()
	r, err := Run(context.Background(), Options{Root: root, ProfileName: "timeout", StateDir: state, lockDirectory: state, Env: os.Environ(), Profile: profile(config.CIStep{Name: "slow", Run: "sleep 1", Timeout: "25ms"})})
	if err == nil || r.Status != "failed" || r.Steps[0].Status != "timed_out" {
		t.Fatalf("report=%#v err=%v", r, err)
	}
}

func TestStatusShowsLiveReceiptThenCompletedReport(t *testing.T) {
	root, state := t.TempDir(), t.TempDir()
	done := make(chan error, 1)
	go func() {
		_, err := Run(context.Background(), Options{Root: root, ProfileName: "live", StateDir: state, lockDirectory: state, Env: os.Environ(), Profile: profile(config.CIStep{Name: "wait", Run: "sleep 0.2"})})
		done <- err
	}()
	deadline := time.Now().Add(time.Second)
	for {
		reports, err := Status(state)
		if err != nil {
			t.Fatal(err)
		}
		if len(reports) == 1 && reports[0].Status == "running" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("live receipt not observed: %#v", reports)
		}
		time.Sleep(5 * time.Millisecond)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	reports, err := Status(state)
	if err != nil || reports[0].Status != "succeeded" {
		t.Fatalf("reports=%#v err=%v", reports, err)
	}
}

func TestRunCheckpointsStartedStepBeforeCompletion(t *testing.T) {
	root, state := t.TempDir(), t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	writer := &scopedReadyWriter{ready: make(chan struct{})}
	done := make(chan struct {
		report Report
		err    error
	}, 1)
	go func() {
		report, err := Run(ctx, Options{
			Root: root, ProfileName: "checkpoint", StateDir: state, lockDirectory: state, Output: writer,
			Profile: profile(config.CIStep{
				Name:        "wait",
				Run:         scopedWorkloadCommand(),
				Environment: map[string]string{"DVA_CIRUN_SCOPED_HELPER": "workload"},
			}),
		})
		done <- struct {
			report Report
			err    error
		}{report, err}
	}()
	select {
	case <-writer.ready:
	case <-time.After(5 * time.Second):
		t.Fatal("step did not reach its ready point")
	}
	reports, err := Status(state)
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 1 || reports[0].Status != "running" || len(reports[0].Steps) != 1 || reports[0].Steps[0].Status != "running" || reports[0].Steps[0].StartedAt.IsZero() || !reports[0].Steps[0].FinishedAt.IsZero() {
		t.Fatalf("running receipt %#v", reports)
	}
	startedAt := reports[0].Steps[0].StartedAt
	cancel()
	var result struct {
		report Report
		err    error
	}
	select {
	case result = <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("canceled checkpoint run did not finish")
	}
	if result.err == nil || result.report.Status != "canceled" {
		t.Fatalf("result=%#v", result)
	}
	reports, err = Status(state)
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 1 || reports[0].Status != "canceled" || len(reports[0].Steps) != 1 || reports[0].Steps[0].Status != "canceled" || reports[0].Steps[0].FinishedAt.IsZero() {
		t.Fatalf("completed receipt %#v", reports)
	}
	if !reports[0].Steps[0].StartedAt.Equal(startedAt) {
		t.Fatalf("started_at changed: running=%s completed=%s", startedAt, reports[0].Steps[0].StartedAt)
	}
}

func TestCheckpointFailureCancelsAndJoinsRunningSteps(t *testing.T) {
	root := canonicalScopedRoot(t, t.TempDir())
	writer := &scopedReadyWriter{ready: make(chan struct{})}
	sentinel := errors.New("checkpoint failed")
	started := time.Now()
	reports, err := execute(context.Background(), root, config.CIProfile{
		MaxParallel: 2,
		Steps: []config.CIStep{
			{Name: "complete", Run: "exit 0"},
			{Name: "block", Run: scopedWorkloadCommand(), Environment: map[string]string{"DVA_CIRUN_SCOPED_HELPER": "workload"}},
		},
	}, nil, writer, func(steps []StepReport) error {
		for _, step := range steps {
			if step.Name == "complete" && step.Status == "succeeded" {
				select {
				case <-writer.ready:
					return sentinel
				case <-time.After(5 * time.Second):
					return errors.New("blocking sibling did not reach its ready point")
				}
			}
		}
		return nil
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("error=%v, want checkpoint sentinel", err)
	}
	if elapsed := time.Since(started); elapsed > 5*time.Second {
		t.Fatalf("checkpoint failure did not join sibling promptly: %s", elapsed)
	}
	if len(reports) != 2 || reports[0].Status != "succeeded" || reports[1].Status != "canceled" || reports[1].FinishedAt.IsZero() {
		t.Fatalf("reports=%#v", reports)
	}
}

func TestRunDetectsInputMutationAndCleansBackgroundGroup(t *testing.T) {
	root, state := t.TempDir(), t.TempDir()
	input := filepath.Join(root, "input")
	if err := os.WriteFile(input, []byte("before"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := runGit(root, "init"); err != nil {
		t.Skipf("git unavailable: %v", err)
	}
	r, err := Run(context.Background(), Options{Root: root, ProfileName: "mutate", StateDir: state, lockDirectory: state, Env: os.Environ(), Profile: profile(config.CIStep{Name: "mutate", Run: "printf after > input; sleep 0.2 &"})})
	if err == nil || r.Status != "stale" {
		t.Fatalf("status=%q err=%v", r.Status, err)
	}
	if !r.Attestation.Available || r.Attestation.Before == r.Attestation.After {
		t.Fatalf("attestation %#v", r.Attestation)
	}
}

func TestFingerprintAttestsUninitializedGitlinkIndexRevision(t *testing.T) {
	root := t.TempDir()
	if err := runGit(root, "init"); err != nil {
		t.Skipf("git unavailable: %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, "module"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := runGit(root, "update-index", "--add", "--cacheinfo", "160000,"+strings.Repeat("1", 40)+",module"); err != nil {
		t.Fatal(err)
	}
	before, err := fingerprint(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if err := runGit(root, "update-index", "--cacheinfo", "160000,"+strings.Repeat("2", 40)+",module"); err != nil {
		t.Fatal(err)
	}
	after, err := fingerprint(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !before.Available || before.Before == after.Before {
		t.Fatalf("gitlink index revision was not attested: before=%#v after=%#v", before, after)
	}
}

func TestFingerprintRejectsAmbiguousGitlinkDirectory(t *testing.T) {
	root := t.TempDir()
	if err := runGit(root, "init"); err != nil {
		t.Skipf("git unavailable: %v", err)
	}
	module := filepath.Join(root, "module")
	if err := os.Mkdir(module, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(module, "not-a-repository"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := runGit(root, "update-index", "--add", "--cacheinfo", "160000,"+strings.Repeat("1", 40)+",module"); err != nil {
		t.Fatal(err)
	}
	if _, err := fingerprint(context.Background(), root); err == nil || !strings.Contains(err.Error(), "gitlink module is not an initialized Git working tree") {
		t.Fatalf("ambiguous gitlink directory err=%v", err)
	}
}

func TestFingerprintPreservesRegularAndSymlinkInputs(t *testing.T) {
	root := t.TempDir()
	if err := runGit(root, "init"); err != nil {
		t.Skipf("git unavailable: %v", err)
	}
	regular := filepath.Join(root, "regular")
	link := filepath.Join(root, "link")
	if err := os.WriteFile(regular, []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("regular", link); err != nil {
		t.Fatal(err)
	}
	if err := runGit(root, "add", "regular", "link"); err != nil {
		t.Fatal(err)
	}
	before, err := fingerprint(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(regular, []byte("two"), 0o600); err != nil {
		t.Fatal(err)
	}
	regularChanged, err := fingerprint(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if before.Before == regularChanged.Before {
		t.Fatal("regular file mutation was not attested")
	}
	if err := os.WriteFile(regular, []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("other", link); err != nil {
		t.Fatal(err)
	}
	linkChanged, err := fingerprint(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if before.Before == linkChanged.Before {
		t.Fatal("symlink target mutation was not attested")
	}
}

func TestFingerprintAttestsIndexOnlyTrackedMutation(t *testing.T) {
	root := t.TempDir()
	if err := runGit(root, "init"); err != nil {
		t.Skipf("git unavailable: %v", err)
	}
	input := filepath.Join(root, "input")
	if err := os.WriteFile(input, []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := runGit(root, "add", "input"); err != nil {
		t.Fatal(err)
	}
	before, err := fingerprint(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(input, []byte("staged-only"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := runGit(root, "add", "input"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(input, []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	after, err := fingerprint(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if before.Before == after.Before {
		t.Fatal("index-only tracked mutation was not attested")
	}
}

func TestFingerprintAttestsTabAndNewlinePath(t *testing.T) {
	root := t.TempDir()
	if err := runGit(root, "init"); err != nil {
		t.Skipf("git unavailable: %v", err)
	}
	name := "tab\tand\nnewline"
	input := filepath.Join(root, name)
	if err := os.WriteFile(input, []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := runGit(root, "add", "--", name); err != nil {
		t.Fatal(err)
	}
	before, err := fingerprint(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(input, []byte("two"), 0o600); err != nil {
		t.Fatal(err)
	}
	after, err := fingerprint(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if before.Before == after.Before {
		t.Fatal("tab/newline path mutation was not attested")
	}
}

func TestFingerprintRejectsUnmergedIndexInput(t *testing.T) {
	root := t.TempDir()
	if err := runGit(root, "init"); err != nil {
		t.Skipf("git unavailable: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "input"), []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := runGit(root, "add", "input"); err != nil {
		t.Fatal(err)
	}
	object, err := gitOutput(context.Background(), root, "rev-parse", ":input")
	if err != nil {
		t.Fatal(err)
	}
	if err := runGit(root, "update-index", "--force-remove", "input"); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "-C", root, "update-index", "--index-info")
	cmd.Stdin = strings.NewReader("100644 " + strings.TrimSpace(string(object)) + " 1\tinput\n100644 " + strings.TrimSpace(string(object)) + " 2\tinput\n")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("create unmerged index: %v: %s", err, output)
	}
	if _, err := fingerprint(context.Background(), root); err == nil || !strings.Contains(err.Error(), "cannot attest unmerged git input input") {
		t.Fatalf("unmerged index err=%v", err)
	}
}

func TestFingerprintAttestsMissingAndDeletedTrackedInput(t *testing.T) {
	root := t.TempDir()
	if err := runGit(root, "init"); err != nil {
		t.Skipf("git unavailable: %v", err)
	}
	input := filepath.Join(root, "input")
	if err := os.WriteFile(input, []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := runGit(root, "add", "input"); err != nil {
		t.Fatal(err)
	}
	before, err := fingerprint(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(input); err != nil {
		t.Fatal(err)
	}
	missing, err := fingerprint(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if before.Before == missing.Before {
		t.Fatal("missing tracked input was not attested")
	}
	if err := runGit(root, "rm", "--cached", "--force", "input"); err != nil {
		t.Fatal(err)
	}
	deleted, err := fingerprint(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if missing.Before == deleted.Before {
		t.Fatal("deleted tracked input was not attested")
	}
}

func TestFingerprintAttestsInitializedGitlinkHeadAndWorkingTree(t *testing.T) {
	root := t.TempDir()
	if err := runGit(root, "init"); err != nil {
		t.Skipf("git unavailable: %v", err)
	}
	module := filepath.Join(root, "module")
	if err := os.Mkdir(module, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := runGit(module, "init"); err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(module, "input")
	if err := os.WriteFile(input, []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := runGit(module, "add", "input"); err != nil {
		t.Fatal(err)
	}
	if err := runGit(module, "-c", "user.name=test", "-c", "user.email=test@example.invalid", "commit", "-m", "initial"); err != nil {
		t.Fatal(err)
	}
	head, err := gitOutput(context.Background(), module, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if err := runGit(root, "update-index", "--add", "--cacheinfo", "160000,"+strings.TrimSpace(string(head))+",module"); err != nil {
		t.Fatal(err)
	}
	before, err := fingerprint(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if err := runGit(root, "update-index", "--cacheinfo", "160000,"+strings.Repeat("3", 40)+",module"); err != nil {
		t.Fatal(err)
	}
	indexOnly, err := fingerprint(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if before.Before == indexOnly.Before {
		t.Fatal("initialized gitlink index-only mutation was not attested")
	}
	if err := runGit(root, "update-index", "--cacheinfo", "160000,"+strings.TrimSpace(string(head))+",module"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(input, []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	changed, err := fingerprint(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if before.Before == changed.Before {
		t.Fatal("gitlink working tree mutation was not attested")
	}
	if err := runGit(module, "add", "input"); err != nil {
		t.Fatal(err)
	}
	if err := runGit(module, "-c", "user.name=test", "-c", "user.email=test@example.invalid", "commit", "-m", "changed"); err != nil {
		t.Fatal(err)
	}
	committed, err := fingerprint(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if changed.Before == committed.Before {
		t.Fatal("gitlink HEAD mutation was not attested")
	}
}

func TestFingerprintAttestsNestedGitlinkWorkingTree(t *testing.T) {
	root := t.TempDir()
	if err := runGit(root, "init"); err != nil {
		t.Skipf("git unavailable: %v", err)
	}
	outer := filepath.Join(root, "outer")
	inner := filepath.Join(outer, "inner")
	if err := os.MkdirAll(inner, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := runGit(inner, "init"); err != nil {
		t.Fatal(err)
	}
	innerInput := filepath.Join(inner, "input")
	if err := os.WriteFile(innerInput, []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := runGit(inner, "add", "input"); err != nil {
		t.Fatal(err)
	}
	if err := runGit(inner, "-c", "user.name=test", "-c", "user.email=test@example.invalid", "commit", "-m", "initial"); err != nil {
		t.Fatal(err)
	}
	innerHead, err := gitOutput(context.Background(), inner, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if err := runGit(outer, "init"); err != nil {
		t.Fatal(err)
	}
	if err := runGit(outer, "update-index", "--add", "--cacheinfo", "160000,"+strings.TrimSpace(string(innerHead))+",inner"); err != nil {
		t.Fatal(err)
	}
	if err := runGit(outer, "-c", "user.name=test", "-c", "user.email=test@example.invalid", "commit", "-m", "nested"); err != nil {
		t.Fatal(err)
	}
	outerHead, err := gitOutput(context.Background(), outer, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if err := runGit(root, "update-index", "--add", "--cacheinfo", "160000,"+strings.TrimSpace(string(outerHead))+",outer"); err != nil {
		t.Fatal(err)
	}
	before, err := fingerprint(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(innerInput, []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	after, err := fingerprint(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if before.Before == after.Before {
		t.Fatal("nested gitlink working tree mutation was not attested")
	}
}

func TestLockReportsActiveRun(t *testing.T) {
	state, root := t.TempDir(), t.TempDir()
	h, err := acquire(state, root, strings.Repeat("a", 32))
	if err != nil {
		t.Fatal(err)
	}
	defer h.release()
	_, err = acquire(state, root, strings.Repeat("c", 32))
	if !errors.Is(err, ErrBusy) || !strings.Contains(err.Error(), strings.Repeat("a", 32)) {
		t.Fatalf("lock err %v", err)
	}
}

// TestLockRejectsAnotherProcess synchronizes on the child's report instead of a deadline, so a
// slow child start on a loaded runner cannot fail it (ISSUE-526): the child writes a line once it
// holds the lock and keeps holding it until the parent closes the child's stdin.
func TestLockRejectsAnotherProcess(t *testing.T) {
	state, root := t.TempDir(), t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=^TestLockHelperProcess$")
	cmd.Env = append(os.Environ(), "DVA_CIRUN_HELPER=1", "DVA_CIRUN_STATE="+state, "DVA_CIRUN_ROOT="+root)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	held, readErr := bufio.NewReader(stdout).ReadString('\n')
	if readErr != nil || held != "held\n" {
		_ = stdin.Close()
		waitErr := cmd.Wait()
		t.Fatalf("child did not report holding the lock: %q, %v; wait: %v; stderr: %s", held, readErr, waitErr, stderr.String())
	}
	h, err := acquire(state, root, "parent")
	if err == nil {
		h.release()
	}
	if closeErr := stdin.Close(); closeErr != nil {
		t.Errorf("close child stdin: %v", closeErr)
	}
	if waitErr := cmd.Wait(); waitErr != nil {
		t.Errorf("child exit: %v", waitErr)
	}
	if !errors.Is(err, ErrBusy) {
		t.Fatalf("acquire while child holds lock error = %v, want ErrBusy", err)
	}
}

func TestLockHelperProcess(t *testing.T) {
	if os.Getenv("DVA_CIRUN_HELPER") != "1" {
		return
	}
	h, err := acquire(os.Getenv("DVA_CIRUN_STATE"), os.Getenv("DVA_CIRUN_ROOT"), "child")
	if err != nil {
		fmt.Fprintf(os.Stderr, "child acquire: %v\n", err)
		os.Exit(2)
	}
	if _, err := os.Stdout.WriteString("held\n"); err != nil {
		os.Exit(3)
	}
	// Hold the lock until the parent closes stdin; EOF or any read error ends the hold.
	_, _ = io.Copy(io.Discard, os.Stdin)
	h.release()
	os.Exit(0)
}

func runGit(dir string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...).Run()
}
