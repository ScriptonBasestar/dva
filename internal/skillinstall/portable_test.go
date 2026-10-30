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

func TestStatusClassifiesBundleDrift(t *testing.T) {
	for _, tc := range []struct{ installed, want string }{
		{"0.4.1", "outdated"}, {"0.4.2", "differs"}, {"0.5.0", "newer"}, {"unknown", "differs"},
	} {
		t.Run(tc.installed, func(t *testing.T) {
			options := Options{Scope: ScopeUser, HomeDir: t.TempDir(), StateRoot: t.TempDir(), Runtimes: []Runtime{RuntimeClaudeCode}, Version: tc.installed}
			if _, err := Install(options); err != nil {
				t.Fatal(err)
			}
			currentBundleSHA = func([]fileHash) string { return strings.Repeat("0", 64) }
			defer func() { currentBundleSHA = sourceBundleSHA }()
			options.Version = "0.4.2"
			status, err := Status(options)
			if err != nil {
				t.Fatal(err)
			}
			if got := status.Destinations[0].Status; got != tc.want {
				t.Fatalf("status = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestInstallRefusesDowngradeUnlessAllowed(t *testing.T) {
	options := Options{Scope: ScopeUser, HomeDir: t.TempDir(), StateRoot: t.TempDir(), Runtimes: []Runtime{RuntimeClaudeCode}, Version: "0.5.0"}
	if _, err := Install(options); err != nil {
		t.Fatal(err)
	}
	currentBundleSHA = func([]fileHash) string { return strings.Repeat("0", 64) }
	defer func() { currentBundleSHA = sourceBundleSHA }()
	options.Version = "0.4.2"
	if _, err := Install(options); err == nil || !strings.Contains(err.Error(), "--allow-downgrade") {
		t.Fatalf("Install() error = %v, want a downgrade refusal", err)
	}
	options.AllowDowngrade = true
	if _, err := Install(options); err != nil {
		t.Fatal(err)
	}
}

func TestAgentMeshRewritesCrossSkillLinks(t *testing.T) {
	contents, err := renderAgentMeshSkill("dva")
	if err != nil {
		t.Fatal(err)
	}
	rendered := string(contents)
	if strings.Contains(rendered, "](../dva-config/") {
		t.Fatal("rendered skill keeps a sibling link that cannot resolve")
	}
	if !strings.Contains(rendered, "(https://github.com/ScriptonBasestar/dva/blob/master/skills/dva-config/references/project-local-tooling.md)") {
		t.Fatal("rendered skill lost the canonical sibling link")
	}
}
