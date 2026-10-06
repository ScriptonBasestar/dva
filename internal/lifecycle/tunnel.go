package lifecycle

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// cloudflared access tunnel support (docs/68, TASK-459).
//
// DVA opens the tunnel declared on a remote kubectl/helm entry before the
// entry runs, waits until it is authenticated and forwarding, runs the entry,
// and closes only the cloudflared process it started itself. A tunnel someone
// else opened — a human or another dva command — is never reused: the port
// conflict check fails closed instead.

const tunnelReadyPollInterval = 250 * time.Millisecond

// tunnelKey dedupes declarations: entries naming the same
// (provider, hostname, local) share one tunnel within a single command.
func tunnelKey(t *config.TunnelConfig) string {
	return strings.Join([]string{t.Provider, t.Hostname, t.Local}, "|")
}

// tunnelTTYAvailable decides whether interactive login can run in the
// foreground. A package var so tests can simulate a non-TTY agent context.
var tunnelTTYAvailable = func() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// tunnelManager owns the tunnels opened during one dva command.
type tunnelManager struct {
	open   map[string]*tunnelProcess
	logger *slog.Logger
}

// tunnelProcess is one cloudflared child DVA started.
type tunnelProcess struct {
	cmd        *exec.Cmd
	pid        int
	stderrTail *tailBuffer
}

func newTunnelManager(logger *slog.Logger) *tunnelManager {
	return &tunnelManager{open: map[string]*tunnelProcess{}, logger: logger}
}

// acquire opens entry.Tunnel unless this manager already holds it. On any
// failure it tears down what it started so a half-open tunnel never outlives
// the call.
func (m *tunnelManager) acquire(entry *config.LifecycleEntry, env *config.Environment) error {
	t := entry.Tunnel
	if t == nil {
		return nil
	}
	key := tunnelKey(t)
	if _, held := m.open[key]; held {
		return nil
	}

	// Interpolate ${...} in the declared hostname and local address the same
	// way runner configs interpolate, so shared declarations stay shared.
	t = &config.TunnelConfig{
		Provider:        t.Provider,
		Hostname:        env.Interpolate(t.Hostname),
		Local:           env.Interpolate(t.Local),
		Auth:            t.Auth,
		ServiceTokenEnv: t.ServiceTokenEnv,
		ReadyTimeout:    t.ReadyTimeout,
	}

	if err := m.authenticate(t); err != nil {
		return err
	}

	// A port already serving is never reused: the owner is unknown, so a
	// successful dial is reported as a conflict and fails closed (docs/68 §5).
	if conn, err := net.DialTimeout("tcp", t.Local, time.Second); err == nil {
		_ = conn.Close()
		return fmt.Errorf("tunnel: local address %s is already in use; not reusing an unowned listener", t.Local)
	}

	proc, err := m.startCloudflared(t, env)
	if err != nil {
		return err
	}

	timeout, err := t.ReadyTimeoutDuration()
	if err != nil {
		_ = proc.terminate()
		return err
	}
	if err := waitForTunnelReady(t.Local, timeout, proc); err != nil {
		_ = proc.terminate()
		return err
	}

	m.open[key] = proc
	m.logger.Info("tunnel ready", "entry", entry.Name, "hostname", t.Hostname, "local", t.Local)
	return nil
}

// Close terminates only the cloudflared processes this manager started.
func (m *tunnelManager) Close() {
	for key, proc := range m.open {
		_ = proc.terminate()
		delete(m.open, key)
	}
}

// authenticate runs the docs/68 §3 checks for the declared auth mode before
// any cloudflared tunnel process is started. Output that may contain a JWT is
// captured and discarded — it is never printed or logged.
func (m *tunnelManager) authenticate(t *config.TunnelConfig) error {
	if t.AuthMode() == config.TunnelAuthServiceToken {
		for _, name := range []string{t.ServiceTokenEnv.ID, t.ServiceTokenEnv.Secret} {
			if os.Getenv(name) == "" {
				return fmt.Errorf("tunnel: environment variable %s is empty or unset; set it before running this command", name)
			}
		}
		return nil
	}

	// interactive: a usable token exits 0 and writes at least one stdout byte.
	// An expired cloudflared token exits 0 and writes nothing (docs/68 §7).
	if err := runCloudflaredDiscard(t.Hostname); err == nil {
		return nil
	}

	// No usable token. With a TTY, the login runs in the foreground and the
	// user completes it in a browser; without one (CI, agents), fail with the
	// command to run.
	if !tunnelTTYAvailable() {
		return fmt.Errorf(
			"tunnel: not authenticated for %s and no terminal is attached; run: cloudflared access login --quiet https://%s",
			t.Hostname, t.Hostname)
	}
	if err := runCloudflaredLogin(t.Hostname); err != nil {
		return fmt.Errorf("tunnel: login for %s failed: %w", t.Hostname, err)
	}
	if err := runCloudflaredDiscard(t.Hostname); err != nil {
		return fmt.Errorf("tunnel: still not authenticated for %s after login", t.Hostname)
	}
	return nil
}

// runCloudflaredDiscard runs `cloudflared access token --app`. Stdout carries
// the JWT, so the writer counts bytes and retains none of them: the token is
// never stored or logged, and whitespace is not trimmed. Authenticated means
// the process succeeded and the count is greater than zero.
func runCloudflaredDiscard(hostname string) error {
	cmd := exec.Command("cloudflared", "access", "token", "--app", "https://"+hostname)
	var stdout stdoutByteCounter
	cmd.Stdout = &stdout
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return err
	}
	if stdout.n == 0 {
		return fmt.Errorf("cloudflared access token exited 0 with empty stdout")
	}
	return nil
}

// stdoutByteCounter counts bytes written to it and keeps none of the contents.
type stdoutByteCounter struct{ n int64 }

func (c *stdoutByteCounter) Write(p []byte) (int, error) {
	c.n += int64(len(p))
	return len(p), nil
}

// runCloudflaredLogin runs `cloudflared access login` in the foreground with
// inherited stdio so the browser flow works.
func runCloudflaredLogin(hostname string) error {
	cmd := exec.Command("cloudflared", "access", "login", "--quiet", "https://"+hostname)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// startCloudflared launches `cloudflared access tcp` in its own process group,
// forwarding local → hostname. Service-token credentials travel only in the
// child's environment.
func (m *tunnelManager) startCloudflared(t *config.TunnelConfig, env *config.Environment) (*tunnelProcess, error) {
	cmd := exec.Command("cloudflared", "access", "tcp", "--hostname", t.Hostname, "--url", t.Local)
	if err := configureProcessGroup(cmd); err != nil {
		return nil, fmt.Errorf("tunnel: %w", err)
	}

	cmd.Env = env.EnvSlice()
	if t.AuthMode() == config.TunnelAuthServiceToken {
		cmd.Env = append(cmd.Env,
			"TUNNEL_SERVICE_TOKEN_ID="+os.Getenv(t.ServiceTokenEnv.ID),
			"TUNNEL_SERVICE_TOKEN_SECRET="+os.Getenv(t.ServiceTokenEnv.Secret))
	}
	cmd.Stdout = io.Discard

	tail := &tailBuffer{max: 4096}
	cmd.Stderr = tail

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("tunnel: starting cloudflared: %w", err)
	}
	return &tunnelProcess{cmd: cmd, pid: cmd.Process.Pid, stderrTail: tail}, nil
}

// waitForTunnelReady polls the local listener until it accepts connections.
// TCP alone is not readiness — the caller has already established the auth
// condition before the process was started.
func waitForTunnelReady(local string, timeout time.Duration, proc *tunnelProcess) error {
	deadline := time.Now().Add(timeout)
	for {
		conn, err := net.DialTimeout("tcp", local, tunnelReadyPollInterval)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("tunnel: %s not ready within %s; cloudflared last said: %s",
				local, timeout, proc.stderrTail.tail())
		}
		time.Sleep(tunnelReadyPollInterval)
	}
}

// terminate signals the child's whole process group and reaps it.
func (p *tunnelProcess) terminate() error {
	if signalableProcessGroupPID(p.pid) {
		_ = terminateProcessGroup(p.pid)
	}
	return p.cmd.Wait()
}

// tailBuffer keeps the last max bytes of cloudflared stderr so a readiness
// timeout can quote the operator's own last line as the cause.
type tailBuffer struct {
	buf []byte
	max int
}

func (b *tailBuffer) Write(p []byte) (int, error) {
	b.buf = append(b.buf, p...)
	if len(b.buf) > b.max {
		b.buf = append([]byte(nil), b.buf[len(b.buf)-b.max:]...)
	}
	return len(p), nil
}

// tail returns the last non-empty line, trimmed; empty when nothing arrived.
func (b *tailBuffer) tail() string {
	lines := strings.Split(strings.TrimRight(string(b.buf), "\n"), "\n")
	for _, line := range slices.Backward(lines) {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}
