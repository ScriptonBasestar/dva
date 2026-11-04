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

func TestStartAndRestartRoutesHonorCallerCancellation(t *testing.T) {
	for _, verb := range []string{"up", "restart"} {
		for _, route := range []string{"whole-stack", "plan", "composition"} {
			t.Run(verb+"/"+route, func(t *testing.T) {
				parent, stop := context.WithTimeout(t.Context(), 5*time.Second)
				defer stop()
				ctx, cancel := context.WithCancel(parent)
				defer cancel()
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					cancel()
					w.WriteHeader(http.StatusServiceUnavailable)
				}))
				defer server.Close()
				yml := fmt.Sprintf(`version: "0.1.0"
stack:
  app:
    default_runner: script
    runners:
      script:
        up: "true"
        stop: "true"
    health_checks:
      check:
        type: http
        url: %q
        ready_timeout: 60
`, server.URL)
				var args []string
				if route != "whole-stack" {
					yml += `plans:
  preview:
    entries:
      - name: app
  composed:
    composes:
      - plan: preview
`
					args = []string{"preview"}
					if route == "composition" {
						args = []string{"composed"}
					}
				}
				useConfig(t, yml)
				cmd := &cobra.Command{}
				cmd.SetContext(ctx)
				var err error
				if verb == "up" {
					err = upCmd.RunE(cmd, args)
				} else {
					err = restartCmd.RunE(cmd, args)
				}
				if !errors.Is(err, context.Canceled) || !strings.Contains(err.Error(), "readiness wait cancelled") {
					t.Fatalf("%s %s: got %v, want caller cancellation from readiness", verb, route, err)
				}
			})
		}
	}
}
