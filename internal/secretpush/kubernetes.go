package secretpush

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const kubeTimeout = 60 * time.Second

// kubernetesSecretKey mirrors the Kubernetes config-key charset: dots, dashes,
// and underscores are legal (tls.crt), unlike GitHub Actions names. Matching is
// case-sensitive because Kubernetes data keys are.
var kubernetesSecretKey = regexp.MustCompile(`^[-._a-zA-Z0-9]{1,253}$`)

// pushKubernetes decrypts a SOPS-encrypted Kubernetes Secret manifest, selects
// only the declared keys, and applies one Secret containing exactly those keys
// to the explicitly declared cluster. Client-side apply merges map keys, so
// data keys DVA never declared survive untouched — ownership of the Secret's
// key set is shared with whatever else wrote to it, while ownership of the
// declared keys is DVA's.
func pushKubernetes(ctx context.Context, opts Options) (Report, error) {
	report, root, source, err := prepareKubernetes(opts)
	if err != nil {
		return report, err
	}
	if opts.DryRun {
		return report, nil
	}
	if _, err := exec.LookPath("sops"); err != nil {
		return report, codeError("sops_unavailable")
	}
	if _, err := exec.LookPath("kubectl"); err != nil {
		return report, codeError("kubectl_unavailable")
	}
	kubeconfig, err := expandKubeconfigPath(opts.Target.Kubeconfig)
	if err != nil {
		return report, err
	}
	plaintext, err := decryptSOPS(ctx, root, source, "yaml", "json")
	if err != nil {
		return report, err
	}
	defer wipe(plaintext)
	values, err := parseKubernetesSecret(plaintext)
	if err != nil {
		return report, err
	}
	defer wipeValues(values)
	for sourceKey := range opts.Target.Keys {
		if _, ok := values[sourceKey]; !ok {
			return report, codeError("source_key_missing")
		}
	}

	state, err := stateDirectory(opts.StateDir)
	if err != nil {
		return report, err
	}
	if err := secureMkdir(state); err != nil {
		return report, err
	}
	unlock, err := lockTarget(state, opts.Target.Context+"/"+opts.Target.Namespace+"/"+opts.Target.SecretName)
	if err != nil {
		return report, err
	}
	defer unlock()
	receipt := filepath.Join(state, report.ID+".json")
	if err := writeReceipt(receipt, report); err != nil {
		return report, err
	}

	// json.Marshal base64-encodes []byte values, which is exactly the Secret
	// data encoding; no separate encoding step ever materializes the values
	// twice outside our control.
	data := make(map[string][]byte, len(opts.Target.Keys))
	defer func() {
		for _, value := range data {
			wipe(value)
		}
	}()
	for sourceKey, destKey := range opts.Target.Keys {
		data[destKey] = append([]byte(nil), values[sourceKey]...)
	}
	manifest, err := json.Marshal(map[string]any{
		"apiVersion": "v1",
		"kind":       "Secret",
		"metadata": map[string]any{
			"name":      opts.Target.SecretName,
			"namespace": opts.Target.Namespace,
		},
		"data": data,
	})
	if err != nil {
		return report, codeError("manifest_marshal_failed")
	}
	defer wipe(manifest)

	// One apply is one object write: after it starts, no interrupted process
	// can prove the cluster rejected or accepted it, so every key moves to
	// unknown before that boundary — the same contract as the GitHub sink.
	for i := range report.Keys {
		report.Keys[i].State = StateUnknown
	}
	if err := writeReceipt(receipt, report); err != nil {
		return report, err
	}
	err = applyKubeManifest(ctx, kubeconfig, opts.Target.Context, manifest)
	if err != nil {
		failState := StateUnknown
		if errors.Is(err, errNotInvoked) {
			failState = StateFailed
		}
		for i := range report.Keys {
			report.Keys[i].State = failState
		}
		if receiptErr := writeReceipt(receipt, report); receiptErr != nil {
			return report, receiptErr
		}
		return report, codeError("secret_push_" + failState)
	}
	for i := range report.Keys {
		report.Keys[i].State = StateAccepted
	}
	if err := writeReceipt(receipt, report); err != nil {
		return report, err
	}
	return report, nil
}

func prepareKubernetes(opts Options) (Report, string, string, error) {
	var report Report
	if opts.Name == "" || opts.Target.Source == "" || len(opts.Target.Keys) == 0 || len(opts.Target.Keys) > maxKeys {
		return report, "", "", codeError("invalid_declaration")
	}
	// Environment is fail-closed at the runtime layer too: config validation
	// already demands "dev", but a direct caller of this package bypasses that.
	if opts.Target.Environment != "dev" {
		return report, "", "", codeError("environment_not_dev")
	}
	if opts.Target.Context == "" || opts.Target.Namespace == "" || opts.Target.SecretName == "" {
		return report, "", "", codeError("invalid_declaration")
	}
	if _, err := expandKubeconfigPath(opts.Target.Kubeconfig); err != nil {
		return report, "", "", err
	}
	for source, destination := range opts.Target.Keys {
		if !kubernetesSecretKey.MatchString(source) || !kubernetesSecretKey.MatchString(destination) {
			return report, "", "", codeError("invalid_secret_key")
		}
	}
	root, err := canonicalRoot(opts.Root)
	if err != nil {
		return report, "", "", err
	}
	source, err := safeSource(root, opts.Target.Source)
	if err != nil {
		return report, "", "", err
	}
	id, err := receiptID()
	if err != nil {
		return report, "", "", err
	}
	destinations := sortedDeclaredKeys(opts.Target.Keys)
	report = Report{
		ID:          id,
		Name:        opts.Name,
		Destination: opts.Target.Context + "/" + opts.Target.Namespace + "/" + opts.Target.SecretName,
		Keys:        make([]KeyReport, len(destinations)),
	}
	for i, key := range destinations {
		report.Keys[i] = KeyReport{Key: key, State: StateNotStarted}
	}
	return report, root, source, nil
}

// kubernetesSource decodes a decrypted Secret manifest. Data and stringData are
// RawMessage so every backing array is owned and wipeable; map[string]string
// would pin plaintext in immutable strings.
type kubernetesSource struct {
	APIVersion string                     `json:"apiVersion"`
	Kind       string                     `json:"kind"`
	Data       map[string]json.RawMessage `json:"data"`
	StringData map[string]json.RawMessage `json:"stringData"`
}

// parseKubernetesSecret merges data (base64) under stringData (raw), the same
// precedence the API server applies.
func parseKubernetesSecret(plaintext []byte) (map[string][]byte, error) {
	var source kubernetesSource
	if err := json.Unmarshal(plaintext, &source); err != nil {
		return nil, codeError("invalid_kubernetes_source")
	}
	if source.Kind != "Secret" || source.APIVersion != "v1" {
		return nil, codeError("invalid_kubernetes_source")
	}
	values := make(map[string][]byte, len(source.Data)+len(source.StringData))
	// encoding/json base64-decodes a JSON string into a []byte target, which
	// is exactly the `data` wire format; invalid base64 is a parse error.
	for key, raw := range source.Data {
		var value []byte
		if err := json.Unmarshal(raw, &value); err != nil || len(value) > maxSecretValue {
			wipe(value)
			wipeValues(values)
			return nil, codeError("invalid_kubernetes_source")
		}
		values[key] = value
	}
	for key, raw := range source.StringData {
		value, ok := rawJSONString(raw)
		if !ok {
			var text string
			if err := json.Unmarshal(raw, &text); err != nil {
				wipeValues(values)
				return nil, codeError("invalid_kubernetes_source")
			}
			// The escaped path leaves an immutable string for the GC; wiping
			// is best-effort, as everywhere a decoder touches plaintext.
			value = []byte(text)
		}
		if len(value) > maxSecretValue {
			wipe(value)
			wipeValues(values)
			return nil, codeError("invalid_kubernetes_source")
		}
		if previous, exists := values[key]; exists {
			wipe(previous)
		}
		values[key] = value
	}
	return values, nil
}

// rawJSONString slices an unescaped JSON string value into its owned backing
// array, keeping the result wipeable. It declines values carrying escapes or
// embedded quotes; the caller then decodes through encoding/json instead.
func rawJSONString(raw json.RawMessage) ([]byte, bool) {
	if len(raw) < 2 || raw[0] != '"' || raw[len(raw)-1] != '"' {
		return nil, false
	}
	inner := raw[1 : len(raw)-1]
	if bytes.IndexByte(inner, '\\') >= 0 || bytes.IndexByte(inner, '"') >= 0 {
		return nil, false
	}
	return inner, true
}

// applyKubeManifest sends the manifest over stdin. The ambient KUBECONFIG is
// stripped so the declared file is the only credential source, and values are
// never present in argv.
func applyKubeManifest(ctx context.Context, kubeconfig, kubeContext string, manifest []byte) error {
	ctx, cancel := context.WithTimeout(ctx, kubeTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "kubectl", "--kubeconfig", kubeconfig, "--context", kubeContext, "apply", "-f", "-")
	cmd.WaitDelay = time.Second
	cmd.Env = withoutAmbiguousKubeEnv(os.Environ())
	cmd.Stdin = bytes.NewReader(manifest)
	cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
	if err := cmd.Start(); err != nil {
		return errNotInvoked
	}
	if err := cmd.Wait(); err != nil {
		return errors.New("secret command failed")
	}
	return nil
}

// expandKubeconfigPath accepts an absolute path or a ~/ prefix; a relative path
// would silently resolve against whatever directory dva happens to run from.
func expandKubeconfigPath(path string) (string, error) {
	if path == "" || strings.ContainsAny(path, "\x00\r\n") {
		return "", codeError("kubeconfig_invalid")
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", codeError("kubeconfig_invalid")
		}
		path = filepath.Join(home, strings.TrimPrefix(path, "~"))
	}
	if !filepath.IsAbs(path) {
		return "", codeError("kubeconfig_invalid")
	}
	return filepath.Clean(path), nil
}

// StatusReport carries declaration metadata only: which keys exist on the
// cluster and which declared keys are missing. Never values.
type StatusReport struct {
	Name        string   `json:"name"`
	Destination string   `json:"destination"`
	Exists      bool     `json:"exists"`
	Keys        []string `json:"keys,omitempty"`
	Missing     []string `json:"missing,omitempty"`
}

// kubectlStatusTemplate asks kubectl to print key names only: values stay in
// kubectl's memory and never enter this process.
const kubectlStatusTemplate = `{{range $k, $_ := .data}}{{$k}}
{{end}}`

// KubernetesStatus compares the declared keys against the live Secret.
// A NotFound answer is a status, not a failure; any other kubectl error is
// reported without its output, which may name credentials paths but never
// carries secret values.
func KubernetesStatus(ctx context.Context, opts Options) (StatusReport, error) {
	report := StatusReport{}
	if _, _, _, err := prepareKubernetes(opts); err != nil {
		return report, err
	}
	if _, err := exec.LookPath("kubectl"); err != nil {
		return report, codeError("kubectl_unavailable")
	}
	kubeconfig, err := expandKubeconfigPath(opts.Target.Kubeconfig)
	if err != nil {
		return report, err
	}
	report.Name = opts.Name
	report.Destination = opts.Target.Context + "/" + opts.Target.Namespace + "/" + opts.Target.SecretName

	ctx, cancel := context.WithTimeout(ctx, kubeTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "kubectl", "--kubeconfig", kubeconfig, "--context", opts.Target.Context,
		"-n", opts.Target.Namespace, "get", "secret", opts.Target.SecretName, "-o", "go-template="+kubectlStatusTemplate)
	cmd.WaitDelay = time.Second
	cmd.Env = withoutAmbiguousKubeEnv(os.Environ())
	stdout := &limitedBuffer{limit: maxOutput, onLimit: cancel}
	stderr := &limitedBuffer{limit: maxDiagnostic}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	runErr := cmd.Run()

	if runErr != nil {
		if bytes.Contains(stderr.buf.Bytes(), []byte("(NotFound)")) {
			report.Missing = sortedDeclaredKeys(opts.Target.Keys)
			return report, nil
		}
		wipe(stdout.buf.Bytes())
		return report, codeError("secret_status_unavailable")
	}
	live := map[string]bool{}
	// Status output carries key names only — the go-template never emits
	// values — so live keys may share the buffer's backing array, and no wipe
	// of it happens here.
	for line := range strings.SplitSeq(stdout.buf.String(), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			live[line] = true
		}
	}
	report.Exists = true
	report.Keys = make([]string, 0, len(live))
	for key := range live {
		report.Keys = append(report.Keys, key)
	}
	sort.Strings(report.Keys)
	missing := sortedDeclaredKeys(opts.Target.Keys)
	kept := missing[:0]
	for _, key := range missing {
		if !live[key] {
			kept = append(kept, key)
		}
	}
	report.Missing = kept
	return report, nil
}

func sortedDeclaredKeys(keys map[string]string) []string {
	destinations := make([]string, 0, len(keys))
	for _, destination := range keys {
		destinations = append(destinations, destination)
	}
	sort.Strings(destinations)
	return destinations
}

// withoutAmbiguousKubeEnv removes KUBECONFIG so the declared kubeconfig flag is
// authoritative and a shell-exported fallback cannot redirect the write.
func withoutAmbiguousKubeEnv(env []string) []string {
	out := make([]string, 0, len(env))
	for _, item := range env {
		if !strings.HasPrefix(item, "KUBECONFIG=") {
			out = append(out, item)
		}
	}
	return out
}
