package lifecycle

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func TestHealthChecker_HTTP_Ready(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	hc := &HealthChecker{}
	results := hc.Check(map[string]config.HealthCheckConfig{
		"web": {Type: "http", URL: srv.URL, Timeout: 2},
	})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Ready {
		t.Error("expected HTTP check to be ready")
	}
}

func TestHealthChecker_HTTP_NotReady(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	hc := &HealthChecker{}
	results := hc.Check(map[string]config.HealthCheckConfig{
		"web": {Type: "http", URL: srv.URL, Timeout: 2},
	})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Ready {
		t.Error("expected HTTP 500 check to not be ready")
	}
}

func TestHealthChecker_HTTP_Unreachable(t *testing.T) {
	hc := &HealthChecker{}
	results := hc.Check(map[string]config.HealthCheckConfig{
		"web": {Type: "http", URL: "http://127.0.0.1:1", Timeout: 1},
	})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Ready {
		t.Error("expected unreachable HTTP check to not be ready")
	}
}

func TestHealthChecker_TCP_Ready(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	// Extract host:port from the test server
	addr := srv.Listener.Addr().String()

	hc := &HealthChecker{}
	results := hc.Check(map[string]config.HealthCheckConfig{
		"db": {Type: "tcp", Address: addr, Timeout: 2},
	})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Ready {
		t.Error("expected TCP check to be ready")
	}
}

func TestHealthChecker_TCP_NotReady(t *testing.T) {
	hc := &HealthChecker{}
	results := hc.Check(map[string]config.HealthCheckConfig{
		"db": {Type: "tcp", Address: "127.0.0.1:1", Timeout: 1},
	})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Ready {
		t.Error("expected unreachable TCP check to not be ready")
	}
}

func TestHealthChecker_Command_Ready(t *testing.T) {
	hc := &HealthChecker{}
	results := hc.Check(map[string]config.HealthCheckConfig{
		"cmd": {Type: "command", Command: "true", Timeout: 2},
	})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Ready {
		t.Error("expected 'true' command check to be ready")
	}
}

func TestHealthChecker_Command_NotReady(t *testing.T) {
	hc := &HealthChecker{}
	results := hc.Check(map[string]config.HealthCheckConfig{
		"cmd": {Type: "command", Command: "false", Timeout: 2},
	})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Ready {
		t.Error("expected 'false' command check to not be ready")
	}
}

func TestHealthChecker_UnknownType(t *testing.T) {
	hc := &HealthChecker{}
	results := hc.Check(map[string]config.HealthCheckConfig{
		"x": {Type: "unknown", Timeout: 1},
	})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Ready {
		t.Error("unknown type should not be ready")
	}
}

func TestHealthChecker_Empty(t *testing.T) {
	hc := &HealthChecker{}
	results := hc.Check(nil)
	if results != nil {
		t.Error("expected nil for empty checks")
	}
}

func TestHealthChecker_ResultsSorted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	hc := &HealthChecker{}
	results := hc.Check(map[string]config.HealthCheckConfig{
		"z-service": {Type: "command", Command: "true", Timeout: 1},
		"a-service": {Type: "command", Command: "true", Timeout: 1},
		"m-service": {Type: "command", Command: "true", Timeout: 1},
	})

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0].Name != "a-service" || results[1].Name != "m-service" || results[2].Name != "z-service" {
		t.Errorf("results not sorted: %v, %v, %v", results[0].Name, results[1].Name, results[2].Name)
	}
}

func TestWaitUntilReady_ImmediateSuccess(t *testing.T) {
	hc := &HealthChecker{}
	results := hc.WaitUntilReady(context.Background(), map[string]config.HealthCheckConfig{
		"cmd": {Type: "command", Command: "true", Timeout: 1},
	})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Ready {
		t.Error("expected immediate ready")
	}
}

func TestWaitUntilReady_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	hc := &HealthChecker{}
	results := hc.WaitUntilReady(ctx, map[string]config.HealthCheckConfig{
		"fail": {Type: "tcp", Address: "127.0.0.1:1", Timeout: 1},
	})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	// Should return not-ready results when context is cancelled
	if results[0].Ready {
		t.Error("expected not ready after context cancel")
	}
}

// TestHealthCheckerExpandsEntryVarsPerVariant binds the compare readiness fix (TASK-592 pilot):
// a url or address written with the runner's port var is probed on the port the entry's vars
// give it, so two compare variants with different WEB_PORT values each probe their own server.
func TestHealthCheckerExpandsEntryVarsPerVariant(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer up.Close()
	port := up.Listener.Addr().(*net.TCPAddr).Port

	down, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	downPort := down.Addr().(*net.TCPAddr).Port
	if err := down.Close(); err != nil {
		t.Fatal(err)
	}

	checks := map[string]config.HealthCheckConfig{
		"http": {Type: "http", URL: "http://127.0.0.1:${WEB_PORT:-1}/", Timeout: 2},
		"tcp":  {Type: "tcp", Address: "127.0.0.1:${WEB_PORT:-1}", Timeout: 2},
	}
	hc := &HealthChecker{}
	for _, tc := range []struct {
		name string
		port int
		want bool
	}{
		{"variant with the running server", port, true},
		{"variant whose server is down", downPort, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := &config.Environment{Vars: map[string]string{"WEB_PORT": strconv.Itoa(tc.port)}}
			results := hc.CheckContext(context.Background(), checks, "", env)
			if len(results) != 2 {
				t.Fatalf("results = %+v, want 2", results)
			}
			for _, r := range results {
				if r.Ready != tc.want {
					t.Errorf("%s ready = %v, want %v (WEB_PORT=%d)", r.Name, r.Ready, tc.want, tc.port)
				}
			}
		})
	}
}
