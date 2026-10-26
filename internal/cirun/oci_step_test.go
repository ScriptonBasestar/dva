package cirun

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func TestOCIStepCannotFallBackToHostShell(t *testing.T) {
	root := t.TempDir()
	root, rootErr := filepath.EvalSymlinks(root)
	if rootErr != nil {
		t.Fatal(rootErr)
	}
	t.Setenv("PATH", t.TempDir())
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	ctx = context.WithValue(ctx, ociScopeKey{}, ociScope{state: filepath.Join(root, "state"), runID: "test-run"})
	step := config.CIStep{Name: "isolated", Timeout: "1s", OCI: &config.CIOCI{
		Image: "sha256:" + strings.Repeat("a", 64), Argv: []string{"/bin/true"}, Source: "input",
		PidsLimit: 16, MemoryBytes: 128 << 20, CPUs: 1, OutputLimitBytes: 4096,
	}}
	report, err := runStep(ctx, root, step, []string{"HOME=/host/home"}, 4, "profile", io.Discard, time.Now())
	if err == nil || report.Status != "failed" || report.OCI == nil || report.OCI.Image != step.OCI.Image {
		t.Fatalf("OCI step used no owning backend: %#v, %v", report, err)
	}
	if report.WorkerBudget != nil {
		t.Fatal("OCI falsely reports inheriting host tool-worker variables")
	}
	preview := WorkerPlan(config.CIProfile{Steps: []config.CIStep{step}}, []string{"GOMAXPROCS=99"})
	if len(preview) != 1 || preview[0].Step != step.Name || preview[0].WorkerBudget != nil {
		t.Fatalf("OCI preview claims an unsupplied host worker budget: %#v", preview)
	}
}

func TestOCIStepRequiresOwningExecution(t *testing.T) {
	_, err := runOCIStep(context.Background(), t.TempDir(), config.CIStep{}, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "owning CI execution") {
		t.Fatalf("unowned OCI step: %v", err)
	}
}
