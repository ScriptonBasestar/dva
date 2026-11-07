//go:build integration && (darwin || linux)

package integration

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestSecretSourceRealSOPS exercises actual sops formats, an editor which
// changes content, and resealing through a nested creation rule from two cwd's.
func TestSecretSourceRealSOPS(t *testing.T) {
	for _, tc := range []struct {
		format  string
		initial []byte
		edited  []byte
	}{
		{"dotenv", []byte("VALUE=before\n"), []byte("VALUE=after\n")},
		{"yaml", []byte("value: before\n"), []byte("value: after\n")},
		{"json", []byte("{\"value\":\"before\"}\n"), []byte("{\"value\":\"after\"}\n")},
		{"ini", []byte("[section]\nvalue=before\n"), []byte("[section]\nvalue=after\n")},
		{"binary", []byte("before\x00\x01\xff"), []byte("after\x00\x01\xff")},
	} {
		t.Run(tc.format, func(t *testing.T) {
			f := newSopsFixture(t, "VALUE=fixture\n")
			sops, keygen := realSops(t), realTool(t, "age-keygen")
			public := func(key string) string {
				t.Helper()
				out, err := exec.Command(keygen, "-y", key).Output()
				if err != nil {
					t.Fatalf("age-keygen public key: %v", err)
				}
				return strings.TrimSpace(string(out))
			}
			otherKey := f.file("other-age-key.txt")
			if out, err := exec.Command(keygen, "-o", otherKey).CombinedOutput(); err != nil {
				t.Fatalf("age-keygen: %v: %s", err, out)
			}
			dir := f.file("nested")
			if err := os.Mkdir(dir, 0700); err != nil {
				t.Fatal(err)
			}
			writeAll(t, f.dir, map[string]string{
				"dva.yml":           "version: \"0.1.45\"\nsecrets:\n  sources:\n    app: {sops: nested/source.enc, format: " + tc.format + "}\n",
				".sops.yaml":        "creation_rules:\n  - age: " + public(f.keyFile) + "\n",
				"nested/.sops.yaml": "creation_rules:\n  - age: " + public(otherKey) + "\n",
			})
			source := filepath.Join(dir, "source.enc")
			encrypt := exec.Command(sops, "encrypt", "--input-type", tc.format, "--output-type", tc.format, "--age", public(f.keyFile), "/dev/stdin")
			encrypt.Stdin = bytes.NewReader(tc.initial)
			cipher, err := encrypt.Output()
			if err != nil {
				t.Fatalf("seed sops encrypt: %v", err)
			}
			if err := os.WriteFile(source, cipher, 0600); err != nil {
				t.Fatal(err)
			}
			replacement := f.file("editor-replacement")
			if err := os.WriteFile(replacement, tc.edited, 0600); err != nil {
				t.Fatal(err)
			}
			editor := f.file("editor.sh")
			if err := os.WriteFile(editor, []byte("#!/bin/sh\ncat \"$DVA_TEST_REPLACEMENT\" > \"$1\"\n"), 0700); err != nil {
				t.Fatal(err)
			}
			run := func(cwd, key string, args ...string) {
				t.Helper()
				cmd := exec.Command(dvaBinary(t), args...)
				cmd.Dir = cwd
				cmd.Env = append(os.Environ(), "PATH="+f.path, "SOPS_AGE_KEY_FILE="+key, "EDITOR="+editor, "DVA_TEST_REPLACEMENT="+replacement)
				out, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("dva %v: %v: %s", args, err, out)
				}
				if bytes.Contains(out, tc.edited) {
					t.Fatal("dva output exposed fixture plaintext")
				}
			}
			decrypt := func(key string) []byte {
				t.Helper()
				cmd := exec.Command(sops, "decrypt", "--input-type", tc.format, "--output-type", tc.format, source)
				cmd.Env = append(os.Environ(), "SOPS_AGE_KEY_FILE="+key)
				out, err := cmd.Output()
				if err != nil {
					t.Fatalf("sops decrypt: %v", err)
				}
				return out
			}
			run(f.dir, f.keyFile, "secret", "edit", "app")
			edited := decrypt(f.keyFile)
			if !bytes.Contains(edited, []byte("after")) || bytes.Contains(edited, []byte("before")) {
				t.Fatal("editor changes did not survive encryption")
			}
			if tc.format == "binary" && !bytes.Equal(edited, tc.edited) {
				t.Fatal("binary edit changed bytes")
			}
			run(f.dir, f.keyFile, "secret", "reseal", "app")
			if got := decrypt(otherKey); !bytes.Equal(got, edited) {
				t.Fatal("reseal changed decrypted content or missed nested rule")
			}
			run(dir, otherKey, "secret", "reseal", "app")
			if got := decrypt(otherKey); !bytes.Equal(got, edited) {
				t.Fatal("reseal from nested cwd changed content or rule")
			}
		})
	}
}
