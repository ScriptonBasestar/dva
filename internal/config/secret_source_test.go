package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSecretSourceFormat(t *testing.T) {
	github := SecretTarget{Provider: "github-actions", Repository: "owner/repo", Source: "app", Keys: map[string]string{"KEY": "KEY"}}
	kubernetes := SecretTarget{Provider: "kubernetes", Environment: "dev", Kubeconfig: "~/.kube/config", Context: "dev", Namespace: "dev", SecretName: "app", Source: "app", Keys: map[string]string{"KEY": "KEY"}}
	for _, tc := range []struct {
		name, format, expected string
		targets                map[string]SecretTarget
		valid                  bool
	}{
		{"default", "", "dotenv", nil, true},
		{"dotenv", "dotenv", "dotenv", nil, true},
		{"yaml", "yaml", "yaml", nil, true},
		{"json", "json", "json", nil, true},
		{"ini", "ini", "ini", nil, true},
		{"binary", "binary", "binary", nil, true},
		{"invalid", "xml", "", nil, false},
		{"github default", "", "dotenv", map[string]SecretTarget{"github": github}, true},
		{"kubernetes inferred", "", "yaml", map[string]SecretTarget{"kube": kubernetes}, true},
		{"github rejects yaml", "yaml", "", map[string]SecretTarget{"github": github}, false},
		{"kubernetes rejects dotenv", "dotenv", "", map[string]SecretTarget{"kube": kubernetes}, false},
		{"shared source incompatible", "", "", map[string]SecretTarget{"github": github, "kube": kubernetes}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := &Config{Secrets: &SecretsConfig{Sources: map[string]SecretSource{"app": {Sops: "secrets.enc", Format: tc.format}}, Targets: tc.targets}}
			err := c.ValidateRemoteDeclarations()
			if (err == nil) != tc.valid {
				t.Fatalf("ValidateRemoteDeclarations() = %v, want valid=%v", err, tc.valid)
			}
			if tc.valid && c.SecretSourceFormat("app") != tc.expected {
				t.Errorf("resolved format = %q, want %q", c.SecretSourceFormat("app"), tc.expected)
			}
		})
	}

	for _, format := range []string{"dotenv", "yaml", "json", "ini", "binary", "xml"} {
		t.Run("schema/"+format, func(t *testing.T) {
			doc := fmt.Sprintf("secrets:\n  sources:\n    app: {sops: secrets.enc, format: %s}\n", format)
			err := validateYAMLSchema([]byte(doc))
			if (err == nil) != (format != "xml") {
				t.Fatalf("format %q schema error = %v", format, err)
			}
		})
	}
}

func TestSecretSourceProject(t *testing.T) {
	for _, tc := range []struct {
		name, project string
		valid         bool
	}{
		{"local source", "", true},
		{"declared project", "api", true},
		{"unknown project", "missing", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := &Config{
				Subprojects: map[string]SubprojectConfig{"api": {Path: "services/api"}},
				Secrets:     &SecretsConfig{Sources: map[string]SecretSource{"app": {Sops: ".env.enc", Project: tc.project}}},
			}
			err := c.ValidateRemoteDeclarations()
			if (err == nil) != tc.valid {
				t.Fatalf("ValidateRemoteDeclarations() = %v, want valid=%v", err, tc.valid)
			}
			if !tc.valid && !strings.Contains(err.Error(), "unknown project") {
				t.Fatalf("missing project diagnostic: %v", err)
			}
		})
	}

	// A source can be edited without declaring a remote destination.
	doc := "secrets:\n  sources:\n    app: {sops: .env.enc, format: dotenv}\n"
	if err := validateYAMLSchema([]byte(doc)); err != nil {
		t.Fatalf("source-only schema rejected: %v", err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(doc), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err != nil {
		t.Fatalf("source-only configuration rejected: %v", err)
	}
}
