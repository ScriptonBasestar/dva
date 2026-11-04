package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func TestDetectConfigDriftWarnings_ParentOwnedComposeIsNotChildDrift(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "apps", "child")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatalf("mkdir child: %v", err)
	}
	writeParentComposeDriftFile(t, filepath.Join(root, "compose.yaml"))
	writeParentComposeDriftFile(t, filepath.Join(root, "compose.observability.yaml"))
	writeParentComposeDriftConfig(t, root, `
subprojects:
  child:
    path: apps/child
stack:
  root:
    default_runner: compose
    runners:
      compose:
        files: [compose.yaml, compose.observability.yaml]
`)
	writeParentComposeDriftConfig(t, child, `
stack:
  shared:
    default_runner: compose
    runners:
      compose:
        files: [../../compose.yaml]
`)

	childConfig, err := config.Load(child, config.SkipVersionCheck())
	if err != nil {
		t.Fatalf("load child: %v", err)
	}
	if warnings := detectConfigDriftWarnings(childConfig); strings.Contains(strings.Join(warnings, "\n"), "compose.observability.yaml") {
		t.Fatalf("parent-owned compose file must not be reported as child drift: %v", warnings)
	}
}

func TestDetectConfigDriftWarnings_ParentDeclarationDoesNotHideChildOrphan(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "child")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatalf("mkdir child: %v", err)
	}
	writeParentComposeDriftFile(t, filepath.Join(root, "compose.yaml"))
	writeParentComposeDriftFile(t, filepath.Join(root, "compose.monitor.yaml"))
	writeParentComposeDriftFile(t, filepath.Join(child, "compose.orphan.yaml"))
	writeParentComposeDriftConfig(t, root, `
subprojects:
  child:
    path: child
stack:
  root:
    default_runner: compose
    runners:
      compose:
        files: [compose.yaml, compose.monitor.yaml]
`)
	writeParentComposeDriftConfig(t, child, `
stack:
  shared:
    default_runner: compose
    runners:
      compose:
        files: [../compose.yaml]
`)

	childConfig, err := config.Load(child, config.SkipVersionCheck())
	if err != nil {
		t.Fatalf("load child: %v", err)
	}
	warnings := strings.Join(detectConfigDriftWarnings(childConfig), "\n")
	if !strings.Contains(warnings, "compose.orphan.yaml") {
		t.Fatalf("unowned compose beside child must remain drift: %s", warnings)
	}
	if strings.Contains(warnings, "compose.monitor.yaml") {
		t.Fatalf("parent-owned compose must not be reported as child drift: %s", warnings)
	}
}

func TestDetectConfigDriftWarnings_AncestorWithoutSubprojectDeclarationDoesNotOwnCompose(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "child")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatalf("mkdir child: %v", err)
	}
	writeParentComposeDriftFile(t, filepath.Join(root, "compose.yaml"))
	writeParentComposeDriftFile(t, filepath.Join(root, "compose.unowned.yaml"))
	writeParentComposeDriftConfig(t, root, `
stack:
  root:
    default_runner: compose
    runners:
      compose:
        files: [compose.yaml, compose.unowned.yaml]
`)
	writeParentComposeDriftConfig(t, child, `
stack:
  shared:
    default_runner: compose
    runners:
      compose:
        files: [../compose.yaml]
`)

	childConfig, err := config.Load(child, config.SkipVersionCheck())
	if err != nil {
		t.Fatalf("load child: %v", err)
	}
	if warnings := strings.Join(detectConfigDriftWarnings(childConfig), "\n"); !strings.Contains(warnings, "compose.unowned.yaml") {
		t.Fatalf("ancestor without declared relationship must not hide compose drift: %s", warnings)
	}
}

func TestDetectConfigDriftWarnings_ParentDvaYamlOwnsCompose(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "child")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatalf("mkdir child: %v", err)
	}
	writeParentComposeDriftFile(t, filepath.Join(root, "compose.yaml"))
	writeParentComposeDriftFile(t, filepath.Join(root, "compose.parent.yaml"))
	writeParentComposeDriftConfig(t, root, `
subprojects:
  child:
    path: child
stack:
  root:
    default_runner: compose
    runners:
      compose:
        files: [compose.yaml, compose.parent.yaml]
`)
	if err := os.Rename(filepath.Join(root, config.FileName), filepath.Join(root, config.FileNameAlt)); err != nil {
		t.Fatalf("rename parent config to dva.yaml: %v", err)
	}
	writeParentComposeDriftConfig(t, child, `
stack:
  shared:
    default_runner: compose
    runners:
      compose:
        files: [../compose.yaml]
`)

	childConfig, err := config.Load(child, config.SkipVersionCheck())
	if err != nil {
		t.Fatalf("load child: %v", err)
	}
	if warnings := strings.Join(detectConfigDriftWarnings(childConfig), "\n"); strings.Contains(warnings, "compose.parent.yaml") {
		t.Fatalf("dva.yaml parent-owned compose must not be reported as child drift: %s", warnings)
	}
}

func TestDetectConfigDriftWarnings_ParentOwnershipIgnoresAmbientDVAFile(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "child")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatalf("mkdir child: %v", err)
	}
	writeParentComposeDriftFile(t, filepath.Join(root, "compose.yaml"))
	writeParentComposeDriftFile(t, filepath.Join(root, "compose.parent.yaml"))
	writeParentComposeDriftConfig(t, root, `
subprojects:
  child:
    path: child
stack:
  root:
    default_runner: compose
    runners:
      compose:
        files: [compose.yaml, compose.parent.yaml]
`)
	writeParentComposeDriftConfig(t, child, `
stack:
  shared:
    default_runner: compose
    runners:
      compose:
        files: [../compose.yaml]
`)

	childConfig, err := config.Load(child, config.SkipVersionCheck())
	if err != nil {
		t.Fatalf("load child: %v", err)
	}
	t.Setenv(config.EnvFileKey, filepath.Join(child, config.FileName))
	if warnings := strings.Join(detectConfigDriftWarnings(childConfig), "\n"); strings.Contains(warnings, "compose.parent.yaml") {
		t.Fatalf("ambient DVA_FILE must not redirect parent ownership lookup: %s", warnings)
	}
}

func TestDetectConfigDriftWarnings_UnloadableParentDoesNotOwnCompose(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "child")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatalf("mkdir child: %v", err)
	}
	writeParentComposeDriftFile(t, filepath.Join(root, "compose.yaml"))
	writeParentComposeDriftFile(t, filepath.Join(root, "compose.parent.yaml"))
	writeParentComposeDriftConfig(t, root, `
subprojects:
  child:
    path: child
stack: [
`)
	writeParentComposeDriftConfig(t, child, `
stack:
  shared:
    default_runner: compose
    runners:
      compose:
        files: [../compose.yaml]
`)

	childConfig, err := config.Load(child, config.SkipVersionCheck())
	if err != nil {
		t.Fatalf("load child: %v", err)
	}
	if warnings := strings.Join(detectConfigDriftWarnings(childConfig), "\n"); !strings.Contains(warnings, "compose.parent.yaml") {
		t.Fatalf("unloadable parent must not hide compose drift: %s", warnings)
	}
}

func TestDetectConfigDriftWarnings_UnsupportedParentDoesNotOwnCompose(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "child")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatalf("mkdir child: %v", err)
	}
	writeParentComposeDriftFile(t, filepath.Join(root, "compose.yaml"))
	writeParentComposeDriftFile(t, filepath.Join(root, "compose.parent.yaml"))
	writeParentComposeDriftConfig(t, root, `
subprojects:
  child:
    path: child
stack:
  root:
    default_runner: compose
    runners:
      compose:
        files: [compose.yaml, compose.parent.yaml]
`)
	if err := os.WriteFile(filepath.Join(root, config.FileName), []byte(strings.ReplaceAll(mustReadParentComposeDriftConfig(t, root), `version: "0.4.2"`, `version: "99.0.0"`)), 0o644); err != nil {
		t.Fatalf("make parent unsupported: %v", err)
	}
	writeParentComposeDriftConfig(t, child, `
stack:
  shared:
    default_runner: compose
    runners:
      compose:
        files: [../compose.yaml]
`)

	childConfig, err := config.Load(child, config.SkipVersionCheck())
	if err != nil {
		t.Fatalf("load child: %v", err)
	}
	if warnings := strings.Join(detectConfigDriftWarnings(childConfig), "\n"); !strings.Contains(warnings, "compose.parent.yaml") {
		t.Fatalf("unsupported parent must not hide compose drift: %s", warnings)
	}
}

func TestDetectConfigDriftWarnings_SymlinkedSubprojectDeclarationOwnsCompose(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "child-real")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatalf("mkdir child: %v", err)
	}
	if err := os.Symlink("child-real", filepath.Join(root, "child-link")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	writeParentComposeDriftFile(t, filepath.Join(root, "compose.yaml"))
	writeParentComposeDriftFile(t, filepath.Join(root, "compose.parent.yaml"))
	writeParentComposeDriftConfig(t, root, `
subprojects:
  child:
    path: child-link
stack:
  root:
    default_runner: compose
    runners:
      compose:
        files: [compose.yaml, compose.parent.yaml]
`)
	writeParentComposeDriftConfig(t, child, `
stack:
  shared:
    default_runner: compose
    runners:
      compose:
        files: [../compose.yaml]
`)

	childConfig, err := config.Load(child, config.SkipVersionCheck())
	if err != nil {
		t.Fatalf("load child: %v", err)
	}
	if warnings := strings.Join(detectConfigDriftWarnings(childConfig), "\n"); strings.Contains(warnings, "compose.parent.yaml") {
		t.Fatalf("canonical symlinked subproject declaration must own compose: %s", warnings)
	}
}

func writeParentComposeDriftConfig(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, config.FileName), []byte("version: \"0.4.2\"\n"+body), 0o644); err != nil {
		t.Fatalf("write dva.yml: %v", err)
	}
}

func writeParentComposeDriftFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir compose parent: %v", err)
	}
	if err := os.WriteFile(path, []byte("services: {}\n"), 0o644); err != nil {
		t.Fatalf("write compose %s: %v", path, err)
	}
}

func mustReadParentComposeDriftConfig(t *testing.T, dir string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, config.FileName))
	if err != nil {
		t.Fatalf("read dva.yml: %v", err)
	}
	return string(data)
}
