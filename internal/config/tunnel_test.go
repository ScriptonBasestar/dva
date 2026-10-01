package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// tunnelFixture is a valid minimal tunnel declaration on a kubectl entry.
// Each rejection case rewrites one field of it via strings.ReplaceAll; Load()
// runs the JSON schema and the Go validator together, so a case fails whenever
// either layer rejects the declaration.
const tunnelFixture = `stack:
  remote-k8s:
    plugin: kubectl
    kubectl:
      manifests: [k8s/dev-tools.yaml]
      context: scripton-cluster-cf
    tunnel:
      provider: cloudflared
      hostname: ${CF_K8S_HOST}
      local: 127.0.0.1:16443
      auth: interactive
`

func TestTunnelConfigValidation(t *testing.T) {
	c := loadTunnelConfig(t, tunnelFixture)
	entry := c.Stack["remote-k8s"]
	if entry.Tunnel == nil {
		t.Fatal("tunnel declaration lost on load")
	}
	if entry.Tunnel.Provider != "cloudflared" || entry.Tunnel.Local != "127.0.0.1:16443" {
		t.Fatalf("lost tunnel fields: %+v", entry.Tunnel)
	}
	if got := entry.Tunnel.AuthMode(); got != TunnelAuthInteractive {
		t.Fatalf("auth default = %q, want interactive", got)
	}
	if got, err := entry.Tunnel.ReadyTimeoutDuration(); err != nil || got != 30*time.Second {
		t.Fatalf("default ready_timeout = %v, %v; want 30s", got, err)
	}

	// Service-token happy path: env var names only, never values.
	withServiceToken := strings.ReplaceAll(tunnelFixture,
		"      auth: interactive\n",
		"      auth: service-token\n      service_token_env: {id: CF_ID, secret: CF_SECRET}\n")
	tun := loadTunnelConfig(t, withServiceToken).Stack["remote-k8s"].Tunnel
	if tun.AuthMode() != TunnelAuthServiceToken ||
		tun.ServiceTokenEnv == nil || tun.ServiceTokenEnv.ID != "CF_ID" || tun.ServiceTokenEnv.Secret != "CF_SECRET" {
		t.Fatalf("lost service-token fields: %+v", tun)
	}

	// Rejection table: each case rewrites the fixture so exactly one rule is
	// violated. Load() must reject every one of them.
	for _, test := range []struct{ name, old, next string }{
		{"unknown provider", "provider: cloudflared", "provider: wireguard"},
		{"missing hostname", "hostname: ${CF_K8S_HOST}", "hostname: \"\""},
		{"non-loopback local", "local: 127.0.0.1:16443", "local: 192.168.1.10:16443"},
		{"local without port", "local: 127.0.0.1:16443", "local: 127.0.0.1"},
		{"service-token without env names", "auth: interactive", "auth: service-token"},
		{"service-token missing secret name", "      local: 127.0.0.1:16443\n",
			"      local: 127.0.0.1:16443\n      auth: service-token\n      service_token_env:\n        id: CF_ID\n"},
		{"interactive with service_token_env", "      local: 127.0.0.1:16443\n",
			"      local: 127.0.0.1:16443\n      service_token_env: {id: CF_ID, secret: CF_SECRET}\n"},
		{"invalid auth value", "auth: interactive", "auth: kerberos"},
		{"invalid ready_timeout", "      local: 127.0.0.1:16443\n",
			"      local: 127.0.0.1:16443\n      ready_timeout: 30minutes\n"},
		{"non-positive ready_timeout", "      local: 127.0.0.1:16443\n",
			"      local: 127.0.0.1:16443\n      ready_timeout: 0s\n"},
		{"tunnel on process entry", "plugin: kubectl",
			"plugin: process\n    process:\n      command: sleep 3600"},
	} {
		t.Run(test.name, func(t *testing.T) {
			out := strings.ReplaceAll(tunnelFixture, test.old, test.next)
			if out == tunnelFixture {
				t.Fatalf("rewrite %q matched nothing in fixture", test.old)
			}
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, FileName), []byte(out), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(dir); err == nil {
				t.Fatalf("accepted invalid tunnel declaration: %s", test.name)
			}
		})
	}
}

// loadTunnelConfig writes one dva.yml and loads it, so every case in this file
// exercises the same schema + Go validation path a real load takes.
func loadTunnelConfig(t *testing.T, body string) *Config {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(dir)
	if err != nil {
		t.Fatalf("valid tunnel declaration rejected: %v", err)
	}
	return c
}
