//go:build darwin || linux || freebsd || netbsd || openbsd

package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestLifecycleCommandContextReceivesSignals(t *testing.T) {
	for _, sig := range []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGHUP} {
		t.Run(sig.String(), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestLifecycleSignalHelper$")
			cmd.Env = append(os.Environ(), "DVA_LIFECYCLE_SIGNAL_HELPER=1")
			cmd.Stderr = os.Stderr
			pipe, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			ready := make(chan string, 1)
			go func() {
				line, _ := bufio.NewReader(pipe).ReadString('\n')
				ready <- line
			}()
			select {
			case line := <-ready:
				if line != "ready\n" {
					cancel()
					_ = cmd.Wait()
					t.Fatalf("helper readiness = %q", line)
				}
			case <-ctx.Done():
				_ = cmd.Wait()
				t.Fatal("helper did not become ready")
			}
			if err := cmd.Process.Signal(sig); err != nil {
				cancel()
				_ = cmd.Wait()
				t.Fatal(err)
			}
			if err := cmd.Wait(); err != nil {
				t.Fatalf("helper did not handle %s: %v", sig, err)
			}
		})
	}
}

func TestLifecycleSignalHelper(t *testing.T) {
	if os.Getenv("DVA_LIFECYCLE_SIGNAL_HELPER") != "1" {
		return
	}
	ctx, stop := lifecycleCommandContext(&cobra.Command{})
	defer stop()
	fmt.Println("ready")
	select {
	case <-ctx.Done():
		if !errors.Is(ctx.Err(), context.Canceled) {
			t.Fatal(ctx.Err())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("signal did not cancel lifecycle context")
	}
}
