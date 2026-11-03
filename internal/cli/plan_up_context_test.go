package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestPlanUpCallerContextReachesReadiness(t *testing.T) {
	for _, route := range []string{"direct", "cobra"} {
		t.Run(route, func(t *testing.T) {
			parent, stop := context.WithTimeout(t.Context(), 5*time.Second)
			defer stop()
			ctx, cancel := context.WithCancel(parent)
			defer cancel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				cancel()
				w.WriteHeader(http.StatusServiceUnavailable)
			}))
			defer server.Close()
			useConfig(t, fmt.Sprintf(`version: "0.1.0"
stack:
  app:
    default_runner: script
    runners:
      script:
        up: "true"
    health_checks:
      check:
        type: http
        url: %q
        ready_timeout: 60
plans:
  preview:
    entries:
      - name: app
`, server.URL))
			start := time.Now()
			var err error
			if route == "cobra" {
				cmd := &cobra.Command{}
				cmd.SetContext(ctx)
				err = upCmd.RunE(cmd, []string{"preview"})
			} else {
				c := mustLoadConfig()
				err = runPlanUp(ctx, c, rootEnvLoad(c), "preview", nil)
			}
			if !errors.Is(err, context.Canceled) || !strings.Contains(err.Error(), "readiness wait cancelled") {
				t.Fatalf("plan up error = %v, want caller cancellation from readiness", err)
			}
			if elapsed := time.Since(start); elapsed >= time.Second {
				t.Errorf("caller cancellation took %s to propagate", elapsed)
			}
		})
	}
}
