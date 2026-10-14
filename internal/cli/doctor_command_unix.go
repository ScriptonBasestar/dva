//go:build darwin || linux || freebsd || netbsd || openbsd

package cli

import (
	"os"
	"os/exec"
	"syscall"
)

// A command check is a shell script, so cancelling only its shell can leave its children
// alive. Give the shell its own process group and cancel that group as one unit.
func configureDoctorCommand(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return os.ErrProcessDone
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
