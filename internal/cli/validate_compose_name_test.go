package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func TestPrintComposeNameWarnings_Missing(t *testing.T) {
	warnings := []config.ComposeNameWarning{
		{File: "compose.yml", DvaName: "myproject", ComposeName: ""},
	}

	var buf bytes.Buffer
	printComposeNameWarnings(&buf, warnings)
	output := buf.String()

	if !strings.Contains(output, "missing top-level") {
		t.Errorf("output should mention 'missing top-level', got: %s", output)
	}
	if !strings.Contains(output, "myproject") {
		t.Errorf("output should contain project name 'myproject', got: %s", output)
	}
}

func TestPrintComposeNameWarnings_Mismatch(t *testing.T) {
	warnings := []config.ComposeNameWarning{
		{File: "compose.yml", DvaName: "myproject", ComposeName: "old-name"},
	}

	var buf bytes.Buffer
	printComposeNameWarnings(&buf, warnings)
	output := buf.String()

	if !strings.Contains(output, "differs from") {
		t.Errorf("output should mention 'differs from', got: %s", output)
	}
	if !strings.Contains(output, "old-name") {
		t.Errorf("output should contain old name, got: %s", output)
	}
}

func TestPrintComposeNameWarnings_Empty(t *testing.T) {
	var buf bytes.Buffer
	printComposeNameWarnings(&buf, nil)
	if buf.Len() > 0 {
		t.Errorf("expected no output for empty warnings, got: %s", buf.String())
	}
}

func TestFixComposeNameWarnings_AddsMissingName(t *testing.T) {
	c := loadTestConfig(t, "version: \"0.1.22\"\nstack:\n  compose:\n    default_runner: compose\n    order: 10\n    runners:\n      compose:\n        project_name: myproject\n        files: [compose.yml]\n")

	// Create compose file without name
	composePath := filepath.Join(c.FileDir(), "compose.yml")
	os.WriteFile(composePath, []byte("services:\n  web:\n    image: nginx\n"), 0644)

	warnings := []config.ComposeNameWarning{
		{File: composePath, DvaName: "myproject", ComposeName: ""},
	}

	fixComposeNameWarnings(c, warnings)

	// Verify compose file was updated
	data, _ := os.ReadFile(composePath)
	if !strings.Contains(string(data), "name:") {
		t.Error("compose file should now contain 'name:' field")
	}
}

func TestFixComposeNameWarnings_Empty(t *testing.T) {
	c := &config.Config{}
	// Should not panic with empty warnings
	fixComposeNameWarnings(c, nil)
}
