//go:build !windows

package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func TestBackendStatusQueriesCancelTheirProcessGroups(t *testing.T) {
	type backend struct {
		name, executable string
		status           func(context.Context, *PluginContext) ([]ServiceStatus, error)
		fallbackStopped  bool
	}

	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.Mkdir(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(root, "entered")
	childPID := filepath.Join(root, "child.pid")
	writeStatusQueryFake(t, bin, []string{"docker", "podman-compose", "kubectl", "helm", "vagrant", "multipass"})
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("DVA_STATUS_MARKER", marker)
	t.Setenv("DVA_STATUS_CHILD_PID", childPID)
	t.Setenv("DVA_STATUS_MODE", "block")

	env := config.NewEnvironment(nil, root, root)
	contextFor := func(entry *config.LifecycleEntry) *PluginContext {
		return &PluginContext{Entry: entry, Env: env, ConfigDir: root}
	}
	backends := []backend{
		{name: "compose", executable: "docker", status: func(ctx context.Context, pctx *PluginContext) ([]ServiceStatus, error) {
			return (&ComposePlugin{}).Status(ctx, pctx)
		}},
		{name: "podman-compose", executable: "podman-compose", status: func(ctx context.Context, pctx *PluginContext) ([]ServiceStatus, error) {
			return (&PodmanComposePlugin{}).Status(ctx, pctx)
		}},
		{name: "docker", executable: "docker", fallbackStopped: true, status: func(ctx context.Context, pctx *PluginContext) ([]ServiceStatus, error) {
			return (&DockerPlugin{}).Status(ctx, pctx)
		}},
		{name: "kubectl", executable: "kubectl", status: func(ctx context.Context, pctx *PluginContext) ([]ServiceStatus, error) {
			return (&KubectlPlugin{}).Status(ctx, pctx)
		}},
		{name: "helm", executable: "helm", fallbackStopped: true, status: func(ctx context.Context, pctx *PluginContext) ([]ServiceStatus, error) {
			return (&HelmPlugin{}).Status(ctx, pctx)
		}},
		{name: "kustomize", executable: "kubectl", status: func(ctx context.Context, pctx *PluginContext) ([]ServiceStatus, error) {
			return (&KustomizePlugin{}).Status(ctx, pctx)
		}},
		{name: "vagrant", executable: "vagrant", status: func(ctx context.Context, pctx *PluginContext) ([]ServiceStatus, error) {
			return (&VagrantPlugin{}).Status(ctx, pctx)
		}},
		{name: "multipass", executable: "multipass", fallbackStopped: true, status: func(ctx context.Context, pctx *PluginContext) ([]ServiceStatus, error) {
			return (&MultipassPlugin{}).Status(ctx, pctx)
		}},
	}

	for _, tc := range backends {
		t.Run(tc.name, func(t *testing.T) {
			resolved, err := exec.LookPath(tc.executable)
			if err != nil || resolved != filepath.Join(bin, tc.executable) {
				t.Fatalf("%s path = %q, %v; want temporary fake", tc.executable, resolved, err)
			}
			entry := statusQueryEntry(tc.name)
			pctx := contextFor(entry)
			_ = os.Remove(marker)
			_ = os.Remove(childPID)

			ctx, cancel := context.WithCancel(t.Context())
			result := make(chan error, 1)
			go func() {
				services, err := tc.status(ctx, pctx)
				if len(services) != 0 {
					result <- fmt.Errorf("canceled status returned services: %+v", services)
					return
				}
				result <- err
			}()
			waitForStatusMarker(t, marker)
			cancel()
			select {
			case err := <-result:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("canceled status error = %v, want context.Canceled", err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("canceled status query did not return promptly")
			}
			waitForStatusChildGone(t, childPID)

			t.Setenv("DVA_STATUS_MODE", "fail")
			pctx = contextFor(entry) // vagrant passes Environment explicitly.
			services, err := tc.status(context.Background(), pctx)
			t.Setenv("DVA_STATUS_MODE", "block")
			if tc.fallbackStopped {
				if err != nil || len(services) != 1 || services[0].State != "stopped" {
					t.Fatalf("ordinary nonzero = (%+v, %v), want stopped fallback", services, err)
				}
			} else if err == nil {
				t.Fatalf("ordinary nonzero = (%+v, nil), want backend error", services)
			}
		})
	}
}

func statusQueryEntry(name string) *config.LifecycleEntry {
	entry := &config.LifecycleEntry{Name: name}
	switch name {
	case "compose":
		entry.Compose = &config.ComposePluginConfig{Files: []string{"compose.yml"}}
	case "podman-compose":
		entry.PodmanCompose = &config.PodmanComposePluginConfig{Files: []string{"compose.yml"}}
	case "docker":
		entry.Docker = &config.DockerPluginConfig{Name: "container"}
	case "kubectl":
		entry.Kubectl = &config.KubectlPluginConfig{Manifests: []string{"manifest.yml"}}
	case "helm":
		entry.Helm = &config.HelmPluginConfig{Release: "release"}
	case "kustomize":
		entry.Kustomize = &config.KustomizePluginConfig{Dir: "."}
	case "vagrant":
		entry.Vagrant = &config.VagrantPluginConfig{Dir: ".", Machine: "vm"}
	case "multipass":
		entry.Multipass = &config.MultipassPluginConfig{Name: "vm"}
	}
	return entry
}

func writeStatusQueryFake(t *testing.T, bin string, names []string) {
	t.Helper()
	const fake = `#!/bin/sh
if [ "$DVA_STATUS_MODE" = fail ]; then
  exit 23
fi
/bin/sleep 30 &
child=$!
printf '%s\n' "$child" > "$DVA_STATUS_CHILD_PID"
: > "$DVA_STATUS_MARKER"
wait "$child"
`
	for _, name := range names {
		path := filepath.Join(bin, name)
		if err := os.WriteFile(path, []byte(fake), 0o755); err != nil {
			t.Fatalf("write fake %s: %v", name, err)
		}
	}
}

func waitForStatusMarker(t *testing.T, marker string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(marker); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("status fake did not enter: %s", marker)
}

func waitForStatusChildGone(t *testing.T, pidFile string) {
	t.Helper()
	raw, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("read status child PID: %v", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil || pid <= 0 {
		t.Fatalf("status child PID = %q, %v", raw, err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if err := syscall.Kill(pid, 0); errors.Is(err, syscall.ESRCH) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("status child %d survived cancellation", pid)
}
