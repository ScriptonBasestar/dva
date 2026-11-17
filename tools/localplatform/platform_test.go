package localplatform

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func fixture(t *testing.T) (string, string) {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	scripts := filepath.Join(root, "scripts")
	if err := os.MkdirAll(scripts, 0o700); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join("..", "..", "scripts", "local-macos-platform.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scripts, "local-macos-platform.sh"), content, 0o700); err != nil {
		t.Fatal(err)
	}
	installer, err := os.ReadFile(filepath.Join("..", "..", "scripts", "ci-install-sops-age.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scripts, "ci-install-sops-age.sh"), installer, 0o700); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(root, "fake-bin")
	if err := os.Mkdir(bin, 0o700); err != nil {
		t.Fatal(err)
	}
	writeExecutable(t, filepath.Join(bin, "uname"), "#!/bin/sh\ncase \"$1\" in -s) echo Darwin;; -m) echo arm64;; esac\n")
	writeExecutable(t, filepath.Join(bin, "git"), "#!/bin/sh\nprintf '%s\\n' \"$DVA_TEST_ROOT\"\n")
	writeExecutable(t, filepath.Join(bin, "go"), "#!/bin/sh\nprintf '%s|%s\\n' \"${DVA_INTEGRATION_TOOL_DIR-}\" \"$*\" >> \"$DVA_TEST_LOG\"\n")
	for _, name := range []string{"curl", "gh", "cosign"} {
		writeExecutable(t, filepath.Join(bin, name), "#!/bin/sh\necho network >> \"$DVA_TEST_LOG\"\nexit 99\n")
	}
	for _, slot := range []string{"floor", "latest"} {
		version := "3.13.1"
		if slot == "latest" {
			version = "3.14.0"
		}
		writeSlot(t, root, slot, version, version)
	}
	return root, bin
}

func TestLocalInstallerRejectsUnsafeDestinations(t *testing.T) {
	root, bin := fixture(t)
	destination := filepath.Join(root, "tmp", "installer", "bin")
	workdir := filepath.Join(root, "tmp", "installer", "downloads")
	for _, dir := range []string{destination, workdir} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	run := func(dest string) ([]byte, error) {
		cmd := exec.Command("bash", filepath.Join(root, "scripts", "ci-install-sops-age.sh"), "3.13.1", "--local", dest, workdir)
		cmd.Dir = root
		cmd.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "DVA_TEST_ROOT="+root, "RUNNER_OS=unsupported", "RUNNER_ARCH=unsupported", "DVA_TEST_LOG="+filepath.Join(root, "go-calls"))
		return cmd.CombinedOutput()
	}
	outside := t.TempDir()
	if out, err := run(outside); err == nil || !strings.Contains(string(out), "destination must be inside project tmp") {
		t.Fatalf("outside destination: %s %v", out, err)
	}
	marker := filepath.Join(destination, "keep")
	if err := os.WriteFile(marker, []byte("preserve"), 0o600); err != nil {
		t.Fatal(err)
	}
	if out, err := run(destination); err == nil || !strings.Contains(string(out), "destination is not empty") {
		t.Fatalf("nonempty destination: %s %v", out, err)
	}
	if got, err := os.ReadFile(marker); err != nil || string(got) != "preserve" {
		t.Fatalf("existing file changed: %q %v", got, err)
	}
	if err := os.Remove(marker); err != nil {
		t.Fatal(err)
	}
	if out, err := run(destination); err == nil || !strings.Contains(string(out), "no sops/age assets declared") {
		t.Fatalf("valid local layout did not reach asset check: %s %v", out, err)
	}
	if _, err := os.Stat(filepath.Join(root, "go-calls")); !os.IsNotExist(err) {
		t.Fatal("failed installer preflight executed network command")
	}
}

func TestInstallerCosignCacheLocation(t *testing.T) {
	root, bin := fixture(t)
	destination := filepath.Join(root, "tmp", "installer", "bin")
	workdir := filepath.Join(root, "tmp", "installer", "downloads")
	for _, dir := range []string{destination, workdir} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	writeExecutable(t, filepath.Join(bin, "gh"), "#!/bin/sh\necho v3.14.0\n")
	writeExecutable(t, filepath.Join(bin, "curl"), "#!/bin/sh\nexit 0\n")
	writeExecutable(t, filepath.Join(bin, "cosign"), "#!/bin/sh\nprintf 'cache=%s\\n' \"${TUF_ROOT-unset}\"\nexit 77\n")
	for _, tc := range []struct {
		name, want string
		args       []string
	}{
		{name: "local", want: filepath.Join(workdir, "tuf"), args: []string{"latest", "--local", destination, workdir}},
		{name: "github", want: "inherited-cache", args: []string{"latest"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command("bash", append([]string{filepath.Join(root, "scripts", "ci-install-sops-age.sh")}, tc.args...)...)
			cmd.Dir = root
			cmd.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "DVA_TEST_ROOT="+root, "RUNNER_OS=macOS", "RUNNER_ARCH=ARM64", "RUNNER_TEMP="+workdir, "HOME="+root, "TUF_ROOT=inherited-cache")
			out, err := cmd.CombinedOutput()
			if err == nil || !strings.Contains(string(out), "cache="+tc.want+"\n") {
				t.Fatalf("cosign cache: %s %v", out, err)
			}
		})
	}
}

func writeExecutable(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o700); err != nil {
		t.Fatal(err)
	}
}

func writeSlot(t *testing.T, root, slot, receiptVersion, reportedVersion string) {
	t.Helper()
	dir := filepath.Join(root, "tmp", "macos-platform", slot, "bin")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	tools := map[string]string{
		"sops":       "#!/bin/sh\nprintf 'sops " + reportedVersion + "\\n'\n",
		"age":        "#!/bin/sh\nprintf 'v1.3.1\\n'\n",
		"age-keygen": "#!/bin/sh\nprintf 'v1.3.1\\n'\n",
	}
	receipt := "sops_version " + receiptVersion + "\n"
	for _, name := range []string{"sops", "age", "age-keygen"} {
		content := tools[name]
		writeExecutable(t, filepath.Join(dir, name), content)
		digest := sha256.Sum256([]byte(content))
		receipt += fmt.Sprintf("%s %x\n", name, digest)
	}
	if err := os.WriteFile(filepath.Join(dir, "receipt"), []byte(receipt), 0o600); err != nil {
		t.Fatal(err)
	}
}

func runScript(root, bin string, args ...string) ([]byte, error) {
	cmd := exec.Command("bash", append([]string{filepath.Join(root, "scripts", "local-macos-platform.sh")}, args...)...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "DVA_TEST_LOG="+filepath.Join(root, "go-calls"))
	return cmd.CombinedOutput()
}

func TestLocalPlatformPreflightAndFailClosed(t *testing.T) {
	root, bin := fixture(t)
	if out, err := runScript(root, bin, "preflight"); err != nil {
		t.Fatalf("valid preflight: %s %v", out, err)
	}
	path := filepath.Join(root, "tmp", "macos-platform", "floor", "bin", "sops")
	writeExecutable(t, path, "#!/bin/sh\necho altered\n")
	if out, err := runScript(root, bin, "integration", "floor"); err == nil || !strings.Contains(string(out), "digest mismatch") {
		t.Fatalf("modified binary: %s %v", out, err)
	}
	if _, err := os.Stat(filepath.Join(root, "go-calls")); !os.IsNotExist(err) {
		t.Fatal("failed preflight ran integration")
	}
	writeSlot(t, root, "floor", "3.13.1", "9.9.9")
	if out, err := runScript(root, bin, "preflight"); err == nil || !strings.Contains(string(out), "unexpected sops version") {
		t.Fatalf("wrong executable version: %s %v", out, err)
	}
	if err := os.Remove(filepath.Join(root, "tmp", "macos-platform", "latest", "bin", "receipt")); err != nil {
		t.Fatal(err)
	}
	if out, err := runScript(root, bin, "integration", "latest"); err == nil || !strings.Contains(string(out), "missing receipt") {
		t.Fatalf("missing slot: %s %v", out, err)
	}
}

func TestLocalPlatformRunsExactChecks(t *testing.T) {
	root, bin := fixture(t)
	for _, args := range [][]string{{"unit"}, {"integration", "floor"}, {"integration", "latest"}} {
		if out, err := runScript(root, bin, args...); err != nil {
			t.Fatalf("%v: %s %v", args, out, err)
		}
	}
	data, err := os.ReadFile(filepath.Join(root, "go-calls"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 3 || lines[0] != "|test -timeout 4m ./internal/cli ./internal/config -count=1" {
		t.Fatalf("unexpected calls: %q", data)
	}
	for i, slot := range []string{"floor", "latest"} {
		want := filepath.Join(root, "tmp", "macos-platform", slot, "bin") + "|test -json -tags=integration -race -timeout 4m -count=1 ./internal/integration/..."
		if lines[i+1] != want {
			t.Fatalf("got %q, want %q", lines[i+1], want)
		}
	}
}
