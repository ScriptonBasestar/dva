package config

import (
	"net"
	"net/url"
)

// Endpoint resolves the host:port a http or tcp check probes, after expanding vars against env
// (nil env uses the value as written). Command checks and unparsable values return "". The
// pre-start port guard and `dva validate` both use it, so they agree on which port a check
// owns (TASK-538, TASK-628).
func (h HealthCheckConfig) Endpoint(env *Environment) string {
	expand := func(v string) string {
		if env == nil {
			return v
		}
		return env.Interpolate(v)
	}
	switch h.Type {
	case "tcp":
		address := expand(h.Address)
		if _, _, err := net.SplitHostPort(address); err != nil {
			return ""
		}
		return address
	case "http":
		u, err := url.Parse(expand(h.URL))
		if err != nil || u.Hostname() == "" {
			return ""
		}
		port := u.Port()
		if port == "" {
			switch u.Scheme {
			case "http":
				port = "80"
			case "https":
				port = "443"
			default:
				return ""
			}
		}
		return net.JoinHostPort(u.Hostname(), port)
	}
	return ""
}
