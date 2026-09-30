package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const kubernetesSecretFixture = `secrets:
  sources:
    primeno1: {sops: deploy/secrets/dev/primeno1-api-secrets.yaml}
  targets:
    primeno1-api:
      provider: kubernetes
      environment: dev
      kubeconfig: ~/.kube/scripton-cluster
      context: scripton-cluster
      namespace: primeno1
      name: primeno1-api-secrets
      source: primeno1
      keys: {DB_PASS: db.password, REDIS_PASSWORD: redis.password}
`

func TestSecretKubernetesTarget(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(kubernetesSecretFixture), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(dir)
	if err != nil {
		t.Fatalf("valid kubernetes target rejected: %v", err)
	}
	target := c.Secrets.Targets["primeno1-api"]
	if target.Environment != "dev" || target.Kubeconfig != "~/.kube/scripton-cluster" || target.Context != "scripton-cluster" || target.Namespace != "primeno1" || target.SecretName != "primeno1-api-secrets" || len(target.Keys) != 2 {
		t.Fatalf("lost kubernetes fields: %+v", target)
	}
	for _, test := range []struct{ name, old, next string }{
		{"missing environment", "      environment: dev\n", ""},
		{"non-dev environment", "environment: dev", "environment: staging"},
		{"github repository on kubernetes", "      context: scripton-cluster", "      repository: owner/images\n      context: scripton-cluster"},
		{"relative kubeconfig", "kubeconfig: ~/.kube/scripton-cluster", "kubeconfig: kube/config"},
		{"invalid context", "context: scripton-cluster", "context: Scripton_Cluster"},
		{"invalid namespace", "namespace: primeno1", "namespace: PRIMENO1"},
		{"invalid name", "name: primeno1-api-secrets", "name: primeno1 api secrets"},
		{"consecutive dots in name", "name: primeno1-api-secrets", "name: a..b"},
		{"overlong label in name", "name: primeno1-api-secrets", "name: " + strings.Repeat("a", 64) + ".b"},
		{"invalid destination key", "db.password", "db password"},
		{"undefined source", "source: primeno1", "source: missing"},
		{"empty keys", "keys: {DB_PASS: db.password, REDIS_PASSWORD: redis.password}", "keys: {}"},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, FileName), []byte(strings.ReplaceAll(kubernetesSecretFixture, test.old, test.next)), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(dir); err == nil {
				t.Fatal("accepted invalid kubernetes secret target")
			}
		})
	}
}
