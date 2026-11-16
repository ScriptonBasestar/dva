package config

import (
	"path/filepath"
	"testing"
)

func TestCIReuseScopeValidation(t *testing.T) {
	for _, tc := range []struct {
		scope, workdir string
		probes         []string
		valid          bool
	}{
		{"", "", nil, true},
		{"checkout", "", []string{"echo local"}, true},
		{"repository", "src", nil, true},
		{"global", "", nil, false},
		{"repository", filepath.Join(t.TempDir(), "work"), nil, false},
		{"repository", "", []string{"echo dynamic"}, false},
	} {
		s := CIStep{Name: "unit", Run: "true", Workdir: tc.workdir,
			Reuse: &CIReuse{Scope: tc.scope, Inputs: []string{"src"}, Tools: []string{"sh"}, MaxAge: "1h", Probes: tc.probes}}
		if err := validateCIReuse(s); (err == nil) != tc.valid {
			t.Errorf("scope=%q workdir=%q probes=%v: %v", tc.scope, tc.workdir, tc.probes, err)
		}
	}
}
