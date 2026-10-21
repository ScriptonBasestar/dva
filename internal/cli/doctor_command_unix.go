//go:build darwin || linux || freebsd || netbsd || openbsd

package cli

import (
	"os"
	"os/exec"
	"strconv"
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

// syscall.Signal.String() is a description ("terminated"); evidence names the signal as
// operators write it. Signals outside this table fall back to their number.
var doctorSignalNames = map[syscall.Signal]string{
	syscall.SIGHUP:    "SIGHUP",
	syscall.SIGINT:    "SIGINT",
	syscall.SIGQUIT:   "SIGQUIT",
	syscall.SIGILL:    "SIGILL",
	syscall.SIGTRAP:   "SIGTRAP",
	syscall.SIGABRT:   "SIGABRT",
	syscall.SIGBUS:    "SIGBUS",
	syscall.SIGFPE:    "SIGFPE",
	syscall.SIGKILL:   "SIGKILL",
	syscall.SIGUSR1:   "SIGUSR1",
	syscall.SIGSEGV:   "SIGSEGV",
	syscall.SIGUSR2:   "SIGUSR2",
	syscall.SIGPIPE:   "SIGPIPE",
	syscall.SIGALRM:   "SIGALRM",
	syscall.SIGTERM:   "SIGTERM",
	syscall.SIGXCPU:   "SIGXCPU",
	syscall.SIGXFSZ:   "SIGXFSZ",
	syscall.SIGVTALRM: "SIGVTALRM",
	syscall.SIGPROF:   "SIGPROF",
	syscall.SIGSYS:    "SIGSYS",
}

// doctorCommandSignal names the signal that terminated the shell, or "" when the wait
// status does not report one.
func doctorCommandSignal(exitErr *exec.ExitError) string {
	status, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok || !status.Signaled() {
		return ""
	}
	sig := status.Signal()
	if name, known := doctorSignalNames[sig]; known {
		return name
	}
	return strconv.Itoa(int(sig))
}
