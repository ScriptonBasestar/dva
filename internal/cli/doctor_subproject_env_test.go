package cli

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func TestDoctorInspectsDirectChildEncryptedTargets(t *testing.T) {
	for _, imported := range []bool{false, true} {
		t.Run(map[bool]string{false: "unimported", true: "imported"}[imported], func(t *testing.T) {
			root := "version: '0.1.45'\nsubprojects:\n  child:\n    path: child\n"
			if imported {
				root += "    import:\n      plans: [{name: dev}]\n"
			}
			child := simpleBridgeYAML(".env", "secrets.env.enc") + "plans:\n  dev: {entries: []}\nchecks:\n  - {name: child-command, type: command, command: 'touch should-not-run'}\n"
			f := newBridgeFixture(t, root, map[string]string{"child/" + config.FileName: child, "child/secrets.env.enc": "ENC"})
			t.Setenv(config.EnvFileKey, f.path(config.FileName))
			rows := checkSubprojectEncryptedEnv(f.cfg)
			if len(rows) != 1 || rows[0].Passed || !strings.Contains(rows[0].Name, "[child: child]") || !strings.Contains(rows[0].FixHint, "In subproject") {
				t.Fatalf("wrong child result: %+v", rows)
			}
			if doctorExitError(rows, false) != nil || doctorExitError(rows, true) == nil {
				t.Fatal("advisory/strict contract changed")
			}
			if _, err := os.Stat(f.path("child/should-not-run")); !errors.Is(err, fs.ErrNotExist) {
				t.Fatal("child doctor command executed")
			}
		})
	}
}

func TestDoctorChildInspectionUsesEffectiveOwner(t *testing.T) {
	t.Setenv("DVA_575_SCOPE", "")
	if err := os.Unsetenv("DVA_575_SCOPE"); err != nil {
		t.Fatal(err)
	}
	root := "version: '0.1.45'\nvars: {DVA_575_SCOPE: private-parent}\nsubprojects: {child: {path: child}}\n"
	child := "version: '0.1.45'\nvars: {DVA_575_SCOPE: private-child}\nmodules: [env]\n"
	module := pathBridgeYAML(".env.${DVA_575_SCOPE}", "secrets.env.enc")
	f := newBridgeFixture(t, root, map[string]string{"child/dva.yml": child, "child/.sb/dva/env.yml": module, "child/.env.private-child": "TOKEN=private-secret\n"})
	if rows := checkSubprojectEncryptedEnv(f.cfg); len(rows) != 0 {
		t.Fatalf("parent scope leaked: %+v", rows)
	}
	writeFixtureFiles(t, f.dir, map[string]string{"child/dva.override.yml": simpleBridgeYAML(".override.env", "override.enc")})
	rows := checkSubprojectEncryptedEnv(f.cfg)
	if len(rows) != 1 || !strings.Contains(rows[0].Name, ".override.env") {
		t.Fatalf("override ignored: %+v", rows)
	}
	bytes, err := json.Marshal(rows)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"private-parent", "private-child", "private-secret", f.dir} {
		if strings.Contains(string(bytes), secret) {
			t.Fatalf("private data leaked: %s", bytes)
		}
	}
}

func TestDoctorChildInspectionReportsUnavailableAndContinues(t *testing.T) {
	root := "version: '0.1.45'\nsubprojects:\n  a-missing: {path: absent}\n  b-malformed: {path: broken}\n  c-version: {path: future}\n  d-valid: {path: valid}\n  e-yaml: {path: alt}\n"
	f := newBridgeFixture(t, root, map[string]string{
		"broken/dva.yml": "version: [private-secret\n",
		"future/dva.yml": "version: '999.0.0'\n",
		"valid/dva.yml":  simpleBridgeYAML(".env", "secrets.env.enc"),
		"alt/dva.yaml":   simpleBridgeYAML(".env", "secrets.env.enc"),
	})
	t.Setenv(config.EnvFileKey, f.path(config.FileName))
	rows := checkSubprojectEncryptedEnv(f.cfg)
	if len(rows) != 5 {
		t.Fatalf("lost sibling inspection: %+v", rows)
	}
	for i, name := range []string{"a-missing", "b-malformed", "c-version", "d-valid", "e-yaml"} {
		if !strings.Contains(rows[i].Name, "[child: "+name+"]") {
			t.Fatalf("unstable order: %+v", rows)
		}
	}
	for _, row := range rows[:3] {
		if row.Passed || !strings.Contains(row.Finding, "not inspected") {
			t.Fatalf("load failure hidden: %+v", row)
		}
	}
	b, _ := json.Marshal(rows)
	if strings.Contains(string(b), "private-secret") || strings.Contains(string(b), f.dir) {
		t.Fatalf("load error leaked: %s", b)
	}
}

func TestDoctorChildInspectionStaysWithinEncryptedDirectScope(t *testing.T) {
	root := "version: '0.1.45'\nsubprojects: {child: {path: child}}\n"
	child := "version: '0.1.45'\nenv_file: [{path: missing.env, required: true}]\nsubprojects: {grand: {path: grand, import: {plans: [{name: missing}]}}}\nstack: {broken: {default_runner: unsupported}}\n"
	f := newBridgeFixture(t, root, map[string]string{"child/dva.yml": child, "child/grand/dva.yml": simpleBridgeYAML(".env", "secrets.env.enc")})
	if rows := checkSubprojectEncryptedEnv(f.cfg); len(rows) != 0 {
		t.Fatalf("expanded beyond encrypted direct scope: %+v", rows)
	}
}

func TestDoctorChildEncryptedInputFailuresAreVisible(t *testing.T) {
	root := "version: '0.1.45'\nsubprojects: {child: {path: child}}\n"
	f := newBridgeFixture(t, root, map[string]string{"child/dva.yml": simpleBridgeYAML(".env", "secrets.env.enc"), "child/.env": "invalid private-secret line\n"})
	rows := checkSubprojectEncryptedEnv(f.cfg)
	if len(rows) != 1 || rows[0].Passed || !strings.Contains(rows[0].Finding, "invalid dotenv syntax") || strings.Contains(rows[0].Finding, "private-secret") {
		t.Fatalf("unsafe or missing failure: %+v", rows)
	}
}
