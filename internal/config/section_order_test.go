package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCanonicalOrder_Correct(t *testing.T) {
	content := `version: "0.1.29"
env_file: .env
stack:
  compose:
    order: 10
modes:
  dev:
    description: Dev
health_checks:
  app:
    type: http
interaction:
  test:
    command: make test
provision:
  default:
    - step: Setup
      run: make setup
`
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	warnings := validateCanonicalOrder(path)
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings for correct order, got %d: %v", len(warnings), warnings)
	}
}

func TestCanonicalOrder_Wrong(t *testing.T) {
	// interaction before stack → out of order
	content := `version: "0.1.29"
interaction:
  test:
    command: make test
stack:
  compose:
    order: 10
`
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	warnings := validateCanonicalOrder(path)
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning for wrong order, got %d", len(warnings))
	}
	if !strings.Contains(warnings[0], "section order") {
		t.Errorf("unexpected warning: %s", warnings[0])
	}
}

func TestCanonicalOrder_SingleSection(t *testing.T) {
	content := `version: "0.1.29"
`
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	warnings := validateCanonicalOrder(path)
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings for single section, got %d", len(warnings))
	}
}
