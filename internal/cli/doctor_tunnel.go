package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// Tunnel doctor rows live in their own file for the same reason the Docker
// ones do (see the header of doctor_docker.go): doctor.go sits above the
// file-size gate, so this cluster is the seam that costs nothing to cut.
// Only runDoctorChecks calls into these functions.
//
// docs/68 §5: doctor reports whether cloudflared is installed, the auth state
// for interactive declarations, and the presence of service-token env vars —
// without printing any value.

func runTunnelDoctorChecks(c *config.Config) []DoctorResult {
	var results []DoctorResult

	entryNames := make([]string, 0, len(c.Stack))
	for name := range c.Stack {
		entryNames = append(entryNames, name)
	}
	sort.Strings(entryNames)

	for _, name := range entryNames {
		entry := c.Stack[name]
		if entry.Tunnel == nil {
			continue
		}
		installed := checkTunnelCloudflaredInstalled(name)
		results = append(results, installed)
		if !installed.Passed {
			// Without the binary every further probe is noise.
			continue
		}
		switch entry.Tunnel.AuthMode() {
		case config.TunnelAuthServiceToken:
			results = append(results, checkTunnelServiceTokenEnv(name, entry.Tunnel))
		default:
			results = append(results, checkTunnelAuthState(name, entry.Tunnel.Hostname))
		}
	}
	return results
}

// checkTunnelCloudflaredInstalled looks the binary up on PATH.
func checkTunnelCloudflaredInstalled(entryName string) DoctorResult {
	r := DoctorResult{Name: fmt.Sprintf("cloudflared installed for stack.%s", entryName)}
	if _, err := exec.LookPath("cloudflared"); err != nil {
		r.Passed = false
		r.Finding = "cloudflared not found on PATH"
		r.FixHint = "Install cloudflared (brew install cloudflared, or see https://developers.cloudflare.com/cloudflared)"
		return r
	}
	r.Passed = true
	return r
}

// checkTunnelAuthState probes the cached Access token for one hostname. The
// probe's stdout carries the JWT, so every byte is discarded — only the exit
// code decides.
func checkTunnelAuthState(entryName, hostname string) DoctorResult {
	r := DoctorResult{Name: fmt.Sprintf("cloudflared access token for stack.%s", entryName)}
	cmd := exec.Command("cloudflared", "access", "token", "--app", "https://"+hostname)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		r.Passed = false
		r.Finding = fmt.Sprintf("no usable Access token for %s", hostname)
		r.FixHint = fmt.Sprintf("Run: cloudflared access login --quiet https://%s", hostname)
		return r
	}
	r.Passed = true
	return r
}

// checkTunnelServiceTokenEnv verifies both named variables are set and
// non-empty. Names are configuration; values are never printed.
func checkTunnelServiceTokenEnv(entryName string, t *config.TunnelConfig) DoctorResult {
	r := DoctorResult{Name: fmt.Sprintf("service token env vars for stack.%s", entryName)}
	for _, name := range []string{t.ServiceTokenEnv.ID, t.ServiceTokenEnv.Secret} {
		if os.Getenv(name) == "" {
			r.Passed = false
			r.Finding = fmt.Sprintf("environment variable %s is empty or unset", name)
			r.FixHint = fmt.Sprintf("Set %s and %s in the environment (values are never written to dva.yml)", t.ServiceTokenEnv.ID, t.ServiceTokenEnv.Secret)
			return r
		}
	}
	r.Passed = true
	return r
}
