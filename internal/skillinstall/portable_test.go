package skillinstall

import (
	"strings"
	"sync"
	"testing"
	"time"
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
			options.bundleSHA = func([]fileHash) string { return strings.Repeat("0", 64) }
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
	options.bundleSHA = func([]fileHash) string { return strings.Repeat("0", 64) }
	options.Version = "0.4.2"
	if _, err := Install(options); err == nil || !strings.Contains(err.Error(), "--allow-downgrade") {
		t.Fatalf("Install() error = %v, want a downgrade refusal", err)
	}
	options.AllowDowngrade = true
	if _, err := Install(options); err != nil {
		t.Fatal(err)
	}
}

func TestBundleDigestStrategyIsOperationLocal(t *testing.T) {
	type testCase struct {
		installed, source, expectedStatus, digestByte string
	}
	cases := []testCase{
		{installed: "0.4.1", source: "0.4.2", expectedStatus: "outdated", digestByte: "a"},
		{installed: "0.5.0", source: "0.4.2", expectedStatus: "newer", digestByte: "b"},
	}

	options := make([]Options, len(cases))
	for i, tc := range cases {
		options[i] = Options{
			Scope: ScopeUser, HomeDir: t.TempDir(), StateRoot: t.TempDir(),
			Runtimes: []Runtime{RuntimeClaudeCode}, Version: tc.installed,
		}
		if _, err := Install(options[i]); err != nil {
			t.Fatalf("install %s: %v", tc.installed, err)
		}
		options[i].Version = tc.source
	}

	entered := make(chan struct{}, len(cases))
	release := make(chan struct{})
	var releaseOnce sync.Once
	closeRelease := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(closeRelease)
	for i, tc := range cases {
		digest := strings.Repeat(tc.digestByte, 64)
		options[i].bundleSHA = func([]fileHash) string {
			entered <- struct{}{}
			<-release
			return digest
		}
	}

	type outcome struct {
		testCase
		result Result
		err    error
	}
	outcomes := make(chan outcome, len(cases))
	for i, tc := range cases {
		go func(options Options, tc testCase) {
			result, err := Status(options)
			outcomes <- outcome{testCase: tc, result: result, err: err}
		}(options[i], tc)
	}
	for range cases {
		select {
		case <-entered:
		case <-time.After(5 * time.Second):
			t.Fatal("status operations did not concurrently reach their bundle digest strategies")
		}
	}
	closeRelease()
	for range cases {
		var outcome outcome
		select {
		case outcome = <-outcomes:
		case <-time.After(5 * time.Second):
			t.Fatal("status operations did not finish after their bundle digest strategies were released")
		}
		if outcome.err != nil {
			t.Fatalf("status %s: %v", outcome.installed, outcome.err)
		}
		if len(outcome.result.Destinations) != 1 {
			t.Fatalf("destinations = %d, want 1", len(outcome.result.Destinations))
		}
		entry := outcome.result.Destinations[0]
		wantDigest := strings.Repeat(outcome.digestByte, 64)
		if entry.SourceBundleSHA != wantDigest || entry.Status != outcome.expectedStatus {
			t.Fatalf("status %s = (%q, %q), want (%q, %q)", outcome.installed, entry.SourceBundleSHA, entry.Status, wantDigest, outcome.expectedStatus)
		}
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
