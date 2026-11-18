package config

import "testing"

func TestAdmissionModes(t *testing.T) {
	for _, tc := range []struct {
		mode  string
		valid bool
	}{{"", true}, {"required", true}, {"when-policy-configured", true}, {"optional", false}} {
		p := CIProfile{Timeout: "1m", MaxParallel: 1, Admission: &CIAdmission{Mode: tc.mode, Class: "heavy", Priority: "normal", WaitTimeout: "1m"}, Steps: []CIStep{{Name: "one", Run: "true"}}}
		if err := validateCIProfile("full", p); (err == nil) != tc.valid {
			t.Errorf("mode %q: %v", tc.mode, err)
		}
	}
}

func TestAdmissionRejectsOCIAndNestedCI(t *testing.T) {
	base := CIProfile{Timeout: "1m", MaxParallel: 1, Admission: &CIAdmission{Class: "heavy", Priority: "normal", WaitTimeout: "1m"}, Steps: []CIStep{{Name: "one", Run: "true"}}}
	if err := validateCIProfile("full", base); err != nil {
		t.Fatal(err)
	}
	base.Steps[0].ChildCI = CIChildIndependentRoot
	if err := validateCIProfile("full", base); err == nil {
		t.Fatal("nested CI accepted")
	}
	base.Steps[0].ChildCI = ""
	base.Steps[0].OCI = &CIOCI{}
	if err := validateCIProfile("full", base); err == nil {
		t.Fatal("OCI accepted")
	}
}
