package resourceadmission

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadPolicyRejectsMalformedAndNormalizesDigest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.json")
	write := func(s string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(s), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(`{"capacity":4,"max_running":3,"max_heavy":1,"weights":{"heavy":2,"light":1},"aging_interval":"5m"}`)
	_, one, err := ReadPolicy(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err = ReadPolicy(path); err != nil {
		t.Fatalf("owner policy with group read rejected: %v", err)
	}
	write(`{"weights":{"light":1,"heavy":2},"max_heavy":1,"max_running":3,"capacity":4,"aging_interval":"5m"}`)
	_, two, err := ReadPolicy(path)
	if err != nil || one != two {
		t.Fatalf("digest mismatch %q %q %v", one, two, err)
	}
	write(`{"capacity":4,"max_running":3,"max_heavy":1,"weights":{"heavy":2,"light":1},"aging_interval":"0s"}`)
	if _, _, err = ReadPolicy(path); err == nil {
		t.Fatal("invalid age accepted")
	}
	write(`{"capacity":2,"max_running":2,"max_heavy":2,"weights":{"heavy":1,"light":2},"aging_interval":"1s"}`)
	if _, _, err = ReadPolicy(path); err == nil {
		t.Fatal("heavy lighter than light accepted")
	}
	if _, _, err = ReadPolicy("relative"); err == nil {
		t.Fatal("relative path accepted")
	}
	link := filepath.Join(dir, "link")
	if err = os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if _, _, err = ReadPolicy(link); err == nil {
		t.Fatal("symlink accepted")
	}
}
