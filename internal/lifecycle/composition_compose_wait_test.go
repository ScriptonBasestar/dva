package lifecycle

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// A composition child's DVA health checks are gated at the wave boundary, which skips
// entries without any. Each compose entry must therefore keep Compose's --wait exactly when
// it declares no DVA health checks, decided per entry so a child mixing both is covered, and
// the composition's --no-wait must drop it.
func TestCompositionUpKeepsComposeWaitOnlyForEntriesWithoutHealthChecks(t *testing.T) {
	tests := []struct {
		name     string
		dbHealth bool
		noWait   bool
		wantWait map[string]bool
	}{
		{name: "no health checks", wantWait: map[string]bool{"db": true, "cache": true}},
		{name: "mixed child", dbHealth: true, wantWait: map[string]bool{"db": false, "cache": true}},
		{name: "no-wait", noWait: true, wantWait: map[string]bool{"db": false, "cache": false}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := loadComposeWaitConfig(t, tt.dbHealth)
			logs := captureDefaultSlog(t)

			if err := dryRunCompositionUp(cfg, "dev", tt.noWait); err != nil {
				t.Fatalf("composition up: %v", err)
			}

			for entry, want := range tt.wantWait {
				line := composeUpLogLine(t, logs.String(), entry)
				if got := strings.Contains(line, "--wait"); got != want {
					t.Errorf("entry %s: compose --wait = %v, want %v\n%s", entry, got, want, line)
				}
			}
		})
	}
}

func dryRunCompositionUp(cfg *config.Config, plan string, noWait bool) error {
	comp, err := ResolveCompositionPlan(cfg, plan)
	if err != nil {
		return err
	}
	exec := &PlanChildExecutor{
		DryRun: true,
		NoWait: noWait,
		Environment: func(child *ExecutionPlan) (*config.Config, *config.Environment, error) {
			return cfg, config.NewEnvironment(nil, cfg.FileDir(), cfg.FileDir()), nil
		},
	}
	for _, entry := range comp.Entries {
		if err := exec.Up(context.Background(), entry.ChildPlan); err != nil {
			return err
		}
	}
	return nil
}

// composeUpLogLine returns the dry-run line of entry's compose up, failing when there is not
// exactly one.
func composeUpLogLine(t *testing.T, logs, entry string) string {
	t.Helper()
	var found []string
	for line := range strings.SplitSeq(logs, "\n") {
		if strings.Contains(line, "entry="+entry+" ") && strings.Contains(line, " up -d") {
			found = append(found, line)
		}
	}
	if len(found) != 1 {
		t.Fatalf("entry %s: want one compose up line, got %d:\n%s", entry, len(found), logs)
	}
	return found[0]
}

func loadComposeWaitConfig(t *testing.T, dbHealth bool) *config.Config {
	t.Helper()
	dir := t.TempDir()
	compose := "name: compose-wait-proof\nservices:\n  postgres: { image: busybox:1.36 }\n  redis: { image: busybox:1.36 }\n"
	if err := os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte(compose), 0o644); err != nil {
		t.Fatal(err)
	}
	health := ""
	if dbHealth {
		health = `
    health_checks:
      pg:
        type: tcp
        address: 127.0.0.1:1
`
	}
	body := `version: "0.1.44"
stack:
  db:
    default_runner: compose
` + health + `    runners:
      compose:
        files: [compose.yaml]
        project_name: compose-wait-proof
        services:
          postgres: {}
  cache:
    default_runner: compose
    runners:
      compose:
        files: [compose.yaml]
        project_name: compose-wait-proof
        services:
          redis: {}
  api:
    default_runner: native
    runners:
      native:
        run: "true"
plans:
  infra:
    entries:
      - name: db
        runner: compose
        services: [postgres]
      - name: cache
        runner: compose
        services: [redis]
  api:
    entries:
      - name: api
        runner: native
  dev:
    composes:
      - plan: infra
        order: 0
      - plan: api
        order: 1
        depends_on: [infra]
`
	if err := os.WriteFile(filepath.Join(dir, config.FileName), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func captureDefaultSlog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(old) })
	return &buf
}
