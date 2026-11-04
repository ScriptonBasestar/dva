package lifecycle

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// HealthCheckResult holds the outcome of a single health check.
type HealthCheckResult struct {
	Name      string `json:"name"`
	Ready     bool   `json:"ready"`
	Started   bool   `json:"started,omitempty"`
	StartHint string `json:"start_hint,omitempty"`
}

// HealthChecker runs health checks against configured endpoints.
type HealthChecker struct{}

// Check runs all configured health checks concurrently and returns results.
func (hc *HealthChecker) Check(checks map[string]config.HealthCheckConfig) []HealthCheckResult {
	return hc.CheckWithContext(checks, "", nil)
}

// CheckWithContext runs checks with the owning project's working directory and
// environment. HTTP/TCP checks ignore them; command checks must observe the same
// project context as the lifecycle entry they guard.
func (hc *HealthChecker) CheckWithContext(checks map[string]config.HealthCheckConfig, dir string, env *config.Environment) []HealthCheckResult {
	return hc.checkWithContext(context.Background(), checks, dir, env)
}

// checkWithContext runs one health-check pass and makes every probe subordinate to
// ctx. The exported CheckWithContext predates readiness cancellation, so it keeps
// its independent background context; WaitUntilReadyWithContext supplies its own.
func (hc *HealthChecker) checkWithContext(ctx context.Context, checks map[string]config.HealthCheckConfig, dir string, env *config.Environment) []HealthCheckResult {
	if len(checks) == 0 {
		return nil
	}

	results := make([]HealthCheckResult, 0, len(checks))
	var mu sync.Mutex
	var wg sync.WaitGroup

	for name, check := range checks {
		if ctx.Err() != nil {
			mu.Lock()
			results = append(results, HealthCheckResult{Name: name, StartHint: check.StartHint})
			mu.Unlock()
			continue
		}
		wg.Add(1)
		go func(name string, check config.HealthCheckConfig) {
			defer wg.Done()

			result := HealthCheckResult{
				Name:      name,
				StartHint: check.StartHint,
			}

			timeout := time.Duration(check.Timeout) * time.Second
			if timeout == 0 {
				timeout = 2 * time.Second
			}

			probeCtx, cancel := healthCheckContext(ctx, timeout, check.Type)
			defer cancel()

			switch check.Type {
			case "http":
				result.Ready = checkHTTPWithContext(probeCtx, check.URL)
			case "tcp":
				result.Ready = checkTCPWithContext(probeCtx, check.Address)
			case "command":
				result.Ready = checkCommandWithContext(probeCtx, check.Command, dir, env)
			default:
				fmt.Fprintf(os.Stderr, "[warn] unknown health check type %q for %s\n", check.Type, name)
			}

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(name, check)
	}

	wg.Wait()

	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})

	return results
}

// WaitUntilReady polls health checks until all pass or context is cancelled.
func (hc *HealthChecker) WaitUntilReady(ctx context.Context, checks map[string]config.HealthCheckConfig) []HealthCheckResult {
	return hc.WaitUntilReadyWithContext(ctx, checks, "", nil)
}

// WaitUntilReadyWithContext is WaitUntilReady with project context for command checks.
func (hc *HealthChecker) WaitUntilReadyWithContext(ctx context.Context, checks map[string]config.HealthCheckConfig, dir string, env *config.Environment) []HealthCheckResult {
	for {
		results := hc.checkWithContext(ctx, checks, dir, env)
		if ctx.Err() != nil {
			return results
		}
		allReady := true
		for _, r := range results {
			if !r.Ready {
				allReady = false
				break
			}
		}
		if allReady {
			return results
		}

		select {
		case <-ctx.Done():
			return results
		case <-time.After(2 * time.Second):
			// poll again
		}
	}
}

// CheckHTTP reports whether an HTTP GET to url returns a 2xx/3xx status within
// timeout. Exported so the CLI endpoint table shares one implementation.
func CheckHTTP(url string, timeout time.Duration) bool {
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode >= 200 && resp.StatusCode < 400
}

func checkHTTPWithContext(ctx context.Context, url string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode >= 200 && resp.StatusCode < 400
}

// CheckHTTPReachable reports whether url answers an HTTP request at all within
// timeout, regardless of status code. Any response — including 4xx/5xx — proves
// a server is listening and speaking HTTP. This is the liveness question the
// endpoint table asks (is something serving here?), as opposed to CheckHTTP's
// stricter 2xx/3xx health verdict: an API server that 404s on "/" is up, not
// down. Only a transport error (connection refused, timeout) counts as down.
func CheckHTTPReachable(url string, timeout time.Duration) bool {
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	_ = resp.Body.Close()
	return true
}

// CheckTCP reports whether a TCP connection to address succeeds within timeout.
func CheckTCP(address string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func checkTCPWithContext(ctx context.Context, address string) bool {
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", address)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func checkCommand(command string, timeout time.Duration, dir string, env *config.Environment) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return checkCommandWithContext(ctx, command, dir, env)
}

func checkCommandWithContext(ctx context.Context, command string, dir string, env *config.Environment) bool {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	if dir != "" {
		cmd.Dir = dir
	}
	if env != nil {
		cmd.Env = env.EnvSlice()
	}
	return cmd.Run() == nil
}

// healthCheckContext preserves each probe type's established timeout behavior
// while making it subordinate to the readiness caller's cancellation.
func healthCheckContext(ctx context.Context, timeout time.Duration, checkType string) (context.Context, context.CancelFunc) {
	if checkType == "http" && timeout <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, timeout)
}

// describeHealthChecks renders a stable, human-readable summary of health checks
// for dry-run output: "<name>=<type> <target> (ready_timeout=Ns), ...".
func describeHealthChecks(checks map[string]config.HealthCheckConfig) string {
	names := make([]string, 0, len(checks))
	for name := range checks {
		names = append(names, name)
	}
	sort.Strings(names)

	parts := make([]string, 0, len(names))
	for _, name := range names {
		c := checks[name]
		target := c.URL
		switch c.Type {
		case "tcp":
			target = c.Address
		case "command":
			target = c.Command
		}
		readyTimeout := c.ReadyTimeout
		if readyTimeout <= 0 {
			readyTimeout = 30
		}
		parts = append(parts, fmt.Sprintf("%s=%s %s (ready_timeout=%ds)", name, c.Type, target, readyTimeout))
	}
	return strings.Join(parts, ", ")
}
