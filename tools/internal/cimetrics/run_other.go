//go:build !darwin && !linux && !freebsd && !netbsd && !openbsd

package cimetrics

import (
	"os"
	"os/exec"
)

func configureGroup(*exec.Cmd) {}
func stopGroup(pid int) error {
	p, e := os.FindProcess(pid)
	if e != nil {
		return e
	}
	return p.Kill()
}
func killGroup(pid int) error            { return stopGroup(pid) }
func signalName(*os.ProcessState) string { return "" }
func groupSupported() bool               { return false }
