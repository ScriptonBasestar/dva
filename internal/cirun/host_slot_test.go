//go:build darwin || linux || freebsd || netbsd || openbsd

package cirun

import (
	"bufio"
	"bytes"
	"context"
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

const hostSlotHelperEnv = "DVA_CIRUN_HOST_SLOT"
const hostSlotHolderID = "aabbccddeeff00112233445566778899"
const hostSlotSecondID = "00112233445566778899aabbccddeeff"

// TestHostSlotHelperProcess holds the real host slot file in a fresh process.
// A separate process exercises the same OS admission boundary as
// independently invoked dva runs on the same host.
func TestHostSlotHelperProcess(t *testing.T) {
	if os.Getenv(hostSlotHelperEnv) != "hold" {
		return
	}
	meta := hostSlotMetadata{
		RunID:   os.Getenv("DVA_CIRUN_ID"),
		PID:     os.Getpid(),
		Root:    os.Getenv("DVA_CIRUN_ROOT"),
		Profile: "full",
		Step:    os.Getenv("DVA_CIRUN_STEP"),
	}
	release, _, err := acquireHostSlot(context.Background(), os.Getenv("DVA_CIRUN_STATE"), meta, meta.Step, os.Stdout, nil)
	if err != nil {
		fmt.Fprintln(os.Stdout, "ERROR", err)
		os.Exit(2)
	}
	// A waiting helper announces contention before READY: acquisition blocks,
	// so READY itself is the proof the slot was granted.
	fmt.Fprintln(os.Stdout, "READY")
	// FD 3 is a one-way release pipe from the parent test. It gives the tests
	// a readiness/release handshake without depending on scheduler timing.
	_, _ = io.ReadAll(os.NewFile(3, "release"))
	release()
	os.Exit(0) // avoid the testing package writing PASS into a closed ready pipe.
}

type hostSlotHolder struct {
	cmd     *exec.Cmd
	release *os.File
	pipe    *os.File
	reader  *bufio.Reader
	lines   []string
}

func startHostSlotHolder(t *testing.T, state, id, step string) *hostSlotHolder {
	t.Helper()
	releaseR, releaseW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	readyR, readyW, err := os.Pipe()
	if err != nil {
		_ = releaseR.Close()
		_ = releaseW.Close()
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestHostSlotHelperProcess$")
	cmd.ExtraFiles = []*os.File{releaseR}
	cmd.Stdout, cmd.Stderr = readyW, os.Stderr
	cmd.Env = append(os.Environ(),
		hostSlotHelperEnv+"=hold",
		"DVA_CIRUN_STATE="+state,
		"DVA_CIRUN_ID="+id,
		"DVA_CIRUN_STEP="+step,
		"DVA_CIRUN_ROOT="+t.TempDir(),
	)
	if err := cmd.Start(); err != nil {
		_ = releaseR.Close()
		_ = releaseW.Close()
		_ = readyR.Close()
		_ = readyW.Close()
		t.Fatal(err)
	}
	_ = releaseR.Close()
	_ = readyW.Close()
	h := &hostSlotHolder{cmd: cmd, release: releaseW, pipe: readyR, reader: bufio.NewReader(readyR)}
	t.Cleanup(func() {
		if h.release != nil {
			_ = h.release.Close()
		}
		_ = h.cmd.Wait()
	})
	return h
}

func (h *hostSlotHolder) line(t *testing.T) string {
	t.Helper()
	if err := h.pipe.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		t.Fatalf("holder read deadline: %v", err)
	}
	raw, err := h.reader.ReadString('\n')
	if err != nil {
		t.Fatalf("holder output ended early: %v collected=%q", err, h.lines)
	}
	trimmed := strings.TrimSpace(raw)
	h.lines = append(h.lines, trimmed)
	return trimmed
}

func (h *hostSlotHolder) ready(t *testing.T) {
	t.Helper()
	for {
		switch line := h.line(t); {
		case line == "READY":
			return
		case strings.HasPrefix(line, "ERROR"):
			t.Fatalf("host slot holder failed: %s", line)
		}
	}
}

func (h *hostSlotHolder) stop(t *testing.T) {
	t.Helper()
	if h.release == nil {
		return
	}
	if err := h.release.Close(); err != nil {
		t.Fatal(err)
	}
	h.release = nil
	if err := h.cmd.Wait(); err != nil {
		t.Fatalf("release holder: %v", err)
	}
}

// A configured slot step must queue behind a real holder process, report the
// holder identity it waited for, and record how long it waited.
func TestHostSlotStepWaitsForRealHolderProcess(t *testing.T) {
	t.Setenv(config.CIParentRunEnv, "")
	state, root := t.TempDir(), t.TempDir()
	holder := startHostSlotHolder(t, state, hostSlotHolderID, "heavy-suite")
	holder.ready(t)
	marker := filepath.Join(root, "executed")
	output := &bytes.Buffer{}
	done := make(chan struct{})
	var report Report
	var runErr error
	go func() {
		defer close(done)
		report, runErr = Run(context.Background(), Options{
			Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, Output: output,
			Env: os.Environ(), Profile: profile(config.CIStep{Name: "heavy", Run: "touch " + marker, HostSlot: true}),
		})
	}()
	// While the holder keeps the slot, the step must not execute.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(marker); err == nil {
			t.Fatal("slot step executed while another process held the host slot")
		}
		time.Sleep(25 * time.Millisecond)
	}
	holder.stop(t)
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("slot step did not finish after the holder released")
	}
	if runErr != nil || report.Status != "succeeded" {
		t.Fatalf("report=%+v err=%v", report, runErr)
	}
	if len(report.Steps) != 1 || report.Steps[0].HostSlotWait <= 0 {
		t.Fatalf("host_slot_wait not recorded: %+v", report.Steps)
	}
	log := output.String()
	if !strings.Contains(log, "host_slot=waiting holder run="+hostSlotHolderID) {
		t.Fatalf("log does not name the holder: %q", log)
	}
	if !strings.Contains(log, "host_slot=acquired waited=") {
		t.Fatalf("log does not record acquisition: %q", log)
	}
	if !strings.Contains(log, "host_slot=released") {
		t.Fatalf("log does not record release: %q", log)
	}
}

// The slot wait must not consume the declared step timeout. A conditioned
// step arms a wrapper deadline over the whole pipeline (condition + step), so
// this shape is where a wait charged to the armed context would time out: the
// holder keeps the slot longer than the step budget, then the step runs for a
// duration that fits the budget only if the wait was excluded.
func TestHostSlotWaitExcludedFromStepTimeout(t *testing.T) {
	t.Setenv(config.CIParentRunEnv, "")
	state, root := t.TempDir(), t.TempDir()
	holder := startHostSlotHolder(t, state, hostSlotHolderID, "heavy-suite")
	holder.ready(t)
	output := &bytes.Buffer{}
	done := make(chan struct{})
	var report Report
	var runErr error
	go func() {
		defer close(done)
		report, runErr = Run(context.Background(), Options{
			Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, Output: output,
			Env: os.Environ(),
			Profile: config.CIProfile{Timeout: "10s", MaxParallel: 1, Steps: []config.CIStep{
				{Name: "heavy", Run: "sleep 0.5", Timeout: "1s", HostSlot: true,
					Condition: &config.CICondition{Run: "printf '{\"selected\": true}'", Timeout: "1s"}},
			}},
		})
	}()
	time.Sleep(2 * time.Second)
	holder.stop(t)
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("slot step did not finish after the holder released")
	}
	if runErr != nil || report.Status != "succeeded" {
		t.Fatalf("serialized step timed out: status=%s err=%v log=%q", report.Status, runErr, output.String())
	}
	step := report.Steps[0]
	if step.Status != "succeeded" || step.HostSlotWait < 1500*time.Millisecond {
		t.Fatalf("step=%+v, want a recorded wait of at least 1.5s", step)
	}
}

// Reviewer reproduction: on the reuse path the wrapper deadline is armed
// before the slot wait. A wait longer than the step timeout must not leave
// the post-execution checks reading an expired wrapper, misreporting a
// successful serialized step as timed_out.
func TestHostSlotReuseWaitExcludedFromStepTimeout(t *testing.T) {
	t.Setenv(config.CIParentRunEnv, "")
	state, root := t.TempDir(), t.TempDir()
	if err := runGit(root, "init"); err != nil {
		t.Skipf("git unavailable: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "input"), []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("count\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	commitAll(t, root)
	holder := startHostSlotHolder(t, state, hostSlotHolderID, "heavy-suite")
	holder.ready(t)
	output := &bytes.Buffer{}
	done := make(chan struct{})
	var report Report
	var runErr error
	go func() {
		defer close(done)
		report, runErr = Run(context.Background(), Options{
			Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, Output: output,
			Env: os.Environ(),
			Profile: config.CIProfile{Timeout: "10s", MaxParallel: 1, Steps: []config.CIStep{
				{Name: "heavy", Run: "printf x >> count", Timeout: "1s", HostSlot: true,
					Reuse: &config.CIReuse{Inputs: []string{"input"}, Tools: []string{"sh"}, Outputs: []string{"count"}, MaxAge: "1h"}},
			}},
		})
	}()
	time.Sleep(2 * time.Second)
	holder.stop(t)
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("slot step did not finish after the holder released")
	}
	if runErr != nil || report.Status != "succeeded" {
		t.Fatalf("serialized reuse step misreported: status=%s err=%v log=%q", report.Status, runErr, output.String())
	}
	step := report.Steps[0]
	if step.Status != "succeeded" || step.HostSlotWait < 1300*time.Millisecond {
		t.Fatalf("step=%+v, want a recorded wait of at least 1.3s", step)
	}
	if step.Reuse == nil || step.Reuse.Disposition != "executed" {
		t.Fatalf("reuse disposition not reported: %+v", step.Reuse)
	}
}

// The exclusion must not disable enforcement: after a wait excluded from the
// budget, a command exceeding the re-armed step timeout still times out
// through the reuse path.
func TestHostSlotReuseExecutionDeadlineStillBites(t *testing.T) {
	t.Setenv(config.CIParentRunEnv, "")
	state, root := t.TempDir(), t.TempDir()
	if err := runGit(root, "init"); err != nil {
		t.Skipf("git unavailable: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "input"), []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	commitAll(t, root)
	holder := startHostSlotHolder(t, state, hostSlotHolderID, "heavy-suite")
	holder.ready(t)
	output := &bytes.Buffer{}
	done := make(chan struct{})
	var report Report
	var runErr error
	go func() {
		defer close(done)
		report, runErr = Run(context.Background(), Options{
			Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, Output: output,
			Env: os.Environ(),
			Profile: config.CIProfile{Timeout: "10s", MaxParallel: 1, Steps: []config.CIStep{
				{Name: "heavy", Run: "sleep 2", Timeout: "1s", HostSlot: true,
					Reuse: &config.CIReuse{Inputs: []string{"input"}, Tools: []string{"sh"}, MaxAge: "1h"}},
			}},
		})
	}()
	time.Sleep(600 * time.Millisecond)
	holder.stop(t)
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("slot step did not finish after the holder released")
	}
	if runErr == nil || report.Status == "succeeded" {
		t.Fatalf("overrunning reuse step was not stopped: status=%s err=%v", report.Status, runErr)
	}
	step := report.Steps[0]
	if step.Status != "timed_out" || step.HostSlotWait <= 0 {
		t.Fatalf("step=%+v, want timed_out after a recorded wait", step)
	}
}

// A failing slot step must release the slot so the next run is admitted.
func TestHostSlotReleasedAfterStepFailure(t *testing.T) {
	t.Setenv(config.CIParentRunEnv, "")
	state, root := t.TempDir(), t.TempDir()
	_, err := Run(context.Background(), Options{
		Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, Env: os.Environ(),
		Profile: profile(config.CIStep{Name: "heavy", Run: "exit 1", HostSlot: true}),
	})
	if err == nil {
		t.Fatal("failing slot step unexpectedly succeeded")
	}
	release, waited, err := acquireHostSlot(context.Background(), state, hostSlotMetadata{RunID: hostSlotHolderID, PID: os.Getpid(), Step: "probe"}, "probe", io.Discard, nil)
	if err != nil {
		t.Fatalf("failed step retained the host slot: %v", err)
	}
	if waited > 500*time.Millisecond {
		t.Fatalf("next acquisition waited %s after slot step failure", waited)
	}
	release()
}

// Unmarked steps never touch the slot: they run while a holder keeps it.
func TestHostSlotNotTakenByUnmarkedSteps(t *testing.T) {
	t.Setenv(config.CIParentRunEnv, "")
	state, root := t.TempDir(), t.TempDir()
	holder := startHostSlotHolder(t, state, hostSlotHolderID, "heavy-suite")
	holder.ready(t)
	defer holder.stop(t)
	marker := filepath.Join(root, "executed")
	output := &bytes.Buffer{}
	report, err := Run(context.Background(), Options{
		Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, Output: output,
		Env: os.Environ(), Profile: profile(config.CIStep{Name: "light", Run: "touch " + marker}),
	})
	if err != nil || report.Status != "succeeded" {
		t.Fatalf("unmarked step blocked by the host slot: status=%s err=%v", report.Status, err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("unmarked step did not execute: %v", err)
	}
	if strings.Contains(output.String(), "host_slot") {
		t.Fatalf("unmarked step emitted host slot logs: %q", output.String())
	}
	if report.Steps[0].HostSlotWait != 0 {
		t.Fatalf("unmarked step recorded a slot wait: %+v", report.Steps[0])
	}
}

// A composed child whose ancestor holds the slot must fail fast instead of
// queueing behind its own parent until the deadline.
func TestHostSlotAncestorHolderFailsFast(t *testing.T) {
	state := t.TempDir()
	holder := startHostSlotHolder(t, state, hostSlotHolderID, "outer-heavy")
	holder.ready(t)
	defer holder.stop(t)
	_, _, err := acquireHostSlot(context.Background(), state, hostSlotMetadata{RunID: hostSlotSecondID, PID: os.Getpid(), Step: "inner-heavy"}, "inner-heavy", io.Discard,
		[]compositionFrame{{RunID: hostSlotHolderID, Root: t.TempDir(), Step: "outer-heavy"}})
	if err == nil || !strings.Contains(err.Error(), "ancestor run "+hostSlotHolderID) {
		t.Fatalf("error = %v, want ancestor holder rejection", err)
	}
}

// Two real processes contend: the second holder process queues behind the
// first — announcing whom it waits for — and acquires only after the first
// releases the OS lock.
func TestHostSlotTwoProcessAcquisition(t *testing.T) {
	state := t.TempDir()
	first := startHostSlotHolder(t, state, hostSlotHolderID, "heavy-a")
	first.ready(t)
	second := startHostSlotHolder(t, state, hostSlotSecondID, "heavy-b")
	if line := second.line(t); !strings.Contains(line, "host_slot=waiting holder run="+hostSlotHolderID) {
		t.Fatalf("second holder did not report waiting behind the first: %q", line)
	}
	first.stop(t)
	// READY only prints after acquisition, so reading through READY proves the
	// handoff; the acquired line in between proves this process took the slot.
	second.ready(t)
	if !strings.Contains(strings.Join(second.lines, "\n"), "host_slot=acquired") {
		t.Fatalf("second holder never logged acquisition: %q", second.lines)
	}
	second.stop(t)
}
