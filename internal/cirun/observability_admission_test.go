package cirun

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func TestConfigurationShapeAdmission(t *testing.T) {
	p := config.CIProfile{Timeout: "1m", MaxParallel: 1}
	legacy := struct {
		Profile     string
		Timeout     string
		MaxParallel int
		ToolJobs    *int
		Steps       []struct {
			Name             string
			DependsOn        []string
			Timeout, Workdir string
		}
	}{Profile: "full", Timeout: "1m", MaxParallel: 1}
	data, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	if got := configurationShapeFingerprint("full", p); got != hex.EncodeToString(sum[:]) {
		t.Fatalf("nil admission changed legacy shape: %s", got)
	}
	p.Admission = &config.CIAdmission{Class: "heavy", Priority: "normal", WaitTimeout: "5m"}
	base := configurationShapeFingerprint("full", p)
	for name, mutate := range map[string]func(*config.CIAdmission){"mode": func(a *config.CIAdmission) { a.Mode = "when-policy-configured" }, "class": func(a *config.CIAdmission) { a.Class = "light" }, "priority": func(a *config.CIAdmission) { a.Priority = "background" }, "wait": func(a *config.CIAdmission) { a.WaitTimeout = "1m" }} {
		a := *p.Admission
		mutate(&a)
		candidate := p
		candidate.Admission = &a
		if configurationShapeFingerprint("full", candidate) == base {
			t.Errorf("%s omitted from fingerprint", name)
		}
	}
}
