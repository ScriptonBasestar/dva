package lifecycle

import (
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// The tunnel tests drive acquire through a fake `cloudflared` shim. Docs/68
// behaviours under test: interactive auth fails closed without a TTY and never
// leaks stdout bytes (§3), readiness requires both auth and a live local
// listener (§4), service-token credentials travel only in the child env (§3),
// and the manager owns — dedupes and tears down — only the processes it
// started itself (§5). Auth is a successful process and a stdout byte count
// above zero. The count is not whitespace-trimmed, and the bytes are not stored.

// tunnelNoTTY pins tunnelTTYAvailable to false for the duration of the test
// and restores the real detector afterwards. The override is required for
// determinism: `go test` run in an interactive terminal has a char-device
// stdin, which would otherwise take the real detector down the TTY path.
func tunnelNoTTY(t *testing.T) {
	t.Helper()

	orig := tunnelTTYAvailable
	tunnelTTYAvailable = func() bool { return false }
	t.Cleanup(func() { tunnelTTYAvailable = orig })
}

// tunnelFreeLocal returns a loopback host:port that is currently free. The
// documented default 127.0.0.1:16443 is deliberately not used here: a
// developer running a real cloudflared access tunnel on this machine (the
// tunnel the feature exists for) occupies exactly that port and would turn
// every success-path test into a false conflict.
func tunnelFreeLocal(t *testing.T) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("find a free loopback port: %v", err)
	}
	defer ln.Close()
	return ln.Addr().String()
}

// tunnelShim installs a fake `cloudflared` first on PATH and returns the shim
// log directory. The shim logs every invocation to $TUNNEL_SHIM_LOG/calls.log
// as "<subcommand> <nanotimestamp>" (the subcommand is $2: DVA always invokes
// `cloudflared access <sub> ...`) and honours:
//
//	TOKEN_EXIT        exit code for `access token` (default 0)
//	TOKEN_STDOUT      stdout of `access token`: synthetic (default, a nonempty
//	                  sentinel that is not a credential), blank (one space),
//	                  or empty (zero bytes). Successful auth fixtures need a
//	                  count above zero. Tests assert the sentinel does not
//	                  reappear in errors.
//	LISTEN            when 1, the tcp subcommand binds 127.0.0.1:$TUNNEL_SHIM_PORT
//	TUNNEL_SHIM_PORT  port the LISTEN listener binds (default 16443)
//
// The LISTEN listener is a nohup'd python3 accept loop, so it outlives the
// shim process the way a real cloudflared forwarder outlives nothing — it
// stays until DVA kills the process group it was started in.
//
// PATH is replaced, not prepended, so a real cloudflared binary on the machine
// cannot decide the result (helm_test.go convention); the script pins its own
// PATH to the system dirs because the test PATH no longer contains them.
func tunnelShim(t *testing.T, local string) string {
	t.Helper()

	dir := t.TempDir()
	_, port, err := net.SplitHostPort(local)
	if err != nil {
		t.Fatalf("split shim local %q: %v", local, err)
	}
	script := `#!/bin/sh
PATH=/usr/bin:/bin:/usr/sbin:/sbin
export PATH
log_dir="$TUNNEL_SHIM_LOG"
mkdir -p "$log_dir"
echo "$2 $(date +%s%N)" >> "$log_dir/calls.log"
case "$2" in
  token)
    case "${TOKEN_STDOUT:-synthetic}" in
      empty)
        ;;
      blank)
        printf ' '
        ;;
      synthetic)
        printf '%s\n' 'synthetic-stdout'
        ;;
      *)
        echo "unknown TOKEN_STDOUT" >&2
        exit 2
        ;;
    esac
    exit "${TOKEN_EXIT:-0}"
    ;;
  login)
    exit 0
    ;;
  tcp)
    {
      echo "id=$TUNNEL_SERVICE_TOKEN_ID"
      echo "secret=$TUNNEL_SERVICE_TOKEN_SECRET"
    } > "$log_dir/service_token.log"
    if [ "$LISTEN" = "1" ]; then
      nohup python3 -c 'import os, socket
s = socket.socket()
s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
s.bind(("127.0.0.1", int(os.environ.get("TUNNEL_SHIM_PORT", "16443"))))
s.listen(8)
while True:
    conn, _ = s.accept()
    conn.close()
' >/dev/null 2>&1 &
      sleep 0.4
    fi
    exit 0
    ;;
  *)
    exit 0
    ;;
esac
`
	if err := os.WriteFile(filepath.Join(dir, "cloudflared"), []byte(script), 0o755); err != nil {
		t.Fatalf("write cloudflared shim: %v", err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("TUNNEL_SHIM_LOG", dir)
	t.Setenv("TUNNEL_SHIM_PORT", port)
	return dir
}

// tunnelEntry builds the stack entry the acquire tests drive, declaring a
// tunnel in the given auth mode forwarding local. Service-token env names
// follow the docs/68 §2 rule: TID/TSECRET name environment variables, they
// never carry the values themselves.
func tunnelEntry(auth, local string) *config.LifecycleEntry {
	tun := &config.TunnelConfig{
		Provider:     config.TunnelProviderCloudflared,
		Hostname:     "app.example.com",
		Local:        local,
		Auth:         auth,
		ReadyTimeout: "5s",
	}
	if auth == config.TunnelAuthServiceToken {
		tun.ServiceTokenEnv = &config.TunnelServiceTokenEnv{ID: "TID", Secret: "TSECRET"}
	}
	return &config.LifecycleEntry{Name: "svc", Tunnel: tun}
}

// tunnelDiscardLogger keeps acquire's readiness log line out of test output.
func tunnelDiscardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// countSubcommand counts calls.log lines whose subcommand is sub. A missing
// calls.log means the shim was never invoked, which counts as zero.
func countSubcommand(t *testing.T, logDir, sub string) int {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(logDir, "calls.log"))
	if os.IsNotExist(err) {
		return 0
	}
	if err != nil {
		t.Fatalf("read calls.log: %v", err)
	}
	n := 0
	for line := range strings.SplitSeq(string(raw), "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] == sub {
			n++
		}
	}
	return n
}

// tunnelPortAnswers reports whether anything currently accepts on local.
func tunnelPortAnswers(t *testing.T, local string) bool {
	t.Helper()

	conn, err := net.DialTimeout("tcp", local, 250*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func TestTunnelAuthInteractiveNoTTY(t *testing.T) {
	local := tunnelFreeLocal(t)
	logDir := tunnelShim(t, local)
	t.Setenv("TOKEN_EXIT", "1")
	t.Setenv("TOKEN_STDOUT", "synthetic")
	tunnelNoTTY(t)

	m := newTunnelManager(tunnelDiscardLogger())
	err := m.acquire(
		tunnelEntry(config.TunnelAuthInteractive, local),
		config.NewEnvironment(map[string]string{}, "/tmp", "/tmp"))

	if err == nil {
		t.Fatal("interactive auth with no usable token and no TTY must fail")
	}
	if !strings.Contains(err.Error(), "cloudflared access login --quiet https://app.example.com") {
		t.Fatalf("error must carry the exact login command hint, got: %v", err)
	}
	// Nonzero exit is unauthenticated even when stdout is nonempty. The shim
	// writes a synthetic payload; runCloudflaredDiscard counts those bytes
	// and discards them, so no failure path may quote the payload back.
	if strings.Contains(err.Error(), "synthetic-stdout") {
		t.Fatalf("error must not include stdout bytes, got: %v", err)
	}
	// Without a TTY the foreground login must never run, and a failed auth
	// must never reach the tunnel process.
	if n := countSubcommand(t, logDir, "token"); n != 1 {
		t.Fatalf("expected exactly one token probe, got %d", n)
	}
	if n := countSubcommand(t, logDir, "login"); n != 0 {
		t.Fatalf("login must not run without a TTY, ran %d times", n)
	}
	if n := countSubcommand(t, logDir, "tcp"); n != 0 {
		t.Fatalf("tcp must not run when auth fails, ran %d times", n)
	}
}

func TestTunnelAuthExpiredTokenEmptyStdout(t *testing.T) {
	local := tunnelFreeLocal(t)
	logDir := tunnelShim(t, local)
	t.Setenv("TOKEN_EXIT", "0")
	t.Setenv("TOKEN_STDOUT", "empty")
	tunnelNoTTY(t)

	m := newTunnelManager(tunnelDiscardLogger())
	err := m.acquire(
		tunnelEntry(config.TunnelAuthInteractive, local),
		config.NewEnvironment(map[string]string{}, "/tmp", "/tmp"))

	if err == nil {
		t.Fatal("exit 0 with empty stdout must be unauthenticated")
	}
	if !strings.Contains(err.Error(), "cloudflared access login --quiet https://app.example.com") {
		t.Fatalf("error must carry the exact login command hint, got: %v", err)
	}
	if strings.Contains(err.Error(), "synthetic-stdout") {
		t.Fatalf("error must not include stdout bytes, got: %v", err)
	}
	// Without a TTY the foreground login must never run, and a failed auth
	// must never reach the tunnel process.
	if n := countSubcommand(t, logDir, "token"); n != 1 {
		t.Fatalf("expected exactly one token probe, got %d", n)
	}
	if n := countSubcommand(t, logDir, "login"); n != 0 {
		t.Fatalf("login must not run without a TTY, ran %d times", n)
	}
	if n := countSubcommand(t, logDir, "tcp"); n != 0 {
		t.Fatalf("tcp must not run when auth fails, ran %d times", n)
	}
}

func TestTunnelReadyRequiresAuthAndTCP(t *testing.T) {
	env := config.NewEnvironment(map[string]string{}, "/tmp", "/tmp")

	t.Run("auth failure wins over an already listening port", func(t *testing.T) {
		local := tunnelFreeLocal(t)
		logDir := tunnelShim(t, local)
		t.Setenv("TOKEN_EXIT", "1")
		tunnelNoTTY(t)

		ln, err := net.Listen("tcp", local)
		if err != nil {
			t.Fatalf("bind the foreign listener: %v", err)
		}
		defer ln.Close()

		m := newTunnelManager(tunnelDiscardLogger())
		err = m.acquire(tunnelEntry(config.TunnelAuthInteractive, local), env)
		if err == nil {
			t.Fatal("acquire must fail when auth fails, even with a listener present")
		}
		// Auth runs before the ownership check, so the failure is the auth
		// one — a listening port must not mask it.
		if !strings.Contains(err.Error(), "not authenticated") {
			t.Fatalf("expected the auth failure, got: %v", err)
		}
		if n := countSubcommand(t, logDir, "tcp"); n != 0 {
			t.Fatalf("tcp must not run when auth fails, ran %d times", n)
		}
	})

	t.Run("auth ok but no listener never becomes ready", func(t *testing.T) {
		local := tunnelFreeLocal(t)
		logDir := tunnelShim(t, local)
		t.Setenv("TOKEN_EXIT", "0")
		t.Setenv("TOKEN_STDOUT", "synthetic")
		// LISTEN unset: the shim accepts `tcp` but never binds the port.

		entry := tunnelEntry(config.TunnelAuthInteractive, local)
		entry.Tunnel.ReadyTimeout = "700ms"

		m := newTunnelManager(tunnelDiscardLogger())
		start := time.Now()
		err := m.acquire(entry, env)
		if err == nil {
			t.Fatal("acquire must fail when the local listener never appears")
		}
		if !strings.Contains(err.Error(), "not ready") {
			t.Fatalf("expected the readiness timeout, got: %v", err)
		}
		if elapsed := time.Since(start); elapsed > 5*time.Second {
			t.Fatalf("acquire overshot the declared ready_timeout: %v", elapsed)
		}
		if n := countSubcommand(t, logDir, "tcp"); n != 1 {
			t.Fatalf("expected exactly one tcp attempt, got %d", n)
		}
	})

	t.Run("listener appears so acquire succeeds", func(t *testing.T) {
		local := tunnelFreeLocal(t)
		tunnelShim(t, local)
		t.Setenv("TOKEN_EXIT", "0")
		// One space is a nonzero count. Trimming would treat it as empty and
		// fail auth before the listener is consulted.
		t.Setenv("TOKEN_STDOUT", "blank")
		t.Setenv("LISTEN", "1")

		m := newTunnelManager(tunnelDiscardLogger())
		defer m.Close()
		if err := m.acquire(tunnelEntry(config.TunnelAuthInteractive, local), env); err != nil {
			t.Fatalf("acquire with a live listener must succeed: %v", err)
		}
	})
}

func TestTunnelAuthServiceToken(t *testing.T) {
	env := config.NewEnvironment(map[string]string{}, "/tmp", "/tmp")

	t.Run("child receives the declared service token env", func(t *testing.T) {
		local := tunnelFreeLocal(t)
		logDir := tunnelShim(t, local)
		t.Setenv("LISTEN", "1")
		t.Setenv("TID", "id-value")
		t.Setenv("TSECRET", "secret-value")
		tunnelNoTTY(t)

		m := newTunnelManager(tunnelDiscardLogger())
		defer m.Close()
		if err := m.acquire(tunnelEntry(config.TunnelAuthServiceToken, local), env); err != nil {
			t.Fatalf("service-token acquire must succeed: %v", err)
		}

		raw, err := os.ReadFile(filepath.Join(logDir, "service_token.log"))
		if err != nil {
			t.Fatalf("read service_token.log: %v", err)
		}
		got := string(raw)
		if !strings.Contains(got, "id=id-value") || !strings.Contains(got, "secret=secret-value") {
			t.Fatalf("child env must carry the declared token values, got: %q", got)
		}
		// Service-token mode never needs a TTY: no token probe, no login,
		// exactly one tunnel process.
		if n := countSubcommand(t, logDir, "token"); n != 0 {
			t.Fatalf("service-token mode must not probe for a cached token, probed %d times", n)
		}
		if n := countSubcommand(t, logDir, "login"); n != 0 {
			t.Fatalf("service-token mode must not run login, ran %d times", n)
		}
		if n := countSubcommand(t, logDir, "tcp"); n != 1 {
			t.Fatalf("expected exactly one tcp invocation, got %d", n)
		}
	})

	t.Run("empty id env fails before cloudflared starts", func(t *testing.T) {
		local := tunnelFreeLocal(t)
		logDir := tunnelShim(t, local)
		t.Setenv("TID", "")
		t.Setenv("TSECRET", "")

		m := newTunnelManager(tunnelDiscardLogger())
		err := m.acquire(tunnelEntry(config.TunnelAuthServiceToken, local), env)
		if err == nil {
			t.Fatal("acquire must fail when the token id env is empty")
		}
		if !strings.Contains(err.Error(), "TID") {
			t.Fatalf("error must name the offending env var, got: %v", err)
		}
		if n := countSubcommand(t, logDir, "tcp"); n != 0 {
			t.Fatalf("cloudflared must not start, ran tcp %d times", n)
		}
	})

	t.Run("empty secret env fails before cloudflared starts", func(t *testing.T) {
		local := tunnelFreeLocal(t)
		logDir := tunnelShim(t, local)
		t.Setenv("TID", "id-value")
		t.Setenv("TSECRET", "")

		m := newTunnelManager(tunnelDiscardLogger())
		err := m.acquire(tunnelEntry(config.TunnelAuthServiceToken, local), env)
		if err == nil {
			t.Fatal("acquire must fail when the token secret env is empty")
		}
		if !strings.Contains(err.Error(), "TSECRET") {
			t.Fatalf("error must name the offending env var, got: %v", err)
		}
		if n := countSubcommand(t, logDir, "tcp"); n != 0 {
			t.Fatalf("cloudflared must not start, ran tcp %d times", n)
		}
	})
}

func TestTunnelLifecycleOwnership(t *testing.T) {
	env := config.NewEnvironment(map[string]string{}, "/tmp", "/tmp")

	t.Run("same declaration opens cloudflared once", func(t *testing.T) {
		local := tunnelFreeLocal(t)
		logDir := tunnelShim(t, local)
		t.Setenv("TOKEN_EXIT", "0")
		t.Setenv("TOKEN_STDOUT", "synthetic")
		t.Setenv("LISTEN", "1")

		entry := tunnelEntry(config.TunnelAuthInteractive, local)
		m := newTunnelManager(tunnelDiscardLogger())
		defer m.Close()
		if err := m.acquire(entry, env); err != nil {
			t.Fatalf("first acquire: %v", err)
		}
		if err := m.acquire(entry, env); err != nil {
			t.Fatalf("second acquire of the same declaration: %v", err)
		}
		if n := countSubcommand(t, logDir, "tcp"); n != 1 {
			t.Fatalf("the same declaration must open cloudflared once, got %d tcp calls", n)
		}
	})

	t.Run("close kills the tunnel dva started", func(t *testing.T) {
		local := tunnelFreeLocal(t)
		tunnelShim(t, local)
		t.Setenv("TOKEN_EXIT", "0")
		t.Setenv("TOKEN_STDOUT", "synthetic")
		t.Setenv("LISTEN", "1")

		m := newTunnelManager(tunnelDiscardLogger())
		if err := m.acquire(tunnelEntry(config.TunnelAuthInteractive, local), env); err != nil {
			t.Fatalf("acquire: %v", err)
		}
		if !tunnelPortAnswers(t, local) {
			t.Fatal("the tunnel port must answer once acquire returned")
		}

		m.Close()
		deadline := time.Now().Add(4 * time.Second)
		for tunnelPortAnswers(t, local) {
			if time.Now().After(deadline) {
				t.Fatal("the tunnel port must stop answering within a couple seconds of Close")
			}
			time.Sleep(100 * time.Millisecond)
		}
	})

	t.Run("foreign listener is a conflict not a reuse", func(t *testing.T) {
		local := tunnelFreeLocal(t)
		logDir := tunnelShim(t, local)
		t.Setenv("TOKEN_EXIT", "0")
		t.Setenv("TOKEN_STDOUT", "synthetic")

		ln, err := net.Listen("tcp", local)
		if err != nil {
			t.Fatalf("bind the foreign listener: %v", err)
		}
		defer ln.Close()

		m := newTunnelManager(tunnelDiscardLogger())
		err = m.acquire(tunnelEntry(config.TunnelAuthInteractive, local), env)
		if err == nil {
			t.Fatal("acquire must fail closed on an unowned listener")
		}
		if !strings.Contains(err.Error(), "already in use") {
			t.Fatalf("expected the port conflict error, got: %v", err)
		}
		if n := countSubcommand(t, logDir, "tcp"); n != 0 {
			t.Fatalf("a conflict must never start cloudflared, ran tcp %d times", n)
		}
	})
}
