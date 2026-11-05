package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemovedModeGuardDoesNotInterpretQuotedMergeKey(t *testing.T) {
	// A quoted << is an ordinary unknown root key, not YAML merge syntax.
	if _, err := decodeConfig([]byte("version: '0.1.0'\n'<<': {modes: {}}\n")); err != nil {
		t.Fatalf("literal key was treated as a merge: %v", err)
	}
}

func TestRemovedModeGuardRejectsEncodedAndExplicitMergeKeys(t *testing.T) {
	for name, source := range map[string]string{
		"encoded modes":        "version: '0.1.0'\n!!binary bW9kZXM=: {}\n",
		"encoded default mode": "version: '0.1.0'\n!!binary ZGVmYXVsdF9tb2Rl: dev\n",
		"explicit merge":       "base: &base {modes: {}}\n!!merge '<<': *base\n",
		"long URI merge":       "base: &base {modes: {}}\n!<tag:yaml.org,2002:merge> '<<': *base\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := decodeConfig([]byte(source)); err == nil || !strings.Contains(err.Error(), "was removed in v0.5.0") {
				t.Fatalf("decodeConfig() error = %v, want removed-key rejection", err)
			}
		})
	}
}

func TestLoadRejectsRemovedModesInEveryLoadedLayer(t *testing.T) {
	for _, layer := range []string{"root", "module", "override"} {
		t.Run(layer, func(t *testing.T) {
			dir := t.TempDir()
			root := "version: '0.1.0'\n"
			legacy := "modes: {}\n"
			switch layer {
			case "root":
				root += legacy
			case "module":
				root += "modules: [legacy]\n"
				if err := os.MkdirAll(filepath.Join(dir, DotDirName), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, DotDirName, "legacy.yml"), []byte(legacy), 0o644); err != nil {
					t.Fatal(err)
				}
			case "override":
				if err := os.WriteFile(filepath.Join(dir, "dva.override.yml"), []byte(legacy), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(filepath.Join(dir, FileName), []byte(root), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(dir); err == nil || !strings.Contains(err.Error(), `top-level "modes" was removed`) {
				t.Fatalf("Load() error = %v, want removed modes rejection", err)
			}
		})
	}
}

func TestLoadRejectsRemovedModesAcrossLoadOptions(t *testing.T) {
	for _, load := range []struct {
		name string
		opts []LoadOption
	}{
		{name: "ordinary"},
		{name: "ci", opts: []LoadOption{ForCI()}},
		{name: "env-inspection", opts: []LoadOption{ForEnvInspection()}},
	} {
		t.Run(load.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, FileName), []byte("version: '0.1.0'\ndefault_mode: null\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(dir, load.opts...); err == nil || !strings.Contains(err.Error(), `top-level "default_mode" was removed`) {
				t.Fatalf("Load() error = %v, want removed default_mode rejection", err)
			}
		})
	}
}

func TestLoadRejectsRemovedModesFromImportedChildButNotLazyChild(t *testing.T) {
	for _, imported := range []bool{false, true} {
		t.Run(fmt.Sprintf("imported=%t", imported), func(t *testing.T) {
			dir := t.TempDir()
			child := filepath.Join(dir, "child")
			if err := os.Mkdir(child, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(child, FileName), []byte("version: '0.1.0'\nmodes: {}\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			parent := "version: '0.1.0'\nsubprojects:\n  child:\n    path: child\n"
			if imported {
				parent += "    import:\n      plans: [child-plan]\n"
			}
			if err := os.WriteFile(filepath.Join(dir, FileName), []byte(parent), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := Load(dir)
			if imported && (err == nil || !strings.Contains(err.Error(), `top-level "modes" was removed`)) {
				t.Fatalf("Load(imported child) error = %v, want child rejection", err)
			}
			if !imported && err != nil {
				t.Fatalf("Load(lazy child) error = %v, want success without child evaluation", err)
			}
		})
	}
}

func TestLoadRejectsRemovedModesThroughMergeAliasWithoutRepeatedTraversal(t *testing.T) {
	var source strings.Builder
	source.WriteString("base: &base {modes: {}}\n")
	for i := range 24 {
		previous := "base"
		if i > 0 {
			previous = fmt.Sprintf("layer%d", i-1)
		}
		fmt.Fprintf(&source, "layer%d: &layer%d {<<: [*%s, *%s]}\n", i, i, previous, previous)
	}
	source.WriteString("<<: *layer23\n")
	if _, err := decodeConfig([]byte(source.String())); err == nil || !strings.Contains(err.Error(), `top-level "modes" was removed`) {
		t.Fatalf("decodeConfig() error = %v, want removed modes rejection", err)
	}
}

func TestLoadRejectsMalformedPresentOverrides(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte("version: '0.1.0'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "dva.override.yml"), []byte("stack: [\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil || !strings.Contains(err.Error(), "loading override") {
		t.Fatalf("Load() error = %v, want present override parse error", err)
	}
}

func TestLoadRejectsDanglingRootOverrideSymlink(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte("version: '0.1.0'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("missing-override.yml", filepath.Join(dir, "dva.override.yml")); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil || !strings.Contains(err.Error(), "loading override") {
		t.Fatalf("Load() error = %v, want dangling root override error", err)
	}
}

func TestLoadRejectsMalformedImportedChildOverride(t *testing.T) {
	dir := t.TempDir()
	child := filepath.Join(dir, "child")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte("version: '0.1.0'\nsubprojects:\n  child:\n    path: child\n    import:\n      plans: [child-plan]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(child, FileName), []byte("version: '0.1.0'\nplans:\n  child-plan:\n    entries: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(child, "dva.override.yml"), []byte("stack: [\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil || !strings.Contains(err.Error(), `loading subproject "child" override`) {
		t.Fatalf("Load() error = %v, want child override parse error", err)
	}
}

func TestLoadRejectsDanglingImportedChildOverrideSymlink(t *testing.T) {
	dir := t.TempDir()
	child := filepath.Join(dir, "child")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte("version: '0.1.0'\nsubprojects:\n  child:\n    path: child\n    import:\n      plans: [child-plan]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(child, FileName), []byte("version: '0.1.0'\nplans:\n  child-plan:\n    entries: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("missing-override.yml", filepath.Join(child, "dva.override.yml")); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil || !strings.Contains(err.Error(), `loading subproject "child" override`) {
		t.Fatalf("Load() error = %v, want dangling child override error", err)
	}
}
