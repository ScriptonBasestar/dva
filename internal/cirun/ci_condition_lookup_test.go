package cirun

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func TestLookupDoesNotRunConditionalSelectionOrReuseProbe(t *testing.T) {
	root, step := reuseFixture(t)
	step.Condition = &config.CICondition{Run: "touch condition-ran", Timeout: "1s"}
	step.Reuse.Probes = []string{"touch probe-ran"}
	r, err := Lookup(context.Background(), Options{Root: root, ProfileName: "full", Profile: profile(step), Env: os.Environ()})
	if err != nil || r.Covered || r.Steps[0].State != "not-reusable" || r.Steps[0].Reason != "condition-not-evaluated" {
		t.Fatalf("lookup=%+v err=%v", r, err)
	}
	for _, name := range []string{"condition-ran", "probe-ran", "artifact"} {
		if _, err := os.Stat(filepath.Join(root, name)); !os.IsNotExist(err) {
			t.Fatalf("%s ran: %v", name, err)
		}
	}
}
