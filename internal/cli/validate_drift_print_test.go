package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func TestPrintConfigDriftWarnings(t *testing.T) {
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

	var buf bytes.Buffer
	printConfigDriftWarnings(&buf, detectConfigDriftWarnings(c))
	output := buf.String()
	if !strings.Contains(output, "[warn] config drift:") {
		t.Fatalf("expected config drift warning prefix, got: %s", output)
	}
}
