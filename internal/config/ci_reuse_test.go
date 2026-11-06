package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestCIReuseValidationAndRoundTrip(t *testing.T) {
	valid := CIReuse{Inputs: []string{"src", "*.go"}, Tools: []string{"go"}, Outputs: []string{"bin/app"}, MaxAge: "12h"}
	for name, mutate := range map[string]func(*CIStep){
		"valid":        func(*CIStep) {},
		"empty-inputs": func(s *CIStep) { s.Reuse.Inputs = nil },
		"empty-tools":  func(s *CIStep) { s.Reuse.Tools = nil },
		"bad-age":      func(s *CIStep) { s.Reuse.MaxAge = "0s" },
		"escape":       func(s *CIStep) { s.Reuse.Inputs = []string{"../outside"} },
		"magic":        func(s *CIStep) { s.Reuse.Inputs = []string{":(top)*"} },
		"git":          func(s *CIStep) { s.Reuse.Outputs = []string{".git/config"} },
		"tool-path":    func(s *CIStep) { s.Reuse.Tools = []string{"../tool"} },
		"volatile-env": func(s *CIStep) { s.Reuse.Environment = []string{CIParentRunEnv} },
		"child":        func(s *CIStep) { s.ChildCI = CIChildIndependentRoot },
		"oci":          func(s *CIStep) { s.OCI = &CIOCI{} },
	} {
		t.Run(name, func(t *testing.T) {
			r := valid
			s := CIStep{Name: "unit", Run: "true", Reuse: &r}
			mutate(&s)
			if err := validateCIReuse(s); (err == nil) != (name == "valid") {
				t.Fatalf("validation: %v", err)
			}
		})
	}
	s := CIStep{Name: "unit", Run: "true", Reuse: &valid}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var decoded CIStep
	if err := json.Unmarshal(data, &decoded); err != nil || decoded.Reuse == nil || decoded.Reuse.MaxAge != "12h" {
		t.Fatalf("JSON roundtrip: %s %v", data, err)
	}
	data, err = yaml.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if err := yaml.Unmarshal(data, &decoded); err != nil || decoded.Reuse == nil || decoded.Reuse.MaxAge != "12h" {
		t.Fatalf("YAML roundtrip: %s %v", data, err)
	}
}

func TestLoadCIReuseAndFresh(t *testing.T) {
	dir := t.TempDir()
	text := "ci:\n  profiles:\n    audit:\n      fresh: true\n      timeout: 1m\n      steps:\n        - name: unit\n          run: true\n          reuse:\n            inputs: [src]\n            tools: [go]\n            max_age: 1h\n"
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	p, err := c.ResolveCIProfile("audit")
	if err != nil || !p.Fresh || p.Steps[0].Reuse == nil {
		t.Fatalf("profile: %+v %v", p, err)
	}
	merged := mergeCIProfile(p, CIProfile{Description: "overlay"})
	if !merged.Fresh {
		t.Fatal("overlay disabled fresh")
	}
	p.Steps[0].Environment = map[string]string{CIFreshEnv: "0"}
	if err := validateCIProfile("audit", p); err == nil {
		t.Fatal("step spoofed fresh")
	}
}
