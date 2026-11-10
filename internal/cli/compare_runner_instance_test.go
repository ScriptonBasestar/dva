//go:build !windows

package cli

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/lifecycle"
)

// The production runner passes the instance down to the lifecycle: two instances of one plan
// entry in one directory leave two pid files, and an instance's down stops only its own process.
func TestLifecycleRunnerRunsInstancesWithSeparatePidFiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("native runner requires Unix process groups")
	}
	dir := t.TempDir()
	yml := `version: "0.1.0"
stack:
  web:
    default_runner: native
    runners:
      native:
        run: sleep 300
plans:
  p:
    entries:
      - name: web
`
	if err := os.WriteFile(filepath.Join(dir, "dva.yml"), []byte(yml), 0o644); err != nil {
		t.Fatal(err)
	}
	r := lifecycleRunner{}
	ctx := context.Background()
	pidOf := func(inst string) int {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(dir, ".sb/dva/pids", inst, "web.pid"))
		if err != nil {
			t.Fatalf("instance %s left no pid file: %v", inst, err)
		}
		pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
		if err != nil {
			t.Fatal(err)
		}
		return pid
	}
	// Registered before the first Up so a failing assertion cannot leak a process: it kills
	// whatever pid files this test's directory holds, wherever the run put them.
	t.Cleanup(func() {
		files, _ := filepath.Glob(filepath.Join(dir, ".sb/dva/pids/*.pid"))
		nested, _ := filepath.Glob(filepath.Join(dir, ".sb/dva/pids/*/*.pid"))
		for _, f := range append(files, nested...) {
			if data, err := os.ReadFile(f); err == nil {
				if p, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil && p > 0 {
					_ = syscall.Kill(-p, syscall.SIGKILL)
					_ = syscall.Kill(p, syscall.SIGKILL)
				}
			}
		}
	})
	for _, inst := range []string{"a", "b"} {
		if _, err := r.Up(ctx, dir, inst, "p", nil, []string{"web"}, false); err != nil {
			t.Fatalf("Up %s: %v", inst, err)
		}
	}
	pa, pb := pidOf("a"), pidOf("b")
	if pa == pb {
		t.Fatalf("instances share pid %d", pa)
	}
	if _, err := os.Stat(filepath.Join(dir, ".sb/dva/pids/web.pid")); err == nil {
		t.Error("an instance wrote the ordinary pid file")
	}
	if err := r.Down(ctx, dir, "a", "p", nil, []string{"web"}); err != nil {
		t.Fatalf("Down a: %v", err)
	}
	if lifecycle.IsProcessRunning(pa) || !lifecycle.IsProcessRunning(pb) {
		t.Errorf("after Down a: a running=%v, b running=%v; want false, true", lifecycle.IsProcessRunning(pa), lifecycle.IsProcessRunning(pb))
	}
	if err := r.Down(ctx, dir, "b", "p", nil, []string{"web"}); err != nil {
		t.Fatalf("Down b: %v", err)
	}
}
