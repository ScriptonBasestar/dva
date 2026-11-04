package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func TestStatusRoutesHonorCallerCancellation(t *testing.T) {
	for _, route := range []string{"whole-stack", "plan", "composition"} {
		for _, format := range []string{"text", "json"} {
			t.Run(route+"/"+format, func(t *testing.T) {
				parent, stop := context.WithTimeout(t.Context(), 5*time.Second)
				defer stop()
				ctx, cancel := context.WithCancel(parent)
				defer cancel()
				probe := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					cancel()
					w.WriteHeader(http.StatusServiceUnavailable)
				}))
				defer probe.Close()
				afterProbe := make(chan struct{}, 1)
				after := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					afterProbe <- struct{}{}
					w.WriteHeader(http.StatusOK)
				}))
				defer after.Close()

				yml := fmt.Sprintf(`version: "0.1.0"
stack:
  app:
    default_runner: script
    runners:
      script:
        up: "true"
    health_checks:
      cancellation:
        type: http
        url: %q
endpoints:
  after-cancellation:
    label: must not be queried
    url: %q
`, probe.URL, after.URL)
				args := []string(nil)
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
				oldJSON := jsonOutput
				jsonOutput = format == "json"
				t.Cleanup(func() { jsonOutput = oldJSON })

				cmd := &cobra.Command{}
				cmd.SetContext(ctx)
				var err error
				out := captureStdout(t, func() { err = statusCmd.RunE(cmd, args) })
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("status error = %v, want context cancellation", err)
				}
				if format == "json" {
					assertSingleStatusJSON(t, out)
				}
				select {
				case <-afterProbe:
					t.Fatal("status started an endpoint probe after cancellation")
				default:
				}
			})
		}
	}
}

func TestPreCanceledStatusRoutesRenderWithoutQueries(t *testing.T) {
	for _, route := range []string{"whole-stack", "plan", "composition"} {
		for _, format := range []string{"text", "json"} {
			t.Run(route+"/"+format, func(t *testing.T) {
				ctx, cancel := context.WithCancel(t.Context())
				cancel()
				healthCalls := make(chan struct{}, 1)
				endpointCalls := make(chan struct{}, 1)
				health := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					healthCalls <- struct{}{}
					w.WriteHeader(http.StatusOK)
				}))
				defer health.Close()
				endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					endpointCalls <- struct{}{}
					w.WriteHeader(http.StatusOK)
				}))
				defer endpoint.Close()

				yml := fmt.Sprintf(`version: "0.1.0"
stack:
  app:
    default_runner: script
    runners:
      script:
        up: "true"
    health_checks:
      should-not-run:
        type: http
        url: %q
endpoints:
  should-not-run:
    label: should not run
    url: %q
`, health.URL, endpoint.URL)
				args := []string(nil)
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
				oldJSON := jsonOutput
				jsonOutput = format == "json"
				t.Cleanup(func() { jsonOutput = oldJSON })

				cmd := &cobra.Command{}
				cmd.SetContext(ctx)
				var err error
				out := captureStdout(t, func() { err = statusCmd.RunE(cmd, args) })
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("status error = %v, want context cancellation", err)
				}
				if format == "json" {
					assertSingleStatusJSON(t, out)
				} else if out == "" {
					t.Fatal("pre-canceled status rendered no text status skeleton")
				}
				for _, calls := range []chan struct{}{healthCalls, endpointCalls} {
					select {
					case <-calls:
						t.Fatal("pre-canceled status started a query")
					default:
					}
				}
			})
		}
	}
}

func TestEndpointCancellationRendersCompletedResultsOnly(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	completed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer completed.Close()
	blockedStarted := make(chan struct{})
	blocked := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(blockedStarted)
		<-r.Context().Done()
	}))
	defer blocked.Close()
	go func() {
		<-blockedStarted
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	endpoints := map[string]config.EndpointConfig{
		"completed": {Label: "completed", URL: completed.URL},
		"blocked":   {Label: "blocked", URL: blocked.URL},
	}
	results := checkEndpointHealthContext(ctx, endpoints)
	if len(results) != 1 || results[0].Name != "completed" || !results[0].Ready {
		t.Fatalf("endpoint results = %+v, want only completed reachable endpoint", results)
	}
	out := captureStdout(t, func() { printEndpointTable(endpoints, nil, results) })
	if !strings.Contains(out, "completed") || !strings.Contains(out, "🟢") {
		t.Fatalf("completed endpoint missing from partial table:\n%s", out)
	}
	if strings.Contains(out, "🔴") {
		t.Fatalf("canceled endpoint was rendered as false down:\n%s", out)
	}

	laterCalls := make(chan struct{}, 1)
	later := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		laterCalls <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	defer later.Close()
	if got := checkEndpointHealthContext(ctx, map[string]config.EndpointConfig{
		"later": {Label: "later", URL: later.URL},
	}); len(got) != 0 {
		t.Fatalf("pre-canceled endpoint results = %+v, want none", got)
	}
	select {
	case <-laterCalls:
		t.Fatal("canceled endpoint pass started a later probe")
	default:
	}
}

func assertSingleStatusJSON(t *testing.T, raw string) {
	t.Helper()
	decoder := json.NewDecoder(strings.NewReader(raw))
	var document map[string]any
	if err := decoder.Decode(&document); err != nil {
		t.Fatalf("status output is not JSON: %v\n%s", err, raw)
	}
	var another any
	if err := decoder.Decode(&another); !errors.Is(err, io.EOF) {
		t.Fatalf("status output contains more than one JSON document: %q", raw)
	}
}
