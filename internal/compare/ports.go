package compare

import (
	"context"
	"errors"
	"fmt"
	"net"
	"slices"
	"sort"
	"syscall"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// ValidName reports whether s is usable as a comparison or variant name. Config applies the
// same rule when it parses a declaration; the command calls this for names that arrive as
// arguments instead, before any of them becomes a path.
func ValidName(s string) bool { return namePattern.MatchString(s) }

// ResolveRef pins ref to the full commit it names in the repository holding dir, without
// creating anything. `compare up --dry-run` and `status` use it to show what a ref means now.
func ResolveRef(ctx context.Context, dir, ref string) (string, error) {
	return resolveCommit(ctx, dir, ref)
}

// AssignPorts gives each variant in fresh the ports of the lowest list position no recorded
// variant already holds. Position i of every port list belongs to one variant, so a variant
// keeps its ports across calls: a later `up` that adds a variant never moves an existing one.
//
// A position is derived from the recorded ports, not stored, because the record holds only
// values. A recorded port that is no longer in its list (the declaration changed) occupies no
// position; the occupancy probe still protects a port that is really in use.
//
// It fails when the lists are too short for the positions needed, naming each short list. It
// never reuses a position or wraps around.
func AssignPorts(cmp *config.CompareConfig, recorded []Variant, fresh []string) (map[string]map[string]int, error) {
	keys := make([]string, 0, len(cmp.Ports))
	for k := range cmp.Ports {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	assigned := make(map[string]map[string]int, len(fresh))
	if len(keys) == 0 {
		for _, name := range fresh {
			assigned[name] = nil
		}
		return assigned, nil
	}

	used := map[int]bool{}
	for _, v := range recorded {
		for _, k := range keys {
			if port, ok := v.Ports[k]; ok {
				if i := slices.Index(cmp.Ports[k], port); i >= 0 {
					used[i] = true
				}
				break
			}
		}
	}

	next := 0
	for _, name := range fresh {
		for used[next] {
			next++
		}
		if err := cmp.CheckPortCapacity(next + 1); err != nil {
			return nil, err
		}
		used[next] = true
		ports := make(map[string]int, len(keys))
		for _, k := range keys {
			ports[k] = cmp.Ports[k][next]
		}
		assigned[name] = ports
	}
	return assigned, nil
}

// PortBusy reports whether something already listens on port. It connects to the loopback
// first, then binds both the loopback and the wildcard address, because a listener on either one
// blocks a server that binds the other.
// Only "address in use" means busy; any other bind failure (a privileged port, say) is an error
// rather than a claim about who holds the port.
//
// It is a point-in-time read: a port that is free now can still be taken before the variant
// starts, and that later failure surfaces as the variant's own start error.
func PortBusy(port int) (bool, error) {
	if port < 1 || port > 65535 {
		return false, fmt.Errorf("port %d is outside 1-65535", port)
	}
	// A bind probe alone misses a listener on some platforms: Go sets SO_REUSEADDR, and on
	// macOS that lets a bind to 127.0.0.1 succeed while another process holds the wildcard
	// address without it. A connect finds that listener whatever flags it bound with.
	for _, host := range []string{"127.0.0.1", "::1"} {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, fmt.Sprint(port)), 300*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return true, nil
		}
	}
	for _, host := range []string{"127.0.0.1", ""} {
		l, err := net.Listen("tcp", net.JoinHostPort(host, fmt.Sprint(port)))
		if errors.Is(err, syscall.EADDRINUSE) {
			return true, nil
		}
		if err != nil {
			return false, fmt.Errorf("probe port %d: %w", port, err)
		}
		_ = l.Close()
	}
	return false, nil
}
