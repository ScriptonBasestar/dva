package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func healthPortWarnings(t *testing.T, yml string) []string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv(EnvFileKey, "")
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(yml), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	var got []string
	for _, d := range cfg.ValidateDiagnostics() {
		if strings.Contains(d.Message, "also claims") {
			got = append(got, d.Message)
		}
	}
	return got
}

func TestHealthPortSharedWithOtherEntryCheck(t *testing.T) {
	got := healthPortWarnings(t, `version: '0.1.0'
stack:
  api:
    process: {command: "./api"}
    health_checks:
      ready: {type: http, url: "http://127.0.0.1:8080/health"}
  web:
    process: {command: "./web"}
    health_checks:
      ready: {type: tcp, address: "localhost:8080"}
`)
	if len(got) != 2 {
		t.Fatalf("warnings = %q, want one per process entry", got)
	}
	if !strings.Contains(got[0], "stack.api.health_checks.ready") || !strings.Contains(got[0], "stack.web.health_checks.ready") {
		t.Fatalf("api warning should name web's check: %q", got[0])
	}
}

func TestHealthPortSharedWithDockerPorts(t *testing.T) {
	got := healthPortWarnings(t, `version: '0.1.0'
stack:
  api:
    process: {command: "./api"}
    health_checks:
      ready: {type: http, url: "http://localhost:${API_PORT}/"}
  db:
    default_runner: docker
    runners:
      docker: {image: postgres, ports: ["127.0.0.1:5432:5432", "9090:80/tcp"]}
environment:
  API_PORT: "9090"
`)
	if len(got) != 1 || !strings.Contains(got[0], `stack.db ports "9090:80/tcp"`) {
		t.Fatalf("warnings = %q, want the docker mapping named", got)
	}
}

func TestHealthPortNoWarningForDistinctOrOwnPorts(t *testing.T) {
	got := healthPortWarnings(t, `version: '0.1.0'
stack:
  api:
    process: {command: "./api"}
    health_checks:
      live: {type: http, url: "http://127.0.0.1:8080/live"}
      ready: {type: tcp, address: "127.0.0.1:8080"}
  web:
    process: {command: "./web"}
    health_checks:
      ready: {type: http, url: "http://127.0.0.1:8081/"}
      cmd: {type: command, command: "true"}
`)
	if len(got) != 0 {
		t.Fatalf("warnings = %q, want none", got)
	}
}

func TestHealthPortDockerHostPortParsing(t *testing.T) {
	for spec, want := range map[string]string{
		"8080:80": "8080", "127.0.0.1:8080:80": "8080", "8080:80/udp": "8080",
		"80": "", "8000-8010:80": "", ":80": "", "abc:80": "",
	} {
		if got := dockerHostPort(spec); got != want {
			t.Errorf("dockerHostPort(%q) = %q, want %q", spec, got, want)
		}
	}
}
