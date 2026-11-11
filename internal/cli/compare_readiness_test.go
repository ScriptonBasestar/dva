package cli

import (
	"testing"

	"github.com/ScriptonBasestar/dva/internal/lifecycle"
)

// A service whose state or health the plugin cannot judge makes the cell "-", never "ready"
// and never "not ready"; a definitely-bad signal elsewhere still wins.
func TestReadinessTextUnjudgeableServices(t *testing.T) {
	svc := func(ss ...lifecycle.ServiceStatus) *lifecycle.AggregatedStatus {
		return &lifecycle.AggregatedStatus{Entries: []lifecycle.EntryStatus{{Name: "e", Services: ss}}}
	}
	for name, tc := range map[string]struct {
		st   *lifecycle.AggregatedStatus
		want string
	}{
		"kubectl running pod":    {svc(lifecycle.ServiceStatus{Name: "web", State: "running", Health: "unknown"}), "-"},
		"kubectl pending pod":    {svc(lifecycle.ServiceStatus{Name: "web", State: "pending", Health: "unknown"}), "not ready"},
		"vagrant running vm":     {svc(lifecycle.ServiceStatus{Name: "default", State: "running", Health: "unknown"}), "-"},
		"vagrant stopped vm":     {svc(lifecycle.ServiceStatus{Name: "default", State: "stopped", Health: "unknown"}), "not ready"},
		"helm deployed":          {svc(lifecycle.ServiceStatus{Name: "rel", State: "deployed", Health: "unknown"}), "-"},
		"helm failed":            {svc(lifecycle.ServiceStatus{Name: "rel", State: "failed", Health: "unknown"}), "not ready"},
		"skaffold state unknown": {svc(lifecycle.ServiceStatus{Name: "app", State: "unknown", Health: "unknown"}), "-"},
		// kubectl/kustomize resources without status.phase (Deployment, Service, ConfigMap).
		"kubectl deployment":          {svc(lifecycle.ServiceStatus{Name: "web", State: "", Health: "unknown"}), "-"},
		"no phase and stopped":        {svc(lifecycle.ServiceStatus{Name: "web", State: "", Health: "unknown"}, lifecycle.ServiceStatus{Name: "db", State: "stopped"}), "not ready"},
		"no phase and healthy":        {svc(lifecycle.ServiceStatus{Name: "web", State: "", Health: "unknown"}, lifecycle.ServiceStatus{Name: "db", State: "running", Health: "healthy"}), "-"},
		"multipass Running":           {svc(lifecycle.ServiceStatus{Name: "vm", State: "Running"}), "ready"},
		"unknown and stopped":         {svc(lifecycle.ServiceStatus{Name: "web", State: "running", Health: "unknown"}, lifecycle.ServiceStatus{Name: "db", State: "stopped"}), "not ready"},
		"unknown and healthy running": {svc(lifecycle.ServiceStatus{Name: "web", State: "running", Health: "unknown"}, lifecycle.ServiceStatus{Name: "db", State: "running", Health: "healthy"}), "-"},
		"all running healthy":         {svc(lifecycle.ServiceStatus{Name: "web", State: "running", Health: "healthy"}, lifecycle.ServiceStatus{Name: "db", State: "running"}), "ready"},
		"failing check and unknown": {&lifecycle.AggregatedStatus{Entries: []lifecycle.EntryStatus{{Name: "e",
			Services: []lifecycle.ServiceStatus{{Name: "web", State: "running", Health: "unknown"}},
			Health:   []lifecycle.HealthCheckResult{{Name: "h"}}}}}, "not ready"},
		"unknown and entry error": {&lifecycle.AggregatedStatus{Entries: []lifecycle.EntryStatus{
			{Name: "e", Services: []lifecycle.ServiceStatus{{Name: "web", State: "running", Health: "unknown"}}},
			{Name: "f", Error: "boom"}}}, "error"},
	} {
		if got := readinessText(tc.st); got != tc.want {
			t.Errorf("%s: readinessText = %q, want %q", name, got, tc.want)
		}
	}
}
