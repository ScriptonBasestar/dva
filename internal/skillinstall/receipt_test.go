package skillinstall

import (
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/skillclaim"
)

func TestReceiptRejectsCrossFormatFilesBeforeStatusOrUninstall(t *testing.T) {
	t.Parallel()
	options := testOptions(t, ScopeProject, RuntimeClaudeCode)
	_, destinations, err := resolve(options)
	if err != nil {
		t.Fatal(err)
	}
	target := destinations[0]
	if err := os.MkdirAll(target.path, 0o755); err != nil {
		t.Fatal(err)
	}
	flat := filepath.Join(target.path, "dva.md")
	if err := os.WriteFile(flat, []byte("foreign flat skill"), 0o644); err != nil {
		t.Fatal(err)
	}
	record := receipt{
		Schema: 1, Scope: options.Scope, Destination: target.path, Runtimes: target.runtimes,
		Version: "test", BundleSHA: strings.Repeat("0", sha256.Size*2),
		Files: []fileHash{{Path: "dva.md", SHA: strings.Repeat("0", sha256.Size*2)}},
	}
	if err := writeReceipt(receiptPath(options.StateRoot, target.path), record); err != nil {
		t.Fatal(err)
	}
	status, err := Status(options)
	if err != nil {
		t.Fatal(err)
	}
	if status.Destinations[0].Status != "invalid-receipt" {
		t.Fatalf("status = %s, want invalid-receipt", status.Destinations[0].Status)
	}
	if _, err := Uninstall(options); err == nil {
		t.Fatal("uninstall accepted a native receipt with flat files")
	}
	if _, err := os.Stat(flat); err != nil {
		t.Fatalf("uninstall removed foreign flat file after receipt rejection: %v", err)
	}
}

func TestReceiptDecoderFailsClosedOnAmbiguousOrUnsafeFiles(t *testing.T) {
	mutations := map[string]func(t *testing.T, path string, contents []byte){
		"duplicate-key": func(t *testing.T, path string, contents []byte) {
			t.Helper()
			contents = bytes.Replace(contents, []byte(`"schema": 3,`), []byte(`"schema": 3, "schema": 3,`), 1)
			if err := os.WriteFile(path, contents, 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"unknown-field": func(t *testing.T, path string, contents []byte) {
			t.Helper()
			contents = bytes.Replace(contents, []byte("{\n"), []byte("{\n  \"unknown\": true,\n"), 1)
			if err := os.WriteFile(path, contents, 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"trailing-json": func(t *testing.T, path string, contents []byte) {
			t.Helper()
			if err := os.WriteFile(path, append(contents, []byte("{}\n")...), 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"loose-mode": func(t *testing.T, path string, _ []byte) {
			t.Helper()
			if err := os.Chmod(path, 0o644); err != nil {
				t.Fatal(err)
			}
		},
		"symlink": func(t *testing.T, path string, contents []byte) {
			t.Helper()
			target := path + ".foreign"
			if err := os.WriteFile(target, contents, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, path); err != nil {
				t.Fatal(err)
			}
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			options := testOptions(t, ScopeProject, RuntimeCodex)
			installed, err := Install(options)
			if err != nil {
				t.Fatal(err)
			}
			path := receiptPath(options.StateRoot, installed.Destinations[0].Destination)
			contents, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			mutate(t, path, contents)
			status, err := Status(options)
			if err != nil {
				t.Fatal(err)
			}
			if status.Destinations[0].Status != "invalid-receipt" {
				t.Fatalf("unsafe receipt status = %s", status.Destinations[0].Status)
			}
			if _, err := Uninstall(options); err == nil {
				t.Fatal("unsafe receipt was accepted for uninstall")
			}
		})
	}
}

func TestLegacyNativeReceiptRemainsReadable(t *testing.T) {
	t.Parallel()
	options := testOptions(t, ScopeUser, RuntimeGrok)
	installed, err := Install(options)
	if err != nil {
		t.Fatal(err)
	}
	target := destination{path: installed.Destinations[0].Destination, runtimes: []Runtime{RuntimeGrok}}
	receiptFile := receiptPath(options.StateRoot, target.path)
	record, found, err := readReceipt(receiptFile)
	if err != nil || !found {
		t.Fatalf("read current receipt = (%v, %t, %v)", record, found, err)
	}
	record.Schema, record.Format = 1, ""
	record.Files = withoutCISkill(record.Files)
	record.BundleSHA = sourceBundleSHA(record.Files)
	if err := os.RemoveAll(filepath.Join(target.path, "dva-ci")); err != nil {
		t.Fatal(err)
	}
	ciClaimDestination, err := skillclaim.CanonicalDestination(filepath.Join(target.path, "dva-ci"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(skillclaim.Path(filepath.Dir(options.StateRoot), ciClaimDestination)); err != nil {
		t.Fatal(err)
	}
	if err := writeReceipt(receiptFile, record); err != nil {
		t.Fatal(err)
	}
	claims, err := projectedClaims(target, options.Scope, record.Runtimes, skillBundle{files: record.Files}, skillclaim.StateActive, "legacy-test")
	if err != nil {
		t.Fatal(err)
	}
	for _, claim := range claims {
		if err := os.Remove(skillclaim.Path(filepath.Dir(options.StateRoot), claim.Destination)); err != nil {
			t.Fatal(err)
		}
	}
	status, err := Status(options)
	if err != nil {
		t.Fatal(err)
	}
	if status.Destinations[0].Status != "legacy-unclaimed" {
		t.Fatalf("legacy native receipt status = %s", status.Destinations[0].Status)
	}
	if _, err := Install(options); err != nil {
		t.Fatalf("migrate legacy receipt during install: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target.path, "dva-ci", "SKILL.md")); err != nil {
		t.Fatalf("migration did not add bundled CI skill: %v", err)
	}
	migrated, found, err := readReceipt(receiptFile)
	if err != nil || !found || migrated.Schema != receiptSchemaCurrent || migrated.Installation != "active" {
		t.Fatalf("migrated receipt = (%#v, %t, %v)", migrated, found, err)
	}
	assertClaimConsumers(t, options, target.path, []string{"grok"})
}
