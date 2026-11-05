package cli

import (
	"errors"
	"io/fs"
	"os"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func TestDoctorEnvSopsDeclaration(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		files   map[string]string
		wantRow bool
	}{
		{
			name:    "evidence without declaration reports a row",
			yaml:    "version: \"0.1.45\"\nenv_file: [.env]\n",
			files:   map[string]string{".sops.yaml": "creation_rules: []\n", "secrets.env.enc": "ENC"},
			wantRow: true,
		},
		{
			name:  "no evidence stays silent",
			yaml:  "version: \"0.1.45\"\nenv_file: [.env]\n",
			files: map[string]string{".env": "A=1\n"},
		},
		{
			name:  "a declared plaintext path is not a candidate",
			yaml:  "version: \"0.1.45\"\nenv_file: [.env.enc]\n",
			files: map[string]string{".env.enc": "A=1\n"},
		},
		{
			name:  "declared source stays silent",
			yaml:  simpleBridgeYAML(".env", "secrets.env.enc"),
			files: map[string]string{".sops.yaml": "creation_rules: []\n", "secrets.env.enc": "ENC"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newBridgeFixture(t, tt.yaml, tt.files)
			rows := checkEnvSopsDeclaration(f.cfg)
			if !tt.wantRow {
				if len(rows) != 0 {
					t.Fatalf("checkEnvSopsDeclaration() = %+v, want no rows", rows)
				}
				return
			}
			if len(rows) != 1 || rows[0].Passed {
				t.Fatalf("checkEnvSopsDeclaration() = %+v, want one failing row", rows)
			}
			if want := "{path: .env, sops_source: secrets.env.enc}"; !strings.Contains(rows[0].FixHint, want) {
				t.Fatalf("FixHint = %q, want it to contain %q", rows[0].FixHint, want)
			}
		})
	}
}

func TestNoEncryptedEntryMessageNamesEvidence(t *testing.T) {
	withEvidence := newBridgeFixture(t, "version: \"0.1.45\"\nenv_file: [.env.local]\n",
		map[string]string{"secrets.sops.env": "ENC"})
	_, err := selectEncryptedEntry(withEvidence.cfg, "")
	if got := errorCode(err); got != codeNoEncryptedEntry {
		t.Fatalf("code = %q, want %q", got, codeNoEncryptedEntry)
	}
	for _, want := range []string{"{path: .env.local, sops_source: secrets.sops.env}", "sops files found: secrets.sops.env"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not contain %q", err, want)
		}
	}

	bare := newBridgeFixture(t, "version: \"0.1.45\"\nenv_file: [.env]\n", nil)
	_, err = selectEncryptedEntry(bare.cfg, "")
	if msg := err.Error(); !strings.Contains(msg, "{path: .env, sops_source: .env.enc}") || strings.Contains(msg, "sops files found") {
		t.Fatalf("error without evidence = %q", msg)
	}
}

func TestDoctorEnvUnsealed(t *testing.T) {
	unsealed := func(f *bridgeFixture) []DoctorResult {
		return checkEnvUnsealed(f.cfg, config.InspectEnvFiles(f.cfg.EnvFile, f.cfg.FileDir(), f.cfg.EnvFileScope()))
	}
	yaml := simpleBridgeYAML(".env", "secrets.env.enc")
	absent := newBridgeFixture(t, yaml, map[string]string{"secrets.env.enc": "ENC"})
	rows := unsealed(absent)
	if len(rows) != 1 || rows[0].Passed || !strings.Contains(rows[0].FixHint, "dva config env unseal .env") {
		t.Fatalf("checkEnvUnsealed() = %+v, want one failing row with the unseal hint", rows)
	}
	found := false
	for _, r := range runDoctorChecks(absent.cfg) {
		found = found || strings.HasPrefix(r.Name, "Encrypted env target present")
	}
	if !found {
		t.Fatal("runDoctorChecks() omits the encrypted target row")
	}

	present := newBridgeFixture(t, yaml, map[string]string{"secrets.env.enc": "ENC", ".env": "A=1\n"})
	if rows := unsealed(present); len(rows) != 0 {
		t.Fatalf("checkEnvUnsealed() with the target present = %+v", rows)
	}

	plain := newBridgeFixture(t, "version: \"0.1.45\"\nenv_file: [.env]\n", nil)
	if rows := unsealed(plain); len(rows) != 0 {
		t.Fatalf("checkEnvUnsealed() for an optional plain file = %+v", rows)
	}
}

func TestUpWarnsOnEveryRouteWhenTargetMissing(t *testing.T) {
	body := simpleBridgeYAML(".env", "secrets.env.enc") + `stack:
  svc:
    default_runner: script
    runners:
      script:
        up: "true"
        stop: "true"
`
	withPlans := body + "plans:\n  one:\n    entries:\n      - name: svc\n  both:\n    composes:\n      - plan: one\n"
	for _, tt := range []struct {
		name string
		yaml string
		args []string
	}{
		{"whole stack", body, nil},
		{"plan", withPlans, []string{"one"}},
		{"composition", withPlans, []string{"both"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newBridgeFixture(t, tt.yaml, map[string]string{"secrets.env.enc": "ENC"})
			t.Chdir(f.cfg.FileDir())
			withEnvPolicyGlobals(t, nil, false)
			var err error
			stderr := captureStderr(t, func() { err = upCmd.RunE(upCmd, tt.args) })
			if err != nil {
				t.Fatalf("up %v: %v", tt.args, err)
			}
			if strings.Count(stderr, "declares sops_source") != 1 || !strings.Contains(stderr, "dva config env unseal .env") {
				t.Fatalf("stderr = %q, want one warning naming the target", stderr)
			}
		})
	}
}

func TestCompositionWarnsForChildOwner(t *testing.T) {
	stack := `stack:
  svc:
    default_runner: script
    runners:
      script:
        up: "true"
        stop: "true"
`
	root := simpleBridgeYAML(".env.root", "root.enc") + `subprojects:
  backend:
    path: backend
    import:
      plans:
        - name: deploy
plans:
  all:
    composes:
      - plan: backend/deploy
`
	child := simpleBridgeYAML(".env.child", "child.enc") + stack + "plans:\n  deploy:\n    entries:\n      - name: svc\n"
	f := newBridgeFixture(t, root, map[string]string{"backend/" + config.FileName: child})
	t.Chdir(f.cfg.FileDir())
	withEnvPolicyGlobals(t, nil, false)
	var err error
	stderr := captureStderr(t, func() { err = upCmd.RunE(upCmd, []string{"all"}) })
	if err != nil {
		t.Fatalf("up all: %v", err)
	}
	if !strings.Contains(stderr, "[child: backend/deploy] env_file .env.child") || strings.Contains(stderr, ".env.root") {
		t.Fatalf("stderr = %q, want only the child owner's target", stderr)
	}
}

func TestRestartWarnsOnEveryRouteWhenTargetMissing(t *testing.T) {
	body := simpleBridgeYAML(".env", "secrets.env.enc") + "stack:\n  svc: {default_runner: script, runners: {script: {up: 'true', stop: 'true'}}}\n"
	plans := body + "plans:\n  one: {entries: [{name: svc}]}\n  both: {composes: [{plan: one}]}\n"
	for _, tt := range []struct {
		name, yaml string
		args       []string
	}{
		{"whole stack", body, nil}, {"plan", plans, []string{"one"}}, {"composition", plans, []string{"both"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newBridgeFixture(t, tt.yaml, map[string]string{"secrets.env.enc": "ENC"})
			t.Chdir(f.dir)
			withEnvPolicyGlobals(t, f.cfg, false)
			dryRun = true
			var err error
			stderr := captureStderr(t, func() { err = restartCmd.RunE(restartCmd, tt.args) })
			if err != nil {
				t.Fatal(err)
			}
			if strings.Count(stderr, "declares sops_source") != 1 || !strings.Contains(stderr, "dva config env unseal .env") {
				t.Fatalf("wrong warning: %q", stderr)
			}
		})
	}
}

func TestRestartWarningsUseChildOwner(t *testing.T) {
	for _, plan := range []string{"backend/deploy", "all"} {
		t.Run(plan, func(t *testing.T) {
			root := simpleBridgeYAML(".env.root", "root.enc") + "subprojects:\n  backend:\n    path: backend\n    import:\n      plans: [{name: deploy}]\nplans:\n  all: {composes: [{plan: backend/deploy}]}\n"
			child := simpleBridgeYAML(".env.child", "child.enc") + "stack:\n  svc: {default_runner: script, runners: {script: {up: 'true', stop: 'true'}}}\nplans:\n  deploy: {entries: [{name: svc}]}\n"
			f := newBridgeFixture(t, root, map[string]string{"backend/" + config.FileName: child})
			t.Chdir(f.dir)
			withEnvPolicyGlobals(t, f.cfg, false)
			dryRun = true
			var err error
			stderr := captureStderr(t, func() { err = restartCmd.RunE(restartCmd, []string{plan}) })
			if err != nil {
				t.Fatal(err)
			}
			if strings.Count(stderr, "declares sops_source") != 1 || !strings.Contains(stderr, ".env.child") || strings.Contains(stderr, ".env.root") {
				t.Fatalf("wrong owner: %q", stderr)
			}
		})
	}
}

func TestUnsealedWarningsFollowTargetValidation(t *testing.T) {
	for _, tt := range []struct {
		verb string
		args []string
	}{
		{"up", []string{"typo"}}, {"up", []string{"--unknown"}}, {"restart", []string{"typo"}}, {"restart", []string{"--unknown"}},
	} {
		t.Run(tt.verb+strings.Join(tt.args, "_"), func(t *testing.T) {
			body := simpleBridgeYAML(".env", "secrets.env.enc") + "stack:\n  svc: {default_runner: script, runners: {script: {up: 'touch started', stop: 'touch stopped'}}}\nplans:\n  one: {entries: [{name: svc}]}\n"
			f := newBridgeFixture(t, body, nil)
			t.Chdir(f.dir)
			withEnvPolicyGlobals(t, f.cfg, false)
			var err error
			stderr := captureStderr(t, func() {
				if tt.verb == "up" {
					err = upCmd.RunE(upCmd, tt.args)
				} else {
					err = restartCmd.RunE(restartCmd, tt.args)
				}
			})
			if err == nil || strings.Contains(stderr, "declares sops_source") {
				t.Fatalf("invalid invocation: %v, %q", err, stderr)
			}
			for _, marker := range []string{"started", "stopped"} {
				if _, err := os.Stat(f.path(marker)); !errors.Is(err, fs.ErrNotExist) {
					t.Fatalf("side effect %s: %v", marker, err)
				}
			}
		})
	}
}

func TestRootEnvFailureStillPrecedesUnknownTarget(t *testing.T) {
	f := newBridgeFixture(t, "version: '0.1.45'\nenv_file: [{path: required.env, required: true}]\nplans:\n  one: {entries: []}\n", nil)
	for _, verb := range []string{"up", "restart"} {
		t.Run(verb, func(t *testing.T) {
			withEnvPolicyGlobals(t, f.cfg, false)
			var err error
			captureStderr(t, func() {
				if verb == "up" {
					err = upCmd.RunE(upCmd, []string{"typo"})
				} else {
					err = restartCmd.RunE(restartCmd, []string{"typo"})
				}
			})
			if err == nil || !strings.Contains(err.Error(), "required.env") || strings.Contains(err.Error(), "not found") {
				t.Fatalf("env failure precedence changed: %v", err)
			}
		})
	}
}
