package config

import (
	"fmt"
	"net"
	"sort"
	"strings"
)

// warnHealthPortOwnedByOther warns when a process/native entry's http or tcp health check
// probes a port that another stack entry also claims, through its own health check or a
// docker port mapping declared in dva.yml. Since TASK-538 `up` refuses to start a process
// whose health port already answers, so such an entry fails whenever the other entry runs.
// Compose file ports are not read: validation does no file I/O (TASK-248). TASK-628.
func (c *Config) warnHealthPortOwnedByOther(env *Environment) []string {
	owners := map[string][]string{} // port -> "stack.<entry>.<source>"
	claim := func(port, source string) {
		if port != "" {
			owners[port] = append(owners[port], source)
		}
	}
	for name, entry := range c.Stack {
		if entry == nil {
			continue
		}
		for check, hc := range entry.HealthChecks {
			claim(endpointPort(hc.Endpoint(env)), fmt.Sprintf("stack.%s.health_checks.%s", name, check))
		}
		for _, spec := range entryDockerPorts(entry) {
			claim(dockerHostPort(env.Interpolate(spec)), fmt.Sprintf("stack.%s ports %q", name, spec))
		}
	}

	var warnings []string
	for name, entry := range c.Stack {
		if entry == nil || entry.DetectPlugin() != "process" {
			continue
		}
		own := "stack." + name + "."
		ownPorts := "stack." + name + " ports"
		for check, hc := range entry.HealthChecks {
			endpoint := hc.Endpoint(env)
			var others []string
			for _, source := range owners[endpointPort(endpoint)] {
				if !strings.HasPrefix(source, own) && !strings.HasPrefix(source, ownPorts) {
					others = append(others, source)
				}
			}
			if len(others) == 0 {
				continue
			}
			sort.Strings(others)
			warnings = append(warnings, fmt.Sprintf(
				"stack.%s.health_checks.%s: probes %s, which %s also claims; `up` refuses to start this process while that port answers — point the check at this entry's own port",
				name, check, endpoint, strings.Join(others, ", ")))
		}
	}
	sort.Strings(warnings)
	return warnings
}

func endpointPort(endpoint string) string {
	if endpoint == "" {
		return ""
	}
	_, port, err := net.SplitHostPort(endpoint)
	if err != nil {
		return ""
	}
	return port
}

// entryDockerPorts returns the port specs an entry declares for docker, as the legacy docker
// plugin block or a runners.docker block.
func entryDockerPorts(entry *LifecycleEntry) []string {
	var specs []string
	if entry.Docker != nil {
		specs = append(specs, entry.Docker.Ports...)
	}
	// decodeRunnerNode maps runners.docker onto DockerPluginConfig; DockerRunnerConfig is
	// accepted too for configs built in code.
	switch d := entry.Runners["docker"].(type) {
	case *DockerPluginConfig:
		if d != entry.Docker && d != nil {
			specs = append(specs, d.Ports...)
		}
	case *DockerRunnerConfig:
		if d != nil {
			specs = append(specs, d.Ports...)
		}
	}
	return specs
}

// dockerHostPort returns the host port of a docker -p spec ("8080:80", "127.0.0.1:8080:80",
// "8080:80/tcp"). Container-only specs, ranges and unparsable values return "".
func dockerHostPort(spec string) string {
	spec, _, _ = strings.Cut(spec, "/")
	parts := strings.Split(spec, ":")
	if len(parts) < 2 {
		return ""
	}
	host := parts[len(parts)-2]
	if host == "" || strings.Contains(host, "-") {
		return ""
	}
	for _, r := range host {
		if r < '0' || r > '9' {
			return ""
		}
	}
	return host
}
