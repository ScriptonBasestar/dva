package secretpush

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPushDecryptFailureMakesNoRemoteWrite(t *testing.T) {
	root, state, log := fixture(t)
	t.Setenv("SOPS_EXIT", "1")
	_, err := Push(context.Background(), options(root, state))
	if code(err) != "sops_decrypt_failed" {
		t.Fatalf("error = %v", err)
	}
	if entries, _ := os.ReadDir(log); len(entries) != 0 {
		t.Fatalf("gh ran after decrypt failure: %v", entries)
	}
	if entries, _ := os.ReadDir(state); len(entries) != 0 {
		t.Fatalf("receipt written before decrypted validation: %v", entries)
	}
}

func TestPushRejectsDuplicateAndMissingSourceBeforeWrites(t *testing.T) {
	root, state, log := fixture(t)
	o := options(root, state)
	o.Target.Keys = map[string]string{"ONE": "DEST", "TWO": "DEST"}
	if _, err := Push(context.Background(), o); code(err) != "duplicate_destination_key" {
		t.Fatalf("duplicate error = %v", err)
	}
	if entries, _ := os.ReadDir(log); len(entries) != 0 {
		t.Fatal("gh ran for duplicate mapping")
	}
	o.Target.Keys = map[string]string{"MISSING": "DEST"}
	if _, err := Push(context.Background(), o); code(err) != "source_key_missing" {
		t.Fatalf("missing error = %v", err)
	}
	if entries, _ := os.ReadDir(log); len(entries) != 0 {
		t.Fatal("gh ran for missing source")
	}
}

func TestPushRejectsGitHubReservedAndOversizedDeclarations(t *testing.T) {
	root, state, _ := fixture(t)
	o := options(root, state)
	o.Target.Keys = map[string]string{"ONE": "GITHUB_TOKEN"}
	if _, err := Push(context.Background(), o); code(err) != "invalid_secret_key" {
		t.Fatalf("reserved key error = %v", err)
	}
	o.Target.Keys = make(map[string]string, maxKeys+1)
	for i := range maxKeys + 1 {
		o.Target.Keys["KEY"+string(rune('A'+i))] = "DEST" + string(rune('A'+i))
	}
	if _, err := Push(context.Background(), o); code(err) != "invalid_declaration" {
		t.Fatalf("too many keys error = %v", err)
	}
}

func TestParseDotenvMatchesConfigLogicalValues(t *testing.T) {
	values, err := parseDotenv([]byte("  export PLAIN= value # comment  \nexport_token=kept\nSINGLE=' one # two '\nDOUBLE=\"line\\n\\t\\r\\\"\\\\ # kept  \"\n"))
	if err != nil {
		t.Fatal(err)
	}
	defer wipeValues(values)
	for key, want := range map[string]string{"PLAIN": "value", "export_token": "kept", "SINGLE": " one # two ", "DOUBLE": "line\n\t\r\"\\ # kept  "} {
		if got := string(values[key]); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
	if _, err := parseDotenv([]byte("DUP=one\nDUP=two\n")); code(err) != "duplicate_source_key" {
		t.Fatalf("duplicate error = %v", err)
	}
}

func TestPushPartialFailureRecordsUnknownAndStops(t *testing.T) {
	root, state, log := fixture(t)
	t.Setenv("GH_FAIL", "1")
	t.Setenv("SOPS_BODY", "ONE=one\nTWO=two")
	o := options(root, state)
	o.Target.Keys = map[string]string{"ONE": "A_DEST", "TWO": "B_DEST"}
	report, err := Push(context.Background(), o)
	if code(err) != "secret_push_unknown" {
		t.Fatalf("error = %v", err)
	}
	if got, want := report.Keys[0].State, StateUnknown; got != want {
		t.Fatalf("first state = %q, want %q", got, want)
	}
	if got, want := report.Keys[1].State, StateNotStarted; got != want {
		t.Fatalf("second state = %q, want %q", got, want)
	}
	entries, _ := os.ReadDir(log)
	if len(entries) != 1 {
		t.Fatalf("gh invocations = %d, want 1", len(entries))
	}
	stored := readReport(t, state)
	if stored.Keys[0].State != StateUnknown {
		t.Fatalf("stored state = %+v", stored.Keys)
	}
}

func TestPushReceiptAndErrorsDoNotLeakValues(t *testing.T) {
	root, state, log := fixture(t)
	const secret = "very-secret-value-never-in-receipt"
	t.Setenv("SOPS_BODY", "ONE="+secret+"\n")
	sopsArgs := filepath.Join(t.TempDir(), "sops-args")
	t.Setenv("SOPS_ARGS_FILE", sopsArgs)
	report, err := Push(context.Background(), options(root, state))
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(state, report.ID+".json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), secret) {
		t.Fatalf("receipt leaked secret: %s", b)
	}
	if strings.Contains(string(b), "SOPS_BODY") {
		t.Fatalf("receipt contains process data: %s", b)
	}
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	args, err := os.ReadFile(sopsArgs)
	if err != nil || string(args) != "--decrypt --input-type dotenv --output-type dotenv "+filepath.Join(canonicalRoot, "secrets.env") {
		t.Fatalf("sops argv = %q, err = %v", args, err)
	}
	ghArgs, err := os.ReadFile(filepath.Join(log, "1"))
	if err != nil || string(ghArgs) != "secret set ONE_DEST --repo github.com/acme/widget" {
		t.Fatalf("gh argv = %q, err = %v", ghArgs, err)
	}
}

func TestPushDryRunDoesNotInvokeToolsOrWriteReceipt(t *testing.T) {
	root, state, log := fixture(t)
	o := options(root, state)
	o.DryRun = true
	report, err := Push(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	if report.Keys[0].State != StateNotStarted {
		t.Fatalf("dry run state = %q", report.Keys[0].State)
	}
	if _, err := os.Stat(state); !os.IsNotExist(err) {
		t.Fatalf("dry run state directory = %v", err)
	}
	if entries, _ := os.ReadDir(log); len(entries) != 0 {
		t.Fatalf("external mutation in dry run: %v", entries)
	}
}

func TestPushRejectsWrongOriginDuringDryRun(t *testing.T) {
	root, state, _ := fixture(t)
	git(t, root, "remote", "set-url", "origin", "https://gitlab.com/acme/widget.git")
	o := options(root, state)
	o.DryRun = true
	if _, err := Push(context.Background(), o); code(err) != "repository_not_allowed" {
		t.Fatalf("wrong origin error = %v", err)
	}
	if _, err := os.Stat(state); !os.IsNotExist(err) {
		t.Fatalf("dry run wrote state: %v", err)
	}
}

func TestPushRejectsSourceSymlinkAndAdversarialDecryptOutput(t *testing.T) {
	root, state, log := fixture(t)
	if err := os.Remove(filepath.Join(root, "secrets.env")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(t.TempDir(), "outside"), filepath.Join(root, "secrets.env")); err != nil {
		t.Fatal(err)
	}
	if _, err := Push(context.Background(), options(root, state)); code(err) != "source_symlink" {
		t.Fatalf("symlink error = %v", err)
	}
	if entries, _ := os.ReadDir(log); len(entries) != 0 {
		t.Fatal("gh ran with symlink source")
	}

	root, state, log = fixture(t)
	priorLimit := maxOutput
	maxOutput = 64
	t.Cleanup(func() { maxOutput = priorLimit })
	buffer := &limitedBuffer{limit: maxOutput}
	_, _ = buffer.Write(make([]byte, 65))
	if !buffer.exceeded {
		t.Fatal("limited buffer did not reject an oversized write")
	}
	if err := os.WriteFile(filepath.Join(root, "oversize.env"), []byte("placeholder"), 0o600); err != nil {
		t.Fatal(err)
	}
	o := options(root, state)
	o.Target.Source = "oversize.env"
	if _, err := Push(context.Background(), o); code(err) != "sops_output_too_large" {
		t.Fatalf("large output error = %v", err)
	}
	if entries, _ := os.ReadDir(log); len(entries) != 0 {
		t.Fatal("gh ran with oversize source")
	}
}

func fixture(t *testing.T) (root, state, log string) {
	t.Helper()
	root, state, log = t.TempDir(), filepath.Join(canonicalTemp(t), "state"), filepath.Join(t.TempDir(), "log")
	if err := os.WriteFile(filepath.Join(root, "secrets.env"), []byte("placeholder"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(log, 0o700); err != nil {
		t.Fatal(err)
	}
	git(t, root, "init")
	git(t, root, "remote", "add", "origin", "https://github.com/acme/widget.git")
	bin := t.TempDir()
	writeExecutable(t, filepath.Join(bin, "sops"), "#!/bin/sh\nif [ -n \"${SOPS_ARGS_FILE:-}\" ]; then printf '%s' \"$*\" > \"$SOPS_ARGS_FILE\"; fi\ncase \"$*\" in *oversize*) head -c 65 /dev/zero | tr '\\000' x ;; *) printf '%s' \"${SOPS_BODY:-ONE=one}\"; printf '\\n' ;; esac\nprintf '%s' \"${SOPS_STDERR:-}\" >&2\nexit \"${SOPS_EXIT:-0}\"\n")
	writeExecutable(t, filepath.Join(bin, "gh"), "#!/bin/sh\n: \"${GH_LOG:?}\"\n: \"${GH_COUNT:=0}\"\nGH_COUNT=$((GH_COUNT + 1)); export GH_COUNT\nprintf '%s' \"$*\" > \"$GH_LOG/$GH_COUNT\"\nif [ \"${GH_FAIL:-0}\" = 1 ]; then printf '%s' \"${GH_STDERR:-}\" >&2; exit 1; fi\nexit 0\n")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("GH_LOG", log)
	return root, state, log
}

func canonicalTemp(t *testing.T) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func options(root, state string) Options {
	return Options{Root: root, Name: "production", StateDir: state, Target: Target{Source: "secrets.env", Repository: "acme/widget", Keys: map[string]string{"ONE": "ONE_DEST"}}}
}
func code(err error) string {
	if err == nil {
		return ""
	}
	if e, ok := err.(*CodeError); ok {
		return e.Code
	}
	return err.Error()
}
func readReport(t *testing.T, state string) Report {
	t.Helper()
	files, err := os.ReadDir(state)
	var receipt string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".json") {
			receipt = file.Name()
		}
	}
	if err != nil || receipt == "" {
		t.Fatalf("receipt files: %v %v", files, err)
	}
	b, err := os.ReadFile(filepath.Join(state, receipt))
	if err != nil {
		t.Fatal(err)
	}
	var r Report
	if err := json.Unmarshal(b, &r); err != nil {
		t.Fatal(err)
	}
	return r
}
func writeExecutable(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}
}
func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
}
