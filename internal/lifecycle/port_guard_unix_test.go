//go:build !windows

package lifecycle

import (
	"context"
	"net"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// Two worktrees with the same default port: the second `up` must refuse rather than start a
// process that cannot bind while its health check reaches the first server (TASK-538 pilot).
func TestProcessUpRefusesHeldHealthPort(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)

	for _, check := range []config.HealthCheckConfig{
		{Type: "http", URL: "http://127.0.0.1:${WEB_PORT:-1}/"},
		{Type: "tcp", Address: "127.0.0.1:${WEB_PORT:-1}"},
	} {
		t.Run(check.Type, func(t *testing.T) {
			dir := t.TempDir()
			pctx := identityPctx(dir, "")
			pctx.Env = config.NewEnvironment(map[string]string{"WEB_PORT": port}, dir, dir)
			pctx.Entry.HealthChecks = map[string]config.HealthCheckConfig{"web": check}

			_, err := (&ProcessPlugin{}).Up(context.Background(), pctx)
			if err == nil || !strings.Contains(err.Error(), "127.0.0.1:"+port) {
				t.Fatalf("up err = %v, want a refusal naming 127.0.0.1:%s", err, port)
			}
			if _, statErr := os.Stat(pctx.PidFile("api")); !os.IsNotExist(statErr) {
				t.Fatalf("pid file after refusal: %v; nothing must be started", statErr)
			}
		})
	}
}

// A free port starts normally: the guard only refuses a port someone already serves.
func TestProcessUpStartsOnFreeHealthPort(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)
	_ = ln.Close()

	dir := t.TempDir()
	pctx := identityPctx(dir, "")
	pctx.Entry.HealthChecks = map[string]config.HealthCheckConfig{
		"web": {Type: "http", URL: "http://127.0.0.1:" + port + "/"},
		"cmd": {Type: "command", Command: "true"},
	}
	if _, err := (&ProcessPlugin{}).Up(context.Background(), pctx); err != nil {
		t.Fatalf("up on a free port: %v", err)
	}
	data, err := os.ReadFile(pctx.PidFile("api"))
	if err != nil {
		t.Fatalf("pid file after up on a free port: %v", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		t.Fatalf("pid file: %v", err)
	}
	killProcessGroupInCleanup(t, pid)
}

func TestHealthCheckAddress(t *testing.T) {
	cases := []struct {
		check config.HealthCheckConfig
		want  string
	}{
		{config.HealthCheckConfig{Type: "http", URL: "http://localhost/x"}, "localhost:80"},
		{config.HealthCheckConfig{Type: "http", URL: "https://example.test/"}, "example.test:443"},
		{config.HealthCheckConfig{Type: "http", URL: "http://[::1]:8080/"}, "[::1]:8080"},
		{config.HealthCheckConfig{Type: "tcp", Address: "127.0.0.1:5432"}, "127.0.0.1:5432"},
		{config.HealthCheckConfig{Type: "tcp", Address: "no-port"}, ""},
		{config.HealthCheckConfig{Type: "command", Command: "true"}, ""},
	}
	for _, c := range cases {
		if got := healthCheckAddress(c.check, nil); got != c.want {
			t.Errorf("healthCheckAddress(%+v) = %q, want %q", c.check, got, c.want)
		}
	}
}
