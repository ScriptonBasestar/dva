package cli

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// DoctorCommandEvidence records execution facts without collecting output or environment
// values, either of which can contain secrets supplied by the caller's ambient shell.
type DoctorCommandEvidence struct {
	ShellArgv           []string `json:"shell_argv"`
	Cwd                 string   `json:"cwd"`
	ExitCode            *int     `json:"exit_code,omitempty"`
	TimedOut            bool     `json:"timed_out,omitempty"`
	Duration            string   `json:"duration"`
	ErrorClassification string   `json:"error_classification,omitempty"`
	LaunchError         string   `json:"launch_error,omitempty"`
}

func doctorCommandFinding(command string, evidence *DoctorCommandEvidence) string {
	if evidence == nil {
		return "command could not be evaluated"
	}
	switch evidence.ErrorClassification {
	case "timeout":
		return "command timed out: " + command
	case "start_error":
		if evidence.LaunchError != "" {
			return "command could not start (" + evidence.LaunchError + "): " + command
		}
		return "command could not start: " + command
	case "invalid_configuration":
		return "command has invalid configuration: " + command
	default:
		// Retain the established wording for ordinary non-zero process exits.
		return "command exited non-zero: " + command
	}
}

func runDoctorCommand(check config.DoctorCheck, configDir string) (bool, *DoctorCommandEvidence) {
	cwd, cwdErr := filepath.Abs(configDir)
	if cwdErr != nil {
		cwd = configDir
	}
	shellPath, shellErr := exec.LookPath("sh")
	if shellErr != nil {
		launchError := "shell_lookup_failed"
		if errors.Is(shellErr, exec.ErrNotFound) {
			launchError = "shell_not_found"
		}
		return false, &DoctorCommandEvidence{Cwd: cwd, ErrorClassification: "start_error", LaunchError: launchError}
	}
	if absoluteShellPath, absErr := filepath.Abs(shellPath); absErr == nil {
		shellPath = absoluteShellPath
	}
	timeout, err := check.CommandTimeout()
	if err != nil {
		return false, &DoctorCommandEvidence{
			ShellArgv:           []string{shellPath, "-c", check.Command},
			Cwd:                 cwd,
			ErrorClassification: "invalid_configuration",
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, shellPath, "-c", check.Command)
	cmd.Dir = cwd
	configureDoctorCommand(cmd)

	started := time.Now()
	err = cmd.Run()
	evidence := &DoctorCommandEvidence{
		ShellArgv: []string{shellPath, "-c", check.Command},
		Cwd:       cwd,
		Duration:  time.Since(started).Round(time.Millisecond).String(),
	}
	if err == nil {
		code := 0
		evidence.ExitCode = &code
		return true, evidence
	}
	if ctx.Err() == context.DeadlineExceeded {
		evidence.TimedOut = true
		evidence.ErrorClassification = "timeout"
		return false, evidence
	}
	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
		code := exitErr.ExitCode()
		evidence.ExitCode = &code
		if code < 0 {
			evidence.ErrorClassification = "signal"
		} else {
			evidence.ErrorClassification = "exit_status"
		}
		return false, evidence
	}
	evidence.ErrorClassification = "start_error"
	evidence.LaunchError = "process_start_failed"
	return false, evidence
}
