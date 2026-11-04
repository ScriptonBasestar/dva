package lifecycle

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func TestStatusHealthCancellationRetainsCompletedProbes(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	answered := make(chan struct{})
	fast := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		close(answered)
	}))
	t.Cleanup(fast.Close)
	started := make(chan struct{})
	slow := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
	}))
	t.Cleanup(slow.Close)
	go func() {
		for _, event := range []<-chan struct{}{answered, started} {
			select {
			case <-event:
			case <-ctx.Done():
				return
			}
		}
		// Let the independent successful response finish before interrupting
		// the other request, which cannot complete without cancellation.
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()
	checks := map[string]config.HealthCheckConfig{
		"ready": {Type: "http", URL: fast.URL, Timeout: 30},
		"slow":  {Type: "http", URL: slow.URL, Timeout: 30},
	}
	results := (&HealthChecker{}).CheckContext(ctx, checks, "", nil)
	if len(results) != 1 || results[0].Name != "ready" || !results[0].Ready {
		t.Fatalf("partial health results = %+v, want completed ready probe only", results)
	}
}

func TestStatusHealthPreCanceledPassStartsNoCommand(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	dir := t.TempDir()
	checks := map[string]config.HealthCheckConfig{"cancelled": {Type: "command", Command: ": > touched"}}
	if got := (&HealthChecker{}).CheckContext(ctx, checks, dir, nil); len(got) != 0 {
		t.Fatalf("pre-cancelled results = %+v, want no health verdict", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "touched")); !os.IsNotExist(err) {
		t.Fatalf("pre-cancelled pass executed its command: %v", err)
	}
	if got := (&HealthChecker{}).checkWithContext(ctx, checks, "", nil); len(got) != 1 || got[0].Ready {
		t.Fatalf("legacy cancelled results changed: %+v", got)
	}
}
