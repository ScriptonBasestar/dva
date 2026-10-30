package skillinstall

import (
	"strings"
	"testing"
)

func TestInstalledSkillsResolveOutsideTheSourceCheckout(t *testing.T) {
	bundle, err := bundledBundle()
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range bundle.files {
		text := string(bundle.contents[file.Path])
		for _, banned := range []string{"skills/dva/references", "skills/dva-config/references", "skills/dva-ci/references", "dva:dva-"} {
			if strings.Contains(text, banned) {
				t.Errorf("%s contains checkout-only reference %q", file.Path, banned)
			}
		}
	}
}

func TestBundledSkillsAvoidRuntimeOnlyInjection(t *testing.T) {
	bundle, err := bundledBundle()
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range bundle.files {
		if strings.Contains("\n"+string(bundle.contents[file.Path]), "\n!`") {
			t.Errorf("%s uses Claude-only command injection", file.Path)
		}
	}
}

func TestStatusReportsOutdatedBundle(t *testing.T) {
	options := Options{Scope: ScopeUser, HomeDir: t.TempDir(), StateRoot: t.TempDir(), Runtimes: []Runtime{RuntimeClaudeCode}, Version: "1"}
	if _, err := Install(options); err != nil {
		t.Fatal(err)
	}
	statusBundleSHA = func([]fileHash) string { return strings.Repeat("0", 64) }
	defer func() { statusBundleSHA = sourceBundleSHA }()
	status, err := Status(options)
	if err != nil {
		t.Fatal(err)
	}
	if got := status.Destinations[0].Status; got != "outdated" {
		t.Fatalf("status = %q, want outdated", got)
	}
}
