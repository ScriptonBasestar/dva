package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func secretStub(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	log := filepath.Join(dir, "args")
	t.Setenv("SECRET_ARGS", log)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	stub := `#!/bin/sh
printf '%s\n' "$PWD|$*" >> "$SECRET_ARGS"
case " $* " in
  *" decrypt "*) if [ "$SECRET_FAIL" = early ]; then dd if=/dev/zero bs=65536 count=64 2>/dev/null; exit; fi; printf 'PRIVATE_SENTINEL'; if [ "$SECRET_FAIL" = decrypt ]; then printf 'PRIVATE_SENTINEL' >&2; exit 19; fi ;;
  *" encrypt "*) if [ "$SECRET_FAIL" = early ]; then printf 'no matching creation rule' >&2; exit 23; fi; cat >/dev/null; if [ "$SECRET_FAIL" = encrypt ]; then printf 'PRIVATE_SENTINEL' >&2; exit 23; fi; if [ "$SECRET_FAIL" != empty ]; then printf 'NEW_CIPHERTEXT'; fi ;;
  *" edit "*) if [ "$SECRET_FAIL" = edit ]; then printf 'PRIVATE_SENTINEL' >&2; exit 17; fi ;;
esac
`
	if err := os.WriteFile(filepath.Join(dir, "sops"), []byte(stub), 0700); err != nil {
		t.Fatal(err)
	}
	return log
}

func writeSecretFixture(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestSecretSourceEditReseal(t *testing.T) {
	for _, format := range []string{"dotenv", "yaml", "json", "ini", "binary"} {
		for _, verb := range []string{"edit", "reseal"} {
			t.Run(format+"/"+verb, func(t *testing.T) {
				log := secretStub(t)
				c := remoteTestConfig(t, "secrets:\n  sources:\n    app: {sops: nested/source.enc, format: "+format+"}\n")
				writeSecretFixture(t, c.FileDir(), "nested/source.enc", "OLD_CIPHERTEXT")
				writeSecretFixture(t, c.FileDir(), "nested/.sops.yaml", "creation_rules: []")
				withEnvPolicyGlobals(t, c, verb == "reseal")
				cmd := remoteTestCommand(t, secretCmd, verb)
				var runErr error
				out := captureStdout(t, func() { runErr = cmd.RunE(cmd, []string{"app"}) })
				if runErr != nil {
					t.Fatal(runErr)
				}
				if strings.Contains(out, "PRIVATE_SENTINEL") {
					t.Fatal("plaintext in output")
				}
				args, err := os.ReadFile(log)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(args), "--input-type "+format+" --output-type "+format) || !strings.Contains(string(args), "--config "+filepath.Join(canonicalFixtureDir(t, c.FileDir()), "nested/.sops.yaml")) {
					t.Fatalf("wrong arguments: %s", args)
				}
				data, err := os.ReadFile(filepath.Join(c.FileDir(), "nested/source.enc"))
				if err != nil {
					t.Fatal(err)
				}
				want := "OLD_CIPHERTEXT"
				if verb == "reseal" {
					want = "NEW_CIPHERTEXT"
				}
				if string(data) != want {
					t.Fatal("unexpected ciphertext")
				}
				entries, err := os.ReadDir(filepath.Join(c.FileDir(), "nested"))
				if err != nil {
					t.Fatal(err)
				}
				if len(entries) != 2 {
					t.Fatal("temporary file remains")
				}
			})
		}
	}
}

func TestSecretSourceResealFailurePreservesSource(t *testing.T) {
	for _, failure := range []string{"decrypt", "encrypt", "empty", "early"} {
		t.Run(failure, func(t *testing.T) {
			secretStub(t)
			t.Setenv("SECRET_FAIL", failure)
			c := remoteTestConfig(t, "secrets:\n  sources:\n    app: {sops: source.enc}\n")
			writeSecretFixture(t, c.FileDir(), "source.enc", "ORIGINAL")
			writeSecretFixture(t, c.FileDir(), ".sops.yaml", "creation_rules: []")
			withEnvPolicyGlobals(t, c, true)
			cmd := remoteTestCommand(t, secretCmd, "reseal")
			var runErr error
			out := captureStdout(t, func() { runErr = cmd.RunE(cmd, []string{"app"}) })
			if failure == "early" && runErr != nil && runErr.Error() != "secret_creation_rule_missing" {
				t.Fatalf("lost cause: %v", runErr)
			}
			if runErr == nil || strings.Contains(runErr.Error()+out, "PRIVATE_SENTINEL") {
				t.Fatal("failure absent or plaintext leaked")
			}
			data, err := os.ReadFile(filepath.Join(c.FileDir(), "source.enc"))
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != "ORIGINAL" {
				t.Fatal("original replaced on failure")
			}
			entries, err := os.ReadDir(c.FileDir())
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 3 {
				t.Fatal("temporary file remains")
			}
		})
	}
}

func TestSecretSourceProject(t *testing.T) {
	log := secretStub(t)
	c := remoteTestConfig(t, "subprojects:\n  api: {path: services/api}\nsecrets:\n  sources:\n    app: {sops: nested/source.enc, project: api}\n")
	writeSecretFixture(t, c.FileDir(), "services/api/nested/source.enc", "ORIGINAL")
	writeSecretFixture(t, c.FileDir(), "services/api/.sops.yaml", "creation_rules: []")
	withEnvPolicyGlobals(t, c, false)
	cmd := remoteTestCommand(t, secretCmd, "reseal")
	if err := cmd.RunE(cmd, []string{"app"}); err != nil {
		t.Fatal(err)
	}
	args, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	child := filepath.Join(canonicalFixtureDir(t, c.FileDir()), "services/api")
	if !strings.Contains(string(args), child+"|--config "+filepath.Join(child, ".sops.yaml")) {
		t.Fatalf("lost child root: %s", args)
	}
	for _, path := range []string{"../outside.enc", "/outside.enc"} {
		source := c.Secrets.Sources["app"]
		source.Sops = path
		c.Secrets.Sources["app"] = source
		if root, _, err := openSecretSource(c, "app"); err == nil {
			root.Close()
			t.Fatal("accepted escape")
		}
	}
	source := c.Secrets.Sources["app"]
	source.Sops = "link/source.enc"
	c.Secrets.Sources["app"] = source
	if err := os.Symlink("nested", filepath.Join(child, "link")); err != nil {
		t.Fatal(err)
	}
	if root, _, err := openSecretSource(c, "app"); err == nil {
		root.Close()
		t.Fatal("accepted symlink")
	}
}

func TestSecretSourceProjectFlagCombination(t *testing.T) {
	c := remoteTestConfig(t, "subprojects:\n  api: {path: api}\nsecrets:\n  sources:\n    parent: {sops: parent.enc}\n")
	writeSecretFixture(t, c.FileDir(), "api/dva.yml", "secrets:\n  sources:\n    own: {sops: own.enc}\n")
	withEnvPolicyGlobals(t, c, false)
	cmd := remoteTestCommand(t, secretCmd, "edit")
	if err := cmd.Flags().Set("project", "api"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Flags().Set("project", "") })
	child, err := remoteOwner(cmd)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := child.Secrets.Sources["parent"]; ok {
		t.Fatal("parent alias visible in child")
	}
	writeSecretFixture(t, c.FileDir(), "api/dva.yml", "subprojects:\n  nested: {path: nested}\nsecrets:\n  sources:\n    own: {sops: own.enc, project: nested}\n")
	if _, err := remoteOwner(cmd); err == nil || !strings.Contains(err.Error(), "cannot be combined") {
		t.Fatalf("nested project accepted: %v", err)
	}
}

func canonicalFixtureDir(t *testing.T, dir string) string {
	t.Helper()
	result, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestSecretSourceDiscoveryAndFlags(t *testing.T) {
	c := remoteTestConfig(t, "secrets:\n  sources:\n    app: {sops: source.enc}\n")
	withEnvPolicyGlobals(t, c, false)
	for _, verb := range []string{"edit", "reseal"} {
		cmd := remoteTestCommand(t, secretCmd, verb)
		m := buildManifest(c).StaticCommands["secret"].Subcommands[verb]
		if m.Type != "mutation" || len(m.Effects) == 0 {
			t.Fatal("missing discovery mutation")
		}
		dryRun = true
		if err := cmd.RunE(cmd, []string{"app"}); err == nil {
			t.Fatal("accepted dry run")
		}
		dryRun = false
	}
	jsonOutput = true
	cmd := remoteTestCommand(t, secretCmd, "edit")
	if err := cmd.RunE(cmd, []string{"app"}); err == nil {
		t.Fatal("accepted JSON edit")
	}
}

func TestSecretSourceRulesAndPushOwnership(t *testing.T) {
	c := remoteTestConfig(t, "subprojects:\n  api: {path: api}\nsecrets:\n  sources:\n    app: {sops: source.enc, project: api}\n  targets:\n    actions:\n      provider: github-actions\n      repository: owner/repo\n      source: app\n      keys: {KEY: KEY}\n")
	writeSecretFixture(t, c.FileDir(), "api/source.enc", "CIPHER")
	writeSecretFixture(t, c.FileDir(), ".sops.yaml", "creation_rules: []")
	writeSecretFixture(t, c.FileDir(), "api/.sops.yaml", "creation_rules: []")
	withEnvPolicyGlobals(t, c, false)
	opts, err := secretOptions(c, "actions")
	if err != nil {
		t.Fatal(err)
	}
	if opts.Root != c.FileDir() || opts.SourceRoot != filepath.Join(canonicalFixtureDir(t, c.FileDir()), "api") || opts.SourceFormat != "dotenv" || opts.Target.Repository != "owner/repo" {
		t.Fatal("source changed destination ownership")
	}
	root, source, err := openSecretSource(c, "app")
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	for _, mode := range []string{"root", "explicit"} {
		c.Sops = &config.SopsConfig{Lookup: "root"}
		if mode == "explicit" {
			c.Sops = &config.SopsConfig{Config: ".sops.yaml"}
		}
		rules, err := secretRules(c, root, source, true)
		if err != nil || rules != filepath.Join(canonicalFixtureDir(t, c.FileDir()), ".sops.yaml") {
			t.Fatalf("policy ignored: %v", err)
		}
	}
	outside := canonicalFixtureDir(t, t.TempDir())
	writeSecretFixture(t, outside, "rules.yaml", "creation_rules: []")
	if err := os.Symlink(outside, filepath.Join(c.FileDir(), "linked")); err != nil {
		t.Fatal(err)
	}
	c.Sops = &config.SopsConfig{Config: "linked/rules.yaml"}
	if _, err := secretRules(c, root, source, true); err == nil {
		t.Fatal("accepted symlinked rules parent")
	}
	c.Sops = nil
	sub := c.Subprojects["api"]
	sub.Path = outside
	c.Subprojects["api"] = sub
	writeSecretFixture(t, outside, "source.enc", "CIPHER")
	external, src, err := openSecretSource(c, "app")
	if err != nil {
		t.Fatal(err)
	}
	defer external.Close()
	if _, err := secretRules(c, external, src, true); err == nil {
		t.Fatal("nearest silently escaped owner root")
	}
}
