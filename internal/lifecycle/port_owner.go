package lifecycle

import (
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// portOwnerExtraKey and portOwnerUnverifiedExtraKey carry the TASK-631 owner check into
// ServiceStatus.Extra.
const (
	portOwnerExtraKey           = "port_owner"
	portOwnerUnverifiedExtraKey = "port_owner_unverified"
	portOwnerExternalExtraKey   = "port_owner_external"
)

// listenerLookup returns the pids holding a TCP listening socket on port. verified is false
// when this platform, or this run, cannot tell — no lsof, no /proc, not permitted — and the
// pids are then meaningless. It is a variable so tests can stand in for the OS.
var listenerLookup = lookupListenerPids

// processGroupOf returns the process group of pid, or false when the platform cannot tell.
var processGroupOf = lookupProcessGroup

// portOwnerReport is the outcome of checking every http/tcp health address of an entry.
type portOwnerReport struct {
	// Mismatched lists "address (pid N, ...)" for listeners outside the tracked group.
	Mismatched []string
	// Unverified lists local addresses whose owner could not be looked up.
	Unverified []string
	// External lists addresses declared `owner: external`; their owner is not compared.
	External []string
	// Remote lists addresses on another host. They can never be checked from here, so they
	// are recorded for status but not worth a warning on every `up` (TASK-633).
	Remote []string
}

// checkHealthPortOwners reports, for each local http/tcp health address, whether the socket
// listening on it belongs to the process group dva started (the tracked pid leads its own
// group, see configureProcessGroup).
//
// A passing probe only proves that someone answers. If the entry failed to bind, or another
// process took the port after start, the probe reaches the wrong server and the entry looks
// ready. Ownership is checked only where the OS can tell; elsewhere the address is reported
// as unverified, never as a pass or a failure (user decision C1). Remote hosts are reported
// separately: a local listener on the same port number proves nothing about them. An address
// declared `owner: external` is not compared at all, but is reported as declared rather than
// passed silently (TASK-633).
func checkHealthPortOwners(checks map[string]config.HealthCheckConfig, env *config.Environment, trackedPid int) portOwnerReport {
	var report portOwnerReport
	seen := map[string]bool{}
	// Visit external declarations first so an address shared with an undeclared check is
	// still skipped: the declaration is about the port, not the check name.
	names := make([]string, 0, len(checks))
	for name := range checks {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		ei, ej := checks[names[i]].Owner == config.HealthOwnerExternal, checks[names[j]].Owner == config.HealthOwnerExternal
		if ei != ej {
			return ei
		}
		return names[i] < names[j]
	})
	for _, name := range names {
		check := checks[name]
		address := healthCheckAddress(check, env)
		if address == "" || seen[address] {
			continue
		}
		seen[address] = true
		if check.Owner == config.HealthOwnerExternal {
			report.External = append(report.External, address)
			continue
		}
		host, portText, err := net.SplitHostPort(address)
		port, convErr := strconv.Atoi(portText)
		if err != nil || convErr != nil {
			report.Unverified = append(report.Unverified, address)
			continue
		}
		if !isLocalHost(host) {
			report.Remote = append(report.Remote, address)
			continue
		}
		pids, verified := listenerLookup(port)
		if !verified || len(pids) == 0 {
			report.Unverified = append(report.Unverified, address)
			continue
		}
		var foreign []string
		unknown := false
		for _, pid := range pids {
			group, ok := processGroupOf(pid)
			if !ok {
				unknown = true
				continue
			}
			if group != trackedPid {
				foreign = append(foreign, strconv.Itoa(pid))
			}
		}
		switch {
		case len(foreign) > 0:
			report.Mismatched = append(report.Mismatched, fmt.Sprintf("%s (pid %s)", address, strings.Join(foreign, ", ")))
		case unknown:
			report.Unverified = append(report.Unverified, address)
		}
	}
	sort.Strings(report.Mismatched)
	sort.Strings(report.Unverified)
	sort.Strings(report.External)
	sort.Strings(report.Remote)
	return report
}

// extra returns the ServiceStatus.Extra entries for this report, or nil when it has none.
// Remote addresses join the unverified key: status still shows them, only `up` stays quiet.
func (r portOwnerReport) extra() map[string]string {
	extra := map[string]string{}
	if len(r.Mismatched) > 0 {
		extra[portOwnerExtraKey] = strings.Join(r.Mismatched, "; ")
	}
	if unverified := append(append([]string{}, r.Unverified...), r.Remote...); len(unverified) > 0 {
		extra[portOwnerUnverifiedExtraKey] = strings.Join(unverified, ", ")
	}
	if len(r.External) > 0 {
		extra[portOwnerExternalExtraKey] = strings.Join(r.External, ", ")
	}
	if len(extra) == 0 {
		return nil
	}
	return extra
}

// isLocalHost reports whether a probe host names this machine.
func isLocalHost(host string) bool {
	switch strings.ToLower(host) {
	case "localhost", "":
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && (ip.IsLoopback() || ip.IsUnspecified())
}

// portOwnerError is the up failure for a health address served by another process.
func portOwnerError(name string, pid int, mismatched []string) error {
	return fmt.Errorf("entry %q health port owner mismatch: %s is served outside the process group dva started (pid %d) — "+
		"another process holds the port; stop it or give the entry another port",
		name, strings.Join(mismatched, "; "), pid)
}
