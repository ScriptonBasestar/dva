//go:build integration

package integration

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The helpers that pick which sops the real-binary tests run decide what a pass
// means, so their edges are pinned here without a sops on the machine.

func TestParseSopsVersion(t *testing.T) {
	for _, tc := range []struct {
		name string
		out  string
		want [3]int
		ok   bool
	}{
		{"release", "sops 3.13.3\n", [3]int{3, 13, 3}, true},
		{"no trailing newline", "sops 3.13.1", [3]int{3, 13, 1}, true},
		{"crlf", "sops 3.13.1\r\n", [3]int{3, 13, 1}, true},
		{"space suffix as CI accepts", "sops 3.13.3 (latest)\n", [3]int{3, 13, 3}, true},
		{"later lines ignored", "sops 3.13.3\n[warning] sops 9.9.9 is available\n", [3]int{3, 13, 3}, true},
		{"prerelease rejected like CI", "sops 3.13.1-rc1\n", [3]int{}, false},
		{"glued suffix rejected", "sops 3.13.1+dirty\n", [3]int{}, false},
		{"wrapper line before the real one", "wrapper: sops 3.14.0\nsops 3.12.0\n", [3]int{}, false},
		{"version on second line only", "\nsops 3.13.3\n", [3]int{}, false},
		{"empty", "", [3]int{}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseSopsVersion(tc.out)
			if tc.ok {
				if err != nil {
					t.Fatalf("parseSopsVersion(%q) error = %v, want %v", tc.out, err, tc.want)
				}
				if got != tc.want {
					t.Fatalf("parseSopsVersion(%q) = %v, want %v", tc.out, got, tc.want)
				}
				return
			}
			if err == nil {
				t.Fatalf("parseSopsVersion(%q) = %v, want an error", tc.out, got)
			}
		})
	}
}

func TestLookPathOutsideShims(t *testing.T) {
	root := t.TempDir()
	mkTool := func(dir string) string {
		t.Helper()
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		p := filepath.Join(dir, "faketool")
		if err := os.WriteFile(p, []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		return p
	}
	miseShims := filepath.Join(root, "mise", "shims")
	asdfShims := filepath.Join(root, ".asdf", "shims")
	real := filepath.Join(root, "real", "bin")
	mkTool(miseShims)
	mkTool(asdfShims)
	want := mkTool(real)

	// A relative entry that does resolve from the current directory: the
	// helper must still not return it, because toolPATH symlinks the result
	// from somewhere else.
	t.Chdir(root)
	mkTool(filepath.Join(root, "relbin"))

	sep := string(os.PathListSeparator)
	t.Run("skips relative, empty and every shims dir", func(t *testing.T) {
		pathList := strings.Join([]string{"", "relbin", miseShims, asdfShims, real}, sep)
		got, err := lookPathOutsideShims("faketool", pathList)
		if err != nil {
			t.Fatalf("lookPathOutsideShims error = %v, want %s", err, want)
		}
		if got != want {
			t.Fatalf("lookPathOutsideShims = %s, want %s", got, want)
		}
		if !filepath.IsAbs(got) {
			t.Fatalf("lookPathOutsideShims = %s, want an absolute path", got)
		}
	})
	t.Run("not found when only skipped entries hold it", func(t *testing.T) {
		pathList := strings.Join([]string{"relbin", miseShims, asdfShims}, sep)
		got, err := lookPathOutsideShims("faketool", pathList)
		if !errors.Is(err, exec.ErrNotFound) {
			t.Fatalf("lookPathOutsideShims = %q, %v; want exec.ErrNotFound", got, err)
		}
	})
}
