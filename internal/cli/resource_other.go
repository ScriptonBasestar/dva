//go:build !darwin && !linux && !freebsd && !netbsd && !openbsd

package cli

import (
	"errors"
	"os/exec"
)

func resourceConfigureProcess(*exec.Cmd)         {}
func resourceTermGroup(int) error                { return errors.New("resource admission unsupported") }
func resourceKillGroup(int) error                { return errors.New("resource admission unsupported") }
func resourceGroupGone(int) bool                 { return false }
func resourceSignalExit(err *exec.ExitError) int { return err.ExitCode() }
