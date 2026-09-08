package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
	"github.com/ScriptonBasestar/dva/internal/runner"
)

// TASK-267 item 1: buildManifestSubprojectCommands used to emit
// `usage_example: "dva <project>:<key>"` unconditionally, the same defect interactionUsage
// already exists to fix for local (non-subproject) keys. The fixture below reproduces the
// shape that exposes it — a parent declaring both a literal `engine:test` interaction key
// AND a same-named `engine` subproject whose own `test` entry the literal key shadows —
// measured against ./bin/dva before this fix:
//
//	dva engine:test                 -> PARENT-LITERAL        (parent's literal key wins)
//	dva engine:build                -> CHILD-ENGINE-BUILD    (colon form reaches the child)
//	dva run --project engine test   -> CHILD-ENGINE-TEST
//
// yet the manifest's "test" entry advertised `usage_example: "dva engine:test"`, which
// provably ran the parent's command instead.
//
// The unshadowed key in that measurement was `build`, and it is spelled `compile` below.
// `build` is a reserved built-in, so the child's own `dva config validate` rejects it, and
// since TASK-342 every parent route refuses it too — the row would come through unroutable
// and stop demonstrating the unshadowed case at all. That is the rule working, not a fixture
// accident: this file predates it and happened to pick a name the child cannot legally
// declare. The measurement above is left in its original spelling because it is a record of
// what was executed.

// writeShadowedSubprojectFixture writes a parent dva.yml with a literal `engine:test` key
// and an `engine` subproject whose `test`/`compile` entries the literal key partially shadows,
// plus a `guard` subproject holding the fourth state: `status` is a reserved built-in, so
// guard's own `dva config validate` rejects that key and — since TASK-342 — every parent
// route refuses it, while `ok` beside it routes normally. Returns the loaded parent config.
//
// The `guard` slot used to hold a subproject named `run`, pinning D1's claim that a
// subproject sharing a name with a built-in still routes and so comes through unmarked. That
// claim was retired with the decision it rested on: TASK-263 §3 (a) makes such a name a hard
// validation error, so the config that test measured can no longer load past validate, and
// its coverage moved to config.TestSubprojectReservedNameRejected. The pair kept here is the
// one that still discriminates — a rejected key and a healthy one in the SAME child, which a
// check that refused whole subprojects rather than single keys would fail on `ok`.
func writeShadowedSubprojectFixture(t *testing.T) *config.Config {
	t.Helper()
	tmpDir := t.TempDir()

	engineDir := filepath.Join(tmpDir, "engine")
	if err := os.MkdirAll(engineDir, 0o755); err != nil {
		t.Fatalf("create engine dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(engineDir, config.FileName), []byte(`
version: "0.1.0"
interaction:
  test:
    description: "child engine test"
    command: "echo CHILD-ENGINE-TEST"
  compile:
    description: "child engine compile"
    command: "echo CHILD-ENGINE-COMPILE"
`), 0o644); err != nil {
		t.Fatalf("write engine dva.yml: %v", err)
	}

	guardDir := filepath.Join(tmpDir, "guard")
	if err := os.MkdirAll(guardDir, 0o755); err != nil {
		t.Fatalf("create guard dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(guardDir, config.FileName), []byte(`
version: "0.1.0"
interaction:
  status:
    description: "child guard status"
    command: "echo CHILD-GUARD-STATUS"
  ok:
    description: "child guard ok"
    command: "echo CHILD-GUARD-OK"
`), 0o644); err != nil {
		t.Fatalf("write guard dva.yml: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, config.FileName), []byte(`
version: "0.1.0"
interaction:
  engine:test:
    description: "parent literal"
    command: "echo PARENT-LITERAL"
subprojects:
  engine:
    path: ./engine
  guard:
    path: ./guard
`), 0o644); err != nil {
		t.Fatalf("write parent dva.yml: %v", err)
	}

	c, err := config.Load(tmpDir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	return c
}

// TestManifestSubprojectShadowedKeyUsesWorkingUsage pins D1/D5: a shadowed subproject key's
// emitted usage_example must invoke the child entry, not the shadowing parent key, and the
// entry must carry shadowed_by_literal_key while carrying NEITHER unroutable NOR
// shadowed_by_builtin (ShadowedByBuiltin names a static_commands entry, which a parent
// interaction key is not; unroutable is now a real state for a subproject command, but a
// different one — see TestManifestSubprojectRejectedChildKeyOffersNoInvocation below).
func TestManifestSubprojectShadowedKeyUsesWorkingUsage(t *testing.T) {
	c := writeShadowedSubprojectFixture(t)
	m := buildManifest(c)

	entry, ok := m.Subprojects["engine"].Commands["test"]
	if !ok {
		t.Fatalf("engine subproject commands = %v, missing 'test'", m.Subprojects["engine"].Commands)
	}
	const wantUsage = "dva run --project engine test"
	if entry.UsageExample != wantUsage {
		t.Errorf("usage_example = %q, want %q — the pre-fix form (\"dva engine:test\") provably runs the parent's PARENT-LITERAL command instead", entry.UsageExample, wantUsage)
	}
	if entry.ShadowedByLiteralKey != "engine:test" {
		t.Errorf("shadowed_by_literal_key = %q, want %q", entry.ShadowedByLiteralKey, "engine:test")
	}
	if entry.Unroutable != "" {
		t.Errorf("unroutable = %q, want empty — `dva run --project engine test` reaches this key; the child's own validator does not reject `test`", entry.Unroutable)
	}
	if entry.ShadowedByBuiltin != "" {
		t.Errorf("shadowed_by_builtin = %q, want empty — that field names a static_commands entry, and the parent's `engine:test` interaction key is not one", entry.ShadowedByBuiltin)
	}
}

// TestManifestSubprojectUnshadowedKeyKeepsColonForm pins the counterpart: a subproject key
// the parent does NOT also declare as a literal interaction key keeps the plain
// `dva <project>:<key>` form and sets no shadow marker at all.
func TestManifestSubprojectUnshadowedKeyKeepsColonForm(t *testing.T) {
	c := writeShadowedSubprojectFixture(t)
	m := buildManifest(c)

	entry, ok := m.Subprojects["engine"].Commands["compile"]
	if !ok {
		t.Fatalf("engine subproject commands = %v, missing 'compile'", m.Subprojects["engine"].Commands)
	}
	const wantUsage = "dva engine:compile"
	if entry.UsageExample != wantUsage {
		t.Errorf("usage_example = %q, want %q", entry.UsageExample, wantUsage)
	}
	if entry.ShadowedByLiteralKey != "" {
		t.Errorf("shadowed_by_literal_key = %q, want empty — the parent declares no literal `engine:compile` key", entry.ShadowedByLiteralKey)
	}
}

// TestManifestSubprojectRejectedChildKeyOffersNoInvocation holds the manifest to the promise
// stated on UsageExample: the string it carries invokes the entry it sits inside.
//
// TASK-342 made the parent refuse a key the child's own validator rejects, and left this loop
// still emitting `usage_example: "dva guard:status"` for one. Measured against that build,
// `dva config validate` reported the parent valid while `dva guard:status` exited 1 — the
// manifest advertised a form the same binary refused, which is the exact defect TASK-267
// item 1 fixed for the shadowed case and this repeats for the unroutable one.
//
// The reason must be the CHILD's own diagnosis, not a sentence composed here: it is what
// `dva config validate` prints inside guard, and a second wording would leave the reader
// reconciling two accounts of one refusal.
func TestManifestSubprojectRejectedChildKeyOffersNoInvocation(t *testing.T) {
	c := writeShadowedSubprojectFixture(t)
	m := buildManifest(c)

	entry, ok := m.Subprojects["guard"].Commands["status"]
	if !ok {
		t.Fatalf("guard subproject commands = %v, missing 'status'", m.Subprojects["guard"].Commands)
	}
	// Listed, not dropped: the author declared it and needs to see dva received it. What it
	// must not carry is a runnable-looking string.
	if entry.UsageExample != "" {
		t.Errorf("usage_example = %q, want empty — every parent route refuses `status`, so no invocation reaches this entry", entry.UsageExample)
	}
	if entry.Unroutable != "status" {
		t.Errorf("unroutable = %q, want %q — the reserved built-in the child's validator rejects the key over", entry.Unroutable, "status")
	}
	if want := config.ConflictAdvice("status"); entry.UnroutableReason != want {
		t.Errorf("unroutable_reason = %q, want the child's own advice %q", entry.UnroutableReason, want)
	}
	if entry.ShadowedByLiteralKey != "" {
		t.Errorf("shadowed_by_literal_key = %q, want empty — the parent declares no literal `guard:status` key, and a key nothing reaches cannot also be shadowed", entry.ShadowedByLiteralKey)
	}
}

// TestManifestSubprojectHealthyKeyBesideRejectedOneStillRoutes is the other half. Without it
// the cheapest way to pass the test above is to mark every key in a child that has one bad
// key — which would contradict the per-key rejection TASK-263 §3 (b) chose, and silently
// retract two working entries for one broken neighbour.
func TestManifestSubprojectHealthyKeyBesideRejectedOneStillRoutes(t *testing.T) {
	c := writeShadowedSubprojectFixture(t)
	m := buildManifest(c)

	entry, ok := m.Subprojects["guard"].Commands["ok"]
	if !ok {
		t.Fatalf("guard subproject commands = %v, missing 'ok'", m.Subprojects["guard"].Commands)
	}
	const wantUsage = "dva guard:ok"
	if entry.UsageExample != wantUsage {
		t.Errorf("usage_example = %q, want %q — `ok` is healthy; its neighbour's rejection must not reach it", entry.UsageExample, wantUsage)
	}
	if entry.Unroutable != "" || entry.UnroutableReason != "" || entry.ShadowedByLiteralKey != "" {
		t.Errorf("entry = %+v, want no marker set", entry)
	}
}

// TestLsProjectMarksRejectedChildKey covers the human listing and the machine one together,
// because they are the two surfaces the manifest test above does not reach and they must not
// disagree — the same reason subprojectUsage is one function.
//
// `dva ls --project guard` used to print `status` as an ordinary row and `--json` carried no
// marker for it, while the root listing already marked such keys. A reader comparing the two
// scopes would conclude the subproject key was fine.
func TestLsProjectMarksRejectedChildKey(t *testing.T) {
	c := writeShadowedSubprojectFixture(t)

	oldDetailed, oldFormat := lsDetailed, lsFormat
	t.Cleanup(func() { lsDetailed, lsFormat = oldDetailed, oldFormat })

	t.Run("table", func(t *testing.T) {
		lsDetailed, lsFormat = false, ""
		var runErr error
		out := captureOutput(t, func() { runErr = runLsProject(c, "guard") })
		if runErr != nil {
			t.Fatalf("runLsProject(guard) error: %v", runErr)
		}
		const wantMark = "(unreachable: subproject 'guard' rejects 'status' — 'status' is a reserved DVA command)"
		if !strings.Contains(out, wantMark) {
			t.Errorf("output = %q, want the rejected row marked: %q", out, wantMark)
		}
		// The healthy row carries no mark at all. Asserting only the mark's presence above
		// would pass a change that marked every row.
		for line := range strings.SplitSeq(out, "\n") {
			if strings.Contains(line, "child guard ok") && strings.Contains(line, "unreachable") {
				t.Errorf("healthy row is marked unreachable: %q", line)
			}
		}
	})

	t.Run("json", func(t *testing.T) {
		lsDetailed, lsFormat = false, "json"
		var runErr error
		out := captureOutput(t, func() { runErr = runLsProject(c, "guard") })
		if runErr != nil {
			t.Fatalf("runLsProject(guard) error: %v", runErr)
		}
		var entries map[string]map[string]any
		if err := json.Unmarshal([]byte(out), &entries); err != nil {
			t.Fatalf("unmarshal ls --project guard -f json: %v\n%s", err, out)
		}
		if got := entries["status"]["unroutable"]; got != "status" {
			t.Errorf("status.unroutable = %v, want %q", got, "status")
		}
		if got, want := entries["status"]["unroutable_reason"], config.ConflictAdvice("status"); got != want {
			t.Errorf("status.unroutable_reason = %v, want the child's own advice %q", got, want)
		}
		// Presence is the signal, the contract every marker field in this listing shares.
		if _, marked := entries["ok"]["unroutable"]; marked {
			t.Errorf("healthy `ok` entry carries unroutable: %v", entries["ok"])
		}
	})
}

// TestLsProjectFlag_Registered pins item 2's other requirement: `--project` must actually be
// registered on lsCmd, not just documented. Before this fix, run.go's recovery hint ("Run
// 'dva ls --project %s'") named a flag that only existed on runCmd, and `dva ls --project x`
// exited non-zero with an unknown-flag error.
func TestLsProjectFlag_Registered(t *testing.T) {
	flag := lsCmd.Flags().Lookup("project")
	if flag == nil {
		t.Fatal("lsCmd has no --project flag registered")
	}
	if flag.Shorthand != "p" {
		t.Errorf("--project shorthand = %q, want %q", flag.Shorthand, "p")
	}
}

// TestLsProject_ListsSubprojectInteractions covers runLsProject end to end: it must load the
// named subproject the same way `dva run --project` does and list its interaction keys,
// applying the same subprojectUsage marking the manifest test above pins. Matching
// printTable's own convention (see printTable's shadowedBy/unroutable marks above), the
// working usage string only surfaces inside a row's parenthetical mark when subprojectUsage
// reports a shadow — an unshadowed row prints just "<key>  # <description>" and never spells
// out "dva <project>:<key>" literally, so this only asserts the mark's presence/absence, not a
// literal usage string on every row.
func TestLsProject_ListsSubprojectInteractions(t *testing.T) {
	c := writeShadowedSubprojectFixture(t)

	oldDetailed := lsDetailed
	lsDetailed = false
	t.Cleanup(func() { lsDetailed = oldDetailed })

	var runErr error
	out := captureOutput(t, func() {
		runErr = runLsProject(c, "engine")
	})
	if runErr != nil {
		t.Fatalf("runLsProject(engine) error: %v", runErr)
	}
	const wantCompileLine = "compile  # child engine compile\n"
	if !strings.Contains(out, wantCompileLine) {
		t.Errorf("output = %q, want the unshadowed row unmarked: %q", out, wantCompileLine)
	}
	const wantTestLine = "test     # child engine test  (parent key 'engine:test' takes this name; run: dva run --project engine test)\n"
	if !strings.Contains(out, wantTestLine) {
		t.Errorf("output = %q, want the shadowed entry's mark naming the working usage form: %q", out, wantTestLine)
	}
}

// TestLsProject_UnknownProject pins the error shape: an unknown project must produce the
// same "subproject `%s` not found. Available: ..." message run.go already uses, because
// loadSubprojectConfig is now the single source both callers go through.
func TestLsProject_UnknownProject(t *testing.T) {
	c := &config.Config{
		Subprojects: map[string]config.SubprojectConfig{
			"engine": {Path: "./engine"},
		},
	}
	err := runLsProject(c, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent subproject")
	}
	const want = "subproject `nonexistent` not found. Available: engine"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

// TestSubprojectUsage covers the predicate directly, including shapes the manifest-level
// tests above do not exercise cheaply.
func TestSubprojectUsage(t *testing.T) {
	parent := &config.Config{Interaction: map[string]*config.InteractionCommand{
		"engine:test": {Command: "echo parent"},
	}}
	sub := &config.Config{Interaction: map[string]*config.InteractionCommand{
		"test":    {Command: "echo child test"},
		"compile": {Command: "echo child compile"},
		"status":  {Command: "echo child status"},
	}}

	usage, shadowed, unroutable, _ := subprojectUsage(parent, sub, "engine", "test", nil)
	if usage != "dva run --project engine test" || shadowed != "engine:test" || unroutable != "" {
		t.Errorf("shadowed case: usage=%q shadowed=%q unroutable=%q", usage, shadowed, unroutable)
	}

	usage, shadowed, unroutable, _ = subprojectUsage(parent, sub, "engine", "compile", nil)
	if usage != "dva engine:compile" || shadowed != "" || unroutable != "" {
		t.Errorf("unshadowed case: usage=%q shadowed=%q unroutable=%q", usage, shadowed, unroutable)
	}

	usage, shadowed, unroutable, reason := subprojectUsage(parent, sub, "engine", "status", nil)
	if usage != "" || shadowed != "" || unroutable != "status" || reason != config.ConflictAdvice("status") {
		t.Errorf("rejected case: usage=%q shadowed=%q unroutable=%q reason=%q", usage, shadowed, unroutable, reason)
	}

	// The declared key decides, not the flattened display key. runSubprojectCommand checks
	// args[0] and never looks at the rest, so `status build` is dead for the same reason
	// `status` is — and a lookup keyed on the display string would find nothing in the
	// child's Interaction map and report the row as runnable.
	usage, _, unroutable, _ = subprojectUsage(parent, sub, "engine", "status build",
		&runner.ResolvedCommand{Name: "build", Path: []string{"status", "build"}})
	if usage != "" || unroutable != "status" {
		t.Errorf("flattened subcommand of a rejected key: usage=%q unroutable=%q", usage, unroutable)
	}
}
