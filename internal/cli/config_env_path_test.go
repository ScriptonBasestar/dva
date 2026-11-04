package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func pathBridgeYAML(target, source string) string {
	return fmt.Sprintf("version: %q\nenv_file: [{path: %q, sops_source: %q}]\n", config.EnvBridgeIntroducedVersion, target, source)
}

func TestConfigEnvUnsealResolvesLoaderTarget(t *testing.T) {
	const key = "DVA_585_STAGE"
	for _, tt := range []struct{ name, layers, osValue, expression, want, cwd string }{
		{"vars", "vars: {DVA_585_STAGE: from-vars}\n", "", "${DVA_585_STAGE}", "from-vars", ""},
		{"environment", "vars: {DVA_585_STAGE: from-vars}\nenvironment: {DVA_585_STAGE: from-environment}\n", "", "${DVA_585_STAGE}", "from-environment", ""},
		{"OS wins", "vars: {DVA_585_STAGE: from-vars}\nenvironment: {DVA_585_STAGE: from-environment}\n", "private-os-value", "${DVA_585_STAGE}", "private-os-value", ""},
		{"default", "", "", "${DVA_585_STAGE:-fallback}", "fallback", ""},
		{"invocation cwd", "", "", "${DVA_WORK_DIR_REL_PATH}", "nested", "nested"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cwd != "" {
				t.Setenv(config.EnvRuntimeWorkDirRelPath, "")
				if err := os.Unsetenv(config.EnvRuntimeWorkDirRelPath); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv(key, tt.osValue)
			if tt.osValue == "" {
				if err := os.Unsetenv(key); err != nil {
					t.Fatal(err)
				}
			}
			declared := tt.expression + "/.env"
			// Source metadata stays literal, even if it contains interpolation syntax.
			const source = "secrets.${DVA_585_SOURCE}.enc"
			t.Setenv("DVA_585_SOURCE", "must-not-expand")
			f := newBridgeFixture(t, pathBridgeYAML(declared, source)+tt.layers, map[string]string{
				source: "ENC", declared: "literal must survive", tt.want + "/.keep": "",
			})
			t.Chdir(f.dir)
			if tt.cwd != "" {
				t.Chdir(f.path(tt.cwd))
			}
			f.install(true)
			out, stderr := captureStreams(t, func() {
				if err := runEnvUnseal(declared, false); err != nil {
					t.Fatal(err)
				}
			})
			if b, err := os.ReadFile(f.path(tt.want + "/.env")); err != nil || string(b) != bridgePayload() {
				t.Fatalf("resolved target = %q, %v", b, err)
			}
			if b, err := os.ReadFile(f.path(declared)); err != nil || string(b) != "literal must survive" {
				t.Fatalf("literal changed: %q, %v", b, err)
			}
			if !strings.Contains(out, declared) || strings.Contains(out+stderr, bridgeSecretValue) || strings.Contains(out+stderr, "private-os-value") {
				t.Fatalf("unsafe output: %s %s", out, stderr)
			}
			if len(f.git.askedAbout()) == 0 {
				t.Fatal("Git guard was not queried")
			}
			for _, ask := range f.git.askedAbout() {
				if ask.dir != f.path(tt.want) || (ask.target != "" && ask.target != ".env") {
					t.Fatalf("Git asked about wrong target: %+v", ask)
				}
			}
			if calls := f.sops.callsMade(); len(calls) != 1 || calls[0] != "decrypt "+f.path(source) {
				t.Fatalf("source expanded: %v", calls)
			}
			wd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			scope := f.cfg.EnvFileScopeAt(wd)
			if report := config.ApplyEnvFiles(f.cfg.EnvFile, f.dir, scope); report.Err() != nil || scope.Vars[bridgeSecretKey] != bridgeSecretValue {
				t.Fatal("loader did not read unsealed target")
			}
			f.assertNoTempResidue()
		})
	}
}

func TestConfigEnvResolvedTargetGuards(t *testing.T) {
	for _, tt := range []struct{ name, value, code string }{
		{"unresolved", "", ""}, {"empty", "empty", ""},
		{"value contains reference", "private-${UNRESOLVED}", ""},
		{"escape", "../private-escape", codePathEscapes},
		{"absolute", "/private-absolute", codeAbsolutePath},
		{"missing parent", "private-parent/.env", codeTargetParentMissing},
		{"source collision", "secrets.env.enc", codeSourceIsTarget},
		{"symlink", "private-parent/.env", codePathComponentSymlnk},
		{"tracked", "private-parent/.env", codeTargetTracked},
		{"not ignored", "private-parent/.env", codeTargetNotIgnored},
	} {
		t.Run(tt.name, func(t *testing.T) {
			const declared = "${DVA_585_TARGET}"
			t.Setenv("DVA_585_TARGET", tt.value)
			if tt.name == "unresolved" {
				if err := os.Unsetenv("DVA_585_TARGET"); err != nil {
					t.Fatal(err)
				}
			}
			if tt.name == "empty" {
				t.Setenv("DVA_585_TARGET", "")
			}
			f := newBridgeFixture(t, pathBridgeYAML(declared, "secrets.env.enc"), map[string]string{"secrets.env.enc": "ENC", declared: "literal"})
			if tt.name == "tracked" || tt.name == "not ignored" {
				writeFixtureFiles(t, f.dir, map[string]string{"private-parent/.keep": ""})
				f.git.tracked = tt.name == "tracked"
				f.git.ignored = tt.name != "not ignored"
			}
			if tt.name == "symlink" {
				if err := os.Symlink(t.TempDir(), f.path("private-parent")); err != nil {
					t.Fatal(err)
				}
			}
			f.install(false)
			err := runEnvUnseal(declared, true)
			if err == nil || errorCode(err) != tt.code {
				t.Fatalf("got %v (%s), want %s", err, errorCode(err), tt.code)
			}
			if strings.Contains(err.Error(), "private-") || strings.Contains(err.Error(), bridgeSecretValue) {
				t.Fatalf("expanded value leaked: %v", err)
			}
			if len(f.sops.callsMade()) != 0 {
				t.Fatal("SOPS ran before guard")
			}
			if b, _ := os.ReadFile(f.path(declared)); string(b) != "literal" {
				t.Fatal("literal target changed")
			}
			if b, _ := os.ReadFile(f.path("secrets.env.enc")); string(b) != "ENC" {
				t.Fatal("encrypted source changed")
			}
			f.assertNoTempResidue()
		})
	}
}

func TestConfigEnvExpandedAnchorRevalidation(t *testing.T) {
	const declared = "${DVA_585_DIRECTORY}/.env"
	t.Setenv("DVA_585_DIRECTORY", "private-parent")
	f := newBridgeFixture(t, pathBridgeYAML(declared, "secrets.env.enc"), map[string]string{"secrets.env.enc": "ENC", "private-parent/.keep": ""})
	f.sops.decrypt = func(_ string, out *os.File) error {
		if err := os.Rename(f.path("private-parent"), f.path("moved")); err != nil {
			return err
		}
		if err := os.Mkdir(f.path("private-parent"), 0o700); err != nil {
			return err
		}
		_, err := out.WriteString(bridgePayload())
		return err
	}
	f.install(false)
	err := runEnvUnseal(declared, false)
	if err == nil || strings.Contains(err.Error(), "private-parent") || !strings.Contains(err.Error(), declared) {
		t.Fatalf("unsafe anchor failure: %v", err)
	}
	for _, p := range []string{"moved/.env", "private-parent/.env"} {
		if _, err := os.Stat(f.path(p)); !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("unexpected target %s: %v", p, err)
		}
	}
	f.assertNoTempResidue()
}

func TestConfigEnvDeclaredPathErrorsAndHints(t *testing.T) {
	const declared = ".env.${STAGE}"
	for _, post := range []bool{false, true} {
		for _, original := range []error{
			&fs.PathError{Op: "write", Path: "private-secret", Err: fs.ErrPermission},
			&os.LinkError{Op: "rename", Old: "private-temp", New: "private-secret", Err: fs.ErrPermission},
		} {
			err := original
			if post {
				err = &postRenameError{target: "private-secret", err: err}
			}
			got := declaredTargetError(err, declared)
			if strings.Contains(got.Error(), "private-") || !strings.Contains(got.Error(), declared) || !errors.Is(got, fs.ErrPermission) {
				t.Fatalf("unsafe error: %v", got)
			}
			if _, ok := errors.AsType[*postRenameError](got); ok != post {
				t.Fatalf("post-rename semantics lost: %v", got)
			}
		}
	}
	for target, want := range map[string]string{
		".env":   "dva config env unseal .env",
		"-env":   "dva config env unseal -- -env",
		declared: "dva config env unseal '.env.${STAGE}'",
		"a'b c":  "dva config env unseal 'a'\"'\"'b c'",
	} {
		if got := unsealCommand(target); got != want {
			t.Errorf("hint %q = %q, want %q", target, got, want)
		}
	}
}

func TestConfigEnvTargetScopeDoesNotLoadOtherEnvFiles(t *testing.T) {
	t.Setenv("DVA_585_STAGE", "")
	if err := os.Unsetenv("DVA_585_STAGE"); err != nil {
		t.Fatal(err)
	}
	const declared = ".env.${DVA_585_STAGE}"
	yaml := fmt.Sprintf("version: %q\nvars: {DVA_585_STAGE: from-vars}\nenv_file:\n  - .scope\n  - {path: %q, sops_source: secrets.env.enc}\n", config.EnvBridgeIntroducedVersion, declared)
	f := newBridgeFixture(t, yaml, map[string]string{".scope": "DVA_585_STAGE=from-file\n", "secrets.env.enc": "ENC"})
	f.install(false)
	captureStreams(t, func() {
		if err := runEnvUnseal(declared, false); err != nil {
			t.Fatal(err)
		}
	})
	if b, err := os.ReadFile(f.path(".env.from-vars")); err != nil || string(b) != bridgePayload() {
		t.Fatalf("wrong scoped target: %q, %v", b, err)
	}
	if _, err := os.Stat(f.path(".env.from-file")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("file-derived path was used: %v", err)
	}
}
