package secretpush

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Values are long and distinctive on purpose: a short probe like "pass" would
// substring-match the key name "db.password" and false-positive the leak check.
var (
	kubeSecretValue = []byte("z3cr3t-pass-value")
	kubeRedisValue  = []byte("redis-plain-t9w")
	// cannedSecretJSON is what `sops --decrypt --input-type yaml --output-type
	// json` emits for a decrypted Secret manifest: data values base64,
	// stringData raw. EXTRA exists only to prove unmapped keys never reach the
	// manifest.
	cannedSecretJSON = fmt.Sprintf(`{"apiVersion":"v1","kind":"Secret","metadata":{"name":"src","namespace":"ns"},"data":{"DB_PASS":%q,"EXTRA":"ZXh0cmE="},"stringData":{"REDIS_PASSWORD":%q}}`,
		base64.StdEncoding.EncodeToString(kubeSecretValue), string(kubeRedisValue))
)

func kubeFixture(t *testing.T) (root, state, log string) {
	t.Helper()
	root, state, log = t.TempDir(), filepath.Join(canonicalTemp(t), "state"), t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "secret.yaml"), []byte("placeholder"), 0o600); err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	writeExecutable(t, filepath.Join(bin, "sops"), "#!/bin/sh\nif [ -n \"${SOPS_ARGS_FILE:-}\" ]; then printf '%s' \"$*\" > \"$SOPS_ARGS_FILE\"; fi\nif [ \"${SOPS_EXIT:-0}\" != 0 ]; then printf '%s' \"${SOPS_STDERR:-}\" >&2; exit \"$SOPS_EXIT\"; fi\nprintf '%s' \"${SOPS_JSON:-}\"\n")
	writeExecutable(t, filepath.Join(bin, "kubectl"), "#!/bin/sh\n: \"${KUBE_LOG:?}\"\n: \"${KUBE_COUNT:=0}\"\nKUBE_COUNT=$((KUBE_COUNT + 1)); export KUBE_COUNT\nprintf '%s\\n' \"$*\" > \"$KUBE_LOG/$KUBE_COUNT\"\nif [ -n \"${KUBE_STDIN_DIR:-}\" ]; then cat > \"$KUBE_STDIN_DIR/$KUBE_COUNT\"; fi\nif [ \"${KUBE_FAIL:-0}\" = 1 ]; then exit 1; fi\nexit 0\n")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return root, state, log
}

func kubeOptions(root, state string) Options {
	return Options{Root: root, Name: "primeno1-api", StateDir: state, Target: Target{
		Kind: "kubernetes", Source: "secret.yaml", Environment: "dev",
		Kubeconfig: "~/kube/config", Context: "c1", Namespace: "ns1", SecretName: "api-secrets",
		Keys: map[string]string{"DB_PASS": "db.password", "REDIS_PASSWORD": "redis.password"},
	}}
}

func TestKubernetesKeyMapping(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root, state, log := kubeFixture(t)
	stdinDir := t.TempDir()
	t.Setenv("SOPS_JSON", cannedSecretJSON)
	t.Setenv("KUBE_LOG", log)
	t.Setenv("KUBE_STDIN_DIR", stdinDir)

	report, err := Push(context.Background(), kubeOptions(root, state))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := report.Destination, "c1/ns1/api-secrets"; got != want {
		t.Fatalf("destination = %q, want %q", got, want)
	}
	for _, key := range report.Keys {
		if key.State != StateAccepted {
			t.Fatalf("key state = %+v", report.Keys)
		}
	}
	args, err := os.ReadFile(filepath.Join(log, "1"))
	if err != nil {
		t.Fatal(err)
	}
	if want := "--kubeconfig " + filepath.Join(home, "kube", "config") + " --context c1 apply -f -"; strings.TrimSuffix(string(args), "\n") != want {
		t.Fatalf("kubectl argv = %q, want %q", args, want)
	}
	manifest, err := os.ReadFile(filepath.Join(stdinDir, "1"))
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		APIVersion string            `json:"apiVersion"`
		Kind       string            `json:"kind"`
		Metadata   map[string]string `json:"metadata"`
		Data       map[string]string `json:"data"`
		StringData map[string]string `json:"stringData"`
	}
	if err := json.Unmarshal(manifest, &decoded); err != nil {
		t.Fatalf("manifest = %s", manifest)
	}
	if decoded.APIVersion != "v1" || decoded.Kind != "Secret" || decoded.Metadata["name"] != "api-secrets" || decoded.Metadata["namespace"] != "ns1" {
		t.Fatalf("manifest identity = %+v", decoded)
	}
	want := map[string]string{"db.password": base64.StdEncoding.EncodeToString(kubeSecretValue), "redis.password": base64.StdEncoding.EncodeToString(kubeRedisValue)}
	if len(decoded.Data) != len(want) {
		t.Fatalf("manifest data = %v, want %v", decoded.Data, want)
	}
	for key, value := range want {
		if decoded.Data[key] != value {
			t.Fatalf("data[%s] = %q, want %q", key, decoded.Data[key], value)
		}
	}
	if _, exists := decoded.StringData["EXTRA"]; exists || len(decoded.StringData) != 0 {
		t.Fatalf("unmapped key reached the manifest: %v", decoded)
	}
}

func TestKubernetesRejectsBadDeclarationsBeforeClusterCalls(t *testing.T) {
	root, state, log := kubeFixture(t)
	t.Setenv("SOPS_JSON", cannedSecretJSON)
	t.Setenv("KUBE_LOG", log)

	o := kubeOptions(root, state)
	o.Target.Environment = "staging"
	if _, err := Push(context.Background(), o); code(err) != "environment_not_dev" {
		t.Fatalf("staging error = %v", err)
	}
	o = kubeOptions(root, state)
	o.Target.Kubeconfig = "kube/config"
	if _, err := Push(context.Background(), o); code(err) != "kubeconfig_invalid" {
		t.Fatalf("relative kubeconfig error = %v", err)
	}
	o = kubeOptions(root, state)
	o.Target.Keys = map[string]string{"DB_PASS": "has space"}
	if _, err := Push(context.Background(), o); code(err) != "invalid_secret_key" {
		t.Fatalf("invalid key error = %v", err)
	}
	o = kubeOptions(root, state)
	o.Target.Keys = map[string]string{"MISSING": "dest"}
	t.Setenv("KUBE_STDIN_DIR", "")
	if _, err := Push(context.Background(), o); code(err) != "source_key_missing" {
		t.Fatalf("missing source error = %v", err)
	}
	o = kubeOptions(root, state)
	o.Target.Keys = map[string]string{"DB_PASS": "dest"}
	t.Setenv("SOPS_JSON", strings.Replace(cannedSecretJSON, `"kind":"Secret"`, `"kind":"ConfigMap"`, 1))
	if _, err := Push(context.Background(), o); code(err) != "invalid_kubernetes_source" {
		t.Fatalf("wrong kind error = %v", err)
	}
	if entries, _ := os.ReadDir(log); len(entries) != 0 {
		t.Fatalf("kubectl ran for rejected declarations: %v", entries)
	}
	if entries, _ := os.ReadDir(state); len(entries) != 0 {
		t.Fatalf("receipt written for rejected declarations: %v", entries)
	}
}

func TestKubernetesNoPlaintextLeak(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root, state, log := kubeFixture(t)
	sopsArgs := filepath.Join(t.TempDir(), "sops-args")
	t.Setenv("SOPS_ARGS_FILE", sopsArgs)
	t.Setenv("SOPS_JSON", cannedSecretJSON)
	t.Setenv("KUBE_LOG", log)

	// A source that is not SOPS-encrypted is refused by sops itself before any
	// cluster call; the plaintext-file case is exactly this failure path.
	t.Setenv("SOPS_EXIT", "1")
	o := kubeOptions(root, state)
	if _, err := Push(context.Background(), o); code(err) != "sops_decrypt_failed" {
		t.Fatalf("decrypt refusal error = %v", err)
	}
	if entries, _ := os.ReadDir(log); len(entries) != 0 {
		t.Fatal("kubectl ran with undecryptable source")
	}
	if entries, _ := os.ReadDir(state); len(entries) != 0 {
		t.Fatalf("receipt written before decryption: %v", entries)
	}

	t.Setenv("SOPS_EXIT", "0")
	if _, err := Push(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	args, err := os.ReadFile(sopsArgs)
	if err != nil || string(args) != "--decrypt --input-type yaml --output-type json "+filepath.Join(canonicalRoot, "secret.yaml") {
		t.Fatalf("sops argv = %q, err = %v", args, err)
	}
	for _, key := range []string{string(kubeSecretValue), string(kubeRedisValue)} {
		for _, dir := range []string{log, state} {
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				b, err := os.ReadFile(filepath.Join(dir, entry.Name()))
				if err != nil {
					t.Fatal(err)
				}
				if strings.Contains(string(b), key) {
					t.Fatalf("%s leaked %q: %s", filepath.Join(dir, entry.Name()), key, b)
				}
			}
		}
	}

	// After kubectl starts, the outcome is provable only as unknown.
	t.Setenv("KUBE_FAIL", "1")
	failed, err := Push(context.Background(), o)
	if code(err) != "secret_push_unknown" {
		t.Fatalf("kubectl failure error = %v", err)
	}
	for _, key := range failed.Keys {
		if key.State != StateUnknown {
			t.Fatalf("state after failed apply = %+v", failed.Keys)
		}
	}
	// Two pushes means two receipts; read the failed run's own receipt by ID
	// rather than whichever file sorts last.
	storedB, err := os.ReadFile(filepath.Join(state, failed.ID+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var stored Report
	if err := json.Unmarshal(storedB, &stored); err != nil {
		t.Fatal(err)
	}
	for _, key := range stored.Keys {
		if key.State != StateUnknown {
			t.Fatalf("stored state after failed apply = %+v", stored.Keys)
		}
	}
}
