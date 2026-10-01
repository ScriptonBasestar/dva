package config

import (
	"fmt"
	"net"
	"strings"
	"time"
)

// Tunnel declaration (docs/68): an access prerequisite attached to a remote
// kubectl/helm entry. The tunnel is not a lifecycle backend — DVA opens it
// before the entry runs, checks it is ready, and closes only the process it
// started itself when the command ends.
const (
	TunnelProviderCloudflared = "cloudflared"

	TunnelAuthInteractive     = "interactive"
	TunnelAuthServiceToken    = "service-token"
	TunnelDefaultReadyTimeout = 30 * time.Second
)

// TunnelServiceTokenEnv names the environment variables that carry Cloudflare
// service-token credentials. The variables hold the values — never this file:
// a token written into dva.yml would land in git.
type TunnelServiceTokenEnv struct {
	ID     string `yaml:"id"`
	Secret string `yaml:"secret"`
}

// TunnelConfig declares how DVA reaches a resource behind Cloudflare Access.
type TunnelConfig struct {
	// Provider is the tunnel binary family. v1 accepts only cloudflared.
	Provider string `yaml:"provider"`
	// Hostname is the Access-protected host cloudflared connects to.
	Hostname string `yaml:"hostname"`
	// Local is the loopback address:port cloudflared forwards to. A non-loopback
	// listener would expose an authenticated tunnel to the LAN, so validation
	// rejects it.
	Local string `yaml:"local"`
	// Auth is interactive (default) or service-token.
	Auth string `yaml:"auth,omitempty"`
	// ServiceTokenEnv is required when Auth is service-token.
	ServiceTokenEnv *TunnelServiceTokenEnv `yaml:"service_token_env,omitempty"`
	// ReadyTimeout bounds the readiness wait ("30s" style). Empty means 30s.
	ReadyTimeout string `yaml:"ready_timeout,omitempty"`
}

// AuthMode returns the effective auth mode, defaulting to interactive.
func (t *TunnelConfig) AuthMode() string {
	if strings.TrimSpace(t.Auth) == "" {
		return TunnelAuthInteractive
	}
	return t.Auth
}

// ReadyTimeoutDuration parses ReadyTimeout, falling back to the documented
// 30s default. An explicit value must be a positive duration.
func (t *TunnelConfig) ReadyTimeoutDuration() (time.Duration, error) {
	raw := strings.TrimSpace(t.ReadyTimeout)
	if raw == "" {
		return TunnelDefaultReadyTimeout, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("tunnel: ready_timeout %q is not a duration (e.g. 30s)", raw)
	}
	if d <= 0 {
		return 0, fmt.Errorf("tunnel: ready_timeout %q must be positive", raw)
	}
	return d, nil
}

// validateEntryTunnel enforces the docs/68 §2 field rules for one entry.
// The plugin gate (kubectl/helm only) lives here too because it is a
// cross-field rule the JSON schema cannot express.
func validateEntryTunnel(entryName string, entry *LifecycleEntry) error {
	t := entry.Tunnel
	if t == nil {
		return nil
	}

	plugin := entry.DetectPlugin()
	if plugin != "kubectl" && plugin != "helm" {
		return fmt.Errorf(
			"stack.%s.tunnel: only kubectl and helm entries may declare a tunnel (entry plugin is %q)",
			entryName, plugin)
	}

	var errs []string
	add := func(format string, args ...any) {
		errs = append(errs, fmt.Sprintf("stack.%s.tunnel: %s", entryName, fmt.Sprintf(format, args...)))
	}

	if t.Provider != TunnelProviderCloudflared {
		add("provider must be %q (got %q)", TunnelProviderCloudflared, t.Provider)
	}
	if strings.TrimSpace(t.Hostname) == "" {
		add("hostname is required")
	}
	if err := validateTunnelLocal(t.Local); err != nil {
		add("%v", err)
	}

	switch t.AuthMode() {
	case TunnelAuthInteractive:
		if t.ServiceTokenEnv != nil {
			add("service_token_env is only valid with auth: %s", TunnelAuthServiceToken)
		}
	case TunnelAuthServiceToken:
		if t.ServiceTokenEnv == nil {
			add("auth: %s requires service_token_env.id and service_token_env.secret (names of environment variables, not values)", TunnelAuthServiceToken)
		} else {
			if strings.TrimSpace(t.ServiceTokenEnv.ID) == "" {
				add("service_token_env.id is required (name of the environment variable holding the token ID)")
			}
			if strings.TrimSpace(t.ServiceTokenEnv.Secret) == "" {
				add("service_token_env.secret is required (name of the environment variable holding the token secret)")
			}
		}
	default:
		add("auth must be %q or %q (got %q)", TunnelAuthInteractive, TunnelAuthServiceToken, t.Auth)
	}

	if _, err := t.ReadyTimeoutDuration(); err != nil {
		add("%v", err)
	}

	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("%s", strings.Join(errs, "; "))
}

// validateTunnelLocal accepts only a loopback host with an explicit port.
// cloudflared binds this listener locally; anything wider than loopback puts
// an Access-authenticated route on the network.
func validateTunnelLocal(local string) error {
	host, port, err := net.SplitHostPort(strings.TrimSpace(local))
	if err != nil {
		return fmt.Errorf("local must be host:port on loopback (e.g. 127.0.0.1:16443), got %q", local)
	}
	switch strings.ToLower(host) {
	case "127.0.0.1", "::1", "localhost":
	default:
		return fmt.Errorf("local must bind a loopback address (127.0.0.1, ::1 or localhost), got %q", host)
	}
	if port == "" {
		return fmt.Errorf("local requires an explicit port, got %q", local)
	}
	return nil
}
