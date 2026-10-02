package config

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
	"testing"
)

// TestMigrationGuideURLTargetsCurrentMigrationSection locks the validate-warning
// migration link to the doc that owns §11 after the docs/40 split (TASK-090).
// The constant is what users click; a stale path is a silent dead end.
func TestMigrationGuideURLTargetsCurrentMigrationSection(t *testing.T) {
	const want = "https://github.com/ScriptonBasestar/dva/blob/master/docs/42-migration-and-compatibility.md#11-migration"
	if migrationGuideURL != want {
		t.Errorf("migrationGuideURL = %q\n  want %q", migrationGuideURL, want)
	}

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Join(filepath.Dir(file), "..", "..")
	docPath := filepath.Join(repoRoot, "docs", "42-migration-and-compatibility.md")
	content, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("migration guide doc missing at %s: %v", docPath, err)
	}
	if !strings.Contains(string(content), "## 11. 마이그레이션 원칙") {
		t.Errorf("%s must retain heading ## 11. 마이그레이션 원칙 (anchor owner for #11-migration)", docPath)
	}
}

func TestNoVersionFloorRatchetWarning(t *testing.T) {
	// `version:` is the minimum DVA a config requires, so a floor below the running
	// binary is the correct, portable state and must produce no warning. Regression
	// guard for the removed warnVersionOutdated, which advised raising the floor to
	// match the binary — stranding users on older DVA and ratcheting every release.
	for _, v := range []string{"0.0.1", Version, ""} {
		c := &Config{Version: v}
		for _, w := range c.ValidateWarnings() {
			if strings.Contains(w, "older than") {
				t.Errorf("version %q must not warn about the floor: %s", v, w)
			}
		}
	}
}

func TestWarnsOnNarrowReplaceHookCandidate(t *testing.T) {
	c := &Config{
		Stack: map[string]*LifecycleEntry{
			"compose": {Compose: &ComposePluginConfig{Files: []string{"compose.yml", "compose.dev.yml"}}},
		},
		Interaction: map[string]*InteractionCommand{
			"logs": {Replace: []ProvisionItem{{Run: "docker compose -f ./compose.yml -f compose.dev.yml logs"}}},
		},
	}

	warnings := c.warnEquivalentReplaceHooks()
	if len(warnings) != 1 {
		t.Fatalf("warnings = %v, want one equivalent replace warning", warnings)
	}
	if !strings.Contains(warnings[0], "interaction.logs.replace") ||
		!strings.Contains(warnings[0], "may duplicate") ||
		!strings.Contains(warnings[0], "review whether") {
		t.Errorf("warning does not present a soft review candidate: %q", warnings[0])
	}
}

func TestReplaceHookWithExtraBehaviourIsNotWarned(t *testing.T) {
	declared := map[string]bool{"compose.yml": true}
	for _, command := range []string{
		"docker compose -f compose.yml logs -f",
		"docker compose -f compose.yml logs api",
		"echo preparing && docker compose -f compose.yml logs",
		"docker compose -f 'compose.yml' logs",
	} {
		if replaceHookIsComposeCandidate(ProvisionItem{Run: command}, "logs", declared) {
			t.Errorf("replaceHookIsComposeCandidate(%q) = true, want false for extra behaviour", command)
		}
	}
}

func TestNewlineReplaceHookIsNotCandidate(t *testing.T) {
	declared := map[string]bool{"compose.yml": true}
	command := "docker compose -f compose.yml logs\necho extra"
	if replaceHookIsComposeCandidate(ProvisionItem{Run: command}, "logs", declared) {
		t.Errorf("replaceHookIsComposeCandidate(%q) = true, want false for newline command", command)
	}
}

func TestSubsetComposeFilesAreNotCandidate(t *testing.T) {
	c := &Config{
		Stack: map[string]*LifecycleEntry{
			"compose": {Compose: &ComposePluginConfig{Files: []string{"compose.yml", "compose.dev.yml"}}},
		},
		Interaction: map[string]*InteractionCommand{
			"logs": {Replace: []ProvisionItem{{Run: "docker compose -f compose.yml logs"}}},
		},
	}
	if warnings := c.warnEquivalentReplaceHooks(); len(warnings) != 0 {
		t.Errorf("subset compose files received a candidate warning: %v", warnings)
	}
}

func TestOnlyBuildAndLogsAreReplaceHookCandidates(t *testing.T) {
	c := &Config{
		Stack: map[string]*LifecycleEntry{
			"compose": {Compose: &ComposePluginConfig{Files: []string{"compose.yml"}}},
		},
		Interaction: map[string]*InteractionCommand{
			"up":      {Replace: []ProvisionItem{{Run: "docker-compose -f compose.yml up"}}},
			"down":    {Replace: []ProvisionItem{{Run: "docker-compose -f compose.yml down"}}},
			"stop":    {Replace: []ProvisionItem{{Run: "docker-compose -f compose.yml stop"}}},
			"restart": {Replace: []ProvisionItem{{Run: "docker-compose -f compose.yml restart"}}},
			"clean":   {Replace: []ProvisionItem{{Run: "docker-compose -f compose.yml clean"}}},
		},
	}
	warnings := c.warnEquivalentReplaceHooks()
	if len(warnings) != 0 {
		t.Errorf("non-build/logs lifecycle hooks received candidate warnings: %v", warnings)
	}
}

// TestWarnDuplicateComposeApplicationOwnership was here. It covered the shape where
// applications.<app>.run.docker.service named a service a compose stack entry already
// owned, so `dva up` started it once through the orchestrator and again through the
// application manager. Both halves of that condition are gone (docs/43): there is no
// second lifecycle owner left to collide with, so the warning had nothing to warn about
// and went with the section it read.

func TestValidateWarnings_Integration(t *testing.T) {
	c := &Config{
		Version: "0.0.1",
		HealthChecks: map[string]HealthCheckConfig{
			"app": {
				Type:      "http",
				Start:     "make run",
				StartHint: "make run",
			},
		},
		Interaction: map[string]*InteractionCommand{
			"test": {
				Command: "make test",
				Subcommands: map[string]*InteractionCommand{
					"unit": {Command: "make test"},
				},
			},
		},
		Stack: map[string]*LifecycleEntry{
			"a": {Order: 10},
			"b": {Order: 10},
		},
	}

	warnings := c.ValidateWarnings()
	// Expect at least: version outdated + health check redundancy + duplicate command + duplicate order
	if len(warnings) < 4 {
		t.Errorf("expected at least 4 warnings, got %d: %v", len(warnings), warnings)
	}
}

// TestInteractionWarningsAreOrderStable pins the sort. Both the interaction tree and each
// node's subcommands are maps, so without sorting the same dva.yml prints its warnings in a
// different order on consecutive runs — measured at 3 distinct orderings across 20 runs of
// `dva config validate` on a 3-warning fixture. That is the defect TASK-107 closed for
// command suggestions, and recursion makes it more likely by raising the per-check count.
func TestInteractionWarningsAreOrderStable(t *testing.T) {
	c := &Config{
		Interaction: map[string]*InteractionCommand{
			"rails": {
				Command: "shared-cmd",
				Subcommands: map[string]*InteractionCommand{
					"aaa": {Command: "shared-cmd"},
					"bbb": {Command: "shared-cmd"},
					"ccc": {Command: "shared-cmd"},
				},
			},
		},
	}

	first := c.warnDuplicateParentSubcommand()
	if len(first) != 3 {
		t.Fatalf("expected 3 warnings, got %d: %v", len(first), first)
	}
	if !sort.StringsAreSorted(first) {
		t.Errorf("warnings are not sorted: %v", first)
	}
	// Repeat: Go randomizes map iteration per range, so an unsorted implementation diverges
	// within a handful of calls rather than needing a separate process.
	for i := range 50 {
		got := c.warnDuplicateParentSubcommand()
		if !slices.Equal(got, first) {
			t.Fatalf("run %d differs from run 0:\n first: %v\n got:   %v", i+1, first, got)
		}
	}
}

// TestFlatMapWarningsAreOrderStable covers the two checks TASK-125 sorted nothing for while
// sorting its three siblings. Both range Go maps directly, so both printed a different order on
// consecutive runs of the same file.
//
// It is a separate test rather than more cases in TestInteractionWarningsAreOrderStable because
// neither check goes through eachInteractionNode: with either sort removed that test stays green
// (measured — the whole package still passed both times), so without this one the two sorts would
// ship with no automated guard and only a hand-run binary probe behind them. TASK-128.
func TestFlatMapWarningsAreOrderStable(t *testing.T) {
	redundant := HealthCheckConfig{Type: "tcp", Address: "localhost:1", Start: "up", StartHint: "up by hand"}

	// depth links of subcommands; calculateSubcommandDepth counts links, so > MaxSubcommandDepth
	// is what makes warnDeepSubcommandNesting fire.
	chain := func(depth int) *InteractionCommand {
		node := &InteractionCommand{Command: "echo leaf"}
		for range depth {
			node = &InteractionCommand{Subcommands: map[string]*InteractionCommand{"n": node}}
		}
		return node
	}
	deep := MaxSubcommandDepth + 1

	cases := []struct {
		name string
		run  func() []string
		want int
	}{
		{
			name: "warnHealthCheckRedundancy",
			run: (&Config{
				HealthChecks: map[string]HealthCheckConfig{"alpha": redundant, "bravo": redundant},
				Stack: map[string]*LifecycleEntry{
					"infra": {HealthChecks: map[string]HealthCheckConfig{"charlie": redundant, "delta": redundant}},
				},
			}).warnHealthCheckRedundancy,
			// Two per source, so an unsorted implementation can interleave the two maps as well
			// as shuffle within each.
			want: 4,
		},
		{
			name: "warnDeepSubcommandNesting",
			run: (&Config{
				Interaction: map[string]*InteractionCommand{
					"aaa": chain(deep), "bbb": chain(deep), "ccc": chain(deep),
				},
			}).warnDeepSubcommandNesting,
			want: 3,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			first := tc.run()
			if len(first) != tc.want {
				t.Fatalf("expected %d warnings, got %d: %v", tc.want, len(first), first)
			}
			if !sort.StringsAreSorted(first) {
				t.Errorf("warnings are not sorted: %v", first)
			}
			for i := range 50 {
				if got := tc.run(); !slices.Equal(got, first) {
					t.Fatalf("run %d differs from run 0:\n first: %v\n got:   %v", i+1, first, got)
				}
			}
		})
	}
}

// TestWarnLiteralKeyShadowsSubproject covers the one ambiguity TASK-167's routing change
// introduces, and — just as importantly — the three shapes it must stay quiet about.
//
// A warning that fired on every colon key would be worse than none: the ordinary case this
// task exists to fix (`mytool:fast`, prefix naming no subproject) is now simply a working
// command, and warning about it would tell authors their correct config is suspect.
func TestWarnLiteralKeyShadowsSubproject(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  *Config
		want string // "" means: no warning at all
	}{
		{
			// The shape that could not exist before TASK-167: the parent declares the
			// literal key AND a subproject of the same prefix, so the child's `test`
			// loses the `engine:test` spelling to it.
			name: "prefix names a subproject: the child's command is shadowed",
			cfg: &Config{
				Subprojects: map[string]SubprojectConfig{"engine": {Path: "./engine"}},
				Interaction: map[string]*InteractionCommand{"engine:test": {Command: "echo parent"}},
			},
			want: "interaction.engine:test: `dva engine:test` runs this key, not subproject " +
				"`engine`'s `test` — the literal key takes precedence; use " +
				"`dva run --project engine test` to reach the subproject",
		},
		{
			// The headline TASK-167 case. Nothing is shadowed, because no subproject
			// named `mytool` exists to lose anything.
			name: "prefix names no subproject: nothing to shadow",
			cfg: &Config{
				Interaction: map[string]*InteractionCommand{"mytool:fast": {Command: "echo ok"}},
			},
		},
		{
			// ValidateReservedCommands already makes this a hard error, and
			// LiteralKeyWins excepts it so the key stays unroutable. A second opinion
			// here would describe a precedence that does not happen.
			name: "reserved prefix: unroutable, so it shadows nothing",
			cfg: &Config{
				Subprojects: map[string]SubprojectConfig{"compose": {Path: "./compose"}},
				Interaction: map[string]*InteractionCommand{"compose:ps": {Command: "echo x"}},
			},
		},
		{
			// The row above read `app:build` against a subproject named `app` while
			// `dva app` existed, and it belonged in the silent group for the reason
			// stated there. Removing the built-in moved it: `app` is no longer reserved,
			// so LiteralKeyWins stops excepting the key, it routes, and it does now take
			// precedence over the subproject — which is exactly the condition this
			// warning exists to name.
			//
			// Kept as its own row rather than folded into the first, because the pair is
			// the evidence: the same key and the same subproject answer differently on
			// either side of the removal, and neither row alone shows that.
			name: "removed built-in's prefix names a subproject: now it does shadow",
			cfg: &Config{
				Subprojects: map[string]SubprojectConfig{"app": {Path: "./app"}},
				Interaction: map[string]*InteractionCommand{"app:build": {Command: "echo x"}},
			},
			want: "interaction.app:build: `dva app:build` runs this key, not subproject " +
				"`app`'s `build` — the literal key takes precedence; use " +
				"`dva run --project app build` to reach the subproject",
		},
		{
			name: "no colon: not a candidate for splitting in the first place",
			cfg: &Config{
				Subprojects: map[string]SubprojectConfig{"engine": {Path: "./engine"}},
				Interaction: map[string]*InteractionCommand{"test": {Command: "echo ok"}},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			warnings := tc.cfg.warnLiteralKeyShadowsSubproject()
			if tc.want == "" {
				if len(warnings) != 0 {
					t.Fatalf("expected no warning, got %v", warnings)
				}
				return
			}
			if len(warnings) != 1 {
				t.Fatalf("expected 1 warning, got %d: %v", len(warnings), warnings)
			}
			// Compared whole, not by substring. The escape hatch is the half of this
			// message the reader acts on, and `dva --project engine test` — the shorter
			// spelling, without the verb — exits 1 with `unknown command "test"`. A
			// Contains check on the first clause would pass while the advice sent the
			// reader to a command that refuses.
			if warnings[0] != tc.want {
				t.Errorf("warning text drifted:\n got:  %s\n want: %s", warnings[0], tc.want)
			}
		})
	}
}
