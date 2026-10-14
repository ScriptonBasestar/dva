package config

import (
	"fmt"
	"strings"
	"time"
)

const DefaultDoctorCommandTimeout = 10 * time.Second

// CommandTimeout returns the configured deadline for a command doctor check.
// The empty declaration preserves the historical ten-second deadline.
func (c DoctorCheck) CommandTimeout() (time.Duration, error) {
	raw := strings.TrimSpace(c.Timeout)
	if raw == "" {
		if c.Timeout == "" {
			return DefaultDoctorCommandTimeout, nil
		}
		return 0, fmt.Errorf("checks.%q: timeout must be a positive Go duration", c.Name)
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("checks.%q: timeout must be a positive Go duration", c.Name)
	}
	return d, nil
}

func (c *Config) validateDoctorChecks() error {
	for _, check := range c.DoctorChecks {
		if check.Timeout == "" {
			continue
		}
		if check.Type != "command" {
			return fmt.Errorf("checks.%q: timeout is only valid for command checks", check.Name)
		}
		if _, err := check.CommandTimeout(); err != nil {
			return err
		}
	}
	return nil
}
