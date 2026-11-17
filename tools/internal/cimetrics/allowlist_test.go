package cimetrics

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAllowed(t *testing.T) {
	root := t.TempDir()
	for _, d := range []string{"cmd", "internal/pkg", "tools", "internal/tmp/hidden"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module "+module+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "internal/pkg/a_test.go"), []byte("package pkg\nfunc TestSafe() {}\nfunc helper() {}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "internal/pkg"), filepath.Join(root, "tools/link")); err != nil {
		t.Fatal(err)
	}
	a, err := LoadAllowed(root)
	if err != nil {
		t.Fatal(err)
	}
	if !a[module+"/internal/pkg"]["TestSafe"] || a[module+"/internal/pkg"]["helper"] || a[module+"/tools"] == nil || a[module+"/tools/link"] != nil || a[module+"/internal/tmp"] != nil {
		t.Fatalf("%v", a)
	}
}

func TestLoadAllowedBoundsAndExcludedPaths(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"cmd", "internal/testdata", "internal/_hidden", "internal/.hidden", "internal/tmp", "internal/vendor", "tools"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module "+module+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	large := []byte(strings.Repeat("x", (1<<20)+1))
	for _, dir := range []string{"internal/testdata", "internal/_hidden", "internal/.hidden", "internal/tmp", "internal/vendor"} {
		if err := os.WriteFile(filepath.Join(root, dir, "large_test.go"), large, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(filepath.Join(root, "internal/testdata/large_test.go"), filepath.Join(root, "tools/link_test.go")); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadAllowed(root); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "tools/large_test.go"), large, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadAllowed(root); err == nil || !strings.Contains(err.Error(), "limit") || strings.Contains(err.Error(), root) {
		t.Fatalf("expected generic size limit: %v", err)
	}
}
