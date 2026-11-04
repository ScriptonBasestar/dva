package lifecycle

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func TestWaitUntilReadyWithContextCancelsHTTPProbe(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	resultCh := make(chan []HealthCheckResult, 1)
	go func() {
		resultCh <- (&HealthChecker{}).WaitUntilReadyWithContext(ctx, map[string]config.HealthCheckConfig{
			"slow": {Type: "http", URL: server.URL, Timeout: 30},
		}, "", nil)
	}()

	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("HTTP probe did not start")
	}
	cancel()
	select {
	case results := <-resultCh:
		if len(results) != 1 || results[0].Ready {
			t.Fatalf("cancelled HTTP results = %+v", results)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("HTTP probe did not stop after cancellation")
	}
}

func TestWaitUntilReadyWithContextCancelsDuringProbeBatch(t *testing.T) {
	started := make(chan struct{})
	var once sync.Once
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		once.Do(func() { close(started) })
		<-r.Context().Done()
	}))
	defer server.Close()

	checks := make(map[string]config.HealthCheckConfig, 256)
	for i := range 256 {
		checks[fmt.Sprintf("slow-%03d", i)] = config.HealthCheckConfig{Type: "http", URL: server.URL, Timeout: 30}
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	resultCh := make(chan []HealthCheckResult, 1)
	go func() { resultCh <- (&HealthChecker{}).WaitUntilReady(ctx, checks) }()

	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("probe batch did not start")
	}
	cancel()
	select {
	case results := <-resultCh:
		if len(results) != len(checks) {
			t.Fatalf("cancelled batch returned %d results, want %d", len(results), len(checks))
		}
		for _, result := range results {
			if result.Ready {
				t.Fatalf("cancelled batch reported ready: %+v", result)
			}
		}
	case <-time.After(2 * time.Second):
		t.Fatal("probe batch did not stop after cancellation")
	}
}

func TestWaitUntilReadyWithContextCancelsCommandProbe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dir := t.TempDir()
	marker := filepath.Join(dir, "started")
	resultCh := make(chan []HealthCheckResult, 1)
	go func() {
		resultCh <- (&HealthChecker{}).WaitUntilReadyWithContext(ctx, map[string]config.HealthCheckConfig{
			"slow": {Type: "command", Command: ": > started; exec sleep 30", Timeout: 30},
		}, dir, nil)
	}()

	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(marker); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("command probe did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	select {
	case results := <-resultCh:
		if len(results) != 1 || results[0].Ready {
			t.Fatalf("cancelled command results = %+v", results)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("command probe did not stop after cancellation")
	}
}

func TestCheckHTTPAndTCPAllowZeroTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	if !CheckHTTP(server.URL, 0) {
		t.Fatal("zero-timeout HTTP probe did not retain its unlimited timeout behavior")
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	if !CheckTCP(listener.Addr().String(), 0) {
		t.Fatal("zero-timeout TCP probe did not retain its unlimited timeout behavior")
	}
}

func TestWaitUntilReadyWithContextDoesNotStartCancelledProbe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	marker := filepath.Join(t.TempDir(), "started")
	results := (&HealthChecker{}).WaitUntilReadyWithContext(ctx, map[string]config.HealthCheckConfig{
		"command": {Type: "command", Command: "touch " + marker, Timeout: 30},
		"tcp":     {Type: "tcp", Address: "127.0.0.1:1", Timeout: 30},
	}, "", nil)
	if len(results) != 2 {
		t.Fatalf("pre-cancelled results = %+v", results)
	}
	for _, result := range results {
		if result.Ready {
			t.Fatalf("cancelled probe reported ready: %+v", result)
		}
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("cancelled probe started: stat error = %v", err)
	}
}
