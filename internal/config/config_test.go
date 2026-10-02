package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestServiceTagsFieldParsing locks that services.<svc>.tags still parse after
// related/hint removal (TASK-036). Sibling Tags remains the live field.
func TestServiceTagsFieldParsing(t *testing.T) {
	tmpDir := t.TempDir()
	dvaYml := filepath.Join(tmpDir, FileName)

	content := `stack:
  compose:
    default_runner: compose
    order: 10
    runners:
      compose:
        files: [docker-compose.yml]
        services:
          api:
            tags: [web]
          worker:
            tags: [background]
`
	os.WriteFile(dvaYml, []byte(content), 0644)

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	services := cfg.ComposeServices()
	api := services["api"]
	if len(api.Tags) != 1 || api.Tags[0] != "web" {
		t.Fatalf("api.tags = %v, want [web]", api.Tags)
	}
	worker := services["worker"]
	if len(worker.Tags) != 1 || worker.Tags[0] != "background" {
		t.Fatalf("worker.tags = %v, want [background]", worker.Tags)
	}
}

func TestDoctorChecksParsing(t *testing.T) {
	tmpDir := t.TempDir()
	dvaYml := filepath.Join(tmpDir, FileName)

	content := `checks:
  - name: Docker accessible
    type: docker_socket
    fix_hint: Start Docker
  - name: .env exists
    type: file_exists
    path: .env
    fix_hint: cp .env.example .env
  - name: Migrations applied
    type: command
    command: make migrate-status
    fix_hint: dva provision setup
`
	os.WriteFile(dvaYml, []byte(content), 0644)

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if len(cfg.DoctorChecks) != 3 {
		t.Fatalf("expected 3 checks, got %d", len(cfg.DoctorChecks))
	}
	if cfg.DoctorChecks[0].Type != "docker_socket" {
		t.Errorf("check[0].type = %q, want docker_socket", cfg.DoctorChecks[0].Type)
	}
	if cfg.DoctorChecks[1].Path != ".env" {
		t.Errorf("check[1].path = %q, want .env", cfg.DoctorChecks[1].Path)
	}
	if cfg.DoctorChecks[2].Command != "make migrate-status" {
		t.Errorf("check[2].command = %q", cfg.DoctorChecks[2].Command)
	}
}

func TestModeProvisionField(t *testing.T) {
	tmpDir := t.TempDir()
	dvaYml := filepath.Join(tmpDir, FileName)

	content := `modes:
  full-stack:
    description: "Everything"
    provision: setup
provision:
  setup:
    - step: Install deps
      run: npm install
`
	os.WriteFile(dvaYml, []byte(content), 0644)

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	m, ok := cfg.Modes["full-stack"]
	if !ok {
		t.Fatal("mode full-stack not found")
	}
	if m.Provision != "setup" {
		t.Errorf("provision = %q, want setup", m.Provision)
	}
}

func TestEndpointsParsing(t *testing.T) {
	tmpDir := t.TempDir()
	dvaYml := filepath.Join(tmpDir, FileName)

	content := `endpoints:
  api:
    url: http://localhost:8080
    label: "API Server"
    tags: [app]
    paths:
      /health: "Health check"
      /api/v1: "REST API"
  git-ssh:
    url: ssh://git@localhost:2222
    label: "Git SSH"
    tags: [app, scm]
`
	os.WriteFile(dvaYml, []byte(content), 0644)

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if len(cfg.Endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(cfg.Endpoints))
	}

	api := cfg.Endpoints["api"]
	if api.URL != "http://localhost:8080" {
		t.Errorf("api.URL = %q", api.URL)
	}
	if api.Label != "API Server" {
		t.Errorf("api.Label = %q", api.Label)
	}
	if len(api.Tags) != 1 || api.Tags[0] != "app" {
		t.Errorf("api.Tags = %v", api.Tags)
	}
	if len(api.Paths) != 2 {
		t.Errorf("api.Paths count = %d, want 2", len(api.Paths))
	}
	if api.Paths["/health"] != "Health check" {
		t.Errorf("api.Paths[/health] = %q", api.Paths["/health"])
	}

	ssh := cfg.Endpoints["git-ssh"]
	if ssh.URL != "ssh://git@localhost:2222" {
		t.Errorf("ssh.URL = %q", ssh.URL)
	}
	if len(ssh.Tags) != 2 {
		t.Errorf("ssh.Tags = %v", ssh.Tags)
	}
}

func TestEndpointsMergeOverride(t *testing.T) {
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, FileName), []byte(`endpoints:
  api:
    url: http://localhost:8080
    label: "API"
  db:
    url: localhost:5432
    label: "DB"
`), 0644)

	os.WriteFile(filepath.Join(tmpDir, "dva.override.yml"), []byte(`endpoints:
  api:
    url: http://localhost:9090
    label: "API Override"
  admin:
    url: http://localhost:3000
    label: "Admin"
`), 0644)

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if len(cfg.Endpoints) != 3 {
		t.Fatalf("expected 3 endpoints after merge, got %d", len(cfg.Endpoints))
	}
	if cfg.Endpoints["api"].URL != "http://localhost:9090" {
		t.Errorf("api.URL = %q, want override", cfg.Endpoints["api"].URL)
	}
	if cfg.Endpoints["db"].URL != "localhost:5432" {
		t.Errorf("db should be preserved from base")
	}
	if cfg.Endpoints["admin"].URL != "http://localhost:3000" {
		t.Errorf("admin should be added from override")
	}
}

func TestModeEndpointTags(t *testing.T) {
	tmpDir := t.TempDir()
	dvaYml := filepath.Join(tmpDir, FileName)

	content := `modes:
  dev:
    description: "Dev mode"
    endpoint_tags: [app, monitoring]
`
	os.WriteFile(dvaYml, []byte(content), 0644)

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	m := cfg.Modes["dev"]
	if len(m.EndpointTags) != 2 {
		t.Fatalf("expected 2 endpoint_tags, got %d", len(m.EndpointTags))
	}
	if m.EndpointTags[0] != "app" || m.EndpointTags[1] != "monitoring" {
		t.Errorf("endpoint_tags = %v", m.EndpointTags)
	}
}

func TestEmptyConfig(t *testing.T) {
	tmpDir := t.TempDir()
	dvaYml := filepath.Join(tmpDir, FileName)

	// Empty file
	os.WriteFile(dvaYml, []byte(""), 0644)

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Empty config should load without error, got: %v", err)
	}
	if cfg == nil {
		t.Fatal("Expected cfg to not be nil")
	}
}

func TestResolveEndpoints_SourceToURL(t *testing.T) {
	cfg := &Config{
		Endpoints: map[string]EndpointConfig{
			"app": {Source: "app:11700", Label: "App HTTP"},
			"db":  {Source: "postgres:15432", Label: "PostgreSQL"},
		},
	}
	cfg.ResolveEndpoints()

	if cfg.Endpoints["app"].URL != "http://localhost:11700" {
		t.Errorf("app.URL = %q, want http://localhost:11700", cfg.Endpoints["app"].URL)
	}
	if cfg.Endpoints["db"].URL != "localhost:15432" {
		t.Errorf("db.URL = %q, want localhost:15432", cfg.Endpoints["db"].URL)
	}
}

func TestResolveEndpoints_URLAlreadySet(t *testing.T) {
	cfg := &Config{
		Endpoints: map[string]EndpointConfig{
			"api": {Source: "app:8080", URL: "https://custom.dev:8080", Label: "API"},
		},
	}
	cfg.ResolveEndpoints()

	if cfg.Endpoints["api"].URL != "https://custom.dev:8080" {
		t.Errorf("should not override existing URL, got %q", cfg.Endpoints["api"].URL)
	}
}

func TestResolveEndpoints_NoSource(t *testing.T) {
	cfg := &Config{
		Endpoints: map[string]EndpointConfig{
			"api": {URL: "http://localhost:8080", Label: "API"},
		},
	}
	cfg.ResolveEndpoints()

	if cfg.Endpoints["api"].URL != "http://localhost:8080" {
		t.Errorf("should keep existing URL, got %q", cfg.Endpoints["api"].URL)
	}
}

func TestResolveEndpoints_NilEndpoints(t *testing.T) {
	cfg := &Config{}
	cfg.ResolveEndpoints() // should not panic
}

func TestResolveEndpoints_NonHTTPServices(t *testing.T) {
	tests := []struct {
		source  string
		wantURL string
	}{
		{"redis:16379", "localhost:16379"},
		{"mysql:13306", "localhost:13306"},
		{"mongo:27017", "localhost:27017"},
		{"kafka:9092", "localhost:9092"},
		{"rabbitmq:5672", "localhost:5672"},
		{"ssh:2222", "localhost:2222"},
		// Common aliases
		{"db:15432", "localhost:15432"},
		{"database:13306", "localhost:13306"},
		{"cache:16379", "localhost:16379"},
		{"mq:5672", "localhost:5672"},
		{"queue:9092", "localhost:9092"},
		{"broker:9092", "localhost:9092"},
		// HTTP services
		{"gitea:3000", "http://localhost:3000"},
		{"nginx:8080", "http://localhost:8080"},
		{"api:3000", "http://localhost:3000"},
	}
	for _, tt := range tests {
		cfg := &Config{
			Endpoints: map[string]EndpointConfig{
				"svc": {Source: tt.source, Label: "test"},
			},
		}
		cfg.ResolveEndpoints()
		if cfg.Endpoints["svc"].URL != tt.wantURL {
			t.Errorf("source=%q → URL=%q, want %q", tt.source, cfg.Endpoints["svc"].URL, tt.wantURL)
		}
	}
}

func TestResolveEndpoints_InvalidSource(t *testing.T) {
	cfg := &Config{
		Endpoints: map[string]EndpointConfig{
			"bad1": {Source: "nocolon", Label: "bad"},
			"bad2": {Source: "svc:", Label: "bad"},
		},
	}
	cfg.ResolveEndpoints()

	if cfg.Endpoints["bad1"].URL != "" {
		t.Errorf("invalid source should not resolve, got %q", cfg.Endpoints["bad1"].URL)
	}
	if cfg.Endpoints["bad2"].URL != "" {
		t.Errorf("empty port should not resolve, got %q", cfg.Endpoints["bad2"].URL)
	}
}

func TestResolveEndpoints_IntegrationWithLoad(t *testing.T) {
	tmpDir := t.TempDir()
	dvaYml := filepath.Join(tmpDir, FileName)

	content := `endpoints:
  app-http:
    source: "app:11700"
    label: "App HTTP"
    tags: [app]
    paths:
      /health: "Health check"
  app-ssh:
    url: "ssh://git@localhost:2222"
    label: "Git SSH"
    tags: [app]
  db:
    source: "postgres:15432"
    label: "PostgreSQL"
    tags: [infra]
`
	os.WriteFile(dvaYml, []byte(content), 0644)

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	// source-resolved HTTP endpoint
	app := cfg.Endpoints["app-http"]
	if app.URL != "http://localhost:11700" {
		t.Errorf("app-http.URL = %q, want http://localhost:11700", app.URL)
	}
	if app.Source != "app:11700" {
		t.Errorf("source should be preserved, got %q", app.Source)
	}

	// explicit URL not touched
	ssh := cfg.Endpoints["app-ssh"]
	if ssh.URL != "ssh://git@localhost:2222" {
		t.Errorf("app-ssh.URL = %q, should not be modified", ssh.URL)
	}

	// non-HTTP service
	db := cfg.Endpoints["db"]
	if db.URL != "localhost:15432" {
		t.Errorf("db.URL = %q, want localhost:15432", db.URL)
	}
}

func TestDefaultMode_ValidReference(t *testing.T) {
	cfg := &Config{
		DefaultMode: "dev",
		Modes: map[string]ModeConfig{
			"dev": {Description: "dev mode"},
		},
	}
	// Valid reference — no error expected from semantic check
	if _, ok := cfg.Modes[cfg.DefaultMode]; !ok {
		t.Errorf("expected default_mode '%s' to exist in modes", cfg.DefaultMode)
	}
}

func TestDefaultMode_InvalidReference(t *testing.T) {
	cfg := &Config{
		DefaultMode: "nonexistent",
		Modes: map[string]ModeConfig{
			"dev": {Description: "dev mode"},
		},
	}
	if _, ok := cfg.Modes[cfg.DefaultMode]; ok {
		t.Errorf("expected default_mode '%s' NOT to exist in modes", cfg.DefaultMode)
	}
}

func TestDefaultMode_Empty(t *testing.T) {
	// Validate() cross-checks default_mode against modes only when it is set
	// (validate.go: `if c.DefaultMode != ""`). Omitting it must therefore pass
	// even though "" is not a key in modes — drop that guard and this fails
	// with `default_mode '' not found in modes`.
	//
	// Asserting through a real load is the point. The previous version built a
	// Config literal and checked that the "" it had just assigned was still "",
	// which held no matter what the validator did and left Modes written but
	// never read — the unused write an analyzer eventually flagged.
	cfg := loadConfigForSchemaTest(t, t.TempDir(), `version: "0.1.44"
modes:
  dev:
    description: dev mode
`)
	if cfg.DefaultMode != "" {
		t.Fatalf("fixture should omit default_mode, got %q", cfg.DefaultMode)
	}
	if len(cfg.Modes) != 1 {
		t.Fatalf("fixture should define exactly 1 mode, got %d", len(cfg.Modes))
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("empty default_mode should validate, got: %v", err)
	}
}
