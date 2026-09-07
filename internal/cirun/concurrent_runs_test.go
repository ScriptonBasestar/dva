package cirun

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

const concurrentModeEnv = "DVA_CIRUN_CONCURRENT_MODE"

// TestConcurrentRunSupervisor is a fresh process which owns a real Run call.
// Its CI step starts another fresh test-binary process; the step only reports
// ready after Run has obtained its locks and reached execute.
func TestConcurrentRunSupervisor(t *testing.T) {
	if os.Getenv(concurrentModeEnv) != "supervisor" {
		return
	}
	if err := os.Unsetenv(config.CIParentRunEnv); err != nil {
		os.Exit(10)
	}
	root, state := os.Getenv("DVA_CIRUN_CONCURRENT_ROOT"), os.Getenv("DVA_CIRUN_CONCURRENT_STATE")
	step := concurrentModeEnv + "=step " + shellQuote(os.Args[0]) + " -test.run='^TestConcurrentRunStep$'"
	p := config.CIProfile{Timeout: "20s", MaxParallel: 1, Steps: []config.CIStep{{Name: "hold", Run: step}}}
	if resource := os.Getenv("DVA_CIRUN_CONCURRENT_RESOURCE"); resource != "" {
		p.Locks = []string{resource}
	}
	if _, err := Run(context.Background(), Options{Root: root, ProfileName: os.Getenv("DVA_CIRUN_CONCURRENT_NAME"), StateDir: state, lockDirectory: state, Env: os.Environ(), Profile: p}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(11)
	}
	os.Exit(0)
}

func TestConcurrentRunStep(t *testing.T) {
	if os.Getenv(concurrentModeEnv) != "step" {
		return
	}
	// This Run observes the parent token injected by the supervisor's runStep.
	// It must fail as nested before it can run its own check command.
	if nestedRoot := os.Getenv("DVA_CIRUN_CONCURRENT_NESTED_ROOT"); nestedRoot != "" {
		_, err := Run(context.Background(), Options{Root: nestedRoot, ProfileName: "nested", StateDir: os.Getenv("DVA_CIRUN_CONCURRENT_STATE"), lockDirectory: os.Getenv("DVA_CIRUN_CONCURRENT_STATE"), Profile: config.CIProfile{Timeout: "5s", MaxParallel: 1, Steps: []config.CIStep{{Name: "never", Run: "exit 0"}}}})
		var busy *BusyError
		if !errors.As(err, &busy) || busy.Conflict.Kind != "nested" {
			fmt.Fprintln(os.Stderr, "nested run was not rejected:", err)
			os.Exit(20)
		}
	}
	conn, err := net.DialTimeout("tcp", os.Getenv("DVA_CIRUN_CONCURRENT_ADDR"), 5*time.Second)
	if err != nil {
		os.Exit(21)
	}
	defer conn.Close()
	if _, err := fmt.Fprintln(conn, "ready"); err != nil {
		os.Exit(22)
	}
	_ = conn.SetReadDeadline(time.Now().Add(20 * time.Second))
	_, _ = bufio.NewReader(conn).ReadByte()
	os.Exit(0) // Prevent the testing package from writing after the supervisor closes pipes.
}

func TestConcurrentRealCIProfilesAcrossGitWorktrees(t *testing.T) {
	t.Setenv(config.CIParentRunEnv, "")
	rootA := t.TempDir()
	if err := concurrentGit(rootA, "init"); err != nil {
		t.Skipf("git unavailable: %v", err)
	}
	if err := concurrentGit(rootA, "config", "user.email", "ci@example.invalid"); err != nil {
		t.Fatal(err)
	}
	if err := concurrentGit(rootA, "config", "user.name", "CI Test"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootA, "tracked"), []byte("fixture\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := concurrentGit(rootA, "add", "tracked"); err != nil {
		t.Fatal(err)
	}
	if err := concurrentGit(rootA, "commit", "-m", "fixture"); err != nil {
		t.Fatal(err)
	}
	rootB := filepath.Join(t.TempDir(), "linked-worktree")
	if err := concurrentGit(rootA, "worktree", "add", "--detach", rootB, "HEAD"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = concurrentGit(rootA, "worktree", "remove", "--force", rootB) })

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	state := t.TempDir()
	nestedRoot := t.TempDir()
	first := startConcurrentSupervisor(t, rootA, state, listener.Addr().String(), "first", "alpha", nestedRoot)
	second := startConcurrentSupervisor(t, rootB, state, listener.Addr().String(), "second", "beta", "")
	connections := acceptConcurrentReady(t, listener, 2)
	t.Cleanup(func() {
		for _, conn := range connections {
			_ = conn.Close()
		}
		stopConcurrentSupervisor(first)
		stopConcurrentSupervisor(second)
	})

	reports, err := Status(state)
	if err != nil {
		t.Fatal(err)
	}
	live := 0
	for _, report := range reports {
		if report.Status == "running" {
			live++
		}
	}
	if live != 2 {
		t.Fatalf("live reports=%#v", reports)
	}
	for _, conn := range connections {
		if _, err := conn.Write([]byte("x")); err != nil {
			t.Fatal(err)
		}
		_ = conn.Close()
	}
	waitConcurrentSupervisor(t, first)
	waitConcurrentSupervisor(t, second)
}

func startConcurrentSupervisor(t *testing.T, root, state, addr, name, resource, nestedRoot string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestConcurrentRunSupervisor$")
	cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr
	cmd.Env = append(os.Environ(),
		concurrentModeEnv+"=supervisor",
		"DVA_CIRUN_CONCURRENT_ROOT="+root,
		"DVA_CIRUN_CONCURRENT_STATE="+state,
		"DVA_CIRUN_CONCURRENT_ADDR="+addr,
		"DVA_CIRUN_CONCURRENT_NAME="+name,
		"DVA_CIRUN_CONCURRENT_RESOURCE="+resource,
		"DVA_CIRUN_CONCURRENT_NESTED_ROOT="+nestedRoot,
	)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stopConcurrentSupervisor(cmd) })
	return cmd
}

func acceptConcurrentReady(t *testing.T, listener net.Listener, count int) []net.Conn {
	t.Helper()
	if tcp, ok := listener.(*net.TCPListener); ok {
		if err := tcp.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	connections := make([]net.Conn, 0, count)
	for range count {
		conn, err := listener.Accept()
		if err != nil {
			t.Fatalf("accept ready step: %v", err)
		}
		t.Cleanup(func() { _ = conn.Close() })
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		line, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil || line != "ready\n" {
			_ = conn.Close()
			t.Fatalf("step readiness line=%q err=%v", line, err)
		}
		connections = append(connections, conn)
	}
	return connections
}

func waitConcurrentSupervisor(t *testing.T, cmd *exec.Cmd) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		_ = cmd.Process.Kill()
		<-done
		t.Fatal("CI supervisor did not finish after release")
	}
}

func stopConcurrentSupervisor(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}
}

func concurrentGit(root string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, "git", append([]string{"-c", "commit.gpgsign=false", "-c", "core.hooksPath=/dev/null", "-C", root}, args...)...).Run()
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
