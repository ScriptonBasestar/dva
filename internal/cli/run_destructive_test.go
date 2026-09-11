package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestConfirmDestructiveCommand(t *testing.T) {
	origConfirmYes := confirmYes
	origStdinReader := stdinReader
	origStdoutWriter := stdoutWriter
	origIsStdinTerminal := isStdinTerminal
	defer func() {
		confirmYes = origConfirmYes
		stdinReader = origStdinReader
		stdoutWriter = origStdoutWriter
		isStdinTerminal = origIsStdinTerminal
	}()

	t.Run("non-destructive passes without prompt", func(t *testing.T) {
		confirmYes = false
		isStdinTerminal = func() bool { return false }
		var out bytes.Buffer
		stdoutWriter = &out

		err := confirmDestructiveCommand("safe-cmd", false)
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		if out.Len() > 0 {
			t.Errorf("expected no output, got %q", out.String())
		}
	})

	t.Run("destructive with confirmYes passes without prompt", func(t *testing.T) {
		confirmYes = true
		isStdinTerminal = func() bool { return false }
		var out bytes.Buffer
		stdoutWriter = &out

		err := confirmDestructiveCommand("danger-cmd", true)
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		if out.Len() > 0 {
			t.Errorf("expected no output, got %q", out.String())
		}
	})

	t.Run("destructive in non-interactive terminal fails immediately", func(t *testing.T) {
		confirmYes = false
		isStdinTerminal = func() bool { return false }
		var out bytes.Buffer
		stdoutWriter = &out

		err := confirmDestructiveCommand("db-reset", true)
		if err == nil {
			t.Fatal("expected error in non-interactive mode, got nil")
		}
		if !strings.Contains(err.Error(), "--yes") {
			t.Errorf("expected error to mention --yes, got %q", err.Error())
		}
	})

	t.Run("destructive interactive with affirmative response passes", func(t *testing.T) {
		for _, response := range []string{"y\n", "Y\n", "yes\n", "YES\n"} {
			confirmYes = false
			isStdinTerminal = func() bool { return true }
			stdinReader = strings.NewReader(response)
			var out bytes.Buffer
			stdoutWriter = &out

			err := confirmDestructiveCommand("db-reset", true)
			if err != nil {
				t.Fatalf("expected nil for response %q, got %v", response, err)
			}
			if !strings.Contains(out.String(), "Warning: command \"db-reset\" is marked as destructive") {
				t.Errorf("prompt missing warning for response %q: %q", response, out.String())
			}
		}
	})

	t.Run("destructive interactive with negative or empty response fails", func(t *testing.T) {
		for _, response := range []string{"n\n", "no\n", "\n", "cancel\n"} {
			confirmYes = false
			isStdinTerminal = func() bool { return true }
			stdinReader = strings.NewReader(response)
			var out bytes.Buffer
			stdoutWriter = &out

			err := confirmDestructiveCommand("db-reset", true)
			if err == nil {
				t.Fatalf("expected error for response %q, got nil", response)
			}
			if !strings.Contains(err.Error(), "cancelled by user") {
				t.Errorf("expected cancelled error, got %q", err.Error())
			}
		}
	})
}
