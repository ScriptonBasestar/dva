package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// doctorSyntheticStdout is the nonempty stdout a successful or nonzero
// access-token shim writes. It is not a credential. Doctor rows must not
// copy it into a finding or fix hint.
const doctorSyntheticStdout = "synthetic-stdout-sentinel"

// cloudflaredDoctorShim replaces PATH with a directory whose cloudflared
// exits tokenExit only for `access token` — the arguments checkTunnelAuthState
// passes (`cloudflared access token --app`). stdoutKind selects that probe's
// stdout: "synthetic" (doctorSyntheticStdout plus a newline), "blank" (one
// space, so trimming would look empty), or "empty" (zero bytes). A shim that
// compared $1 to "token" never saw that argv: $1 is "access", so the supplied
// exit was ignored and the fallback `exit 0` made every auth probe look
// healthy. Any other argv still exits 0. The returned log receives one line
// per invocation so a passing row can show it took the access-token branch.
func cloudflaredDoctorShim(t *testing.T, tokenExit, stdoutKind string) string {
	t.Helper()
	if !tokenExitDigits(tokenExit) {
		t.Fatalf("token exit must be an unsigned integer, got %q", tokenExit)
	}
	stdoutLine := accessTokenStdoutLine(t, stdoutKind)
	dir := t.TempDir()
	logPath := filepath.Join(dir, "argv.log")
	script := "#!/bin/sh\n" +
		"printf '%s\\n' \"$*\" >> " + shellSingleQuote(logPath) + "\n" +
		"if [ \"$1\" = \"access\" ] && [ \"$2\" = \"token\" ]; then\n" +
		stdoutLine +
		"  exit " + tokenExit + "\n" +
		"fi\n" +
		"exit 0\n"
	if err := shimWrite(dir, "cloudflared", script); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	return logPath
}

// accessTokenStdoutLine is the shell that writes the probe stdout. An empty
// kind writes zero bytes; printf is used so a blank payload stays one space.
func accessTokenStdoutLine(t *testing.T, kind string) string {
	t.Helper()
	switch kind {
	case "synthetic":
		return "  printf '%s\\n' " + shellSingleQuote(doctorSyntheticStdout) + "\n"
	case "blank":
		return "  printf ' '\n"
	case "empty":
		return ""
	default:
		t.Fatalf("stdout kind must be synthetic, blank, or empty, got %q", kind)
		return ""
	}
}

func tokenExitDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
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

	// Installed, and `cloudflared access token` exits 0 with nonempty stdout:
	// both rows pass. The argv log shows the probe was `access token`, so the
	// pass is that result and not the shim's fallback for any other command.
	logPath := cloudflaredDoctorShim(t, "0", "synthetic")
	results = runTunnelDoctorChecks(c)
	if len(results) != 2 {
		t.Fatalf("installed: got %d rows, want 2 (installed + auth)", len(results))
	}
	if !results[0].Passed || results[0].Name != "cloudflared installed for stack.remote-k8s" {
		t.Fatalf("installed row: %+v", results[0])
	}
	if !results[1].Passed || results[1].Name != "cloudflared access token for stack.remote-k8s" {
		t.Fatalf("auth row on token exit 0: %+v", results[1])
	}
	assertAccessTokenProbe(t, logPath, "app.example.com")
	assertNoDoctorJWT(t, results)

	// One space is a nonzero byte count. Trimming would mark the same probe
	// unauthenticated. The payload is not a credential and is not printed.
	logPath = cloudflaredDoctorShim(t, "0", "blank")
	results = runTunnelDoctorChecks(c)
	if len(results) != 2 || !results[1].Passed || results[1].Name != "cloudflared access token for stack.remote-k8s" {
		t.Fatalf("auth row on whitespace stdout: %+v", results)
	}
	assertAccessTokenProbe(t, logPath, "app.example.com")
	assertNoDoctorJWT(t, results)
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
	_ = cloudflaredDoctorShim(t, "0", "synthetic")

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
		if strings.Contains(r.Finding, "id-value") || strings.Contains(r.FixHint, "secret-value") ||
			strings.Contains(r.Finding, doctorSyntheticStdout) || strings.Contains(r.FixHint, doctorSyntheticStdout) {
			t.Fatalf("value leaked into doctor output: %+v", r)
		}
	}
}

func TestDoctorTunnelInteractiveAuthFailure(t *testing.T) {
	c := tunnelDoctorConfig(&config.TunnelConfig{
		Provider: "cloudflared",
		Hostname: "app.example.com",
		Local:    "127.0.0.1:16443",
	})
	logPath := cloudflaredDoctorShim(t, "1", "synthetic")
	results := runTunnelDoctorChecks(c)
	if len(results) != 2 {
		t.Fatalf("auth failure: got %d rows, want 2", len(results))
	}
	if !results[0].Passed {
		t.Fatalf("installed row failed while cloudflared was on PATH: %+v", results[0])
	}
	auth := results[1]
	if auth.Passed {
		t.Fatalf("auth row passed on token exit 1: %+v", auth)
	}
	if !strings.Contains(auth.Finding, "no usable Access token for app.example.com") {
		t.Fatalf("finding = %q", auth.Finding)
	}
	const wantHint = "Run: cloudflared access login --quiet https://app.example.com"
	if auth.FixHint != wantHint {
		t.Fatalf("fix hint = %q, want %q", auth.FixHint, wantHint)
	}
	assertNoDoctorJWT(t, results)
	assertAccessTokenProbe(t, logPath, "app.example.com")
}

func TestDoctorTunnelAuthEmptyStdout(t *testing.T) {
	c := tunnelDoctorConfig(&config.TunnelConfig{
		Provider: "cloudflared",
		Hostname: "app.example.com",
		Local:    "127.0.0.1:16443",
	})
	logPath := cloudflaredDoctorShim(t, "0", "empty")
	results := runTunnelDoctorChecks(c)
	if len(results) != 2 {
		t.Fatalf("empty stdout: got %d rows, want 2", len(results))
	}
	if !results[0].Passed {
		t.Fatalf("installed row failed while cloudflared was on PATH: %+v", results[0])
	}
	auth := results[1]
	if auth.Passed || auth.Name != "cloudflared access token for stack.remote-k8s" {
		t.Fatalf("auth row passed on exit 0 with empty stdout: %+v", auth)
	}
	if !strings.Contains(auth.Finding, "no usable Access token for app.example.com") {
		t.Fatalf("finding = %q", auth.Finding)
	}
	const wantHint = "Run: cloudflared access login --quiet https://app.example.com"
	if auth.FixHint != wantHint {
		t.Fatalf("fix hint = %q, want %q", auth.FixHint, wantHint)
	}
	// Passed false is the [FAIL] row. The printed line is the finding, which
	// must not carry stdout bytes. This probe wrote zero of them.
	if got := "[FAIL] " + auth.failureLine(); !strings.Contains(got, "[FAIL] no usable Access token for app.example.com") {
		t.Fatalf("failure line = %q", got)
	}
	if strings.Contains(auth.failureLine(), doctorSyntheticStdout) {
		t.Fatalf("stdout bytes leaked into the failure line: %q", auth.failureLine())
	}
	assertNoDoctorJWT(t, results)
	assertAccessTokenProbe(t, logPath, "app.example.com")
}

// TestDoctorTunnelConfigEndToEnd loads a dva.yml through config.Load and runs
// runDoctorChecks, the same aggregation `dva doctor` uses. The cloudflared on
// PATH is the local shim (exit 0, synthetic nonempty stdout). The test does
// not log in, open a tunnel, or read a credential. A doctor run against a
// live tunnel config stays a human check.
func TestDoctorTunnelConfigEndToEnd(t *testing.T) {
	logPath := cloudflaredDoctorShim(t, "0", "synthetic")
	dir := t.TempDir()
	const body = `stack:
  remote-interactive:
    plugin: kubectl
    kubectl:
      manifests: [k8s/dev-tools.yaml]
    tunnel:
      provider: cloudflared
      hostname: app.example.test
      local: 127.0.0.1:16443
      auth: interactive
  remote-token:
    plugin: kubectl
    kubectl:
      manifests: [k8s/dev-tools.yaml]
    tunnel:
      provider: cloudflared
      hostname: token.example.test
      local: 127.0.0.1:16444
      auth: service-token
      service_token_env:
        id: DVA_DOCTOR_E2E_ID
        secret: DVA_DOCTOR_E2E_SECRET
`
	if err := os.WriteFile(filepath.Join(dir, config.FileName), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(dir)
	if err != nil {
		t.Fatalf("load tunnel config: %v", err)
	}
	interactive := c.Stack["remote-interactive"]
	if interactive == nil || interactive.Tunnel == nil ||
		interactive.Tunnel.Hostname != "app.example.test" ||
		interactive.Tunnel.AuthMode() != config.TunnelAuthInteractive {
		t.Fatalf("parsed interactive tunnel: %+v", interactive)
	}
	tokenEntry := c.Stack["remote-token"]
	if tokenEntry == nil || tokenEntry.Tunnel == nil ||
		tokenEntry.Tunnel.AuthMode() != config.TunnelAuthServiceToken ||
		tokenEntry.Tunnel.ServiceTokenEnv == nil ||
		tokenEntry.Tunnel.ServiceTokenEnv.ID != "DVA_DOCTOR_E2E_ID" ||
		tokenEntry.Tunnel.ServiceTokenEnv.Secret != "DVA_DOCTOR_E2E_SECRET" {
		t.Fatalf("parsed service-token tunnel: %+v", tokenEntry)
	}

	t.Setenv("DVA_DOCTOR_E2E_ID", "")
	t.Setenv("DVA_DOCTOR_E2E_SECRET", "")
	results := runDoctorChecks(c)
	_ = doctorRow(t, results, "Docker daemon accessible")

	if row := doctorRow(t, results, "cloudflared installed for stack.remote-interactive"); !row.Passed {
		t.Fatalf("installed: %+v", row)
	}
	if row := doctorRow(t, results, "cloudflared access token for stack.remote-interactive"); !row.Passed {
		t.Fatalf("auth on token exit 0: %+v", row)
	}
	assertAccessTokenProbe(t, logPath, "app.example.test")
	if row := doctorRow(t, results, "cloudflared installed for stack.remote-token"); !row.Passed {
		t.Fatalf("service-token installed: %+v", row)
	}
	envRow := doctorRow(t, results, "service token env vars for stack.remote-token")
	if envRow.Passed {
		t.Fatalf("unset service-token env passed: %+v", envRow)
	}
	if !strings.Contains(envRow.Finding, "DVA_DOCTOR_E2E_ID") {
		t.Fatalf("finding = %q, want the missing variable name", envRow.Finding)
	}
	if !strings.Contains(envRow.FixHint, "DVA_DOCTOR_E2E_ID") || !strings.Contains(envRow.FixHint, "DVA_DOCTOR_E2E_SECRET") {
		t.Fatalf("fix hint = %q, want both variable names", envRow.FixHint)
	}
	assertNoDoctorJWT(t, results)

	const idValue = "e2e-id-SENTINEL"
	const secretValue = "e2e-secret-SENTINEL"
	t.Setenv("DVA_DOCTOR_E2E_ID", idValue)
	t.Setenv("DVA_DOCTOR_E2E_SECRET", secretValue)
	results = runDoctorChecks(c)
	envRow = doctorRow(t, results, "service token env vars for stack.remote-token")
	if !envRow.Passed {
		t.Fatalf("set service-token env failed: %+v", envRow)
	}
	if row := doctorRow(t, results, "cloudflared access token for stack.remote-interactive"); !row.Passed {
		t.Fatalf("auth row dropped on the second aggregation: %+v", row)
	}
	for _, r := range results {
		if strings.Contains(r.Finding, idValue) || strings.Contains(r.FixHint, idValue) ||
			strings.Contains(r.Finding, secretValue) || strings.Contains(r.FixHint, secretValue) ||
			strings.Contains(r.Finding, doctorSyntheticStdout) || strings.Contains(r.FixHint, doctorSyntheticStdout) {
			t.Fatalf("value leaked into doctor output: %+v", r)
		}
	}
	logged, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(logged), "token.example.test") {
		t.Fatalf("service-token entry ran access token: %s", logged)
	}
}

func assertNoDoctorJWT(t *testing.T, results []DoctorResult) {
	t.Helper()
	for _, r := range results {
		if strings.Contains(r.Finding, doctorSyntheticStdout) || strings.Contains(r.FixHint, doctorSyntheticStdout) {
			t.Fatalf("stdout bytes leaked into doctor output: %+v", r)
		}
	}
}

func assertAccessTokenProbe(t *testing.T, logPath, hostname string) {
	t.Helper()
	body, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	want := "access token --app https://" + hostname
	if !strings.Contains(string(body), want) {
		t.Fatalf("argv log = %q, want probe %q", body, want)
	}
}

func doctorRow(t *testing.T, results []DoctorResult, name string) DoctorResult {
	t.Helper()
	for _, r := range results {
		if r.Name == name {
			return r
		}
	}
	names := make([]string, 0, len(results))
	for _, r := range results {
		names = append(names, r.Name)
	}
	t.Fatalf("missing doctor row %q in %q", name, names)
	return DoctorResult{}
}
