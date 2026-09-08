package jobrun

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"github.com/ScriptonBasestar/dva/internal/ociverify"
)

type receipt struct {
	Version    int               `json:"version"`
	ID         string            `json:"id"`
	Root       string            `json:"root"`
	Name       string            `json:"name"`
	Definition Definition        `json:"definition"`
	Resolved   Definition        `json:"resolved"`
	Inputs     map[string]string `json:"inputs"`
	Repository string            `json:"repository"`
	Ref        string            `json:"ref"`
	HeadSHA    string            `json:"head_sha"`
	Runs       []receiptRun      `json:"runs"`
}

const maxReceiptBytes = 1 << 20

type receiptRun struct {
	Name           string             `json:"name"`
	Workflow       string             `json:"workflow"`
	WorkflowID     string             `json:"workflow_id"`
	Inputs         map[string]string  `json:"inputs"`
	ResultArtifact string             `json:"result_artifact"`
	Images         []Image            `json:"images"`
	RunID          int64              `json:"run_id"`
	RunAttempt     int                `json:"run_attempt"`
	URL            string             `json:"url"`
	DispatchState  string             `json:"dispatch_state"`
	Status         string             `json:"status"`
	Conclusion     string             `json:"conclusion"`
	Verified       bool               `json:"verified"`
	Verification   []ociverify.Result `json:"verification,omitempty"`
}

func defaultStateDir() (string, error) {
	if x := os.Getenv("XDG_STATE_HOME"); x != "" {
		return filepath.Join(x, "dva", "jobs"), nil
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, ".local", "state", "dva", "jobs"), nil
}
func statePath(dir, id string) (string, error) {
	if !validID(id) {
		return "", fmt.Errorf("invalid job receipt id")
	}
	if dir == "" {
		var err error
		dir, err = defaultStateDir()
		if err != nil {
			return "", err
		}
	}
	clean := filepath.Clean(dir)
	if clean == "." || clean == string(filepath.Separator) {
		return "", fmt.Errorf("unsafe job state directory")
	}
	return filepath.Join(clean, id+".json"), nil
}
func validID(id string) bool {
	if len(id) != 32 {
		return false
	}
	for _, c := range id {
		if !strings.ContainsRune("0123456789abcdef", c) {
			return false
		}
	}
	return true
}
func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func readReceipt(dir, id string) (receipt, string, error) {
	p, err := statePath(dir, id)
	if err != nil {
		return receipt{}, "", err
	}
	fi, err := os.Lstat(p)
	if err != nil {
		return receipt{}, "", err
	}
	if err := rejectSymlinkAncestors(filepath.Dir(p)); err != nil {
		return receipt{}, "", err
	}
	if parent, err := os.Lstat(filepath.Dir(p)); err != nil || parent.Mode().Perm()&0o022 != 0 {
		return receipt{}, "", fmt.Errorf("unsafe job state directory")
	}
	if fi.Mode()&fs.ModeSymlink != 0 || !fi.Mode().IsRegular() || fi.Mode().Perm()&0o077 != 0 {
		return receipt{}, "", fmt.Errorf("unsafe job receipt path")
	}
	f, err := os.Open(p)
	if err != nil {
		return receipt{}, "", err
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, maxReceiptBytes+1))
	if err != nil {
		return receipt{}, "", err
	}
	if len(data) > maxReceiptBytes {
		return receipt{}, "", fmt.Errorf("job receipt exceeds size limit")
	}
	var r receipt
	if err = json.Unmarshal(data, &r); err != nil {
		return receipt{}, "", fmt.Errorf("invalid job receipt: %w", err)
	}
	if r.ID != id || r.Version != 1 {
		return receipt{}, "", fmt.Errorf("invalid job receipt identity")
	}
	if err := validateReceipt(r); err != nil {
		return receipt{}, "", err
	}
	return r, p, nil
}
func writeReceipt(path string, r receipt) error {
	parent := filepath.Dir(path)
	if err := os.MkdirAll(parent, 0700); err != nil {
		return err
	}
	if err := rejectSymlinkAncestors(parent); err != nil {
		return err
	}
	if fi, err := os.Lstat(parent); err != nil || fi.Mode()&fs.ModeSymlink != 0 || fi.Mode().Perm()&0o022 != 0 {
		return fmt.Errorf("unsafe job state directory")
	}
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	if len(data) > maxReceiptBytes {
		return fmt.Errorf("job receipt exceeds size limit")
	}
	f, err := os.CreateTemp(parent, ".job-")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer func() { _ = os.Remove(tmp) }()
	err = f.Chmod(0600)
	if err == nil {
		_, err = f.Write(data)
	}
	if err == nil {
		err = f.Sync()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	d, err := os.Open(parent)
	if err != nil {
		return err
	}
	defer func() { _ = d.Close() }()
	return d.Sync()
}
func rejectSymlinkAncestors(dir string) error {
	clean, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	for {
		fi, err := os.Lstat(clean)
		if err != nil {
			return err
		}
		if fi.Mode()&fs.ModeSymlink != 0 && !allowedSystemSymlink(clean) {
			return fmt.Errorf("unsafe job state directory")
		}
		next := filepath.Dir(clean)
		if next == clean {
			return nil
		}
		clean = next
	}
}
func allowedSystemSymlink(path string) bool {
	if runtime.GOOS != "darwin" || path != "/var" {
		return false
	}
	target, err := filepath.EvalSymlinks(path)
	return err == nil && target == "/private/var"
}
func validateReceipt(r receipt) error {
	if r.Root == "" || r.Name == "" || !validRepository(r.Repository) || r.Ref == "" || r.HeadSHA == "" {
		return fmt.Errorf("invalid job receipt fields")
	}
	resolved, err := Resolve(r.Definition, r.Inputs)
	if err != nil {
		return fmt.Errorf("invalid job receipt definition: %w", err)
	}
	if !reflect.DeepEqual(resolved, r.Resolved) || resolved.Repository != r.Repository || resolved.Ref != r.Ref || len(r.Runs) != len(resolved.Runs) {
		return fmt.Errorf("invalid job receipt resolution")
	}
	for i, x := range r.Runs {
		if x.Name != resolved.Runs[i].Name || x.Workflow != resolved.Runs[i].Workflow || !reflect.DeepEqual(x.Inputs, resolved.Runs[i].Inputs) || !reflect.DeepEqual(x.Images, resolved.Runs[i].Images) || x.ResultArtifact != resolved.Runs[i].ResultArtifact || (!strings.HasSuffix(x.Workflow, ".yml") && !strings.HasSuffix(x.Workflow, ".yaml")) || x.RunID < 0 {
			return fmt.Errorf("invalid job receipt run")
		}
		if x.WorkflowID != "" {
			if _, err := strconv.ParseInt(x.WorkflowID, 10, 64); err != nil {
				return fmt.Errorf("invalid job receipt workflow id")
			}
		}
		if x.DispatchState == "dispatched" && (x.RunID <= 0 || x.WorkflowID == "" || x.RunAttempt != 1) {
			return fmt.Errorf("invalid dispatched job receipt run")
		}
		if x.DispatchState != "not_started" && x.DispatchState != "unknown_dispatch" && x.DispatchState != "dispatched" {
			return fmt.Errorf("invalid job receipt dispatch state")
		}
	}
	return nil
}
func withReceiptLock(path string, fn func() error) error {
	lockPath := path + ".lock"
	if fi, err := os.Lstat(lockPath); err == nil && fi.Mode()&fs.ModeSymlink != 0 {
		return fmt.Errorf("unsafe job receipt lock path")
	}
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	return lockAndRun(f, fn)
}
func report(r receipt) Report {
	out := Report{ID: r.ID, Name: r.Name, Repository: r.Repository, Ref: r.Ref, HeadSHA: r.HeadSHA, Runs: make([]RunReport, len(r.Runs))}
	for i, x := range r.Runs {
		out.Runs[i] = RunReport{Name: x.Name, Workflow: x.Workflow, RunID: x.RunID, URL: x.URL, Status: x.Status, Conclusion: x.Conclusion, JobSucceeded: x.Conclusion == "success", Verified: x.Verified, DispatchState: x.DispatchState, Images: append([]ociverify.Result(nil), x.Verification...)}
	}
	return out
}
