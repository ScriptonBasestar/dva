package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/ScriptonBasestar/dva/internal/lifecycle"
)

// Docker's doctor rows live here rather than in doctor.go for a mechanical reason: doctor.go
// was over the file-size gate's error limit, so the gate refused every edit to it, including
// the one-line ones. `on_error: require_split` says what to do about that, and this cluster is
// the seam that costs nothing to cut — the four functions below talk to each other and to
// nothing else in doctor.go, and only runDoctorChecks calls into them.

func runDockerDoctorChecks(
	daemonCheck func() DoctorResult,
	socketCheck func() DoctorResult,
	goos string,
) []DoctorResult {
	daemon := daemonCheck()
	results := []DoctorResult{daemon}
	if !daemon.Passed && goos == "linux" {
		results = append(results, socketCheck())
	}
	return results
}

func checkDocker() DoctorResult {
	r := DoctorResult{Name: "Docker daemon accessible"}

	// Shared with the compose lifecycle path, which consults the same probe after a
	// failed command so the daemon diagnosis a failing `dva up` prints and the one
	// doctor reports here cannot drift apart. nil env: doctor probes the ambient
	// environment, as it always has.
	r.Passed = lifecycle.DockerDaemonReachable(nil)

	if !r.Passed {
		r.Finding = "Docker daemon is NOT accessible ('docker info' failed)"
		r.FixHint = "Start Docker Desktop or ensure dockerd is running"
	}
	return r
}

// resolveDockerSocketPath maps DOCKER_HOST to a local Unix socket filesystem path.
//
// Split out of checkDockerSocketPermissions so the mapping is assertable without a
// daemon (TASK-180). Empty dockerHost uses the historical default; unix:// yields that
// path; any other scheme (tcp, ssh, npipe, fd) yields an empty path because there is no
// local socket file to open — the check then falls through to the daemon probe.
func resolveDockerSocketPath(dockerHost string) (path string, fromEnv bool) {
	dockerHost = strings.TrimSpace(dockerHost)
	if dockerHost == "" {
		return "/var/run/docker.sock", false
	}
	const unix = "unix://"
	if after, ok := strings.CutPrefix(dockerHost, unix); ok {
		return after, true
	}
	return "", true
}

func checkDockerSocketPermissions() DoctorResult {
	return evaluateDockerSocket(os.Getenv("DOCKER_HOST"), func() bool {
		return lifecycle.DockerDaemonReachable(nil)
	})
}

// evaluateDockerSocket is the pure body of type: docker_socket / the Linux built-in
// socket diagnostic.
//
// Decision (TASK-180): honour DOCKER_HOST's unix socket when one is configured, and when
// there is no local socket path to measure — default path missing, or a non-unix
// DOCKER_HOST — fall through to the same daemon probe checkDocker uses. That is the
// choice that keeps the two verdicts from disagreeing on a healthy Colima/Podman/Desktop
// host: a path check alone would still hard-fail when the default layout is absent, and
// a daemon-only check would drop the permission finding the type name still implies.
// When an explicit unix:// DOCKER_HOST points at a missing file, that is reported as the
// failure (docker info would fail the same way).
func evaluateDockerSocket(dockerHost string, daemonOK func() bool) DoctorResult {
	path, fromEnv := resolveDockerSocketPath(dockerHost)

	if path != "" {
		if _, err := os.Stat(path); err == nil {
			f, err := os.Open(path)
			if err != nil {
				// Same agreement as a missing default path (TASK-180): docker CLI
				// can still reach the daemon when this file is leftover, root-only,
				// or otherwise unopenable. Fail on the permission finding only
				// when the daemon probe also fails.
				if daemonOK != nil && daemonOK() {
					return DoctorResult{
						Name:   "Docker socket permissions",
						Passed: true,
					}
				}
				return DoctorResult{
					Name:    "Docker socket permissions",
					Finding: fmt.Sprintf("%s exists but this user cannot open it", path),
					Passed:  false,
					FixHint: "Add user to docker group or use sudo",
				}
			}
			_ = f.Close()
			return DoctorResult{
				Name:   "Docker socket permissions",
				Passed: true,
			}
		}
		if fromEnv {
			return DoctorResult{
				Name:    "Docker socket accessible",
				Finding: fmt.Sprintf("no Docker socket at %s (from DOCKER_HOST)", path),
				Passed:  false,
				FixHint: "Start the Docker daemon or set DOCKER_HOST to a live unix socket",
			}
		}
	}

	// Default path absent, or DOCKER_HOST is not a local Unix socket: align with the
	// portable probe so this check cannot pass while checkDocker fails, or the reverse.
	if daemonOK != nil && daemonOK() {
		return DoctorResult{
			Name:   "Docker socket permissions",
			Passed: true,
		}
	}

	finding := "Docker daemon is not reachable ('docker info' failed)"
	if path != "" {
		finding = fmt.Sprintf("no Docker socket at %s and 'docker info' failed", path)
	} else if strings.TrimSpace(dockerHost) != "" {
		finding = fmt.Sprintf("DOCKER_HOST=%s is not a local Unix socket and 'docker info' failed", dockerHost)
	}
	return DoctorResult{
		Name:    "Docker socket accessible",
		Finding: finding,
		Passed:  false,
		FixHint: "Start Docker Desktop, colima, or dockerd; if DOCKER_HOST is set, check it",
	}
}
