//go:build !windows

package cli

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestConfigEnvSealReadsResolvedPlaintext(t *testing.T) {
	const declared = ".env.${DVA_585_STAGE}"
	t.Setenv("DVA_585_STAGE", "private-stage")
	f := newBridgeFixture(t, pathBridgeYAML(declared, "secrets.env.enc")+"env_bridge: {allow_seal: true}\n", map[string]string{
		".env.private-stage": bridgePayload(), declared: "literal unchanged", ".sops.yaml": sopsCreationRuleYAML,
	})
	f.install(false)
	out, stderr := captureStreams(t, func() {
		if err := runEnvSeal(declared, true); err != nil {
			t.Fatal(err)
		}
	})
	if b, err := os.ReadFile(f.path("secrets.env.enc")); err != nil || string(b) != bridgePayload() {
		t.Fatalf("wrong encrypted input: %q, %v", b, err)
	}
	if b, err := os.ReadFile(f.path(declared)); err != nil || string(b) != "literal unchanged" {
		t.Fatalf("literal target changed: %q, %v", b, err)
	}
	if b, err := os.ReadFile(f.path(".env.private-stage")); err != nil || string(b) != bridgePayload() {
		t.Fatal("plaintext target changed")
	}
	if strings.Contains(out+stderr, "private-stage") || strings.Contains(out+stderr, bridgeSecretValue) {
		t.Fatalf("private value leaked: %s %s", out, stderr)
	}
	f.assertNoTempResidue()
}

func TestConfigEnvSealRejectsUnsafeResolvedTarget(t *testing.T) {
	for _, tt := range []struct{ value, code string }{{"../private-escape", codePathEscapes}, {"/private-absolute", codeAbsolutePath}, {"secrets.env.enc", codeSourceIsTarget}, {"", ""}, {"private-parent/.env", codePathComponentSymlnk}} {
		t.Run(tt.value, func(t *testing.T) {
			t.Setenv("DVA_585_TARGET", tt.value)
			const declared = "${DVA_585_TARGET}"
			f := newBridgeFixture(t, pathBridgeYAML(declared, "secrets.env.enc")+"env_bridge: {allow_seal: true}\n", map[string]string{".sops.yaml": sopsCreationRuleYAML, declared: "literal unchanged"})
			if tt.code == codePathComponentSymlnk {
				if err := os.Symlink(t.TempDir(), f.path("private-parent")); err != nil {
					t.Fatal(err)
				}
			}
			f.install(false)
			err := runEnvSeal(declared, true)
			if err == nil || errorCode(err) != tt.code || strings.Contains(err.Error(), "private-") {
				t.Fatalf("unsafe seal outcome: %v (%s)", err, errorCode(err))
			}
			if len(f.sops.callsMade()) != 0 {
				t.Fatal("SOPS ran before rejection")
			}
			if b, err := os.ReadFile(f.path(declared)); err != nil || string(b) != "literal unchanged" {
				t.Fatal("literal target changed")
			}
			f.assertNoTempResidue()
		})
	}
}

func TestConfigEnvSourceOnlyOperationsKeepLiteralMetadata(t *testing.T) {
	const declared = ".env.${DVA_585_UNDEFINED}"
	t.Setenv("DVA_585_UNDEFINED", "")
	if err := os.Unsetenv("DVA_585_UNDEFINED"); err != nil {
		t.Fatal(err)
	}
	for _, verb := range []string{"edit", "reseal"} {
		t.Run(verb, func(t *testing.T) {
			f := newBridgeFixture(t, pathBridgeYAML(declared, "secrets.env.enc"), map[string]string{"secrets.env.enc": "ENC", declared: "literal unchanged", ".sops.yaml": testRules})
			f.install(false)
			captureStreams(t, func() {
				var err error
				if verb == "edit" {
					err = runEnvEdit(declared)
				} else {
					err = runEnvReseal(declared)
				}
				if err != nil {
					t.Fatal(err)
				}
			})
			if b, err := os.ReadFile(f.path(declared)); err != nil || string(b) != "literal unchanged" {
				t.Fatal("source-only verb touched plaintext")
			}
			if len(f.sops.callsMade()) == 0 {
				t.Fatal("source operation did not run")
			}
			f.assertNoTempResidue()
		})
	}
}

func TestConfigEnvHintPreservesShellSelector(t *testing.T) {
	t.Setenv("DVA_585_HINT", "must-not-expand")
	for _, declared := range []string{".env.${DVA_585_HINT}", "some path/a'b", "file; printf BAD"} {
		arg := strings.TrimPrefix(unsealCommand(declared), "dva config env unseal ")
		out, err := exec.Command("/bin/sh", "-c", "set -- "+arg+"; printf '%s' \"$1\"").CombinedOutput()
		if err != nil || string(out) != declared {
			t.Fatalf("selector was not preserved: %q, %v", out, err)
		}
	}
}
