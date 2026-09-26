package cli

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestConfigEnvResealReplacesSourceAndLeavesTargetUntouched(t *testing.T) {
	f := defaultFixture(t)
	f.install(false)
	if err := os.WriteFile(f.path(".env"), []byte("KEEP=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	stdout, stderr := captureStreams(t, func() {
		if err := runEnvReseal(""); err != nil {
			t.Errorf("reseal: %v", err)
		}
	})

	calls := f.sops.callsMade()
	source := f.path("secrets.env.enc")
	want := []string{"decrypt " + source, "encrypt " + source}
	if strings.Join(calls, "\n") != strings.Join(want, "\n") {
		t.Errorf("sops calls = %v, want %v", calls, want)
	}
	got, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != bridgePayload() {
		t.Errorf("source = %q, want encrypt output %q", got, bridgePayload())
	}
	target, err := os.ReadFile(f.path(".env"))
	if err != nil {
		t.Fatal(err)
	}
	if string(target) != "KEEP=1\n" {
		t.Errorf("plaintext target changed: %q", target)
	}
	if !strings.Contains(stdout, "resealed secrets.env.enc") {
		t.Errorf("stdout = %q", stdout)
	}
	if strings.Contains(stdout, bridgeSecretValue) || strings.Contains(stderr, bridgeSecretValue) {
		t.Errorf("reseal leaked the decrypted value\nstdout: %q\nstderr: %q", stdout, stderr)
	}
	if len(f.git.askedAbout()) != 0 {
		t.Errorf("reseal asked git about the plaintext target: %+v", f.git.askedAbout())
	}
	f.assertNoTempResidue()
}

func TestConfigEnvResealRefusesAMissingSourceWithoutCallingSops(t *testing.T) {
	f := newBridgeFixture(t, simpleBridgeYAML(".env", "secrets.env.enc"), nil)
	f.install(false)

	requireCode(t, runEnvReseal(""), codeSourceMissing)
	if len(f.sops.callsMade()) != 0 {
		t.Errorf("sops calls = %v, want none", f.sops.callsMade())
	}
	if _, err := os.Lstat(f.path(".env")); err == nil {
		t.Error("reseal created the plaintext target")
	}
}

func TestConfigEnvResealLeavesSourceUnchangedWhenDecryptFails(t *testing.T) {
	f := defaultFixture(t)
	f.install(false)
	f.sops.decrypt = func(string, *os.File) error { return errors.New("key") }

	requireCode(t, runEnvReseal(""), codeDecryptFailed)
	got, err := os.ReadFile(f.path("secrets.env.enc"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "ENC" {
		t.Errorf("source changed after a failed decrypt: %q", got)
	}
	f.assertNoTempResidue()
}
