//go:build windows

package cli

import "os/exec"

func configureDoctorCommand(cmd *exec.Cmd) {}

// Windows wait statuses carry no terminating signal.
func doctorCommandSignal(*exec.ExitError) string { return "" }
