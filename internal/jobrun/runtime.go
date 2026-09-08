package jobrun

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"path"
	"strings"
	"time"

	"github.com/ScriptonBasestar/dva/internal/ociverify"
	"github.com/ScriptonBasestar/dva/internal/remotetarget"
)

const maxArtifactBytes int64 = 16 << 20

var verifyOCI = ociverify.Verify

func Run(ctx context.Context, opts Options) (Report, error) {
	if !lockingSupported() {
		return Report{}, errors.New("dva jobs are unsupported on this operating system")
	}
	if opts.Name == "" || opts.Root == "" {
		return Report{}, errors.New("job name and root are required")
	}
	resolved, err := Resolve(opts.Definition, opts.Inputs)
	if err != nil {
		return Report{}, err
	}
	timeout, _ := time.ParseDuration(resolved.Timeout)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := remotetarget.Validate(ctx, opts.Root, resolved.Repository); err != nil {
		return Report{}, err
	}
	if opts.Verify && !hasImages(resolved) {
		return Report{}, errors.New("job verification requires at least one declared image")
	}
	if opts.DryRun {
		return report(receipt{Name: opts.Name, Definition: opts.Definition, Resolved: resolved, Repository: resolved.Repository, Ref: resolved.Ref, Runs: receiptRuns(resolved)}), nil
	}
	p := opts.Provider
	if p == nil {
		p = GitHubProvider{}
	}
	head, err := p.HeadSHA(ctx, resolved.Repository, resolved.Ref)
	if err != nil {
		return Report{}, err
	}
	id, err := newID()
	if err != nil {
		return Report{}, err
	}
	sp, err := statePath(opts.StateDir, id)
	if err != nil {
		return Report{}, err
	}
	r := receipt{Version: 1, ID: id, Root: opts.Root, Name: opts.Name, Definition: opts.Definition, Resolved: resolved, Inputs: copyMap(opts.Inputs), Repository: resolved.Repository, Ref: resolved.Ref, HeadSHA: head, Runs: receiptRuns(resolved)}
	for i := range r.Runs {
		workflowID, err := p.ResolveWorkflow(ctx, r.Repository, r.Runs[i].Workflow)
		if err != nil {
			return report(r), err
		}
		r.Runs[i].WorkflowID = workflowID
	}
	if err := writeReceipt(sp, r); err != nil {
		return Report{}, err
	}
	if opts.BeforeDispatch != nil {
		if err := opts.BeforeDispatch(ctx); err != nil {
			return report(r), err
		}
	}
	for i := range r.Runs {
		if err := withReceiptLock(sp, func() error {
			latest, _, err := readReceipt(opts.StateDir, id)
			if err != nil {
				return err
			}
			r = latest
			r.Runs[i].DispatchState = "unknown_dispatch"
			return writeReceipt(sp, r)
		}); err != nil {
			return report(r), err
		}
		res, err := p.Dispatch(ctx, DispatchRequest{Repository: r.Repository, Workflow: r.Runs[i].WorkflowID, Ref: r.Ref, Inputs: r.Runs[i].Inputs})
		if err != nil {
			return report(r), fmt.Errorf("dispatch job run %q: %w", r.Runs[i].Name, err)
		}
		if err := withReceiptLock(sp, func() error {
			latest, _, err := readReceipt(opts.StateDir, id)
			if err != nil {
				return err
			}
			r = latest
			r.Runs[i].RunID = res.RunID
			r.Runs[i].RunAttempt = 1
			r.Runs[i].URL = res.HTMLURL
			if r.Runs[i].URL == "" {
				r.Runs[i].URL = res.RunURL
			}
			r.Runs[i].DispatchState = "dispatched"
			return writeReceipt(sp, r)
		}); err != nil {
			return report(r), err
		}
	}
	if opts.Wait || opts.Verify {
		return poll(ctx, opts.StateDir, id, opts.Verify, p, resolved.Timeout)
	}
	return report(r), nil
}
func Status(ctx context.Context, stateDir, id string) (Report, error) {
	return refresh(ctx, stateDir, id, false, GitHubProvider{})
}
func Resume(ctx context.Context, stateDir, id string, verify bool) (Report, error) {
	r, _, err := readReceipt(stateDir, id)
	if err != nil {
		return Report{}, err
	}
	if verify && !hasImages(r.Resolved) {
		return Report{}, errors.New("job verification requires at least one declared image")
	}
	return poll(ctx, stateDir, id, verify, GitHubProvider{}, r.Resolved.Timeout)
}

func poll(ctx context.Context, stateDir, id string, verify bool, p Provider, timeout string) (Report, error) {
	d, err := time.ParseDuration(timeout)
	if err != nil || d <= 0 {
		return Report{}, errors.New("job timeout must be a positive duration")
	}
	ctx, cancel := context.WithTimeout(ctx, d)
	defer cancel()
	for {
		out, err := refresh(ctx, stateDir, id, verify, p)
		if err != nil {
			return out, err
		}
		done := true
		for _, x := range out.Runs {
			if x.RunID <= 0 || x.DispatchState != "dispatched" || x.Status != "completed" {
				done = false
				break
			}
		}
		if done {
			for _, x := range out.Runs {
				if x.Conclusion != "success" {
					return out, fmt.Errorf("job run %q concluded %q", x.Name, x.Conclusion)
				}
			}
			return out, nil
		}
		select {
		case <-ctx.Done():
			return out, ctx.Err()
		case <-time.After(time.Second):
		}
	}
}
func Verify(ctx context.Context, stateDir, id string) (Report, error) {
	r, _, err := readReceipt(stateDir, id)
	if err != nil {
		return Report{}, err
	}
	if !hasImages(r.Resolved) {
		return Report{}, errors.New("job verification requires at least one declared image")
	}
	out, err := refresh(ctx, stateDir, id, true, GitHubProvider{})
	if err != nil {
		return out, err
	}
	for _, run := range out.Runs {
		if run.Status != "completed" || run.Conclusion != "success" || (len(r.Runs) > 0 && hasRunImages(r, run.Name) && !run.Verified) {
			return out, fmt.Errorf("job run %q is incomplete for verification", run.Name)
		}
	}
	return out, nil
}
func hasRunImages(r receipt, name string) bool {
	for _, x := range r.Runs {
		if x.Name == name {
			return len(x.Images) > 0
		}
	}
	return false
}

func refresh(ctx context.Context, stateDir, id string, verify bool, p Provider) (Report, error) {
	if p == nil {
		p = GitHubProvider{}
	}
	r, sp, err := readReceipt(stateDir, id)
	if err != nil {
		return Report{}, err
	}
	err = withReceiptLock(sp, func() error {
		fail := func(cause error) error {
			if saveErr := writeReceipt(sp, r); saveErr != nil {
				return saveErr
			}
			return cause
		}
		latest, _, err := readReceipt(stateDir, id)
		if err != nil {
			return err
		}
		r = latest
		for i := range r.Runs {
			x := &r.Runs[i]
			if verify {
				x.Verified = false
				x.Verification = nil
			}
			if x.RunID <= 0 {
				return fail(fmt.Errorf("job run %q has no confirmed remote run ID", x.Name))
			}
			remote, err := p.GetRun(ctx, r.Repository, x.RunID)
			if err != nil {
				return fail(fmt.Errorf("read job run %q: %w", x.Name, err))
			}
			if err := validateRemote(r, *x, remote); err != nil {
				x.Verified = false
				x.Verification = nil
				return fail(err)
			}
			x.Status = remote.Status
			x.Conclusion = remote.Conclusion
			if remote.Status != "completed" || remote.Conclusion != "success" {
				x.Verified = false
				x.Verification = nil
			}
			if remote.HTMLURL != "" {
				x.URL = remote.HTMLURL
			}
			if verify && len(x.Images) > 0 && remote.Status == "completed" {
				x.Verified = false
				x.Verification = nil
				if remote.Conclusion != "success" {
					return fail(fmt.Errorf("job run %q is not successful", x.Name))
				}
				results, err := verifyArtifact(ctx, p, r, *x)
				if err != nil {
					return fail(err)
				}
				x.Verified = true
				x.Verification = results
			}
		}
		return writeReceipt(sp, r)
	})
	return report(r), err
}
func hasImages(d Definition) bool {
	for _, x := range d.Runs {
		if len(x.Images) > 0 {
			return true
		}
	}
	return false
}
func receiptRuns(d Definition) []receiptRun {
	o := make([]receiptRun, len(d.Runs))
	for i, x := range d.Runs {
		o[i] = receiptRun{Name: x.Name, Workflow: x.Workflow, Inputs: copyMap(x.Inputs), ResultArtifact: x.ResultArtifact, Images: append([]Image(nil), x.Images...), DispatchState: "not_started"}
	}
	return o
}
func copyMap(m map[string]string) map[string]string {
	o := map[string]string{}
	maps.Copy(o, m)
	return o
}
func validateRemote(r receipt, x receiptRun, remote RemoteRun) error {
	// Actions reports head_branch inconsistently for tag dispatches. The receipt pins
	// the dispatch ref before mutation and the returned run must instead match its
	// immutable head SHA, workflow ID, event, repository, and exact response ID.
	if remote.ID != x.RunID || remote.Attempt != x.RunAttempt || !strings.EqualFold(remote.Repository, r.Repository) || remote.Event != "workflow_dispatch" || remote.HeadSHA == "" || remote.Workflow != x.WorkflowID {
		return fmt.Errorf("GitHub run %d does not match its receipt", x.RunID)
	}
	if r.HeadSHA != "" && remote.HeadSHA != r.HeadSHA {
		return fmt.Errorf("GitHub run %d head SHA differs from receipt", x.RunID)
	}
	return nil
}

type artifactResult struct {
	SchemaVersion int             `json:"schema_version"`
	HeadSHA       string          `json:"head_sha"`
	Images        []artifactImage `json:"images"`
}
type artifactImage struct {
	Reference string `json:"reference"`
	Digest    string `json:"digest"`
}

func verifyArtifact(ctx context.Context, p Provider, r receipt, x receiptRun) ([]ociverify.Result, error) {
	arts, err := p.ListArtifacts(ctx, r.Repository, x.RunID)
	if err != nil {
		return nil, err
	}
	var found []Artifact
	for _, a := range arts {
		if a.Name == x.ResultArtifact {
			found = append(found, a)
		}
	}
	if len(found) != 1 {
		return nil, fmt.Errorf("result artifact %q is missing or ambiguous", x.ResultArtifact)
	}
	if found[0].Expired || found[0].Size < 1 || found[0].Size > maxArtifactBytes {
		return nil, fmt.Errorf("result artifact %q is unavailable or too large", x.ResultArtifact)
	}
	data, err := p.DownloadArtifact(ctx, r.Repository, found[0].ID)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxArtifactBytes {
		return nil, errors.New("result artifact exceeds size limit")
	}
	result, err := parseArtifact(data)
	if err != nil {
		return nil, err
	}
	if result.SchemaVersion != 1 || result.HeadSHA != r.HeadSHA {
		return nil, errors.New("result artifact has invalid schema or head SHA")
	}
	if len(result.Images) != len(x.Images) {
		return nil, errors.New("result artifact images do not match declaration")
	}
	seen := map[string]bool{}
	verified := make([]ociverify.Result, 0, len(result.Images))
	for _, got := range result.Images {
		if got.Reference == "" || !strings.HasPrefix(got.Digest, "sha256:") || len(got.Digest) != 71 || seen[got.Reference] {
			return nil, errors.New("result artifact has invalid image digest")
		}
		seen[got.Reference] = true
		var image *Image
		for i := range x.Images {
			if x.Images[i].Reference == got.Reference {
				image = &x.Images[i]
				break
			}
		}
		if image == nil {
			return nil, errors.New("result artifact image reference differs from declaration")
		}
		verifiedResult, err := verifyOCI(ctx, ociverify.Options{Reference: image.Reference, ExpectedDigest: got.Digest, Platforms: image.Platforms})
		if err != nil {
			return nil, fmt.Errorf("verify image %q: %w", got.Reference, err)
		}
		verified = append(verified, verifiedResult)
	}
	return verified, nil
}
func parseArtifact(data []byte) (artifactResult, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return artifactResult{}, errors.New("result artifact is not a ZIP")
	}
	if len(zr.File) != 1 {
		return artifactResult{}, errors.New("result artifact must contain exactly result.json")
	}
	f := zr.File[0]
	if f.Name != "result.json" || path.Clean(f.Name) != f.Name || strings.Contains(f.Name, "\\") || f.UncompressedSize64 > 1<<20 {
		return artifactResult{}, errors.New("unsafe result artifact ZIP")
	}
	rc, err := f.Open()
	if err != nil {
		return artifactResult{}, err
	}
	defer func() { _ = rc.Close() }()
	b, err := io.ReadAll(io.LimitReader(rc, 1<<20+1))
	if err != nil || len(b) > 1<<20 {
		return artifactResult{}, errors.New("result artifact result.json too large")
	}
	var result artifactResult
	if err := json.Unmarshal(b, &result); err != nil {
		return artifactResult{}, errors.New("invalid result.json")
	}
	return result, nil
}
