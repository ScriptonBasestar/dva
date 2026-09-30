//go:build integration && (darwin || linux)

package integration

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/secretpush"
)

// TestSecretKubernetesStatus proves the status pipeline against real sops
// decryption and a fake kubectl: real encrypted YAML in, key names out, values
// nowhere. The kubernetes sink needs no git origin contract, so the fixture is
// only a directory holding the encrypted source.
func TestSecretKubernetesStatus(t *testing.T) {
	sops, keygen := realTool(t, "sops"), realTool(t, "age-keygen")
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	keyFile := filepath.Join(dir, "age-identity.txt")
	out, err := exec.Command(keygen, "-o", keyFile).CombinedOutput()
	if err != nil {
		t.Fatalf("age-keygen: %v: %s", err, out)
	}
	pub := agePublicKey(t, string(out))
	plain := filepath.Join(dir, "plain.yaml")
	const plainYAML = "apiVersion: v1\nkind: Secret\nmetadata:\n  name: api-secrets\n  namespace: ns1\ndata:\n  DB_PASS: cGFzcw==\n  REDIS_PASSWORD: cmVkaXMtcGxhaW4=\n"
	if err := os.WriteFile(plain, []byte(plainYAML), 0o600); err != nil {
		t.Fatal(err)
	}
	enc, err := exec.Command(sops, "encrypt", "--input-type", "yaml", "--output-type", "yaml", "--age", pub, plain).Output()
	if err != nil {
		t.Fatalf("sops encrypt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "secret.yaml"), enc, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(plain); err != nil {
		t.Fatal(err)
	}

	bin, log := t.TempDir(), filepath.Join(t.TempDir(), "kubectl-args")
	kubectl := filepath.Join(bin, "kubectl")
	if err := os.WriteFile(kubectl, []byte("#!/bin/sh\nprintf '%s\\n' \"$*\" > \"$KUBE_ARGS\"\nif [ \"${KUBE_NOT_FOUND:-0}\" = 1 ]; then printf '%s' 'Error from server (NotFound): secrets \"api-secrets\" not found' >&2; exit 1; fi\nprintf '%s\\n' 'db.password' 'redis.password'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+toolPATH(t, map[string]string{"sops": sops}))
	t.Setenv("SOPS_AGE_KEY_FILE", keyFile)
	t.Setenv("KUBE_ARGS", log)

	opts := secretpush.Options{Root: dir, Name: "primeno1-api", Target: secretpush.Target{
		Kind: "kubernetes", Source: "secret.yaml", Environment: "dev",
		Kubeconfig: "~/.kube/scripton-cluster", Context: "scripton-cluster", Namespace: "primeno1", SecretName: "api-secrets",
		Keys: map[string]string{"DB_PASS": "db.password", "REDIS_PASSWORD": "redis.password", "NEW_KEY": "new.key"},
	}}
	report, err := secretpush.KubernetesStatus(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Exists || strings.Join(report.Keys, ",") != "db.password,redis.password" || strings.Join(report.Missing, ",") != "new.key" {
		t.Fatalf("status = %+v", report)
	}
	args, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	expanded, err := expandHomeForTest()
	if err != nil {
		t.Fatal(err)
	}
	if want := "--kubeconfig " + expanded + " --context scripton-cluster -n primeno1 get secret api-secrets -o go-template="; !strings.HasPrefix(strings.TrimSuffix(string(args), "\n"), want) {
		t.Fatalf("kubectl argv = %q, want prefix %q", args, want)
	}
	for _, value := range []string{"pass", "plain"} {
		if strings.Contains(strings.TrimSuffix(string(args), "\n"), value) {
			t.Fatalf("kubectl argv leaked plaintext: %q", args)
		}
	}

	t.Setenv("KUBE_NOT_FOUND", "1")
	report, err = secretpush.KubernetesStatus(context.Background(), opts)
	if err != nil {
		t.Fatalf("not-found should be a status, not an error: %v", err)
	}
	if report.Exists || strings.Join(report.Missing, ",") != "db.password,new.key,redis.password" {
		t.Fatalf("not-found status = %+v", report)
	}
}

func expandHomeForTest() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".kube", "scripton-cluster"), nil
}
