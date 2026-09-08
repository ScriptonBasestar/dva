package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const remoteFixture = `secrets:
  sources:
    dockerhub: {sops: secrets/dockerhub.env.enc}
  targets:
    dockerhub-actions:
      provider: github-actions
      repository: owner/images
      source: dockerhub
      keys: {DOCKER_USERNAME: DOCKER_USERNAME, DOCKER_PASSWORD: DOCKER_PASSWORD}
jobs:
  postgres-extensions:
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
        inputs: {pg_version: "{{input.pg_version}}", variant: essential}
        result_artifact: dva-result
        images:
          - reference: "docker.io/owner/postgres:{{input.pg_version}}-essential"
            platforms: [linux/amd64, linux/arm64]
`

func TestRemoteConfigLoadAndSchema(t *testing.T) {
	c := loadConfigForSchemaTest(t, t.TempDir(), remoteFixture)
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	if c.Jobs["postgres-extensions"].Inputs["pg_version"].Default != "18" {
		t.Fatal("lost job input")
	}
	if c.Secrets.Sources["dockerhub"].Sops != "secrets/dockerhub.env.enc" {
		t.Fatal("lost secret source")
	}
}

func TestRemoteConfigRejectsInvalidContracts(t *testing.T) {
	for _, test := range []struct{ name, old, next string }{
		{"undefined source", "source: dockerhub", "source: missing"},
		{"traversal", "sops: secrets/dockerhub.env.enc", "sops: ../private.env.enc"},
		{"duplicate destination", "DOCKER_PASSWORD: DOCKER_PASSWORD", "DOCKER_PASSWORD: docker_username"},
		{"reserved destination", "DOCKER_PASSWORD: DOCKER_PASSWORD", "DOCKER_PASSWORD: GITHUB_TOKEN"},
		{"unsupported provider", "provider: github-actions", "provider: gitlab"},
		{"unknown template", "input.pg_version", "input.missing"},
		{"invalid timeout", "timeout: 30m", "timeout: 0s"},
		{"wrong default", "default: \"18\"", "default: \"19\""},
		{"missing target", "secret_targets: [dockerhub-actions]", "secret_targets: [missing]"},
		{"cross repository", "    ref: artifact-source", "    repository: other/images\n    ref: artifact-source"},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, FileName), []byte(strings.ReplaceAll(remoteFixture, test.old, test.next)), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(dir); err == nil {
				t.Fatal("accepted invalid remote declaration")
			}
		})
	}
}

func TestRemoteMergeCannotWidenKeySelectionOrRedirect(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(remoteFixture), 0600); err != nil {
		t.Fatal(err)
	}
	override := `secrets:
  targets:
    dockerhub-actions:
      keys: {DOCKER_PASSWORD: DOCKER_PASSWORD}
jobs:
  postgres-extensions:
    secret_targets: []
    inputs:
      pg_version: {default: "19"}
`
	if err := os.WriteFile(filepath.Join(dir, "dva.override.yml"), []byte(override), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Secrets.Targets["dockerhub-actions"].Keys) != 1 || len(c.Jobs["postgres-extensions"].SecretTargets) != 0 || len(c.Jobs["postgres-extensions"].Inputs["pg_version"].Values) != 0 {
		t.Fatal("authority or validation contract was incorrectly merged")
	}
	for _, section := range []string{"secrets:\n  targets:\n    dockerhub-actions:", "jobs:\n  postgres-extensions:"} {
		indent := "    "
		if strings.HasPrefix(section, "secrets") {
			indent = "      "
		}
		if err := os.WriteFile(filepath.Join(dir, "dva.override.yml"), []byte(section+"\n"+indent+"repository: other/images\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(dir); err == nil || !strings.Contains(err.Error(), "restricted field") {
			t.Fatalf("redirect accepted: %v", err)
		}
	}
}

func TestRemoteSchemaRejectsPlaintextAndUnknownFields(t *testing.T) {
	for _, value := range []string{
		strings.Replace(remoteFixture, "sops: secrets/dockerhub.env.enc", "sops: secrets/dockerhub.env.enc, value: forbidden", 1),
		strings.Replace(remoteFixture, "timeout: 30m", "timeout: 30m\n    command: forbidden", 1),
	} {
		if err := validateYAMLSchema([]byte(value)); err == nil {
			t.Fatal("schema accepted unrecognized execution/secret field")
		}
	}
}
