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
