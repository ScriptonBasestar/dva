package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
	"github.com/ScriptonBasestar/dva/internal/lifecycle"
)

func TestParsePortMappingsManifestVariants(t *testing.T) {
	for _, name := range portMappingsManifestNames {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			writeFixture(t, dir, map[string]string{name: `services:
  application:
    web:
      port: 12345
      env: WEB_PORT
      description: Web UI
`})
			mappings, source, err := parsePortMappingsManifest(dir)
			if err != nil {
				t.Fatalf("parsePortMappingsManifest: %v", err)
			}
			if source != name || len(mappings) != 1 {
				t.Fatalf("parsePortMappingsManifest = (%+v, %q), want one mapping from %q", mappings, source, name)
			}
			got := mappings[0]
			if got.Name != "web" || got.Category != "application" || got.Port != 12345 || got.Env != "WEB_PORT" || got.Description != "Web UI" {
				t.Fatalf("mapping = %+v", got)
			}
		})
	}

	t.Run("canonical filename has deterministic precedence", func(t *testing.T) {
		dir := t.TempDir()
		writeFixture(t, dir, map[string]string{
			"PORT_MAPPINGS.yaml": "services: {application: {canonical: {port: 10001}}}\n",
			"port_mappings.yml":  "services: {application: {fallback: {port: 10002}}}\n",
		})
		mappings, source, err := parsePortMappingsManifest(dir)
		if err != nil {
			t.Fatalf("parsePortMappingsManifest: %v", err)
		}
		if source != "PORT_MAPPINGS.yaml" || len(mappings) != 1 || mappings[0].Name != "canonical" {
			t.Fatalf("precedence result = (%+v, %q)", mappings, source)
		}
	})
}

func TestGzGitWorkspacesBecomeSubprojects(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, map[string]string{".gz-git.yaml": `kind: workspace
workspaces:
  target-form: {targetPath: packages/target}
  path-form: {path: ./packages/path}
  default-form: {url: ssh://example/default.git}
  unsafe: {path: ../outside}
`})
	subprojects, err := parseGzGitWorkspaces(dir)
	if err != nil {
		t.Fatalf("parseGzGitWorkspaces: %v", err)
	}
	want := map[string]string{
		"target-form":  "packages/target",
		"path-form":    "packages/path",
		"default-form": "default-form",
	}
	if len(subprojects) != len(want) {
		t.Fatalf("subprojects = %+v, want %+v", subprojects, want)
	}
	for name, path := range want {
		if subprojects[name] != path {
			t.Errorf("subprojects[%q] = %q, want %q", name, subprojects[name], path)
		}
	}
}

func TestMakefileTargetsYieldNativeRunner(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, map[string]string{"Makefile": `APP_DIR := apps/web
.PHONY: dev-web build-web build test
dev-web: ## Start web
	cd $(APP_DIR) && pnpm dev
build-web: ## Build web directly
	cd $(APP_DIR) && pnpm build:entry
build: ## Build web
	cd $(APP_DIR) && pnpm build:shared
test: ## Test web
	cd $(APP_DIR) && pnpm test
`})
	discovery, err := discoverNativeScaffold(dir)
	if err != nil {
		t.Fatalf("discoverNativeScaffold: %v", err)
	}
	if len(discovery.entries) != 1 {
		t.Fatalf("entries = %+v", discovery.entries)
	}
	entry := discovery.entries[0]
	if entry.Name != "web" || entry.Dir != "apps/web" || entry.Run != "pnpm dev" || entry.Build != "pnpm build:entry" {
		t.Fatalf("native entry = %+v", entry)
	}
	if !discovery.hasTest {
		t.Fatal("declared test target was not detected")
	}
}

func TestMakefileImmediateAssignmentKeepsAssignmentTimeValue(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, map[string]string{"Makefile": `BASE := old
APP := $(BASE)
BASE := new
.PHONY: dev-app
dev-app:
	cd $(APP) && run-app
`})
	discovery, err := discoverNativeScaffold(dir)
	if err != nil {
		t.Fatalf("discoverNativeScaffold: %v", err)
	}
	if len(discovery.entries) != 1 || discovery.entries[0].Dir != "old" {
		t.Fatalf("immediate assignment resolved after reassignment: %+v", discovery.entries)
	}
}

func TestMakefileAutomaticReferenceIsRejected(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, map[string]string{"Makefile": `.PHONY: dev-app
dev-app:
	cd apps/$@ && run-app
`})
	_, err := discoverNativeScaffold(dir)
	if err == nil || !strings.Contains(err.Error(), "dev-app") || !strings.Contains(err.Error(), "$@") {
		t.Fatalf("discoverNativeScaffold error = %v, want target and automatic reference", err)
	}
}

func TestMakefileEscapedDollarPreservesShellDollar(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, map[string]string{"Makefile": `.PHONY: dev-app
dev-app:
	cd apps && echo $$APP_ENV
`})
	discovery, err := discoverNativeScaffold(dir)
	if err != nil {
		t.Fatalf("discoverNativeScaffold: %v", err)
	}
	if len(discovery.entries) != 1 || discovery.entries[0].Run != "echo $APP_ENV" {
		t.Fatalf("escaped dollar recipe = %+v, want shell dollar", discovery.entries)
	}
}

func TestScriptonDashboardFixtureYieldsTwoNativeEntries(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, scriptonDashboardRootFiles)

	created, err := scaffoldDvaYml(dir, "")
	if err != nil || !created {
		t.Fatalf("scaffoldDvaYml = (%v, %v), want (true, nil)", created, err)
	}
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	if len(cfg.Stack) != 2 || cfg.Stack["components"] == nil || cfg.Stack["dashboard"] == nil {
		t.Fatalf("stack = %+v, want components and dashboard", cfg.Stack)
	}
	if len(cfg.Plans) != 1 || cfg.Plans["dev"] == nil || len(cfg.Plans["dev"].Entries) != 2 {
		t.Fatalf("plans = %+v, want one two-entry dev plan", cfg.Plans)
	}
	if len(cfg.Endpoints) != 1 || cfg.Endpoints["dashboard"].URL != "http://localhost:11600" {
		t.Fatalf("endpoints = %+v, want dashboard on 11600", cfg.Endpoints)
	}
	if len(cfg.Subprojects) != 2 {
		t.Fatalf("subprojects = %+v, want two .gz-git workspaces", cfg.Subprojects)
	}
	generated, err := os.ReadFile(filepath.Join(dir, config.FileName))
	if err != nil {
		t.Fatalf("read generated config: %v", err)
	}
	text := string(generated)
	for _, want := range []string{"dir: \"scripton-ui-components\"", "build: \"pnpm build\"", "run: \"pnpm dev\"", "DASHBOARD_PORT", "confirm the application consumes it"} {
		if !strings.Contains(text, want) {
			t.Errorf("generated config missing %q:\n%s", want, text)
		}
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("generated config does not validate: %v", err)
	}
	if warnings := cfg.ValidateWarnings(); len(warnings) > 0 {
		t.Fatalf("generated config has semantic warnings: %v", warnings)
	}
}

func TestInitHybridIncludesDeclaredNativeEntries(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, map[string]string{
		"compose.yaml": "services: {db: {image: postgres}}\n",
		"package.json": "{}\n",
		"Makefile": `.PHONY: dev-api
dev-api: ## Start API
	cd api && go run .
test: ## Test through repository Makefile
	go test ./...
`,
	})
	created, err := scaffoldDvaYml(dir, "")
	if err != nil || !created {
		t.Fatalf("scaffoldDvaYml = (%v, %v), want (true, nil)", created, err)
	}
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	if len(cfg.Stack) != 2 || cfg.Stack["compose"] == nil || cfg.Stack["api"] == nil {
		t.Fatalf("hybrid stack = %+v, want compose and api", cfg.Stack)
	}
	if cfg.DefaultPlan() != "dev" {
		t.Fatalf("DefaultPlan() = %q, want dev", cfg.DefaultPlan())
	}
	resolved, err := lifecycle.ResolvePlan(cfg, cfg.DefaultPlan(), nil)
	if err != nil {
		t.Fatalf("ResolvePlan(DefaultPlan()): %v", err)
	}
	if len(resolved.Entries) != 2 || resolved.Entries[0].Name != "compose" || resolved.Entries[1].Name != "api" {
		t.Fatalf("resolved default plan entries = %+v, want compose then api", resolved.Entries)
	}
	if test := cfg.Interaction["test"]; test == nil || test.Command != "make test" || test.Service != "" {
		t.Fatalf("interaction.test = %+v, want direct Makefile command without template service", test)
	}
}

func TestHybridNativeComposeNameCollisionResolvesComposeRunner(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, map[string]string{
		"compose.yaml": "services: {db: {image: postgres}}\n",
		"Makefile": `.PHONY: dev-compose
dev-compose:
	./run-compose-native
`,
	})
	created, err := scaffoldDvaYml(dir, "minimal")
	if err != nil || !created {
		t.Fatalf("scaffoldDvaYml = (%v, %v), want (true, nil)", created, err)
	}
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	resolved, err := lifecycle.ResolvePlan(cfg, cfg.DefaultPlan(), nil)
	if err != nil {
		t.Fatalf("ResolvePlan(DefaultPlan()): %v", err)
	}
	if len(resolved.Entries) != 1 || resolved.Entries[0].Name != "compose" || resolved.Entries[0].Runner != "compose" {
		t.Fatalf("resolved collision = %+v, want authoritative compose runner", resolved.Entries)
	}
}
