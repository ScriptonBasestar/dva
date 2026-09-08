package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
	"github.com/ScriptonBasestar/dva/internal/runner"
)

// writeImportedItemIdentityFixture builds a parent dva.yml that imports one interaction
// from a subproject under an `as:` alias, alongside a locally declared interaction. TASK-333
// needs a real import: block (not just a bare `subprojects:` entry, which the shadowed-key
// fixture in subproject_usage_test.go uses) so resolveSubprojectImports actually stamps
// SubprojectName/CanonicalAddress onto the cloned *config.InteractionCommand these tests
// assert on — a hand-built ResolvedCommand map would only prove buildCommandEntries reads
// those fields correctly, not that subproject.go ever sets them.
func writeImportedItemIdentityFixture(t *testing.T) *config.Config {
	t.Helper()
	tmpDir := t.TempDir()

	engineDir := filepath.Join(tmpDir, "engine")
	if err := os.MkdirAll(engineDir, 0o755); err != nil {
		t.Fatalf("create engine dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(engineDir, config.FileName), []byte(`
version: "0.1.0"
interaction:
  compile:
    description: "child engine compile"
    command: "echo CHILD-ENGINE-COMPILE"
`), 0o644); err != nil {
		t.Fatalf("write engine dva.yml: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, config.FileName), []byte(`
version: "0.1.0"
interaction:
  local-task:
    description: "declared directly in the root dva.yml"
    command: "echo ROOT-LOCAL-TASK"
subprojects:
  engine:
    path: ./engine
    import:
      interactions:
        - name: compile
          as: fast-compile
`), 0o644); err != nil {
		t.Fatalf("write parent dva.yml: %v", err)
	}

	c, err := config.Load(tmpDir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	return c
}

// TestImportedItemOwnerField pins TASK-333 item 1: every item in `dva ls --json` and in
// `dva manifest` names the subproject it was imported from, or "root" for a locally
// declared item. It exercises the real import pipeline (config.Load ->
// resolveSubprojectImports -> InteractionTree.List -> buildCommandEntries/buildManifest),
// not a hand-built ResolvedCommand map, so it would fail if subproject.go stopped stamping
// SubprojectName or if list.go/manifest.go stopped reading it.
func TestImportedItemOwnerField(t *testing.T) {
	c := writeImportedItemIdentityFixture(t)

	tree := runner.NewInteractionTree(c.Interaction)
	commands := tree.List()
	keys := sortedKeys(commands)

	entries := buildCommandEntries(c, commands, keys)

	local, ok := entries["local-task"].(map[string]any)
	if !ok {
		t.Fatalf("entries missing local-task: %#v", entries)
	}
	if local["owner"] != rootOwnerName {
		t.Errorf("local-task owner = %v, want %q", local["owner"], rootOwnerName)
	}

	imported, ok := entries["engine/compile"].(map[string]any)
	if !ok {
		t.Fatalf("entries missing engine/compile: %#v", entries)
	}
	if imported["owner"] != "engine" {
		t.Errorf("engine/compile owner = %v, want \"engine\"", imported["owner"])
	}

	// The `as:` alias is a second map key pointing at the same import, so it carries the
	// same owner as its canonical address.
	alias, ok := entries["fast-compile"].(map[string]any)
	if !ok {
		t.Fatalf("entries missing fast-compile (alias of engine/compile): %#v", entries)
	}
	if alias["owner"] != "engine" {
		t.Errorf("fast-compile (alias) owner = %v, want \"engine\"", alias["owner"])
	}

	// Same field, same values, via `dva manifest`'s own build path.
	m := buildManifest(c)
	if got := m.DynamicCommands["local-task"].Owner; got != rootOwnerName {
		t.Errorf("manifest local-task owner = %q, want %q", got, rootOwnerName)
	}
	if got := m.DynamicCommands["engine/compile"].Owner; got != "engine" {
		t.Errorf("manifest engine/compile owner = %q, want \"engine\"", got)
	}
	if got := m.DynamicCommands["fast-compile"].Owner; got != "engine" {
		t.Errorf("manifest fast-compile (alias) owner = %q, want \"engine\"", got)
	}
}

// TestImportedItemCanonicalAndAliasMarkers pins TASK-333 item 2: an item reachable under
// more than one address states which address is canonical (`engine/compile`, the
// `<subproject>/<name>` import route) and which are aliases (`fast-compile`, the `as:` name) —
// in both `dva ls --json` and `dva manifest`. Like the owner test above, this runs the real
// import pipeline so it fails if subproject.go stops stamping CanonicalAddress or if
// interactionAliasGroups/buildCommandEntries/buildManifest stop deriving the markers from it.
func TestImportedItemCanonicalAndAliasMarkers(t *testing.T) {
	c := writeImportedItemIdentityFixture(t)

	tree := runner.NewInteractionTree(c.Interaction)
	commands := tree.List()
	keys := sortedKeys(commands)

	entries := buildCommandEntries(c, commands, keys)

	canonical, ok := entries["engine/compile"].(map[string]any)
	if !ok {
		t.Fatalf("entries missing engine/compile: %#v", entries)
	}
	if _, has := canonical["alias_of"]; has {
		t.Errorf("engine/compile carries alias_of %v, want none (it is the canonical address)", canonical["alias_of"])
	}
	aliases, _ := canonical["aliases"].([]string)
	if len(aliases) != 1 || aliases[0] != "fast-compile" {
		t.Errorf("engine/compile aliases = %v, want [\"fast-compile\"]", canonical["aliases"])
	}

	alias, ok := entries["fast-compile"].(map[string]any)
	if !ok {
		t.Fatalf("entries missing fast-compile: %#v", entries)
	}
	if _, has := alias["aliases"]; has {
		t.Errorf("fast-compile carries aliases %v, want none (it is the alias)", alias["aliases"])
	}
	if alias["alias_of"] != "engine/compile" {
		t.Errorf("fast-compile alias_of = %v, want \"engine/compile\"", alias["alias_of"])
	}

	// A locally declared, never-imported item is reachable under exactly one address and
	// must carry neither marker.
	local, ok := entries["local-task"].(map[string]any)
	if !ok {
		t.Fatalf("entries missing local-task: %#v", entries)
	}
	if _, has := local["aliases"]; has {
		t.Errorf("local-task carries aliases %v, want none", local["aliases"])
	}
	if _, has := local["alias_of"]; has {
		t.Errorf("local-task carries alias_of %v, want none", local["alias_of"])
	}

	// Same distinction via `dva manifest`.
	m := buildManifest(c)
	canonicalCmd := m.DynamicCommands["engine/compile"]
	if canonicalCmd.AliasOf != "" {
		t.Errorf("manifest engine/compile alias_of = %q, want \"\"", canonicalCmd.AliasOf)
	}
	if len(canonicalCmd.Aliases) != 1 || canonicalCmd.Aliases[0] != "fast-compile" {
		t.Errorf("manifest engine/compile aliases = %v, want [\"fast-compile\"]", canonicalCmd.Aliases)
	}
	aliasCmd := m.DynamicCommands["fast-compile"]
	if aliasCmd.AliasOf != "engine/compile" {
		t.Errorf("manifest fast-compile alias_of = %q, want \"engine/compile\"", aliasCmd.AliasOf)
	}
	if len(aliasCmd.Aliases) != 0 {
		t.Errorf("manifest fast-compile aliases = %v, want none", aliasCmd.Aliases)
	}
}

// TestCompletionOffersAllThreeAddressForms pins TASK-333 item 3: shell completion offers
// all three address forms a subproject interaction is reachable under —
// `--project <name>` (the flag value itself), `<name>:<key>` (the colon shorthand,
// run.go:141), and `<name>/<key>` (the import route, subproject.go). It calls the
// registered cobra completion functions directly rather than asserting on completion.go's
// source text, so it fails if either function stops offering the form it is responsible for.
func TestCompletionOffersAllThreeAddressForms(t *testing.T) {
	tmpDir := t.TempDir()

	engineDir := filepath.Join(tmpDir, "engine")
	if err := os.MkdirAll(engineDir, 0o755); err != nil {
		t.Fatalf("create engine dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(engineDir, config.FileName), []byte(`
version: "0.1.0"
interaction:
  compile:
    description: "child engine compile"
    command: "echo CHILD-ENGINE-COMPILE"
`), 0o644); err != nil {
		t.Fatalf("write engine dva.yml: %v", err)
	}

	// `broken` points at a directory that does not exist, and carries no import: block
	// so that config.Load below still succeeds — the same shape as a sibling checkout a
	// developer has not cloned, or one pinned to a config version this binary refuses.
	// It is in this fixture, rather than a test of its own, so that every assertion
	// below doubles as a regression test: completionCandidates used to call
	// config.LoadSubprojects over the whole map at once, and that function discards
	// partial results on any failure, so this one absent path erased `engine:compile`
	// too while `dva run engine:compile` still routed it.
	if err := os.WriteFile(filepath.Join(tmpDir, config.FileName), []byte(`
version: "0.1.0"
subprojects:
  engine:
    path: ./engine
    import:
      interactions:
        - name: compile
  broken:
    path: ./nowhere
`), 0o644); err != nil {
		t.Fatalf("write parent dva.yml: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("restore Chdir: %v", err)
		}
	}()

	// Form 1: `--project <name>` — the flag's own completion function must offer the
	// subproject name.
	projectFn, ok := runCmd.GetFlagCompletionFunc("project")
	if !ok || projectFn == nil {
		t.Fatal("runCmd has no registered completion function for --project")
	}
	projectNames, _ := projectFn(runCmd, nil, "")
	if !contains(projectNames, "engine") {
		t.Errorf("--project completion = %v, want it to include \"engine\"", projectNames)
	}

	// Forms 2 and 3: `<name>:<key>` and `<name>/<key>` are both positional-argument
	// completions for `dva run`.
	if runCmd.ValidArgsFunction == nil {
		t.Fatal("runCmd has no ValidArgsFunction registered")
	}
	positional, _ := runCmd.ValidArgsFunction(runCmd, nil, "")
	if !contains(positional, "engine:compile") {
		t.Errorf("run completion = %v, want it to include \"engine:compile\"", positional)
	}
	if !contains(positional, "engine/compile") {
		t.Errorf("run completion = %v, want it to include \"engine/compile\"", positional)
	}

	// The unloadable subproject costs its own colon forms and no others. Asserted as a
	// prefix scan rather than one literal key because `broken`'s dva.yml cannot be read,
	// so the set of keys it would have contributed is not knowable from here.
	for _, name := range positional {
		if strings.HasPrefix(name, "broken:") {
			t.Errorf("run completion = %v, want no candidate from the unloadable subproject", positional)
			break
		}
	}

	// --project still offers it: the name is declared, and `dva run --project broken k`
	// reporting "loading subproject" beats completion staying silent about a name the
	// user wrote in their own dva.yml.
	if !contains(projectNames, "broken") {
		t.Errorf("--project completion = %v, want it to include \"broken\"", projectNames)
	}
}
