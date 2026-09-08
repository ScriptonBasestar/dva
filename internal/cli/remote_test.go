package cli

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
	"github.com/ScriptonBasestar/dva/internal/jobrun"
	"github.com/ScriptonBasestar/dva/internal/secretpush"
	"github.com/spf13/cobra"
)

const remoteCLIConfig = `secrets:
  sources:
    dockerhub: {sops: dockerhub.env.enc}
  targets:
    dockerhub-actions:
      provider: github-actions
      repository: owner/images
      source: dockerhub
      keys: {DOCKER_PASSWORD: DOCKER_PASSWORD}
jobs:
  extensions:
    provider: github-actions
    repository: owner/images
    ref: artifact-source
    timeout: 30m
    inputs:
      pg_version: {default: "18", values: ["17", "18"]}
    secret_targets: [dockerhub-actions]
    runs:
      - name: essential
        workflow: build.yml
        inputs: {pg_version: "{{input.pg_version}}"}
        result_artifact: dva-result
        images:
          - reference: "docker.io/owner/postgres:{{input.pg_version}}-essential"
`

func remoteTestCommand(t *testing.T, parent *cobra.Command, name string) *cobra.Command {
	t.Helper()
	for _, cmd := range parent.Commands() {
		if cmd.Name() == name {
			oldContext := cmd.Context()
			cmd.SetContext(t.Context())
			t.Cleanup(func() { cmd.SetContext(oldContext) })
			return cmd
		}
	}
	t.Fatalf("missing command %s", name)
	return nil
}

func remoteTestConfig(t *testing.T, text string) *config.Config {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "dva.yml"), []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestRemoteDiscoveryAndPreviewDoNotExecute(t *testing.T) {
	c := remoteTestConfig(t, remoteCLIConfig)
	for _, args := range [][]string{{"init", c.FileDir()}, {"-C", c.FileDir(), "remote", "add", "origin", "https://github.com/owner/images.git"}} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git: %v %s", err, out)
		}
	}
	withEnvPolicyGlobals(t, c, true)
	m := buildManifest(c)
	if len(m.Jobs) != 1 || len(m.SecretTargets) != 1 || len(m.StaticCommands["secret"].Subcommands["push"].Effects) == 0 {
		t.Fatal("missing remote discovery/effects")
	}
	cmd := remoteTestCommand(t, jobCmd, "run")
	oldRun := runRemoteJob
	runRemoteJob = func(context.Context, jobrun.Options) (jobrun.Report, error) {
		t.Fatal("preview executed remote job")
		return jobrun.Report{}, nil
	}
	t.Cleanup(func() { runRemoteJob = oldRun })
	dryRun = true
	var runErr error
	data := captureStdout(t, func() { runErr = cmd.RunE(cmd, []string{"extensions"}) })
	if runErr != nil {
		t.Fatal(runErr)
	}
	if !json.Valid([]byte(data)) || !strings.Contains(data, "18-essential") || strings.Contains(data, "{{input.") {
		t.Fatalf("preview lost resolved inputs: %s", data)
	}
}

func TestRemoteJobSecretsAreExplicitAndFailureStopsDispatch(t *testing.T) {
	c := remoteTestConfig(t, remoteCLIConfig)
	withEnvPolicyGlobals(t, c, true)
	dryRun = false
	cmd := remoteTestCommand(t, jobCmd, "run")
	oldRun, oldPush := runRemoteJob, pushRemoteSecret
	t.Cleanup(func() {
		runRemoteJob = oldRun
		pushRemoteSecret = oldPush
		_ = cmd.Flags().Set("with-secrets", "false")
	})
	pushes, dispatches := 0, 0
	pushRemoteSecret = func(_ context.Context, opts secretpush.Options) (secretpush.Report, error) {
		pushes++
		if opts.Root != c.FileDir() || opts.Target.Repository != "owner/images" {
			t.Fatal("secret owner changed")
		}
		return secretpush.Report{Name: opts.Name}, errors.New("secret_push_unknown")
	}
	runRemoteJob = func(ctx context.Context, opts jobrun.Options) (jobrun.Report, error) {
		if opts.BeforeDispatch != nil {
			if err := opts.BeforeDispatch(ctx); err != nil {
				return jobrun.Report{ID: "prepared"}, err
			}
		}
		dispatches++
		return jobrun.Report{ID: "dispatched"}, nil
	}
	var runErr error
	captureStdout(t, func() { runErr = cmd.RunE(cmd, []string{"extensions"}) })
	if runErr != nil || pushes != 0 || dispatches != 1 {
		t.Fatalf("implicit secret mutation: %v %d %d", runErr, pushes, dispatches)
	}
	if err := cmd.Flags().Set("with-secrets", "true"); err != nil {
		t.Fatal(err)
	}
	data := captureStdout(t, func() { runErr = cmd.RunE(cmd, []string{"extensions"}) })
	if runErr == nil || pushes != 1 || dispatches != 1 || !json.Valid([]byte(data)) || !strings.Contains(data, "secret_push_unknown") {
		t.Fatalf("partial failure lost: %v %d %d %s", runErr, pushes, dispatches, data)
	}
}

func TestRemoteChildUsesOwnSecretsAndDirectory(t *testing.T) {
	c := remoteTestConfig(t, "subprojects:\n  images: {path: child}\n")
	child := filepath.Join(c.FileDir(), "child")
	if err := os.Mkdir(child, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(child, "dva.yml"), []byte(remoteCLIConfig), 0600); err != nil {
		t.Fatal(err)
	}
	withEnvPolicyGlobals(t, c, true)
	dryRun = false
	cmd := remoteTestCommand(t, secretCmd, "push")
	oldPush := pushRemoteSecret
	t.Cleanup(func() { pushRemoteSecret = oldPush; _ = cmd.Flags().Set("project", "") })
	pushRemoteSecret = func(_ context.Context, opts secretpush.Options) (secretpush.Report, error) {
		if opts.Root != child || opts.Target.Source != "dockerhub.env.enc" {
			t.Fatalf("wrong child context: %#v", opts)
		}
		return secretpush.Report{Name: opts.Name}, nil
	}
	if err := cmd.Flags().Set("project", "images"); err != nil {
		t.Fatal(err)
	}
	var runErr error
	captureStdout(t, func() { runErr = cmd.RunE(cmd, []string{"dockerhub-actions"}) })
	if runErr != nil {
		t.Fatal(runErr)
	}
}

func TestRemoteReceiptCommandsRejectDryRun(t *testing.T) {
	withEnvPolicyGlobals(t, nil, true)
	dryRun = true
	for _, name := range []string{"status", "resume", "verify"} {
		cmd := remoteTestCommand(t, jobCmd, name)
		if err := cmd.RunE(cmd, []string{"not-a-receipt"}); err == nil || !strings.Contains(err.Error(), "--dry-run") {
			t.Fatalf("%s accepted preview: %v", name, err)
		}
	}
}

func TestRemoteChildInvalidProviderCannotPush(t *testing.T) {
	c := remoteTestConfig(t, "subprojects:\n  images: {path: child}\n")
	child := filepath.Join(c.FileDir(), "child")
	if err := os.Mkdir(child, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(child, "dva.yml"), []byte(strings.ReplaceAll(remoteCLIConfig, "provider: github-actions", "provider: unsupported")), 0600); err != nil {
		t.Fatal(err)
	}
	withEnvPolicyGlobals(t, c, true)
	dryRun = false
	cmd := remoteTestCommand(t, secretCmd, "push")
	old := pushRemoteSecret
	pushRemoteSecret = func(context.Context, secretpush.Options) (secretpush.Report, error) {
		t.Fatal("invalid child invoked provider")
		return secretpush.Report{}, nil
	}
	t.Cleanup(func() { pushRemoteSecret = old; _ = cmd.Flags().Set("project", "") })
	if err := cmd.Flags().Set("project", "images"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.RunE(cmd, []string{"dockerhub-actions"}); err == nil {
		t.Fatal("child bypassed remote validation")
	}
}

func TestRemoteInvalidImagePreviewFailsBeforeEffects(t *testing.T) {
	for _, replacement := range []string{"docker.io/owner/postgres", "docker.io/owner/../postgres:v1"} {
		c := remoteTestConfig(t, strings.ReplaceAll(remoteCLIConfig, "docker.io/owner/postgres:{{input.pg_version}}-essential", replacement))
		withEnvPolicyGlobals(t, c, true)
		dryRun = true
		cmd := remoteTestCommand(t, jobCmd, "run")
		if err := cmd.RunE(cmd, []string{"extensions"}); err == nil {
			t.Fatal("invalid image passed preview")
		}
	}
}
