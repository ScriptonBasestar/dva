package config

import (
	"slices"
	"strings"
	"testing"
)

func TestWarnDuplicateParentSubcommand(t *testing.T) {
	// Same command → warning
	c := &Config{
		Interaction: map[string]*InteractionCommand{
			"build": {
				Command: "cargo build",
				Subcommands: map[string]*InteractionCommand{
					"ce": {Command: "cargo build"},
				},
			},
		},
	}
	warnings := c.warnDuplicateParentSubcommand()
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
	if !strings.Contains(warnings[0], "identical to parent") {
		t.Errorf("unexpected warning: %s", warnings[0])
	}

	// Different command → no warning
	c = &Config{
		Interaction: map[string]*InteractionCommand{
			"build": {
				Command: "cargo build",
				Subcommands: map[string]*InteractionCommand{
					"all": {Command: "cargo build --workspace"},
				},
			},
		},
	}
	warnings = c.warnDuplicateParentSubcommand()
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d", len(warnings))
	}

	// No subcommands → no warning
	c = &Config{
		Interaction: map[string]*InteractionCommand{
			"test": {Command: "cargo test"},
		},
	}
	warnings = c.warnDuplicateParentSubcommand()
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d", len(warnings))
	}
}

func TestWarnChildOverridesParentCritical(t *testing.T) {
	c := &Config{
		Interaction: map[string]*InteractionCommand{
			"app": {
				Runner: "local",
				Pod:    "app-pod",
				Subcommands: map[string]*InteractionCommand{
					"dev": {
						Runner: "docker-compose",
						Pod:    "dev-pod", // both overridden
					},
					"test": {
						Runner: "local",
						Pod:    "app-pod", // same as parent, no warning
					},
				},
			},
		},
	}

	warnings := c.warnChildOverridesParentCritical()
	if len(warnings) != 2 {
		t.Fatalf("expected 2 warnings, got %d", len(warnings))
	}

	hasRunnerWarn := false
	hasPodWarn := false
	for _, w := range warnings {
		if strings.Contains(w, "overrides parent runner") {
			hasRunnerWarn = true
		}
		if strings.Contains(w, "overrides parent pod") {
			hasPodWarn = true
		}
	}
	if !hasRunnerWarn || !hasPodWarn {
		t.Errorf("missing expected warnings: %v", warnings)
	}
}

func TestWarnDeepSubcommandNesting(t *testing.T) {
	c := &Config{
		Interaction: map[string]*InteractionCommand{
			"level0": {
				Subcommands: map[string]*InteractionCommand{
					"level1": {
						Subcommands: map[string]*InteractionCommand{
							"level2": {
								Subcommands: map[string]*InteractionCommand{
									"level3": {
										Subcommands: map[string]*InteractionCommand{
											"level4": {
												Subcommands: map[string]*InteractionCommand{
													"level5": {
														Subcommands: map[string]*InteractionCommand{
															"level6": {
																Command: "echo too deep",
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"shallow": {
				Subcommands: map[string]*InteractionCommand{
					"sub": {Command: "echo ok"},
				},
			},
		},
	}

	warnings := c.warnDeepSubcommandNesting()
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
	if !strings.Contains(warnings[0], "nested 6 levels deep") {
		t.Errorf("unexpected warning text: %s", warnings[0])
	}
}

func TestWarnUnreachableCommands(t *testing.T) {
	svc := "my-service"
	c := &Config{
		Interaction: map[string]*InteractionCommand{
			"unreachable": {
				Subcommands: map[string]*InteractionCommand{
					"sub": {Command: "echo ok"},
				},
			},
			"reachable_with_cmd": {
				Command: "echo hi",
				Subcommands: map[string]*InteractionCommand{
					"sub": {Command: "echo ok"},
				},
			},
			"reachable_with_svc": {
				Service: svc,
				Subcommands: map[string]*InteractionCommand{
					"sub": {Command: "echo ok"},
				},
			},
			"reachable_with_hooks": {
				Replace: []ProvisionItem{{Run: "echo replaced"}},
				Subcommands: map[string]*InteractionCommand{
					"sub": {Command: "echo ok"},
				},
			},
			// Was "reachable_without_subs", asserted here as producing no warning because the
			// check returned early on leaves. The name was the tell: nothing about this node is
			// reachable. `dva run dead_leaf` resolves, runs `sh -c ""` and exits 0. TASK-165
			// made it the second shape this check reports, so the row is renamed to what it is
			// and now asserts the warning rather than its absence.
			"dead_leaf": {},
			// The false positive that shape could produce, and the reason the check consults
			// inherited state rather than the raw node: this leaf sets nothing either, but
			// `dva run reachable_with_cmd inherits_target` runs `echo hi`. Warning about it
			// would make every correctly-factored config noisy — the mistake TASK-128 fixed for
			// the group shape, which this one must not reintroduce.
			"leaf_parent": {
				Command: "echo hi",
				Subcommands: map[string]*InteractionCommand{
					"inherits_target": {},
				},
			},
			// default_args alone executes: exec.buildCommandLine joins command and args and,
			// in shell mode, `sh -c` gets the args as the whole line. Measured — a node with
			// only `default_args: "echo reached"` prints `reached`. So this is a target, and a
			// warning here would be false.
			"args_only": {DefaultArgs: "echo reached"},
		},
	}

	warnings := c.warnUnreachableCommands()
	want := []string{
		"interaction.dead_leaf: has no execution target and no subcommands",
		"interaction.unreachable: has subcommands but is not directly callable",
	}
	if len(warnings) != len(want) {
		t.Fatalf("expected %d warnings, got %d: %v", len(want), len(warnings), warnings)
	}
	for _, w := range want {
		if !slices.ContainsFunc(warnings, func(got string) bool { return strings.Contains(got, w) }) {
			t.Errorf("missing warning %q, got: %v", w, warnings)
		}
	}
	// Named separately from the count: a count alone would not say which node went missing if
	// one of these ever started warning.
	for _, w := range warnings {
		for _, mustNotWarn := range []string{"inherits_target", "args_only"} {
			if strings.Contains(w, mustNotWarn) {
				t.Errorf("%s has an execution target (own or inherited) and must not warn: %s", mustNotWarn, w)
			}
		}
	}
}

// nestedInteractionConfig places the three mistakes the interaction-tree warnings look for
// one level below where the depth-1 versions of those checks could see them: `db` duplicates
// its own child's command and disagrees with its runner, and `grp.mid` is a group node with
// no execution target anywhere above it. A check that only walks the top level finds none of
// them.
//
// `rails.tools.nested` is the opposite case and is why it is here: it also sets no execution
// target, but inherits `tools-group` from its parent, so `dva run rails tools nested` runs.
// Warning about it is a false positive — the one this fixture originally asserted as correct
// (TASK-125), fixed in TASK-128.
func nestedInteractionConfig() *Config {
	return &Config{
		Interaction: map[string]*InteractionCommand{
			"rails": {
				Command: "bundle exec rails",
				Subcommands: map[string]*InteractionCommand{
					"db": {
						Command: "db-group",
						Runner:  "local",
						Subcommands: map[string]*InteractionCommand{
							"migrate": {Command: "db-group", Runner: "docker"},
						},
					},
					"tools": {
						Command: "tools-group",
						Subcommands: map[string]*InteractionCommand{
							"nested": {
								Subcommands: map[string]*InteractionCommand{
									"leaf": {Command: "echo leaf"},
								},
							},
						},
					},
				},
			},
			// Nothing in this chain supplies anything to execute, so `grp` and `grp.mid`
			// are both genuinely uncallable. `mid` is the depth-2 one.
			"grp": {
				Subcommands: map[string]*InteractionCommand{
					"mid": {
						Subcommands: map[string]*InteractionCommand{
							"leaf": {Command: "echo leaf"},
						},
					},
				},
			},
		},
	}
}

// TestInteractionWarningsRecurseIntoNestedSubcommands is the contract these three checks were
// missing. `subcommands` is recursive in the schema and the runner executes it to unbounded
// depth, so a warning that stops at depth 1 reports the shallow mistake and stays silent on
// the identical deep one. Measured before the fix: 3 warnings at depth 1, 0 at depth 2.
func TestInteractionWarningsRecurseIntoNestedSubcommands(t *testing.T) {
	c := nestedInteractionConfig()

	cases := []struct {
		name      string
		got       []string
		wantCount int
		wantPath  string
		wantText  string
	}{
		{
			name:      "warnDuplicateParentSubcommand",
			got:       c.warnDuplicateParentSubcommand(),
			wantCount: 1,
			wantPath:  "interaction.rails.subcommands.db.subcommands.migrate",
			wantText:  "identical to parent",
		},
		{
			name:      "warnChildOverridesParentCritical",
			got:       c.warnChildOverridesParentCritical(),
			wantCount: 1,
			wantPath:  "interaction.rails.subcommands.db.subcommands.migrate",
			wantText:  "overrides parent runner",
		},
		{
			// Two: `grp` at depth 0 and `grp.mid` at depth 2. The depth-2 one is the
			// contract; `grp` is included so the count is exact rather than a floor.
			name:      "warnUnreachableCommands",
			got:       c.warnUnreachableCommands(),
			wantCount: 2,
			wantPath:  "interaction.grp.subcommands.mid",
			wantText:  "not directly callable",
		},
	}

	for _, tc := range cases {
		if len(tc.got) != tc.wantCount {
			t.Errorf("%s: expected %d warning(s), got %d: %v", tc.name, tc.wantCount, len(tc.got), tc.got)
			continue
		}
		// The path must be the full YAML location, not just the top-level entry name —
		// a user cannot act on a warning that does not say which node it means.
		var found string
		for _, w := range tc.got {
			if strings.Contains(w, tc.wantPath) {
				found = w
				break
			}
		}
		if found == "" {
			t.Errorf("%s: want path %q in warnings, got: %v", tc.name, tc.wantPath, tc.got)
			continue
		}
		if !strings.Contains(found, tc.wantText) {
			t.Errorf("%s: want text %q in warning, got: %s", tc.name, tc.wantText, found)
		}
	}

	// The false positive this fixture used to assert as correct: `rails.tools.nested` sets no
	// execution target but inherits one, so it must not be reported. Asserted separately from
	// the count above because a count alone would not say which node went missing.
	for _, w := range c.warnUnreachableCommands() {
		if strings.Contains(w, "tools.subcommands.nested") {
			t.Errorf("inherited execution target must not warn as unreachable, got: %s", w)
		}
	}
}

// TestChildOverrideComparesInheritedRunner pins the fix for a false negative that only shows
// up below depth 1: the check read the parent's *raw* runner, but a middle node that sets no
// runner still passes its own parent's down.
//
// The two configs below are runtime-identical — `db` resolves to `local` either way, `migrate`
// to `docker` — and differ only in whether the author redundantly restated `runner: local` on
// the middle node. Before TASK-128 only the redundant one warned, making the trigger condition
// "did the author happen to type the value twice" rather than "does the backend change".
func TestChildOverrideComparesInheritedRunner(t *testing.T) {
	build := func(midRunner string) *Config {
		mid := &InteractionCommand{
			Subcommands: map[string]*InteractionCommand{
				"migrate": {Command: "db:migrate", Runner: "docker"},
			},
		}
		mid.Runner = midRunner
		return &Config{
			Interaction: map[string]*InteractionCommand{
				"rails": {
					Command:     "bundle exec rails",
					Runner:      "local",
					Subcommands: map[string]*InteractionCommand{"db": mid},
				},
			},
		}
	}

	for _, tc := range []struct{ name, midRunner string }{
		{"runner inherited implicitly", ""},
		{"runner restated redundantly", "local"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := build(tc.midRunner).warnChildOverridesParentCritical()
			if len(got) != 1 {
				t.Fatalf("expected 1 warning, got %d: %v", len(got), got)
			}
			const wantPath = "interaction.rails.subcommands.db.subcommands.migrate"
			if !strings.Contains(got[0], wantPath) {
				t.Errorf("want path %q, got: %s", wantPath, got[0])
			}
			// The effective parent runner, so both spellings report the same transition.
			if !strings.Contains(got[0], "(local → docker)") {
				t.Errorf("want the effective transition (local → docker), got: %s", got[0])
			}
		})
	}
}

// TestInteractionWarningsDepth1WordingIsUnchanged pins the exact depth-1 strings.
//
// The three pre-existing tests for these checks assert a phrase and a count
// (`strings.Contains(w, "identical to parent")`), never the config path, so every one of them
// would still pass if the recursion rewrite had mangled the `interaction.x.subcommands.y`
// prefix into something wrong. They are kept as-is, and this test supplies the byte-identity
// they do not cover: the fixture is the same shape used to measure the before/after output of
// the real binary, so a diff here is a diff a user would have seen.
func TestInteractionWarningsDepth1WordingIsUnchanged(t *testing.T) {
	c := &Config{
		Interaction: map[string]*InteractionCommand{
			"rails": {
				Command: "bundle exec rails",
				Runner:  "local",
				Subcommands: map[string]*InteractionCommand{
					"console": {Command: "bundle exec rails", Runner: "docker"},
				},
			},
			"grp": {
				Subcommands: map[string]*InteractionCommand{
					"leaf": {Command: "echo leaf"},
				},
			},
		},
	}

	cases := []struct {
		name string
		got  []string
		want []string
	}{
		{
			name: "warnDuplicateParentSubcommand",
			got:  c.warnDuplicateParentSubcommand(),
			want: []string{`interaction.rails.subcommands.console: command "bundle exec rails" is identical to parent; subcommand is redundant`},
		},
		{
			name: "warnChildOverridesParentCritical",
			got:  c.warnChildOverridesParentCritical(),
			want: []string{"interaction.rails.subcommands.console: overrides parent runner (local → docker); this may change execution backend unexpectedly"},
		},
		{
			name: "warnUnreachableCommands",
			got:  c.warnUnreachableCommands(),
			want: []string{"interaction.grp: has subcommands but is not directly callable; add an execution target or remove subcommands"},
		},
	}

	for _, tc := range cases {
		if !slices.Equal(tc.got, tc.want) {
			t.Errorf("%s: depth-1 output changed\n want: %q\n got:  %q", tc.name, tc.want, tc.got)
		}
	}
}
