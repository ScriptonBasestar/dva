// Package secretpush decrypts an explicitly declared dotenv file and publishes
// selected keys as GitHub repository secrets. It never persists secret material.
package secretpush

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/ScriptonBasestar/dva/internal/remotetarget"
)

const (
	StateAccepted   = "accepted"
	StateFailed     = "failed"
	StateUnknown    = "unknown"
	StateNotStarted = "not_started"
	maxDiagnostic   = 4096
	maxKeys         = 64
	maxSecretValue  = 48 * 1024
	sopsTimeout     = 2 * time.Minute
	ghTimeout       = 30 * time.Second
)

var maxOutput = 1 << 20 // test seam; limits plaintext retained in memory.

var secretName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,255}$`)

// Target declares one encrypted dotenv source and its explicit GitHub mapping.
type Target struct {
	Source     string
	Repository string
	Keys       map[string]string // dotenv source key -> GitHub destination secret
}

type Options struct {
	Root, Name, StateDir string
	Target               Target
	DryRun               bool
}

// KeyReport deliberately contains declaration metadata only; never a value,
// value length, digest, command output, or diagnostic text.
type KeyReport struct {
	Key   string `json:"key"`
	State string `json:"state"`
}

// Report is both the returned safe receipt and the on-disk receipt format.
type Report struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Repository string      `json:"repository"`
	Keys       []KeyReport `json:"keys"`
}

// CodeError identifies failure categories without exposing tool output or secrets.
type CodeError struct{ Code string }

func (e *CodeError) Error() string { return e.Code }

func codeError(code string) error { return &CodeError{Code: code} }

// Push validates the complete plaintext before it invokes gh. A receipt is
// atomically persisted before the first remote mutation and after every result.
func Push(ctx context.Context, opts Options) (Report, error) {
	root, source, mappings, report, err := prepare(ctx, opts)
	if err != nil {
		return report, err
	}
	if opts.DryRun {
		return report, nil
	}
	if _, err := exec.LookPath("sops"); err != nil {
		return report, codeError("sops_unavailable")
	}
	if _, err := exec.LookPath("gh"); err != nil {
		return report, codeError("gh_unavailable")
	}
	plaintext, err := decrypt(ctx, root, source)
	if err != nil {
		return report, err
	}
	defer wipe(plaintext)
	values, err := parseDotenv(plaintext)
	if err != nil {
		return report, err
	}
	for sourceKey := range mappings.forward {
		if _, ok := values[sourceKey]; !ok {
			wipeValues(values)
			return report, codeError("source_key_missing")
		}
	}
	defer wipeValues(values)

	state, err := stateDirectory(opts.StateDir)
	if err != nil {
		return report, err
	}
	if err := secureMkdir(state); err != nil {
		return report, err
	}
	unlock, err := lockTarget(state, opts.Target.Repository)
	if err != nil {
		return report, err
	}
	defer unlock()
	receipt := filepath.Join(state, report.ID+".json")
	if err := writeReceipt(receipt, report); err != nil {
		return report, err
	}
	for i := range report.Keys {
		key := report.Keys[i].Key
		value := values[mappings.reverse[key]]
		// Once gh is started, an interrupted process cannot prove whether GitHub
		// accepted the value. Persist unknown before that boundary.
		report.Keys[i].State = StateUnknown
		if err := writeReceipt(receipt, report); err != nil {
			return report, err
		}
		err = setSecret(ctx, root, opts.Target.Repository, key, value)
		wipe(value)
		if err != nil {
			report.Keys[i].State = StateUnknown
			if errors.Is(err, errNotInvoked) {
				report.Keys[i].State = StateFailed
			}
			if receiptErr := writeReceipt(receipt, report); receiptErr != nil {
				return report, receiptErr
			}
			return report, codeError("secret_push_" + report.Keys[i].State)
		}
		report.Keys[i].State = StateAccepted
		if err := writeReceipt(receipt, report); err != nil {
			return report, err
		}
	}
	return report, nil
}

type mappingSet struct {
	forward map[string]string
	reverse map[string]string
}

func prepare(ctx context.Context, opts Options) (string, string, mappingSet, Report, error) {
	var report Report
	if opts.Name == "" || opts.Target.Repository == "" || opts.Target.Source == "" || len(opts.Target.Keys) == 0 || len(opts.Target.Keys) > maxKeys {
		return "", "", mappingSet{}, report, codeError("invalid_declaration")
	}
	root, err := canonicalRoot(opts.Root)
	if err != nil {
		return "", "", mappingSet{}, report, err
	}
	source, err := safeSource(root, opts.Target.Source)
	if err != nil {
		return "", "", mappingSet{}, report, err
	}
	mappings, err := validateMappings(opts.Target.Keys)
	if err != nil {
		return "", "", mappingSet{}, report, err
	}
	if err := remotetarget.Validate(ctx, root, opts.Target.Repository); err != nil {
		return "", "", mappingSet{}, report, codeError("repository_not_allowed")
	}
	id, err := receiptID()
	if err != nil {
		return "", "", mappingSet{}, report, err
	}
	keys := make([]string, 0, len(mappings.reverse))
	for destination := range mappings.reverse {
		keys = append(keys, destination)
	}
	sort.Strings(keys)
	report = Report{ID: id, Name: opts.Name, Repository: opts.Target.Repository, Keys: make([]KeyReport, len(keys))}
	for i, key := range keys {
		report.Keys[i] = KeyReport{Key: key, State: StateNotStarted}
	}
	return root, source, mappings, report, nil
}

func canonicalRoot(root string) (string, error) {
	if root == "" {
		return "", codeError("root_required")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return "", codeError("root_invalid")
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", codeError("root_invalid")
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return "", codeError("root_invalid")
	}
	return root, nil
}

func safeSource(root, source string) (string, error) {
	if filepath.IsAbs(source) || source == "." {
		return "", codeError("source_outside_root")
	}
	clean := filepath.Clean(source)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", codeError("source_outside_root")
	}
	path := filepath.Join(root, clean)
	part := root
	for item := range strings.SplitSeq(clean, string(filepath.Separator)) {
		part = filepath.Join(part, item)
		info, err := os.Lstat(part)
		if err != nil {
			return "", codeError("source_unreadable")
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", codeError("source_symlink")
		}
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return "", codeError("source_not_regular")
	}
	return path, nil
}

func validateMappings(keys map[string]string) (mappingSet, error) {
	result := mappingSet{forward: make(map[string]string, len(keys)), reverse: make(map[string]string, len(keys))}
	for source, destination := range keys {
		if !secretName.MatchString(source) || !secretName.MatchString(destination) || strings.HasPrefix(destination, "GITHUB_") {
			return mappingSet{}, codeError("invalid_secret_key")
		}
		if _, exists := result.reverse[destination]; exists {
			return mappingSet{}, codeError("duplicate_destination_key")
		}
		result.forward[source], result.reverse[destination] = destination, source
	}
	return result, nil
}

func decrypt(ctx context.Context, root, source string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, sopsTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sops", "--decrypt", "--input-type", "dotenv", "--output-type", "dotenv", source)
	cmd.WaitDelay = time.Second
	cmd.Dir = root
	cmd.Env = withoutGHRouting(os.Environ())
	stdout, stderr := &limitedBuffer{limit: maxOutput, onLimit: cancel}, &limitedBuffer{limit: maxDiagnostic}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	if err := cmd.Run(); stdout.exceeded {
		wipe(stdout.buf.Bytes())
		return nil, codeError("sops_output_too_large")
	} else if err != nil {
		wipe(stdout.buf.Bytes())
		return nil, codeError("sops_decrypt_failed")
	}
	return stdout.take(), nil
}

var errNotInvoked = errors.New("secret command not invoked")

func setSecret(ctx context.Context, root, repository, name string, value []byte) error {
	ctx, cancel := context.WithTimeout(ctx, ghTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gh", "secret", "set", name, "--repo", "github.com/"+repository)
	cmd.WaitDelay = time.Second
	cmd.Dir = root
	cmd.Env = withoutGHRouting(os.Environ())
	cmd.Stdin = bytes.NewReader(value)
	stderr := &limitedBuffer{limit: maxDiagnostic}
	cmd.Stdout, cmd.Stderr = io.Discard, stderr
	if err := cmd.Start(); err != nil {
		return errNotInvoked
	}
	if err := cmd.Wait(); err != nil {
		return errors.New("secret command failed")
	}
	return nil
}

func parseDotenv(data []byte) (map[string][]byte, error) {
	values := make(map[string][]byte)
	for raw := range bytes.SplitSeq(data, []byte{'\n'}) {
		line := bytes.TrimSpace(bytes.TrimSuffix(raw, []byte{'\r'}))
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		if len(line) > len("export") && bytes.HasPrefix(line, []byte("export")) && dotenvSpace(line[len("export")]) {
			line = bytes.TrimLeftFunc(line[len("export"):], func(r rune) bool { return r == ' ' || r == '\t' || r == '\r' || r == '\n' || r == '\f' || r == '\v' })
		}
		i := bytes.IndexByte(line, '=')
		if i < 1 || !secretName.Match(line[:i]) {
			wipeValues(values)
			return nil, codeError("invalid_dotenv")
		}
		key := string(line[:i])
		if _, exists := values[key]; exists {
			wipeValues(values)
			return nil, codeError("duplicate_source_key")
		}
		value := bytes.TrimSpace(line[i+1:])
		if len(value) == 0 || (value[0] != '\'' && value[0] != '"') {
			if comment := bytes.Index(value, []byte(" #")); comment >= 0 {
				value = bytes.TrimSpace(value[:comment])
			}
		}
		value = unquoteDotenvValue(value)
		if len(value) > maxSecretValue {
			wipe(value)
			wipeValues(values)
			return nil, codeError("invalid_dotenv_value")
		}
		values[key] = append([]byte(nil), value...)
		wipe(value)
	}
	return values, nil
}

func dotenvSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\r' || b == '\n' || b == '\f' || b == '\v'
}

// unquoteDotenvValue mirrors config.unquoteEnvValue without materializing a
// string. It intentionally does not interpolate values or consult process env.
func unquoteDotenvValue(value []byte) []byte {
	if len(value) < 2 || (value[0] != '\'' && value[0] != '"') || value[len(value)-1] != value[0] {
		return value
	}
	inner := value[1 : len(value)-1]
	if value[0] == '\'' {
		return inner
	}
	out := make([]byte, 0, len(inner))
	for i := 0; i < len(inner); i++ {
		if inner[i] == '\\' && i+1 < len(inner) {
			switch inner[i+1] {
			case '\\':
				out = append(out, '\\')
				i++
				continue
			case 'n':
				out = append(out, '\n')
				i++
				continue
			case 't':
				out = append(out, '\t')
				i++
				continue
			case 'r':
				out = append(out, '\r')
				i++
				continue
			case '"':
				out = append(out, '"')
				i++
				continue
			}
		}
		out = append(out, inner[i])
	}
	return out
}

func stateDirectory(dir string) (string, error) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		return "", codeError("state_platform_unsupported")
	}
	if dir != "" {
		resolved, err := filepath.Abs(dir)
		if err != nil {
			return "", codeError("state_directory_unavailable")
		}
		return resolved, nil
	}
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		resolved, err := filepath.Abs(filepath.Join(xdg, "dva", "secrets"))
		if err != nil {
			return "", codeError("state_directory_unavailable")
		}
		return resolved, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", codeError("state_directory_unavailable")
	}
	return filepath.Join(home, ".local", "state", "dva", "secrets"), nil
}

func secureMkdir(dir string) error {
	volume := filepath.VolumeName(dir)
	current := volume + string(filepath.Separator)
	for part := range strings.SplitSeq(strings.TrimPrefix(dir, current), string(filepath.Separator)) {
		if part == "" {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			if err := os.Mkdir(current, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
				return codeError("state_directory_failed")
			}
			info, err = os.Lstat(current)
		}
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o022 != 0 {
			return codeError("state_directory_failed")
		}
	}
	info, err := os.Lstat(dir)
	if err != nil || info.Mode().Perm() != 0o700 {
		return codeError("state_directory_failed")
	}
	return nil
}

func writeReceipt(path string, report Report) error {
	b, err := json.Marshal(report)
	if err != nil {
		return codeError("receipt_write_failed")
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".secret-receipt-*")
	if err != nil {
		return codeError("receipt_write_failed")
	}
	name := tmp.Name()
	defer func() { _ = os.Remove(name) }()
	if _, err = tmp.Write(b); err == nil {
		err = tmp.Chmod(0o600)
	}
	if err == nil {
		err = tmp.Sync()
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return codeError("receipt_write_failed")
	}
	if err := os.Rename(name, path); err != nil {
		return codeError("receipt_write_failed")
	}
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return codeError("receipt_write_failed")
	}
	err = dir.Sync()
	closeErr := dir.Close()
	if err != nil || closeErr != nil {
		return codeError("receipt_write_failed")
	}
	return nil
}

func receiptID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", codeError("receipt_id_failed")
	}
	return hex.EncodeToString(b), nil
}

type limitedBuffer struct {
	buf      bytes.Buffer
	limit    int
	size     int
	exceeded bool
	onLimit  func()
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	b.size += len(p)
	if b.size > b.limit {
		b.exceeded = true
		if b.onLimit != nil {
			b.onLimit()
			b.onLimit = nil
		}
		remaining := b.limit - b.buf.Len()
		if remaining > 0 {
			_, _ = b.buf.Write(p[:remaining])
		}
		return len(p), nil
	}
	return b.buf.Write(p)
}
func (b *limitedBuffer) take() []byte {
	out := append([]byte(nil), b.buf.Bytes()...)
	b.buf.Reset()
	return out
}
func withoutGHRouting(env []string) []string {
	out := make([]string, 0, len(env))
	for _, item := range env {
		if !strings.HasPrefix(item, "GH_DEBUG=") && !strings.HasPrefix(item, "GH_HOST=") && !strings.HasPrefix(item, "GH_REPO=") {
			out = append(out, item)
		}
	}
	return out
}

func wipe(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
func wipeValues(values map[string][]byte) {
	for key, value := range values {
		wipe(value)
		delete(values, key)
	}
}
