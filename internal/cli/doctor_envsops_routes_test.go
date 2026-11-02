package cli

import (
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

const scriptStack = `stack:
  svc:
    default_runner: script
    runners:
      script:
        up: "true"
        stop: "true"
`

func upStderr(t *testing.T, f *bridgeFixture, args []string) string {
	t.Helper()
	t.Chdir(f.cfg.FileDir())
	withEnvPolicyGlobals(t, nil, false)
	var err error
	stderr := captureStderr(t, func() { err = upCmd.RunE(upCmd, args) })
	if err != nil {
		t.Fatalf("up %v: %v", args, err)
	}
	return stderr
}

func TestUpWarnsForImportedPlanOwner(t *testing.T) {
	root := simpleBridgeYAML(".env.root", "root.enc") + `subprojects:
  backend:
    path: backend
    import:
      plans:
        - name: deploy
`
	child := simpleBridgeYAML(".env.child", "child.enc") + scriptStack + "plans:\n  deploy:\n    entries:\n      - name: svc\n"
	f := newBridgeFixture(t, root, map[string]string{"backend/" + config.FileName: child})
	stderr := upStderr(t, f, []string{"backend/deploy"})
	if strings.Count(stderr, "declares sops_source") != 1 || !strings.Contains(stderr, "env_file .env.child") || strings.Contains(stderr, ".env.root") {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestCompositionWarnsOncePerOwner(t *testing.T) {
	root := "version: \"0.1.45\"\n" + `subprojects:
  backend:
    path: backend
    import:
      plans:
        - name: deploy
        - name: migrate
plans:
  all:
    composes:
      - plan: backend/deploy
      - plan: backend/migrate
`
	child := simpleBridgeYAML(".env.child", "child.enc") + scriptStack + "plans:\n  deploy:\n    entries:\n      - name: svc\n  migrate:\n    entries:\n      - name: svc\n"
	f := newBridgeFixture(t, root, map[string]string{"backend/" + config.FileName: child})
	stderr := upStderr(t, f, []string{"all"})
	if n := strings.Count(stderr, "declares sops_source"); n != 1 {
		t.Fatalf("warnings = %d, stderr = %q", n, stderr)
	}
}

func TestCompositionWarnsEachOwner(t *testing.T) {
	root := "version: \"0.1.45\"\n" + `subprojects:
  a:
    path: a
    import:
      plans:
        - name: deploy
  b:
    path: b
    import:
      plans:
        - name: deploy
plans:
  all:
    composes:
      - plan: a/deploy
      - plan: b/deploy
`
	mk := func(target, src string) string {
		return simpleBridgeYAML(target, src) + scriptStack + "plans:\n  deploy:\n    entries:\n      - name: svc\n"
	}
	f := newBridgeFixture(t, root, map[string]string{
		"a/" + config.FileName: mk(".env.a", "a.enc"),
		"b/" + config.FileName: mk(".env.b", "b.enc"),
	})
	stderr := upStderr(t, f, []string{"all"})
	if !strings.Contains(stderr, "[child: a/deploy] env_file .env.a") || !strings.Contains(stderr, "[child: b/deploy] env_file .env.b") {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestUpWarnsForWrapperForm(t *testing.T) {
	yaml := "version: \"0.1.45\"\nenv_file:\n  files:\n    - {path: .env, sops_source: secrets.env.enc}\n" + scriptStack
	f := newBridgeFixture(t, yaml, map[string]string{"secrets.env.enc": "ENC"})
	stderr := upStderr(t, f, nil)
	if strings.Count(stderr, "declares sops_source") != 1 {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestDoctorStrictFailsOnUnsealed(t *testing.T) {
	f := newBridgeFixture(t, simpleBridgeYAML(".env", "secrets.env.enc"), map[string]string{"secrets.env.enc": "ENC"})
	rows := checkEnvUnsealed(f.cfg, config.ApplyEnvFiles(f.cfg.EnvFile, f.cfg.FileDir(), f.cfg.EnvFileScope()))
	if err := doctorExitError(rows, false); err != nil {
		t.Fatalf("default exit = %v, want nil", err)
	}
	if err := doctorExitError(rows, true); err == nil {
		t.Fatal("strict exit = nil, want error")
	} else {
		t.Logf("strict: %v", err)
	}
}
