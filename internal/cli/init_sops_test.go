package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// TestInitSopsDeclaresCandidate is TASK-441's core contract: a sops-encrypted
// candidate at the init target root becomes a declared env_file entry, with
// sops_source naming the file that actually exists and path naming the
// plaintext DVA derives from it.
func TestInitSopsDeclaresCandidate(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, map[string]string{
		"compose.yaml": "services: {db: {image: postgres}}\n",
		".env.enc":     "ENC",
	})

	created, err := scaffoldDvaYml(dir, "")
	if err != nil || !created {
		t.Fatalf("scaffoldDvaYml() = (%v, %v), want (true, nil)", created, err)
	}

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	entries := cfg.AllEnvFileConfigs()
	if len(entries) != 1 {
		t.Fatalf("env_file entries = %+v, want exactly one", entries)
	}
	if entries[0].Path != ".env" || entries[0].SopsSource != ".env.enc" {
		t.Fatalf("env_file entry = %+v, want {path: .env, sops_source: .env.enc}", entries[0])
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("generated config does not validate: %v", err)
	}

	generated, err := os.ReadFile(filepath.Join(dir, config.FileName))
	if err != nil {
		t.Fatalf("read generated config: %v", err)
	}
	text := string(generated)
	if !strings.Contains(text, "env_file:") || !strings.Contains(text, "sops_source: \".env.enc\"") {
		t.Fatalf("generated dva.yml missing env_file declaration:\n%s", text)
	}
}

// TestInitSopsCreationRulesOnlyAddsNothing is the decided-design guardrail: a
// bare .sops.yaml with no matching candidate file names nothing that exists,
// so init must not invent a sops_source pointing at a file it never saw.
func TestInitSopsCreationRulesOnlyAddsNothing(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, map[string]string{
		"compose.yaml": "services: {db: {image: postgres}}\n",
		".sops.yaml":   "creation_rules: []\n",
	})

	created, err := scaffoldDvaYml(dir, "")
	if err != nil || !created {
		t.Fatalf("scaffoldDvaYml() = (%v, %v), want (true, nil)", created, err)
	}

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	if entries := cfg.AllEnvFileConfigs(); len(entries) != 0 {
		t.Fatalf("env_file entries = %+v, want none (.sops.yaml alone is not a candidate)", entries)
	}

	generated, err := os.ReadFile(filepath.Join(dir, config.FileName))
	if err != nil {
		t.Fatalf("read generated config: %v", err)
	}
	if strings.Contains(string(generated), "env_file:") {
		t.Fatalf("generated dva.yml must not declare env_file from creation rules alone:\n%s", generated)
	}
}

// TestInitSopsNoEvidence is the regression guard: a directory with no sops
// evidence at all gets no env_file entry, exactly as before TASK-441.
func TestInitSopsNoEvidence(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, map[string]string{
		"compose.yaml": "services: {db: {image: postgres}}\n",
	})

	created, err := scaffoldDvaYml(dir, "")
	if err != nil || !created {
		t.Fatalf("scaffoldDvaYml() = (%v, %v), want (true, nil)", created, err)
	}

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	if entries := cfg.AllEnvFileConfigs(); len(entries) != 0 {
		t.Fatalf("env_file entries = %+v, want none", entries)
	}
}

// TestInitSopsNativeOnlyKeepsGuidanceAndDeclaresEnvFile is the review-fix
// regression check for finding 1a: a language manifest with no declared
// Makefile targets and no Compose file must still get
// generateNativeOnlyConfigIn's "no stack entry generated" guidance — sops
// evidence must not switch it to the discovered-config writer's unrelated "0
// declared Makefile native entries" framing — but the env_file entry must
// still be declared.
func TestInitSopsNativeOnlyKeepsGuidanceAndDeclaresEnvFile(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, map[string]string{
		"go.mod":   "module example.com/x\n",
		".env.enc": "ENC",
	})

	var created bool
	var err error
	stdout := captureStdout(t, func() {
		created, err = scaffoldDvaYml(dir, "")
	})
	if err != nil || !created {
		t.Fatalf("scaffoldDvaYml() = (%v, %v), want (true, nil)", created, err)
	}
	if !strings.Contains(stdout, "no stack entry generated") {
		t.Fatalf("announcement must keep the native-only guidance, got:\n%s", stdout)
	}

	generated, err := os.ReadFile(filepath.Join(dir, config.FileName))
	if err != nil {
		t.Fatalf("read generated config: %v", err)
	}
	text := string(generated)
	if !strings.Contains(text, "no `stack:` entry was generated") {
		t.Fatalf("generated dva.yml must keep the native-only guidance comment, got:\n%s", text)
	}

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	entries := cfg.AllEnvFileConfigs()
	if len(entries) != 1 || entries[0].Path != ".env" || entries[0].SopsSource != ".env.enc" {
		t.Fatalf("env_file entries = %+v, want [{path: .env, sops_source: .env.enc}]", entries)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("generated config does not validate: %v", err)
	}
}

// TestInitSopsLoneCandidateStillErrors is the review-fix regression check for
// finding 1b: a directory with only a sops candidate — no Compose file, no
// language manifest, no other discovered evidence — must still refuse to
// create dva.yml exactly as before TASK-441. Sops evidence alone does not
// justify scaffolding a config that would otherwise be empty.
func TestInitSopsLoneCandidateStillErrors(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, map[string]string{".env.enc": "ENC"})

	created, err := scaffoldDvaYml(dir, "")
	if created {
		t.Fatal("scaffoldDvaYml() created a config from sops evidence alone")
	}
	if !strings.Contains(err.Error(), errComposeFileNotFound.Error()) {
		t.Fatalf("scaffoldDvaYml() error = %v, want it to name %v", err, errComposeFileNotFound)
	}
	if _, statErr := os.Stat(filepath.Join(dir, config.FileName)); !os.IsNotExist(statErr) {
		t.Fatalf("dva.yml was created or could not be checked: %v", statErr)
	}
}

// TestInitSopsCandidateEqualsTargetFallsBackToEnv covers the derivation
// safety net: a candidate whose name is nothing but the sops suffix (".sops")
// strips down to an empty target, which discoverSopsEnvFileEntry must not
// declare as the path — it falls back to ".env" instead.
func TestInitSopsCandidateEqualsTargetFallsBackToEnv(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, map[string]string{
		"compose.yaml": "services: {db: {image: postgres}}\n",
		".sops":        "ENC",
	})

	created, err := scaffoldDvaYml(dir, "")
	if err != nil || !created {
		t.Fatalf("scaffoldDvaYml() = (%v, %v), want (true, nil)", created, err)
	}

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	entries := cfg.AllEnvFileConfigs()
	if len(entries) != 1 || entries[0].Path != ".env" || entries[0].SopsSource != ".sops" {
		t.Fatalf("env_file entries = %+v, want [{path: .env, sops_source: .sops}]", entries)
	}
}

// TestInitSopsBaseEnvFileWins is the review-fix check for finding 2's "base
// template env_file wins" requirement. No shipped init template declares
// env_file today, so this exercises mergeInitDiscovery directly — the same
// function every writing path in init_scaffold.go calls — to prove the rule
// mergeInitDiscovery already has: a base mapping that already declares
// env_file is left alone, because the merge loop only replaces a key whose
// base and overlay values are both YAML mappings, and env_file's value is
// always a sequence.
func TestInitSopsBaseEnvFileWins(t *testing.T) {
	base := "version: \"0.1.45\"\nenv_file:\n  - path: .env\n    sops_source: base.enc\n"
	discovered := "version: \"0.1.45\"\n\nenv_file:\n  - path: .env\n    sops_source: discovered.enc\n"

	merged, err := mergeInitDiscovery(base, discovered)
	if err != nil {
		t.Fatalf("mergeInitDiscovery: %v", err)
	}
	if !strings.Contains(merged, "base.enc") {
		t.Fatalf("base env_file declaration must win, got:\n%s", merged)
	}
	if strings.Contains(merged, "discovered.enc") {
		t.Fatalf("discovered env_file declaration must not override the base one, got:\n%s", merged)
	}
}
