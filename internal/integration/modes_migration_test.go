//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func runModeMigrationCLI(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	binary := dvaBinary(t)
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "DVA_FILE="+filepath.Join(dir, "dva.yml"))
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestModesMigrationRealBinary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dva.yml")
	source := "version: '0.1.44'\nstack:\n  probe:\n    default_runner: script\n    runners:\n      script: {command: 'true'}\nmodes:\n  dev: {stack: [probe]}\ndefault_mode: dev\n"
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	if out, err := runModeMigrationCLI(t, dir, "config", "validate"); err == nil || !strings.Contains(out, "removed in v0.5.0") {
		t.Fatalf("legacy config accepted: err=%v output=%s", err, out)
	}
	if out, err := runModeMigrationCLI(t, dir, "config", "migrate", "--write"); err != nil {
		t.Fatalf("raw legacy input could not be migrated: %v\n%s", err, out)
	}
	out, err := runModeMigrationCLI(t, dir, "manifest", "-f", "json")
	if err != nil {
		t.Fatalf("migrated config failed normal CLI loading: %v\n%s", err, out)
	}
	var manifest struct {
		DefaultPlan string                     `json:"default_plan"`
		Plans       map[string]json.RawMessage `json:"plans"`
	}
	if err := json.Unmarshal([]byte(out), &manifest); err != nil || manifest.DefaultPlan != "dev" || manifest.Plans["dev"] == nil {
		t.Fatalf("migrated manifest is not a dev plan: %v\n%s", err, out)
	}
	for _, verb := range []string{"up", "down", "stop", "restart", "build"} {
		if out, err := runModeMigrationCLI(t, dir, verb, "--mode=dev"); err == nil || !strings.Contains(out, "removed in v0.5.0") {
			t.Fatalf("%s accepted removed selector: err=%v output=%s", verb, err, out)
		}
	}
}

func TestPartialModesMigrationRealBinaryPreservesOriginal(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dva.yml")
	source := "version: '0.1.44'\nmodes:\n  dev: {compose_profiles: [tools]}\n"
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	if out, err := runModeMigrationCLI(t, dir, "config", "migrate", "--write"); err == nil || !strings.Contains(out, "nothing was written") || !strings.Contains(out, "modes.dev") {
		t.Fatalf("partial migration lost failure/guidance: err=%v output=%s", err, out)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != source {
		t.Fatalf("partial migration changed its source: %v", err)
	}
}
