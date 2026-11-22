package lifecycle

import (
	"context"
	"net"
	"sort"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// portGuardDialTimeout bounds each pre-start probe. A listener on loopback answers at once;
// the bound only keeps an unroutable host from stalling `up`.
const portGuardDialTimeout = 500 * time.Millisecond

// heldHealthEndpoints returns the http/tcp health check addresses that already accept
// connections before the entry's process has started.
//
// A process entry's health check is the only declaration of the port it serves. If that port
// answers before the process exists, someone else owns it — another worktree's dev server, a
// stale run, an unrelated service — and starting anyway is worse than failing: the new process
// cannot bind, while every health probe reaches the other server and reports this entry ready.
// The TASK-538 pilot hit exactly that with two worktrees on one default port.
func heldHealthEndpoints(ctx context.Context, checks map[string]config.HealthCheckConfig, env *config.Environment) []string {
	var held []string
	seen := map[string]bool{}
	for _, check := range checks {
		address := healthCheckAddress(check, env)
		if address == "" || seen[address] {
			continue
		}
		seen[address] = true
		dialCtx, cancel := context.WithTimeout(ctx, portGuardDialTimeout)
		conn, err := (&net.Dialer{}).DialContext(dialCtx, "tcp", address)
		cancel()
		if err == nil {
			_ = conn.Close()
			held = append(held, address)
		}
	}
	sort.Strings(held)
	return held
}

// healthCheckAddress resolves the host:port a http or tcp check probes, after the same var
// expansion the probe applies. Command checks and unparsable values have no address.
func healthCheckAddress(check config.HealthCheckConfig, env *config.Environment) string {
	return check.Endpoint(env)
}
