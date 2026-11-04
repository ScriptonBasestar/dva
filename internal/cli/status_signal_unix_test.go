//go:build !windows

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestStatusCommandReceivesSignals(t *testing.T) {
	if os.Getenv("DVA_TEST_STATUS_SIGNAL_HELPER") == "1" {
		jsonOutput = true
		if err := statusCmd.RunE(&cobra.Command{}, nil); !errors.Is(err, context.Canceled) {
			fmt.Fprintf(os.Stderr, "status did not return cancellation: %v\n", err)
			os.Exit(2)
		}
		os.Exit(0)
	}
	for _, signal := range []syscall.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP} {
		t.Run(signal.String(), func(t *testing.T) {
			started := make(chan struct{}, 1)
			server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				started <- struct{}{}
				<-r.Context().Done()
			}))
			t.Cleanup(server.Close)
			path := filepath.Join(t.TempDir(), "dva.yml")
			body := fmt.Sprintf("version: \"0.1.0\"\nstack:\n  app:\n    default_runner: script\n    runners:\n      script: {}\n    health_checks:\n      slow:\n        type: http\n        url: %q\n        timeout: 30\n", server.URL)
			if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestStatusCommandReceivesSignals$")
			cmd.Env = append(os.Environ(), "DVA_TEST_STATUS_SIGNAL_HELPER=1", "DVA_FILE="+path)
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = cmd.Process.Kill() })
			done := make(chan error, 1)
			go func() { done <- cmd.Wait() }()
			select {
			case <-started:
			case <-time.After(5 * time.Second):
				t.Fatal("status did not start its health probe")
			}
			if err := cmd.Process.Signal(signal); err != nil {
				t.Fatal(err)
			}
			select {
			case err := <-done:
				if err != nil {
					t.Fatalf("status signal helper failed: %v: %s", err, stderr.String())
				}
			case <-time.After(5 * time.Second):
				t.Fatal("status did not return after its signal")
			}
			decoder := json.NewDecoder(&stdout)
			var document map[string]any
			if err := decoder.Decode(&document); err != nil {
				t.Fatalf("invalid partial status JSON: %v", err)
			}
			if _, ok := document["stack"]; !ok {
				t.Fatalf("partial status lost its stack shape: %+v", document)
			}
			var extra any
			if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
				t.Fatalf("status emitted more than one document: %v", err)
			}
		})
	}
}
