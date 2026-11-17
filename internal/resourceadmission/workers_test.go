package resourceadmission

import (
	"strings"
	"testing"
)

func TestWorkerCapsRespectSmallerAndUnknownValues(t *testing.T) {
	env := CapWorkerEnvironment([]string{"CARGO_BUILD_JOBS=4", "GOMAXPROCS=1", "RUST_TEST_THREADS=unknown", "GOFLAGS=-p=8"}, 2)
	joined := strings.Join(env, "\n")
	for _, want := range []string{"CARGO_BUILD_JOBS=2", "GOMAXPROCS=1", "RUST_TEST_THREADS=unknown", "GOFLAGS=-p=8", "PYTEST_MAX_WORKERS=2"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %s in %s", want, joined)
		}
	}
}
