package config

import (
	"strings"
	"testing"
)

// owner accepts only `external` (TASK-633), at the top level and on stack entries, and
// survives decoding into HealthCheckConfig.
func TestHealthCheckOwnerSchema(t *testing.T) {
	valid := `version: "0.1.0"
health_checks:
  top:
    type: tcp
    address: localhost:5432
    owner: external
stack:
  web:
    process:
      command: docker run --rm -p 8080:80 nginx
    health_checks:
      ready:
        type: tcp
        address: localhost:8080
        owner: external
`
	cfg, _, err := ValidateConfigBytes([]byte(valid))
	if err != nil {
		t.Fatalf("valid owner rejected: %v", err)
	}
	if got := cfg.HealthChecks["top"].Owner; got != HealthOwnerExternal {
		t.Fatalf("top-level owner = %q", got)
	}
	if got := cfg.Stack["web"].HealthChecks["ready"].Owner; got != HealthOwnerExternal {
		t.Fatalf("stack owner = %q", got)
	}

	invalid := strings.Replace(valid, "owner: external\nstack", "owner: docker\nstack", 1)
	if _, _, err := ValidateConfigBytes([]byte(invalid)); err == nil {
		t.Fatal("owner: docker accepted, want a schema error")
	}
}

func TestMergeHealthCheckOwner(t *testing.T) {
	merged := mergeHealthCheckConfig(HealthCheckConfig{Type: "tcp"}, HealthCheckConfig{Owner: HealthOwnerExternal})
	if merged.Owner != HealthOwnerExternal || merged.Type != "tcp" {
		t.Fatalf("merged = %+v", merged)
	}
}
