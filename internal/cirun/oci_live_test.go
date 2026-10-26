package cirun

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

type cancelOnChildReady struct {
	bytes.Buffer
	cancel context.CancelFunc
}

func (w *cancelOnChildReady) Write(p []byte) (int, error) {
	n, err := w.Buffer.Write(p)
	if bytes.Contains(w.Bytes(), []byte("DVA_OCI_CHILD_READY")) {
		w.cancel()
	}
	return n, err
}

// Default CI skips this opt-in acceptance. A required run never silently skips.
// The cached, trusted, digest-pinned Linux image needs sh, timeout and a C compiler.
func TestOCILiveKernelLimits(t *testing.T) {
	image := os.Getenv("DVA_OCI_TEST_IMAGE")
	if image == "" {
		if os.Getenv("DVA_OCI_TEST_REQUIRED") == "1" {
			t.Fatal("required OCI acceptance has no pinned image")
		}
		t.Skip("opt-in Docker/cgroup-v2 acceptance; ordinary CI is not kernel evidence")
	}
	stateRoot := os.Getenv("DVA_OCI_TEST_STATE_DIR")
	if !filepath.IsAbs(stateRoot) {
		t.Fatal("OCI acceptance requires an absolute persistent DVA_OCI_TEST_STATE_DIR for unresolved cleanup evidence")
	}
	probe, err := os.ReadFile("testdata/oci/pids.c")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, script, want string
		cancel             bool
	}{
		{"limits", `test "$HOME" = /tmp/home; test -z "${DVA_HOST_ONLY_SENTINEL+x}"; cc -O0 /workspace/pids.c -o /tmp/probe; /tmp/probe`, "success", false},
		{"exit", `exit 7`, "exit", false},
		{"timeout_reparented", `cc -O0 /workspace/pids.c -o /tmp/probe; exec /tmp/probe detach`, "timed_out", false},
		{"cancel_reparented", `cc -O0 /workspace/pids.c -o /tmp/probe; exec /tmp/probe detach`, "canceled", true},
		{"output", `i=0; while [ "$i" -lt 2048 ]; do printf '0123456789012345678901234567890123456789012345678901234567890123456789'; i=$((i+1)); done`, "output", false},
	}
	for _, tc := range cases {
		passed := t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			root, err = filepath.EvalSymlinks(root)
			if err != nil {
				t.Fatal(err)
			}
			input := filepath.Join(root, "input")
			if err := os.Mkdir(input, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(input, "pids.c"), probe, 0o600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("DVA_HOST_ONLY_SENTINEL", "host-only")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			out := &cancelOnChildReady{cancel: func() {}}
			if tc.cancel {
				out.cancel = cancel
			}
			profile := config.CIProfile{Timeout: "30s", MaxParallel: 1, Steps: []config.CIStep{{Name: "kernel", Timeout: "5s", OCI: &config.CIOCI{
				Image: image, Argv: []string{"/bin/sh", "-eu", "-c", tc.script}, Source: "input",
				PidsLimit: 16, MemoryBytes: 128 << 20, CPUs: 1, OutputLimitBytes: 4096,
			}}}}
			report, runErr := Run(ctx, Options{Root: root, ProfileName: "kernel", StateDir: filepath.Join(stateRoot, tc.name), lockDirectory: filepath.Join(root, "locks"), Profile: profile, Output: out})
			if len(report.Steps) != 1 || report.Steps[0].OCI == nil {
				t.Fatalf("missing OCI receipt: %#v, %v", report, runErr)
			}
			evidence := report.Steps[0].OCI
			t.Logf("OCI evidence: %+v", evidence)
			if evidence.Kernel == nil || evidence.Cleanup != "removed" || evidence.CleanupError != "" {
				t.Fatalf("no kernel/absence proof: %+v; %v; %s", evidence, runErr, out.String())
			}
			switch tc.want {
			case "success":
				if runErr != nil || !strings.Contains(out.String(), "DVA_OCI_PID_PROBE denied=") || evidence.ExitCode == nil || *evidence.ExitCode != 0 {
					t.Fatalf("PID fixture: %v; %s", runErr, out.String())
				}
			case "exit":
				if runErr == nil || evidence.ExitCode == nil || *evidence.ExitCode != 7 {
					t.Fatalf("lost workload exit: %+v %v", evidence, runErr)
				}
			case "timed_out", "canceled":
				if runErr == nil || report.Steps[0].Status != tc.want || !strings.Contains(out.String(), "DVA_OCI_CHILD_READY") {
					t.Fatalf("lost cancellation/child proof: %#v; %v; %s", report.Steps[0], runErr, out.String())
				}
			case "output":
				if runErr == nil || !evidence.OutputTruncated || evidence.OutputBytes > 4096 {
					t.Fatalf("unbounded output: %+v %v", evidence, runErr)
				}
			}
		})
		if !passed {
			t.Fatal("OCI acceptance stopped; inspect retained state before any later live attempt")
		}
	}
}
