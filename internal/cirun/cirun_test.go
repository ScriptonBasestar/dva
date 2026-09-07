package cirun

import (
	"context"
	"errors"
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
	if err != nil || string(log) != "firstsecond" {
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

func TestLockReportsActiveRun(t *testing.T) {
	state, root := t.TempDir(), t.TempDir()
	h, err := acquire(state, root, "aabb")
	if err != nil {
		t.Fatal(err)
	}
	defer h.release()
	_, err = acquire(state, root, "ccdd")
	if !errors.Is(err, ErrBusy) || !strings.Contains(err.Error(), "aabb") {
		t.Fatalf("lock err %v", err)
	}
}

func TestLockRejectsAnotherProcess(t *testing.T) {
	state, root := t.TempDir(), t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=TestLockHelperProcess")
	cmd.Env = append(os.Environ(), "DVA_CIRUN_HELPER=1", "DVA_CIRUN_STATE="+state, "DVA_CIRUN_ROOT="+root)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer cmd.Wait()
	deadline := time.Now().Add(time.Second)
	for {
		h, err := acquire(state, root, "parent")
		if errors.Is(err, ErrBusy) {
			return
		}
		if err == nil {
			h.release()
		}
		if time.Now().After(deadline) {
			t.Fatalf("child never held lock: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestLockHelperProcess(t *testing.T) {
	if os.Getenv("DVA_CIRUN_HELPER") != "1" {
		return
	}
	h, err := acquire(os.Getenv("DVA_CIRUN_STATE"), os.Getenv("DVA_CIRUN_ROOT"), "child")
	if err != nil {
		os.Exit(2)
	}
	time.Sleep(500 * time.Millisecond)
	h.release()
	os.Exit(0)
}

func runGit(dir string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...).Run()
}
