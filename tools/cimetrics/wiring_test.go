package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLintPhasesPreserveGuardOrderAndFailure(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.Mkdir(bin, 0o700); err != nil {
		t.Fatal(err)
	}
	write := func(path, body string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(body), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(dir, "VERSION"), "go1.fixture\n")
	write(filepath.Join(bin, "mise"), "#!/bin/sh\ncase \"$1\" in which) exit 0;; exec) shift; shift; exec \"$@\";; esac\nexit 2\n")
	write(filepath.Join(bin, "go"), "#!/bin/sh\ncase \"$1\" in version) echo 'go version go1.fixture fixture/fixture';; env) printf '%s\\n' \"$FAKE_ROOT\";; *) exit 2;; esac\n")
	write(filepath.Join(bin, "golangci-lint"), "#!/bin/sh\necho golangci >> \"$TRACE\"\nexit \"${LINT_RC:-0}\"\n")
	write(filepath.Join(bin, "gopls"), "#!/bin/sh\necho gopls >> \"$TRACE\"\nprintf '%s' \"${GOPLS_FINDING:-}\"\nexit \"${GOPLS_RC:-0}\"\n")
	trace := filepath.Join(dir, "trace")
	run := func(phase, lintRC, goplsRC, finding string) (string, string, error) {
		t.Helper()
		if err := os.WriteFile(trace, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command("/bin/sh", filepath.Join(root, "scripts", "ci-lint.sh"), phase)
		cmd.Dir = root
		cmd.Env = []string{"PATH=" + bin + ":/usr/bin:/bin", "FAKE_ROOT=" + dir, "TRACE=" + trace,
			"GOLANGCI_LINT_CACHE=" + filepath.Join(dir, "cache"), "LINT_RC=" + lintRC,
			"GOPLS_RC=" + goplsRC, "GOPLS_FINDING=" + finding}
		out, err := cmd.CombinedOutput()
		steps, readErr := os.ReadFile(trace)
		if readErr != nil {
			t.Fatal(readErr)
		}
		return string(out), string(steps), err
	}
	if out, steps, err := run("all", "0", "0", ""); err != nil || steps != "golangci\ngopls\n" {
		t.Fatalf("all: %q, %q, %v", out, steps, err)
	}
	if _, steps, err := run("preflight", "0", "0", ""); err != nil || steps != "" {
		t.Fatalf("preflight executed analyzers: %q, %v", steps, err)
	}
	if _, steps, err := run("all", "42", "0", ""); err == nil || steps != "golangci\n" {
		t.Fatalf("golangci failure did not stop chain: %q, %v", steps, err)
	}
	if out, _, err := run("gopls", "0", "0", "fixture finding"); err == nil || !strings.Contains(out, "gopls check found issues") {
		t.Fatalf("findings must fail despite tool exit 0: %q, %v", out, err)
	}
	if out, _, err := run("gopls", "0", "23", ""); err == nil || !strings.Contains(out, "could not run (exit 23)") {
		t.Fatalf("tool failure must fail despite empty findings: %q, %v", out, err)
	}
	write(filepath.Join(dir, "VERSION"), "go1.different\n")
	if out, steps, err := run("all", "0", "0", ""); err == nil || steps != "" || !strings.Contains(out, "go and GOROOT disagree") {
		t.Fatalf("pairing mismatch must stop before analyzers: %q, %q, %v", out, steps, err)
	}
}

func TestNativeLintDependencyChainPreservesBothAnalyzers(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "dva.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var d struct {
		CI struct {
			Profiles map[string]struct {
				Steps []struct {
					Name    string
					Run     string
					Depends []string `yaml:"depends_on"`
				}
			}
		}
	}
	if err := yaml.Unmarshal(b, &d); err != nil {
		t.Fatal(err)
	}
	for _, profile := range []string{"core", "full", "audit"} {
		steps := d.CI.Profiles[profile].Steps
		depends := map[string][]string{}
		commands := map[string]string{}
		for _, s := range steps {
			depends[s.Name], commands[s.Name] = s.Depends, s.Run
		}
		for name, required := range map[string]string{
			"lint-preflight": "vet", "lint-golangci": "lint-preflight", "lint-gopls": "lint-golangci", "test": "lint-gopls",
		} {
			found := false
			for _, dep := range depends[name] {
				found = found || dep == required
			}
			if !found {
				t.Errorf("%s/%s can bypass %s", profile, name, required)
			}
		}
		if !strings.Contains(commands["test"], "-race -cover") || !strings.Contains(commands["test"], "-count=1") {
			t.Errorf("%s lost race/coverage/fresh test contract", profile)
		}
	}
}
