package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// cloudflaredDoctorShim replaces PATH with a directory whose cloudflared
// reports its subcommand through its exit code: token exits TOKEN_EXIT
// (default 0). Output is never checked — the doctor must decide on the exit
// code only, and its stdout/stderr must stay free of the fake JWT the shim
// prints.
func cloudflaredDoctorShim(t *testing.T, tokenExit string) {
	t.Helper()
	dir := t.TempDir()
	script := "#!/bin/sh\n" +
		"echo fake.jwt.token\n" +
		"if [ \"$1\" = \"token\" ]; then\n" +
		"  exit " + tokenExit + "\n" +
		"fi\n" +
		"exit 0\n"
	if err := shimWrite(dir, "cloudflared", script); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
}

func shimWrite(dir, name, body string) error {
	return os.WriteFile(filepath.Join(dir, name), []byte(body), 0o755)
}

// tunnelDoctorConfig builds the in-memory config one kubectl entry with a
// tunnel declaration feeds runTunnelDoctorChecks.
func tunnelDoctorConfig(tunnel *config.TunnelConfig) *config.Config {
	return &config.Config{Stack: map[string]*config.LifecycleEntry{
		"remote-k8s": {
			Name:    "remote-k8s",
			Plugin:  "kubectl",
			Kubectl: &config.KubectlPluginConfig{Manifests: []string{"k8s/x.yaml"}},
			Tunnel:  tunnel,
		},
	}}
}

func TestDoctorTunnelInstalledCheck(t *testing.T) {
	c := tunnelDoctorConfig(&config.TunnelConfig{
		Provider: "cloudflared",
		Hostname: "app.example.com",
		Local:    "127.0.0.1:16443",
	})

	// Missing binary: one failing row with an install hint.
	t.Setenv("PATH", t.TempDir())
	results := runTunnelDoctorChecks(c)
	if len(results) != 1 || results[0].Passed {
		t.Fatalf("missing cloudflared: got %+v", results)
	}
	if !strings.Contains(results[0].Finding, "not found on PATH") {
		t.Fatalf("finding = %q, want a not-found finding", results[0].Finding)
	}

	// Installed: the row passes. The shim prints a fake JWT on every call; the
	// token probe in the next test must never let it reach captured output.
	cloudflaredDoctorShim(t, "0")
	results = runTunnelDoctorChecks(c)
	if len(results) != 2 {
		t.Fatalf("installed: got %d rows, want 2 (installed + auth)", len(results))
	}
	if !results[0].Passed || !results[1].Passed {
		t.Fatalf("installed rows: %+v", results)
	}
}

func TestDoctorTunnelServiceTokenEnvCheck(t *testing.T) {
	c := tunnelDoctorConfig(&config.TunnelConfig{
		Provider:        "cloudflared",
		Hostname:        "app.example.com",
		Local:           "127.0.0.1:16443",
		Auth:            config.TunnelAuthServiceToken,
		ServiceTokenEnv: &config.TunnelServiceTokenEnv{ID: "DOC_TID", Secret: "DOC_TSECRET"},
	})

	// The shim makes the installed row deterministic regardless of the host
	// (helm_test.go convention): PATH is replaced, not prepended.
	cloudflaredDoctorShim(t, "0")

	// Neither variable set: one failing row naming the missing variable.
	results := runTunnelDoctorChecks(c)
	if len(results) != 2 || results[1].Passed {
		t.Fatalf("unset env: got %+v", results)
	}
	if !strings.Contains(results[1].Finding, "DOC_TID") {
		t.Fatalf("finding = %q, want the missing variable name", results[1].Finding)
	}

	// Both set: the row passes, and neither the finding nor the fix hint may
	// quote a value — only the declared names.
	t.Setenv("DOC_TID", "id-value")
	t.Setenv("DOC_TSECRET", "secret-value")
	results = runTunnelDoctorChecks(c)
	if len(results) != 2 || !results[1].Passed {
		t.Fatalf("set env: got %+v", results)
	}
	for _, r := range results {
		if strings.Contains(r.Finding, "id-value") || strings.Contains(r.FixHint, "secret-value") {
			t.Fatalf("value leaked into doctor output: %+v", r)
		}
	}
}
