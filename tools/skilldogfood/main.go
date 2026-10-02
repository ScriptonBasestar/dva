// Command skilldogfood verifies a selected SHA-pinned DVA executable's skill
// installer without AI runtimes.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"
)

const allSkillRuntimes = "claude-code,codex,opencode,grok,antigravity,agent-mesh"
const commandStderrLimit = 8 * 1024

var runtimeDestinations = map[string][]string{
	".agent-mesh/skills/dva": {"agent-mesh"},
	".agents/skills":         {"antigravity", "codex"},
	".claude/skills":         {"claude-code"},
	".grok/skills":           {"grok"},
	".opencode/skills":       {"opencode"},
}

type commandResult struct {
	Operation string              `json:"operation"`
	DryRun    bool                `json:"dry_run"`
	Scope     string              `json:"scope"`
	Results   []destinationResult `json:"results"`
}

type destinationResult struct {
	Destination     string          `json:"destination"`
	Runtimes        []string        `json:"runtimes"`
	Status          string          `json:"status"`
	RuntimeStatuses []runtimeStatus `json:"runtime_statuses"`
	TakeoverBackup  string          `json:"takeover_backup"`
	BackupStatus    string          `json:"backup_status"`
}

type runtimeStatus struct {
	Runtime string `json:"runtime"`
	Status  string `json:"status"`
}
type receiptRecord struct {
	Schema       int        `json:"schema"`
	Installation string     `json:"installation"`
	Format       string     `json:"format"`
	Scope        string     `json:"scope"`
	Destination  string     `json:"destination"`
	Runtimes     []string   `json:"runtimes"`
	Version      string     `json:"version"`
	BundleSHA    string     `json:"bundle_sha256"`
	Files        []fileHash `json:"files"`
}
type fileHash struct {
	Path string `json:"path"`
	SHA  string `json:"sha256"`
}
type claimRecord struct {
	Schema       int        `json:"schema"`
	Name         string     `json:"name"`
	Kind         string     `json:"kind"`
	State        string     `json:"state"`
	OperationID  string     `json:"operation_id"`
	Generation   uint64     `json:"generation"`
	Destination  string     `json:"destination"`
	Producer     string     `json:"producer"`
	Format       string     `json:"format"`
	Scope        string     `json:"scope"`
	Consumers    []string   `json:"consumers"`
	SourceDigest string     `json:"source_digest"`
	Files        []fileHash `json:"files"`
}

// treeEntry records the runtime-path facts that a dry-run must preserve.
type treeEntry struct {
	Path       string
	Type       string
	Mode       fs.FileMode
	ContentSHA string
	LinkTarget string
	ModTime    time.Time
}

// gitTreeState captures content that porcelain status alone cannot distinguish.
// Runtime destinations are snapshotted separately because they are commonly ignored.
type gitTreeState struct {
	Status           string
	WorktreeDiffSHA  string
	IndexDiffSHA     string
	UntrackedEntries []treeEntry
}

type invocation struct {
	binary string
	env    []string
}

func main() {
	var binary, expectedSHA, flowRoot string
	flags := flag.NewFlagSet("skilldogfood", flag.ExitOnError)
	flags.StringVar(&binary, "dva-bin", "", "absolute path to the selected dva executable")
	flags.StringVar(&expectedSHA, "expected-sha256", os.Getenv("DVA_SHA256"), "independently recorded SHA-256 of the selected dva executable")
	flags.StringVar(&flowRoot, "flow-root", "", "absolute path to a flow Git repository root whose state will remain stable")
	if err := flags.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if err := run(binary, expectedSHA, flowRoot, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: skill installer dogfood failed: %v\n", err)
		os.Exit(1)
	}
}

func run(binaryArg, expectedSHA, flowArg string, out io.Writer) (err error) {
	binary, err := executableFile(binaryArg)
	if err != nil {
		return fmt.Errorf("DVA_BIN: %w", err)
	}
	flowRoot, err := gitRoot(flowArg)
	if err != nil {
		return fmt.Errorf("FLOW_ROOT: %w", err)
	}

	if err := validSHA256(expectedSHA); err != nil {
		return fmt.Errorf("DVA_SHA256: %w", err)
	}
	executed, sha, cleanup, err := immutableExecutableCopy(binary, expectedSHA)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, cleanup()) }()
	base := invocation{binary: executed, env: os.Environ()}
	version, err := base.output("version")
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "DVA binary (original): %s\nDVA SHA-256 (executed immutable copy): %s\ndva version:\n%s", binary, sha, version); err != nil {
		return err
	}
	if !strings.HasSuffix(version, "\n") {
		if _, err := fmt.Fprintln(out); err != nil {
			return err
		}
	}

	if err := verifyFlowDryRun(base, flowRoot); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "flow dry-run: unchanged %s\n", flowRoot); err != nil {
		return err
	}

	fixture, err := os.MkdirTemp("", "dva-skill-dogfood-")
	if err != nil {
		return fmt.Errorf("create fixture: %w", err)
	}
	defer func() {
		err = errors.Join(err, removeAll("clean fixture "+fixture, fixture))
	}()
	fixture, err = filepath.EvalSymlinks(fixture)
	if err != nil {
		return fmt.Errorf("resolve fixture path: %w", err)
	}

	project := filepath.Join(fixture, "project")
	if err := os.MkdirAll(project, 0o755); err != nil {
		return fmt.Errorf("create fixture project: %w", err)
	}
	fixtureEnv := withEnvironment(os.Environ(), map[string]string{
		"HOME":            filepath.Join(fixture, "home"),
		"XDG_STATE_HOME":  filepath.Join(fixture, "state"),
		"XDG_CONFIG_HOME": filepath.Join(fixture, "config"),
		"XDG_DATA_HOME":   filepath.Join(fixture, "data"),
		"XDG_CACHE_HOME":  filepath.Join(fixture, "cache"),
	})
	isolated := invocation{binary: executed, env: fixtureEnv}
	if err := verifyFixtureRoundTrip(isolated, project, filepath.Join(fixture, "state", "dva")); err != nil {
		return err
	}
	if err := verifyTakeoverLifecycle(isolated, filepath.Join(project, "takeover")); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out, "real_target_dry_run: passed"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out, "fixture_round_trip: passed"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out, "shared_runtime_unlink: passed"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out, "takeover_lifecycle: passed"); err != nil {
		return err
	}
	return nil
}

func validSHA256(value string) error {
	if len(value) != sha256.Size*2 {
		return fmt.Errorf("must be a %d-character SHA-256 hex string", sha256.Size*2)
	}
	if _, err := hex.DecodeString(value); err != nil {
		return fmt.Errorf("must be hexadecimal: %w", err)
	}
	return nil
}

func (inv invocation) output(args ...string) (string, error) {
	return commandOutput(inv.env, inv.binary, args...)
}

func (inv invocation) json(directory string, args ...string) (commandResult, error) {
	args = append([]string{"--json"}, args...)
	output, err := commandOutputInDir(inv.env, directory, inv.binary, args...)
	if err != nil {
		return commandResult{}, err
	}
	var result commandResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		return commandResult{}, fmt.Errorf("decode JSON response %q: %w", output, err)
	}
	return result, nil
}

func commandOutput(env []string, command string, args ...string) (string, error) {
	return commandOutputInDir(env, "", command, args...)
}

func commandOutputInDir(env []string, directory, command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	cmd.Dir = directory
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %w\n%s", strings.Join(append([]string{command}, args...), " "), err, output)
	}
	return string(output), nil
}

func requireEmptyDirectory(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	if len(entries) != 0 {
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		sort.Strings(names)
		return fmt.Errorf("contains %s", strings.Join(names, ", "))
	}
	return nil
}

func requireEmptyOrMissingDirectory(path string) error {
	err := requireEmptyDirectory(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func fileSHA256(path string) (digest string, err error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func withEnvironment(base []string, replacements map[string]string) []string {
	keys := make([]string, 0, len(replacements))
	for key := range replacements {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]string, 0, len(base)+len(keys))
	for _, entry := range base {
		key, _, found := strings.Cut(entry, "=")
		if !found {
			result = append(result, entry)
			continue
		}
		if _, replaced := replacements[key]; !replaced {
			result = append(result, entry)
		}
	}
	for _, key := range keys {
		result = append(result, key+"="+replacements[key])
	}
	return result
}

func requireEnvelope(result commandResult, operation string, dryRun bool) error {
	if result.Operation != operation || result.DryRun != dryRun || result.Scope != "project" {
		return fmt.Errorf("got operation=%q dry_run=%t scope=%q", result.Operation, result.DryRun, result.Scope)
	}
	return nil
}

func requireDestinations(project string, result commandResult, status string) error {
	expected := map[string]string{
		".agent-mesh/skills/dva": "", ".agents/skills": "", ".claude/skills": "", ".grok/skills": "", ".opencode/skills": "",
	}
	for suffix := range expected {
		expected[suffix] = status
	}
	return requireDestinationStatusSet(project, result, expected)
}

func requireDestinationStatusSet(project string, result commandResult, expected map[string]string) error {
	if len(result.Results) != len(expected) {
		return fmt.Errorf("got %d destinations, want %d", len(result.Results), len(expected))
	}
	seen := make(map[string]bool, len(expected))
	for _, entry := range result.Results {
		suffix, ok := destinationSuffix(entry.Destination)
		if !ok {
			return fmt.Errorf("unexpected destination %q", entry.Destination)
		}
		if entry.Destination != filepath.Join(project, filepath.FromSlash(suffix)) {
			return fmt.Errorf("destination %q is not exact project path", entry.Destination)
		}
		want, ok := expected[suffix]
		if !ok {
			return fmt.Errorf("unexpected destination suffix %q", suffix)
		}
		if seen[suffix] {
			return fmt.Errorf("duplicate destination suffix %q", suffix)
		}
		seen[suffix] = true
		if entry.Status != want {
			return fmt.Errorf("%s has status %q, want %q", suffix, entry.Status, want)
		}
		wantRuntimes := runtimeDestinations[suffix]
		if !sameStrings(entry.Runtimes, wantRuntimes) {
			return fmt.Errorf("%s has runtimes=%v statuses=%d, want runtimes=%v", suffix, entry.Runtimes, len(entry.RuntimeStatuses), wantRuntimes)
		}
		wantStatuses := make(map[string]string, len(wantRuntimes))
		for _, runtime := range wantRuntimes {
			wantStatuses[runtime] = entry.Status
		}
		if result.Operation == "uninstall" && suffix == ".agents/skills" && entry.Status == "uninstalled" {
			wantStatuses["codex"] = "not-installed"
		}
		if err := requireRuntimeStatuses(entry, wantStatuses); err != nil {
			return err
		}
	}
	return nil
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	left, right = append([]string(nil), left...), append([]string(nil), right...)
	sort.Strings(left)
	sort.Strings(right)
	return reflect.DeepEqual(left, right)
}

func requireOnlyDestination(project string, result commandResult, suffix, status string) error {
	if len(result.Results) != 1 {
		return fmt.Errorf("got %d destinations, want 1", len(result.Results))
	}
	entry := result.Results[0]
	actual, ok := destinationSuffix(entry.Destination)
	if !ok || actual != suffix {
		return fmt.Errorf("got destination %q, want suffix %q", entry.Destination, suffix)
	}
	if entry.Destination != filepath.Join(project, filepath.FromSlash(suffix)) {
		return fmt.Errorf("destination %q is not exact project path", entry.Destination)
	}
	if entry.Status != status {
		return fmt.Errorf("%s has status %q, want %q", suffix, entry.Status, status)
	}
	return nil
}

func destinationSuffix(path string) (string, bool) {
	for _, suffix := range []string{".agent-mesh/skills/dva", ".agents/skills", ".claude/skills", ".grok/skills", ".opencode/skills"} {
		if strings.HasSuffix(filepath.ToSlash(path), suffix) {
			return suffix, true
		}
	}
	return "", false
}

func receiptFormatForSuffix(suffix string) string {
	if suffix == ".agent-mesh/skills/dva" {
		return "agent-mesh-flat-markdown"
	}
	return "agent-skills-directory"
}

func requireRuntimeStatuses(entry destinationResult, expected map[string]string) error {
	if len(entry.RuntimeStatuses) != len(expected) {
		return fmt.Errorf("%s has %d runtime statuses, want %d", entry.Destination, len(entry.RuntimeStatuses), len(expected))
	}
	actual := make(map[string]string, len(entry.RuntimeStatuses))
	for _, status := range entry.RuntimeStatuses {
		if _, duplicate := actual[status.Runtime]; duplicate {
			return fmt.Errorf("%s repeats runtime status %q", entry.Destination, status.Runtime)
		}
		actual[status.Runtime] = status.Status
	}
	for runtime, want := range expected {
		if actual[runtime] != want {
			return fmt.Errorf("%s runtime %q has status %q, want %q", entry.Destination, runtime, actual[runtime], want)
		}
	}
	return nil
}
