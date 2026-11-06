package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSopsTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeRules(t *testing.T, path string) { writeSopsTestFile(t, path, "creation_rules: []\n") }

// The default follows the encrypted file, not the working directory: the
// nearest .sops.yaml at or above the file's directory, bounded by root.
func TestResolveSopsConfigNearest(t *testing.T) {
	root := t.TempDir()
	writeRules(t, filepath.Join(root, SopsConfigFileName))
	writeRules(t, filepath.Join(root, "svc", SopsConfigFileName))

	cases := map[string]string{
		".env.enc":                 filepath.Join(root, SopsConfigFileName),
		"svc/.env.enc":             filepath.Join(root, "svc", SopsConfigFileName),
		"svc/deep/nested/.env.enc": filepath.Join(root, "svc", SopsConfigFileName),
		"other/.env.enc":           filepath.Join(root, SopsConfigFileName),
	}
	for target, want := range cases {
		for _, s := range []*SopsConfig{nil, {}, {Lookup: SopsLookupNearest}} {
			got, err := ResolveSopsConfig(root, s, filepath.Join(root, target))
			if err != nil || got != want {
				t.Errorf("%s (%+v): got %q, %v; want %q", target, s, got, err, want)
			}
		}
	}
}

func TestResolveSopsConfigRootIgnoresNested(t *testing.T) {
	root := t.TempDir()
	writeRules(t, filepath.Join(root, SopsConfigFileName))
	writeRules(t, filepath.Join(root, "svc", SopsConfigFileName))

	got, err := ResolveSopsConfig(root, &SopsConfig{Lookup: SopsLookupRoot}, filepath.Join(root, "svc", ".env.enc"))
	if err != nil || got != filepath.Join(root, SopsConfigFileName) {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestResolveSopsConfigExplicit(t *testing.T) {
	root := t.TempDir()
	writeRules(t, filepath.Join(root, "svc", SopsConfigFileName))
	writeRules(t, filepath.Join(root, "ops", "sops.yaml"))

	got, err := ResolveSopsConfig(root, &SopsConfig{Config: "ops/sops.yaml"}, filepath.Join(root, "svc", ".env.enc"))
	if err != nil || got != filepath.Join(root, "ops", "sops.yaml") {
		t.Fatalf("got %q, %v", got, err)
	}

	// A named file that is absent is a broken declaration, never "none applies":
	// a decrypting caller must not be able to shrug it off.
	_, err = ResolveSopsConfig(root, &SopsConfig{Config: "ops/missing.yaml"}, filepath.Join(root, ".env.enc"))
	if err == nil || errors.Is(err, ErrNoSopsConfig) {
		t.Fatalf("missing explicit config: got %v, want a non-ErrNoSopsConfig error", err)
	}
}

func TestResolveSopsConfigNone(t *testing.T) {
	parent := t.TempDir()
	// A rules file above the config root is not the project's and must not apply.
	writeRules(t, filepath.Join(parent, SopsConfigFileName))
	root := filepath.Join(parent, "proj")
	if err := os.MkdirAll(filepath.Join(root, "svc"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SOPS_CONFIG", filepath.Join(parent, SopsConfigFileName))

	for _, s := range []*SopsConfig{nil, {Lookup: SopsLookupRoot}} {
		_, err := ResolveSopsConfig(root, s, filepath.Join(root, "svc", ".env.enc"))
		if !errors.Is(err, ErrNoSopsConfig) {
			t.Errorf("%+v: got %v, want ErrNoSopsConfig", s, err)
		}
	}
}

func TestResolveSopsConfigRefusesSymlinkAndOutside(t *testing.T) {
	root := t.TempDir()
	real := filepath.Join(t.TempDir(), "rules.yaml")
	writeRules(t, real)
	if err := os.Symlink(real, filepath.Join(root, SopsConfigFileName)); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	_, err := ResolveSopsConfig(root, nil, filepath.Join(root, ".env.enc"))
	if err == nil || errors.Is(err, ErrNoSopsConfig) || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("symlinked rules: got %v", err)
	}

	if _, err := ResolveSopsConfig(root, nil, filepath.Join(filepath.Dir(root), "x.enc")); err == nil {
		t.Fatal("target outside root resolved")
	}
}

func TestValidateSopsConfig(t *testing.T) {
	good := []*SopsConfig{nil, {}, {Lookup: "nearest"}, {Lookup: "root"}, {Config: "ops/sops.yaml"}}
	for _, s := range good {
		if err := validateSopsConfig(s); err != nil {
			t.Errorf("%+v: %v", s, err)
		}
	}
	bad := []*SopsConfig{
		{Lookup: "cwd"},
		{Lookup: "root", Config: "x.yaml"},
		{Config: "/etc/sops.yaml"},
		{Config: "../sops.yaml"},
		{Config: "."},
		{Config: "a\\b.yaml"},
	}
	for _, s := range bad {
		if err := validateSopsConfig(s); err == nil {
			t.Errorf("%+v accepted", s)
		}
	}
}

// sops: is root-only. A module or override that sets it would let a file the
// root does not show change which keys encrypt the project's secrets.
func TestSopsSectionRootOnly(t *testing.T) {
	t.Run("module", func(t *testing.T) {
		dir := t.TempDir()
		writeSopsTestFile(t, filepath.Join(dir, FileName), "modules: [extra]\n")
		writeSopsTestFile(t, filepath.Join(dir, DotDirName, "extra.yml"), "sops:\n  lookup: root\n")
		if _, err := Load(dir); err == nil || !strings.Contains(err.Error(), "sops:") {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("override", func(t *testing.T) {
		dir := t.TempDir()
		writeSopsTestFile(t, filepath.Join(dir, FileName), "interaction: {}\n")
		writeSopsTestFile(t, filepath.Join(dir, "dva.override.yml"), "sops:\n  lookup: root\n")
		if _, err := Load(dir); err == nil || !strings.Contains(err.Error(), "sops:") {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("root accepted", func(t *testing.T) {
		dir := t.TempDir()
		writeSopsTestFile(t, filepath.Join(dir, FileName), "sops:\n  lookup: root\n")
		c, err := Load(dir)
		if err != nil || c.Sops == nil || c.Sops.Lookup != SopsLookupRoot {
			t.Fatalf("got %+v, %v", c, err)
		}
	})
	t.Run("invalid rejected at load", func(t *testing.T) {
		dir := t.TempDir()
		writeSopsTestFile(t, filepath.Join(dir, FileName), "sops:\n  lookup: root\n  config: x.yaml\n")
		if _, err := Load(dir); err == nil {
			t.Fatal("config+lookup accepted")
		}
	})
}

func TestSopsSectionSchema(t *testing.T) {
	for doc, ok := range map[string]bool{
		"sops:\n  lookup: nearest\n":           true,
		"sops:\n  config: ops/sops.yaml\n":     true,
		"sops:\n  lookup: cwd\n":               false,
		"sops:\n  lookup: root\n  config: x\n": false,
		"sops:\n  unknown: 1\n":                false,
	} {
		err := validateYAMLSchema([]byte(doc))
		if (err == nil) != ok {
			t.Errorf("%q: schema err=%v, want ok=%v", doc, err, ok)
		}
	}
}
