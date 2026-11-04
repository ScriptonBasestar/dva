package lifecycle

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func TestProcessPluginDoesNotSpawnForCancelledCaller(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := (&ProcessPlugin{}).Up(ctx, &PluginContext{
		Entry: &config.LifecycleEntry{Name: "app", Process: &config.ProcessPluginConfig{Command: ": > executed"}},
		Env:   config.NewEnvironment(nil, dir, dir), ConfigDir: dir, Logger: slog.Default(),
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled process up error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, config.DotDirName)); !os.IsNotExist(err) {
		t.Fatalf("cancelled process created state: %v", err)
	}
}

func TestOrchestratorCancelledCallerDoesNotStartOrStopEntries(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, config.FileName), []byte(`version: "0.1.0"
stack:
  app:
    default_runner: native
    runners:
      native:
        run: ": > executed"
`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	orch := NewOrchestrator(cfg, config.NewEnvironment(nil, dir, dir))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	for _, verb := range []string{"up", "stop", "restart"} {
		t.Run(verb, func(t *testing.T) {
			var err error
			switch verb {
			case "up":
				err = orch.Up(ctx, UpOptions{})
			case "stop":
				err = orch.Stop(ctx, StopOptions{})
			case "restart":
				err = orch.Restart(ctx, UpOptions{})
			}
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("cancelled %s error = %v", verb, err)
			}
			if _, err := os.Stat(filepath.Join(dir, config.DotDirName)); !os.IsNotExist(err) {
				t.Fatalf("cancelled %s created process state: %v", verb, err)
			}
		})
	}
}
