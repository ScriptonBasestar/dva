package cli

import (
	"path/filepath"
	"reflect"
	"testing"
)

const testRules = "creation_rules: []\n"

// Regression: sops on its own searches for .sops.yaml upward from the working
// directory, so the same reseal re-encrypted with different keys depending on
// where it was typed. The bridge now always names the rules file, chosen from
// the encrypted file's location, and the cwd plays no part.
func TestConfigEnvResealPassesNearestRulesRegardlessOfCwd(t *testing.T) {
	f := newBridgeFixture(t, simpleBridgeYAML(".env", "svc/secrets.env.enc"), map[string]string{
		"svc/secrets.env.enc": "ENC",
		".sops.yaml":          testRules,
		"svc/.sops.yaml":      testRules,
		"tools/.sops.yaml":    testRules,
	})
	f.install(false)
	t.Chdir(f.path("tools"))

	if err := runEnvReseal(""); err != nil {
		t.Fatalf("reseal: %v", err)
	}
	want := f.path("svc/.sops.yaml")
	if got := f.sops.configsUsed(); !reflect.DeepEqual(got, []string{want, want}) {
		t.Errorf("--config per call = %v, want %s for decrypt and encrypt", got, want)
	}
}

func TestConfigEnvResealHonoursRootLookup(t *testing.T) {
	yaml := simpleBridgeYAML(".env", "svc/secrets.env.enc") + "sops:\n  lookup: root\n"
	f := newBridgeFixture(t, yaml, map[string]string{
		"svc/secrets.env.enc": "ENC",
		".sops.yaml":          testRules,
		"svc/.sops.yaml":      testRules,
	})
	f.install(false)

	if err := runEnvReseal(""); err != nil {
		t.Fatalf("reseal: %v", err)
	}
	want := filepath.Join(f.dir, ".sops.yaml")
	if got := f.sops.configsUsed(); !reflect.DeepEqual(got, []string{want, want}) {
		t.Errorf("--config per call = %v, want %s", got, want)
	}
}

// With no rules file, re-encrypting has nothing to apply: reseal refuses under
// the existing creation-rule code before sops runs, while unseal only decrypts
// and proceeds without --config.
func TestConfigEnvNoRulesFile(t *testing.T) {
	t.Run("reseal refuses", func(t *testing.T) {
		f := defaultFixture(t)
		f.install(false)
		requireCode(t, runEnvReseal(""), codeSopsCreationRuleMissing)
		if calls := f.sops.callsMade(); len(calls) != 0 {
			t.Errorf("sops calls = %v, want none", calls)
		}
		f.assertNoTempResidue()
	})
	t.Run("unseal proceeds", func(t *testing.T) {
		f := defaultFixture(t)
		f.install(false)
		if err := runEnvUnseal("", false); err != nil {
			t.Fatalf("unseal: %v", err)
		}
		if got := f.sops.configsUsed(); !reflect.DeepEqual(got, []string{""}) {
			t.Errorf("--config per call = %q, want none", got)
		}
	})
}

// A rules file the project names but does not have is a broken dva.yml, so
// even a decrypt-only command refuses instead of silently using no rules.
func TestConfigEnvUnsealRefusesMissingDeclaredRules(t *testing.T) {
	yaml := simpleBridgeYAML(".env", "secrets.env.enc") + "sops:\n  config: ops/sops.yaml\n"
	f := newBridgeFixture(t, yaml, map[string]string{"secrets.env.enc": "ENC"})
	f.install(false)

	requireCode(t, runEnvUnseal("", false), codeDecryptFailed)
	if calls := f.sops.callsMade(); len(calls) != 0 {
		t.Errorf("sops calls = %v, want none", calls)
	}
}

func TestSopsArgsPutsConfigBeforeSubcommand(t *testing.T) {
	got := sopsArgs("/p/.sops.yaml", "encrypt", "--filename-override", "x")
	want := []string{"--config", "/p/.sops.yaml", "encrypt", "--filename-override", "x"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if got := sopsArgs("", "decrypt", "x"); !reflect.DeepEqual(got, []string{"decrypt", "x"}) {
		t.Errorf("empty config: got %v", got)
	}
}
