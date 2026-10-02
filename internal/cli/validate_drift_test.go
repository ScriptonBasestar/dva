package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func TestDetectConfigDriftWarnings_ComposeFilesMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "docker-compose.yml"), []byte("services:\n  app:\n    image: nginx\n"), 0644); err != nil {
		t.Fatalf("write compose: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "docker-compose.override.yml"), []byte("services:\n  app:\n    environment:\n      FOO: bar\n"), 0644); err != nil {
		t.Fatalf("write override: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, config.FileName), []byte("version: \"0.1.0\"\nstack:\n  compose:\n    default_runner: compose\n    order: 10\n    runners:\n      compose:\n        files:\n          - docker-compose.yml\n"), 0644); err != nil {
		t.Fatalf("write dva.yml: %v", err)
	}

	c, err := config.Load(".")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	warnings := detectConfigDriftWarnings(c)
	if len(warnings) == 0 {
		t.Fatal("expected drift warning for compose.files mismatch")
	}
	if !strings.Contains(warnings[0], "compose.files") {
		t.Fatalf("unexpected warning: %s", warnings[0])
	}
}

func TestDetectConfigDriftWarnings_ModernComposeOverlaysMatch(t *testing.T) {
	tmpDir := t.TempDir()
	for _, name := range []string{"compose.yaml", "compose.tools.yaml", "compose.monitor.yaml"} {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte("services: {}\n"), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	if err := os.WriteFile(filepath.Join(tmpDir, config.FileName), []byte(`version: "0.1.44"
stack:
  compose:
    default_runner: compose
    runners:
      compose:
        files: [compose.yaml, compose.tools.yaml, compose.monitor.yaml]
`), 0644); err != nil {
		t.Fatalf("write dva.yml: %v", err)
	}

	c, err := config.Load(tmpDir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if warnings := detectConfigDriftWarnings(c); len(warnings) != 0 {
		t.Fatalf("expected no compose drift warning, got %v", warnings)
	}
}

// TestDetectConfigDriftWarnings_DetectsUnregisteredDashNamedComposeFile is TASK-316 Finding 2:
// autodiscovery previously only recognized the dotted overlay style (compose.tools.yaml), so
// a dashed file like compose-ha.yaml sat beside dva.yml completely invisible to drift
// detection. It must now be detected and, being unregistered, warned about.
func TestDetectConfigDriftWarnings_DetectsUnregisteredDashNamedComposeFile(t *testing.T) {
	tmpDir := t.TempDir()
	for _, name := range []string{"compose.yaml", "compose-ha.yaml"} {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte("services: {}\n"), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	if err := os.WriteFile(filepath.Join(tmpDir, config.FileName), []byte(`version: "0.1.44"
stack:
  compose:
    default_runner: compose
    runners:
      compose:
        files: [compose.yaml]
`), 0644); err != nil {
		t.Fatalf("write dva.yml: %v", err)
	}

	c, err := config.Load(tmpDir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	warnings := detectConfigDriftWarnings(c)
	joined := strings.Join(warnings, "\n")
	if !strings.Contains(joined, "compose-ha.yaml") || !strings.Contains(joined, "beside dva.yml") || !strings.Contains(joined, "compose.files") {
		t.Fatalf("expected unregistered compose-ha.yaml warning, got %v", warnings)
	}
}

// TestDetectConfigDriftWarnings_RegisteredDashNamedComposeFileNoWarning is the other half of
// Finding 2: once compose-ha.yaml is registered under runners.compose.files, it must not
// produce a warning either — the old symmetric sameStringSlice comparison flagged exactly this
// case (declared but, before the prefix fix, never autodiscovered) as drift.
func TestDetectConfigDriftWarnings_RegisteredDashNamedComposeFileNoWarning(t *testing.T) {
	tmpDir := t.TempDir()
	for _, name := range []string{"compose.yaml", "compose-ha.yaml"} {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte("services: {}\n"), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	if err := os.WriteFile(filepath.Join(tmpDir, config.FileName), []byte(`version: "0.1.44"
stack:
  compose:
    default_runner: compose
    runners:
      compose:
        files: [compose.yaml, compose-ha.yaml]
`), 0644); err != nil {
		t.Fatalf("write dva.yml: %v", err)
	}

	c, err := config.Load(tmpDir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if warnings := detectConfigDriftWarnings(c); len(warnings) != 0 {
		t.Fatalf("expected registered compose-ha.yaml to produce no drift warning, got %v", warnings)
	}
}

// TestDetectConfigDriftWarnings_DetectsUnregisteredSubdirectoryComposeFile is TASK-316 Finding
// 3: the scan set previously never left the dva.yml directory, so a project keeping its
// compose corpus under a subdirectory (env/docker-compose/, as in the primeno1 dogfood
// evidence) never had that subdirectory's other files checked. Registering one file there
// must widen the scan to that directory, surfacing an unregistered sibling.
func TestDetectConfigDriftWarnings_DetectsUnregisteredSubdirectoryComposeFile(t *testing.T) {
	tmpDir := t.TempDir()
	composeDir := filepath.Join(tmpDir, "env", "docker-compose")
	if err := os.MkdirAll(composeDir, 0755); err != nil {
		t.Fatalf("mkdir env/docker-compose: %v", err)
	}
	for _, name := range []string{"docker-compose.yml", "docker-compose.verify.yml"} {
		if err := os.WriteFile(filepath.Join(composeDir, name), []byte("services: {}\n"), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	if err := os.WriteFile(filepath.Join(tmpDir, config.FileName), []byte(`version: "0.1.44"
stack:
  compose:
    default_runner: compose
    runners:
      compose:
        files: [env/docker-compose/docker-compose.yml]
`), 0644); err != nil {
		t.Fatalf("write dva.yml: %v", err)
	}

	c, err := config.Load(tmpDir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	warnings := detectConfigDriftWarnings(c)
	joined := strings.Join(warnings, "\n")
	if !strings.Contains(joined, "docker-compose.verify.yml") || !strings.Contains(joined, "in env/docker-compose/") {
		t.Fatalf("expected unregistered subdirectory compose file warning, got %v", warnings)
	}
}

func TestDetectConfigDriftWarnings_DetectsUnregisteredComposeFileBesideIncludedFile(t *testing.T) {
	// flow-pipechain shape: the root compose file is a shim that only `include:`s the real
	// file from a subdirectory. That subdirectory is in scan scope because the include chain
	// reaches it, so an unregistered sibling there is drift (TASK-316 Finding 3).
	tmpDir := t.TempDir()
	deployDir := filepath.Join(tmpDir, "deploy", "local")
	if err := os.MkdirAll(deployDir, 0755); err != nil {
		t.Fatalf("mkdir deploy/local: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "compose.yaml"), []byte("include:\n  - deploy/local/docker-compose.yml\n"), 0644); err != nil {
		t.Fatalf("write root shim: %v", err)
	}
	for _, name := range []string{"docker-compose.yml", "docker-compose.extra.yml"} {
		if err := os.WriteFile(filepath.Join(deployDir, name), []byte("services: {}\n"), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	if err := os.WriteFile(filepath.Join(tmpDir, config.FileName), []byte(`version: "0.1.44"
stack:
  compose:
    default_runner: compose
    runners:
      compose:
        files: [compose.yaml]
`), 0644); err != nil {
		t.Fatalf("write dva.yml: %v", err)
	}

	c, err := config.Load(tmpDir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	warnings := detectConfigDriftWarnings(c)
	joined := strings.Join(warnings, "\n")
	if !strings.Contains(joined, "docker-compose.extra.yml") || !strings.Contains(joined, "in deploy/local/") {
		t.Fatalf("expected unregistered compose file warning beside the included file, got %v", warnings)
	}
	if strings.Contains(joined, "docker-compose.yml,") || strings.Contains(joined, " docker-compose.yml ") {
		t.Fatalf("include-reached file must not be reported as unregistered, got %v", warnings)
	}
}

func TestDetectConfigDriftWarnings_IgnoresConfiguredSubdirectoryComposeFiles(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(tmpDir, "compose"), 0755); err != nil {
		t.Fatalf("mkdir compose: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "compose.yaml"), []byte("services: {}\n"), 0644); err != nil {
		t.Fatalf("write root compose: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "compose", "minor-guardian-e2e.yaml"), []byte("services: {}\n"), 0644); err != nil {
		t.Fatalf("write isolated compose: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, config.FileName), []byte(`version: "0.1.44"
stack:
  minor-guardian-e2e-infra:
    default_runner: compose
    runners:
      compose:
        files: [compose/minor-guardian-e2e.yaml]
        project_name: isolated-e2e
  compose:
    default_runner: compose
    runners:
      compose:
        files: [compose.yaml]
        project_name: app
plans:
  local-infra:
    entries:
      - name: compose
`), 0644); err != nil {
		t.Fatalf("write dva.yml: %v", err)
	}

	c, err := config.Load(tmpDir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if warnings := detectConfigDriftWarnings(c); len(warnings) != 0 {
		t.Fatalf("expected no compose drift warning for configured subdirectory file, got %v", warnings)
	}
}

func TestDetectConfigDriftWarnings_MissingConfiguredRootComposeFile(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, config.FileName), []byte(`version: "0.1.44"
stack:
  compose:
    default_runner: compose
    runners:
      compose:
        files: [compose.yaml]
`), 0644); err != nil {
		t.Fatalf("write dva.yml: %v", err)
	}

	c, err := config.Load(tmpDir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	// Finding 2 (TASK-316): a configured file that does not exist is not "unregistered" —
	// missingConfiguredComposeFiles already reports it below, and detectUnregisteredComposeFileWarnings
	// has nothing to report since autodiscovery finds nothing in an empty directory.
	warnings := detectConfigDriftWarnings(c)
	if len(warnings) != 1 || !strings.Contains(warnings[0], `compose file "compose.yaml" is configured by dva.yml but does not exist`) {
		t.Fatalf("expected missing root compose drift warning, got %v", warnings)
	}
}

func TestDetectConfigDriftWarnings_MissingConfiguredSubdirectoryAndAbsoluteComposeFiles(t *testing.T) {
	tmpDir := t.TempDir()
	missingAbsolute := filepath.Join(t.TempDir(), "missing-absolute.yaml")
	if err := os.Mkdir(filepath.Join(tmpDir, "compose"), 0755); err != nil {
		t.Fatalf("mkdir compose: %v", err)
	}
	dvaConfig := fmt.Sprintf(`version: "0.1.44"
stack:
  subdirectory:
    default_runner: compose
    runners:
      compose:
        files: [compose/missing.yaml, compose/missing.yaml]
  absolute:
    default_runner: compose
    runners:
      compose:
        files: [%q]
`, missingAbsolute)
	if err := os.WriteFile(filepath.Join(tmpDir, config.FileName), []byte(dvaConfig), 0644); err != nil {
		t.Fatalf("write dva.yml: %v", err)
	}

	c, err := config.Load(tmpDir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	warnings := detectConfigDriftWarnings(c)
	if len(warnings) != 2 {
		t.Fatalf("expected one warning per missing resolved path, got %v", warnings)
	}
	if !strings.Contains(strings.Join(warnings, "\n"), `compose file "compose/missing.yaml" is configured by dva.yml but does not exist`) {
		t.Fatalf("missing subdirectory compose warning not found: %v", warnings)
	}
	if !strings.Contains(strings.Join(warnings, "\n"), missingAbsolute) {
		t.Fatalf("missing absolute compose warning not found: %v", warnings)
	}
}

func TestDetectConfigDriftWarnings_ResolvesPathSourcedComposeFilesPerEntry(t *testing.T) {
	tmpDir := t.TempDir()
	for _, source := range []string{"source-a", "source-b"} {
		if err := os.Mkdir(filepath.Join(tmpDir, source), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", source, err)
		}
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "source-a", "compose.yaml"), []byte("services: {}\n"), 0644); err != nil {
		t.Fatalf("write source-a compose: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, config.FileName), []byte(`version: "0.1.44"
stack:
  source-a:
    default_runner: compose
    source:
      path: source-a
    runners:
      compose:
        files: [compose.yaml]
  source-b:
    default_runner: compose
    source:
      path: source-b
    runners:
      compose:
        files: [compose.yaml]
`), 0644); err != nil {
		t.Fatalf("write dva.yml: %v", err)
	}

	c, err := config.Load(tmpDir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	warnings := detectConfigDriftWarnings(c)
	if len(warnings) != 1 {
		t.Fatalf("expected missing compose only for source-b, got %v", warnings)
	}
	if !strings.Contains(warnings[0], `compose file "compose.yaml (stack entry \"source-b\")"`) {
		t.Fatalf("missing source-b warning not found: %v", warnings)
	}
}

func TestDetectConfigDriftWarnings_DefersUnavailableGitSource(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, config.FileName), []byte(`version: "0.1.44"
stack:
  remote:
    default_runner: compose
    source:
      git: https://example.com/shared-infra.git
      ref: v1
    runners:
      compose:
        files: [compose.yaml]
`), 0644); err != nil {
		t.Fatalf("write dva.yml: %v", err)
	}

	c, err := config.Load(tmpDir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if warnings := detectConfigDriftWarnings(c); len(warnings) != 0 {
		t.Fatalf("expected unavailable git source to defer to source readiness, got %v", warnings)
	}
}

func TestDetectConfigDriftWarnings_SourceStackFixtureHasNoFalseComposeWarnings(t *testing.T) {
	fixture := filepath.Join("..", "integration", "testdata", "fixtures", "source-stack")
	c, err := config.Load(fixture)
	if err != nil {
		t.Fatalf("load source-stack fixture: %v", err)
	}
	if warnings := detectConfigDriftWarnings(c); len(warnings) != 0 {
		t.Fatalf("expected no root or missing compose warnings for source-stack fixture, got %v", warnings)
	}
}

func TestDetectConfigDriftWarnings_ResolvesComposeFilesWithEntryVars(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "compose.dev.yml"), []byte("services:\n  api:\n    image: nginx\n"), 0644); err != nil {
		t.Fatalf("write entry-resolved compose: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, config.FileName), []byte(`version: "0.1.44"
stack:
  compose:
    default_runner: compose
    vars:
      STAGE: dev
    runners:
      compose:
        files: ["compose.${STAGE}.yml"]
interaction:
  api-shell:
    service: api
    command: sh
`), 0644); err != nil {
		t.Fatalf("write dva.yml: %v", err)
	}

	c, err := config.Load(tmpDir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if warnings := detectConfigDriftWarnings(c); len(warnings) != 0 {
		t.Fatalf("expected entry vars to resolve compose path and service discovery, got %v", warnings)
	}
	services, complete := configuredComposeServices(c)
	if !complete || !services["api"] {
		t.Fatal("service discovery did not use entry-resolved compose file")
	}
}

func TestDetectConfigDriftWarnings_DefersPlanOrSiteDrivenComposePath(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "compose.yaml"), []byte("services: {}\n"), 0644); err != nil {
		t.Fatalf("write root compose: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, config.FileName), []byte(`version: "0.1.44"
stack:
  compose:
    default_runner: compose
    runners:
      compose:
        files: ["compose.${STAGE}.yml"]
plans:
  local:
    site: local
    entries:
      - name: compose
        vars:
          STAGE: dev
sites:
  local:
    vars:
      STAGE: dev
`), 0644); err != nil {
		t.Fatalf("write dva.yml: %v", err)
	}

	c, err := config.Load(tmpDir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if warnings := detectConfigDriftWarnings(c); len(warnings) != 0 {
		t.Fatalf("expected unresolved plan/site path to defer static drift checks, got %v", warnings)
	}
}

func TestMissingConfiguredComposeFiles_DeduplicatesEntryResolvedPath(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, config.FileName), []byte(`version: "0.1.44"
stack:
  compose:
    default_runner: compose
    vars:
      STAGE: dev
    runners:
      compose:
        files: ["compose.${STAGE}.yml", compose.dev.yml]
`), 0644); err != nil {
		t.Fatalf("write dva.yml: %v", err)
	}

	c, err := config.Load(tmpDir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	missing := missingConfiguredComposeFiles(c)
	if len(missing) != 1 || missing[0] != "compose.${STAGE}.yml" {
		t.Fatalf("expected one missing resolved path, got %v", missing)
	}
}

func TestDetectConfigDriftWarnings_DefersServiceComparisonForDynamicPlanCorpus(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "compose.yaml"), []byte("services:\n  postgres:\n    image: postgres\n"), 0644); err != nil {
		t.Fatalf("write root compose: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, config.FileName), []byte(`version: "0.1.44"
stack:
  postgres:
    default_runner: compose
    runners:
      compose:
        files: [compose.yaml]
  api:
    default_runner: compose
    runners:
      compose:
        files: ["compose.${STAGE}.yaml"]
plans:
  local:
    site: local
    entries:
      - name: postgres
      - name: api
        vars:
          STAGE: dev
sites:
  local:
    vars:
      STAGE: dev
interaction:
  api-shell:
    service: api
    command: sh
`), 0644); err != nil {
		t.Fatalf("write dva.yml: %v", err)
	}

	c, err := config.Load(tmpDir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if warnings := detectConfigDriftWarnings(c); len(warnings) != 0 {
		t.Fatalf("expected incomplete plan corpus to defer api service comparison, got %v", warnings)
	}
}

func TestDetectConfigDriftWarnings_DefersServiceComparisonForUnavailableSource(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "compose.yaml"), []byte("services:\n  postgres:\n    image: postgres\n"), 0644); err != nil {
		t.Fatalf("write root compose: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, config.FileName), []byte(`version: "0.1.44"
stack:
  postgres:
    default_runner: compose
    runners:
      compose:
        files: [compose.yaml]
  source-api:
    default_runner: compose
    source:
      git: https://example.com/api.git
      ref: v1
    runners:
      compose:
        files: [compose.yaml]
interaction:
  api-shell:
    service: api
    command: sh
`), 0644); err != nil {
		t.Fatalf("write dva.yml: %v", err)
	}

	c, err := config.Load(tmpDir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if warnings := detectConfigDriftWarnings(c); len(warnings) != 0 {
		t.Fatalf("expected unavailable source corpus to defer api service comparison, got %v", warnings)
	}
}

func TestDetectConfigDriftWarnings_CompleteCorpusStillReportsMissingService(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "compose.yaml"), []byte("services:\n  postgres:\n    image: postgres\n"), 0644); err != nil {
		t.Fatalf("write root compose: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, config.FileName), []byte(`version: "0.1.44"
stack:
  postgres:
    default_runner: compose
    runners:
      compose:
        files: [compose.yaml]
interaction:
  api-shell:
    service: api
    command: sh
`), 0644); err != nil {
		t.Fatalf("write dva.yml: %v", err)
	}

	c, err := config.Load(tmpDir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	warnings := detectConfigDriftWarnings(c)
	if len(warnings) != 1 || !strings.Contains(warnings[0], `references compose service "api"`) {
		t.Fatalf("expected complete corpus missing-service warning, got %v", warnings)
	}
}

func TestDetectConfigDriftWarnings_MissingService(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "docker-compose.yml"), []byte("services:\n  web:\n    image: nginx\n"), 0644); err != nil {
		t.Fatalf("write compose: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, config.FileName), []byte("version: \"0.1.0\"\nstack:\n  compose:\n    default_runner: compose\n    order: 10\n    runners:\n      compose:\n        files:\n          - docker-compose.yml\ninteraction:\n  test:\n    service: app\n    command: go test ./...\n"), 0644); err != nil {
		t.Fatalf("write dva.yml: %v", err)
	}

	c, err := config.Load(".")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	warnings := detectConfigDriftWarnings(c)
	if len(warnings) == 0 {
		t.Fatal("expected service drift warning")
	}

	found := false
	for _, warning := range warnings {
		if strings.Contains(warning, `references compose service "app"`) {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected missing-service warning, got: %v", warnings)
	}
}
