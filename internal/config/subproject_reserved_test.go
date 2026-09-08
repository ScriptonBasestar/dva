package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// writeSubprojectFixture lays down a parent dva.yml plus one child directory named by
// childDir, and returns the parent directory.
//
// Both files are written by one helper because the two rules TASK-263 §3 froze are about
// the pair: a name the parent declares and a key the child declares. A test that built only
// one side would have to assume the other, which is how the parent and the child came to
// disagree about which keys exist in the first place.
func writeSubprojectFixture(t *testing.T, parentYAML, childDir, childYAML string) string {
	t.Helper()

	parentDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(parentDir, FileName), []byte(parentYAML), 0o644); err != nil {
		t.Fatalf("write parent config: %v", err)
	}
	if childYAML != "" {
		childPath := filepath.Join(parentDir, childDir)
		if err := os.MkdirAll(childPath, 0o755); err != nil {
			t.Fatalf("create child dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(childPath, FileName), []byte(childYAML), 0o644); err != nil {
			t.Fatalf("write child config: %v", err)
		}
	}
	return parentDir
}

// TestSubprojectReservedNameRejected covers TASK-263 §3 decision (a).
//
// The rejection is `config validate`'s, not the router's, which is what the decision says
// and what the divergence needs. Measured on v0.1.48 against exactly this shape, `dva
// up:web` ran the child's `web` while the same invocation logged ConflictAdvice's claim
// that "no invocation reaches this key ... it fails with subproject 'up' not found". The
// advice is true of every config that passes validate only once validate refuses this one.
//
// The free-name leg is not filler: a check that rejected every subproject whose name
// happens to appear somewhere in the CLI, or that rejected on a substring, would pass the
// reserved leg alone and quietly break every ordinary config.
func TestSubprojectReservedNameRejected(t *testing.T) {
	for _, tc := range []struct {
		name       string
		subproject string
		wantErr    bool
	}{
		{name: "reserved built-in", subproject: "up", wantErr: true},
		{name: "another reserved built-in", subproject: "config", wantErr: true},
		{name: "free name", subproject: "engine", wantErr: false},
		{name: "free name containing a reserved one", subproject: "up-stream", wantErr: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parentDir := writeSubprojectFixture(t,
				"version: \"0.1.0\"\nsubprojects:\n  "+tc.subproject+":\n    path: child\n",
				"", "")

			cfg, err := Load(parentDir)
			if err != nil {
				t.Fatalf("Load error: %v", err)
			}

			err = cfg.Validate()
			if !tc.wantErr {
				if err != nil {
					t.Fatalf("subproject %q was rejected: %v", tc.subproject, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("subproject %q validated clean; a reserved built-in name must be rejected", tc.subproject)
			}
			// The reserved word itself, because the author's fix is to rename, and a message
			// that says "reserved" without saying which word leaves them guessing at a set
			// that is 29 names long.
			if !strings.Contains(err.Error(), "'"+tc.subproject+"' is a reserved DVA command") {
				t.Errorf("error does not name the reserved word %q: %v", tc.subproject, err)
			}
		})
	}
}

// TestSubprojectRejectionNamesRuleAndDeclaration holds both rejection paths to the third
// thing TASK-263 asked for: a message a reader can act on without opening the source.
//
// It asserts on the errors the real paths return, not on the two advice constructors. The
// constructors are what the code shares; the errors are what the reader gets, and the gap
// between them is exactly where a wrapper drops the declaration — which is what happened
// before Validate started prefixing `subprojects.<name>`.
func TestSubprojectRejectionNamesRuleAndDeclaration(t *testing.T) {
	t.Run("reserved subproject name", func(t *testing.T) {
		parentDir := writeSubprojectFixture(t,
			"version: \"0.1.0\"\nsubprojects:\n  up:\n    path: child\n", "", "")

		cfg, err := Load(parentDir)
		if err != nil {
			t.Fatalf("Load error: %v", err)
		}
		err = cfg.Validate()
		if err == nil {
			t.Fatal("expected a rejection")
		}

		// Declaration: which key in the file. Rule: why it is refused.
		if !strings.Contains(err.Error(), "subprojects.up") {
			t.Errorf("error does not name the declaration `subprojects.up`: %v", err)
		}
		if !strings.Contains(err.Error(), "reserved subproject name") {
			t.Errorf("error does not name the rule that fired: %v", err)
		}
	})

	// TASK-333. `root` is refused for a different reason than the names above and by a
	// different check, so this asserts the two messages stay distinct: the reserved-name
	// text sends a reader to reservedCommands, and `root` is deliberately not in it.
	t.Run("owner sentinel as a subproject name", func(t *testing.T) {
		parentDir := writeSubprojectFixture(t,
			"version: \"0.1.0\"\nsubprojects:\n  "+RootOwnerName+":\n    path: child\n", "", "")

		cfg, err := Load(parentDir)
		if err != nil {
			t.Fatalf("Load error: %v", err)
		}
		err = cfg.Validate()
		if err == nil {
			t.Fatal("expected a rejection")
		}

		if !strings.Contains(err.Error(), "subprojects."+RootOwnerName) {
			t.Errorf("error does not name the declaration `subprojects.%s`: %v", RootOwnerName, err)
		}
		// The rule, in the reader's terms: the field this name would make ambiguous.
		if !strings.Contains(err.Error(), "owner") {
			t.Errorf("error does not name the owner field it protects: %v", err)
		}
		// Not the other message. Sharing it would point the reader at a list `root` is
		// deliberately absent from, which is the disagreement this separation avoids.
		if strings.Contains(err.Error(), "reserved subproject name") {
			t.Errorf("owner-sentinel rejection reused the reserved-name message: %v", err)
		}
	})

	t.Run("child key rejected through the import route", func(t *testing.T) {
		parentDir := writeSubprojectFixture(t,
			"version: \"0.1.0\"\nsubprojects:\n  engine:\n    path: child\n    import:\n      interactions:\n        - status\n",
			"child",
			"version: \"0.1.0\"\ninteraction:\n  status:\n    command: echo CHILD-STATUS\n")

		_, err := Load(parentDir)
		if err == nil {
			t.Fatal("the import succeeded; a key the child's own validator rejects must not become `engine/status`")
		}

		if !strings.Contains(err.Error(), "subprojects.engine") {
			t.Errorf("error does not name the declaring subproject: %v", err)
		}
		if !strings.Contains(err.Error(), "interaction `status`") {
			t.Errorf("error does not name the offending key: %v", err)
		}
		if !strings.Contains(err.Error(), "rejected by `dva config validate` inside subproject") {
			t.Errorf("error does not name the rule that fired: %v", err)
		}
		// The child's own diagnosis travels with it: the reader is standing in the parent
		// directory and has no reason to have run validate in the child.
		if !strings.Contains(err.Error(), ConflictAdvice("status")) {
			t.Errorf("error drops the child's own advice, leaving no fix: %v", err)
		}
	})
}

// TestSubprojectImportAcceptsKeysTheChildValidatorAllows is the other half of the import
// rule. Rejecting per key rather than per config is a deliberate choice — a child with one
// bad key keeps every other key importable — and without this the cheapest way to pass the
// test above is to refuse any child that has a conflict anywhere.
func TestSubprojectImportAcceptsKeysTheChildValidatorAllows(t *testing.T) {
	parentDir := writeSubprojectFixture(t,
		"version: \"0.1.0\"\nsubprojects:\n  engine:\n    path: child\n    import:\n      interactions:\n        - test\n",
		"child",
		"version: \"0.1.0\"\ninteraction:\n  test:\n    command: echo CHILD-TEST\n  status:\n    command: echo CHILD-STATUS\n")

	cfg, err := Load(parentDir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if _, ok := cfg.Interaction["engine/test"]; !ok {
		t.Fatal("`engine/test` was not imported; the child's unrelated `status` conflict must not block a healthy key")
	}
	if _, ok := cfg.Interaction["engine/status"]; ok {
		t.Fatal("`engine/status` appeared without being imported")
	}
}

// TestSubprojectImportIgnoresExcludeTags pins the asymmetry USAGE.md's `exclude_tags`
// section documents, and pins it as behavior rather than as a sentence. The parent has
// four ways to reach a child interaction and `exclude_tags` reaches three of them:
// `dva p:k` and `dva run --project p k` both land in runSubprojectCommand, and
// `dva ls --project p` filters separately, all three through
// subCfg.FilterInteractions(sub.ExcludeTags). The fourth, an `import:` of the name, does
// not — resolveSubprojectImports reads subCfg.Interaction[name] directly.
//
// Whether that asymmetry is right is a separate question; an import names one key
// explicitly, so honoring a tag filter over an explicit name is arguable either way. What
// is not arguable is that it must not change by accident: someone who hides an interaction
// with exclude_tags and then imports it by name gets it back silently, and the only place
// that is written down is the document this test is bound to.
//
// Both halves are asserted together on purpose. The import half alone would still pass if
// FilterInteractions stopped filtering at all, which is the failure most likely to go
// unnoticed.
func TestSubprojectImportIgnoresExcludeTags(t *testing.T) {
	parentDir := writeSubprojectFixture(t,
		"version: \"0.1.0\"\nsubprojects:\n  engine:\n    path: child\n    exclude_tags: [infra]\n    import:\n      interactions:\n        - dbshell\n",
		"child",
		"version: \"0.1.0\"\ninteraction:\n  dbshell:\n    command: echo CHILD-DBSHELL\n    tags: [infra]\n  test:\n    command: echo CHILD-TEST\n    tags: [app]\n")

	cfg, err := Load(parentDir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if _, ok := cfg.Interaction["engine/dbshell"]; !ok {
		keys := make([]string, 0, len(cfg.Interaction))
		for k := range cfg.Interaction {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		t.Errorf("`engine/dbshell` is missing: the import route now honors exclude_tags, "+
			"which contradicts the exclude_tags section of USAGE.md; got keys %v", keys)
	}

	childCfg, err := Load(filepath.Join(parentDir, "child"))
	if err != nil {
		t.Fatalf("Load child error: %v", err)
	}
	filtered := childCfg.FilterInteractions([]string{"infra"})
	if _, ok := filtered["dbshell"]; ok {
		t.Error("FilterInteractions kept `dbshell` despite its `infra` tag: the three " +
			"routes that do filter no longer filter, so the documented asymmetry is gone")
	}
	if _, ok := filtered["test"]; !ok {
		t.Error("FilterInteractions dropped `test`, which carries no excluded tag")
	}
}
