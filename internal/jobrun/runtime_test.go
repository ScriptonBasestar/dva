package jobrun

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/ScriptonBasestar/dva/internal/ociverify"
)

type fakeProvider struct {
	dispatches int
	failAt     int
	run        RemoteRun
	artifacts  []Artifact
	zip        []byte
}

func (f *fakeProvider) HeadSHA(context.Context, string, string) (string, error) { return "abc", nil }
func (f *fakeProvider) ResolveWorkflow(context.Context, string, string) (string, error) {
	return "1", nil
}
func (f *fakeProvider) Dispatch(_ context.Context, r DispatchRequest) (DispatchResponse, error) {
	f.dispatches++
	if f.failAt == f.dispatches {
		return DispatchResponse{}, errors.New("lost ack")
	}
	return DispatchResponse{RunID: int64(f.dispatches), HTMLURL: "https://example.test/run"}, nil
}
func (f *fakeProvider) GetRun(context.Context, string, int64) (RemoteRun, error) {
	x := f.run
	if x.Attempt == 0 {
		x.Attempt = 1
	}
	return x, nil
}
func (f *fakeProvider) ListArtifacts(context.Context, string, int64) ([]Artifact, error) {
	return f.artifacts, nil
}
func (f *fakeProvider) DownloadArtifact(context.Context, string, int64) ([]byte, error) {
	return f.zip, nil
}
func testRoot(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	for _, a := range [][]string{{"init"}, {"remote", "add", "origin", "https://github.com/acme/repo.git"}} {
		if out, e := exec.Command("git", append([]string{"-C", d}, a...)...).CombinedOutput(); e != nil {
			t.Fatal(string(out), e)
		}
	}
	return d
}
func def(runs int) Definition {
	d := Definition{Provider: "github-actions", Repository: "acme/repo", Ref: "branch", Timeout: "1s", Inputs: map[string]Input{"tag": {Default: "v1"}}}
	for i := range runs {
		d.Runs = append(d.Runs, RunDefinition{Name: string(rune('a' + i)), Workflow: "build.yml", Inputs: map[string]string{"tag": "{{input.tag}}"}, ResultArtifact: "result"})
	}
	return d
}

func TestResolveRejectsUnknownAndMissingTemplates(t *testing.T) {
	d := def(1)
	d.Runs[0].Inputs["x"] = "{{input.nope}}"
	if _, e := Resolve(d, nil); e == nil {
		t.Fatal("accepted unknown template")
	}
	d = def(1)
	if _, e := Resolve(d, map[string]string{"nope": "x"}); e == nil {
		t.Fatal("accepted unknown input")
	}
}
func TestRunPersistsUnknownDispatchAndResumeDoesNotRedispatch(t *testing.T) {
	p := &fakeProvider{failAt: 1}
	state := t.TempDir()
	r, e := Run(context.Background(), Options{Root: testRoot(t), Name: "n", StateDir: state, Definition: def(1), Provider: p})
	if e == nil || r.ID == "" {
		t.Fatalf("want persisted failure: %#v %v", r, e)
	}
	if p.dispatches != 1 {
		t.Fatal("wrong dispatch count")
	}
	_, _ = refresh(context.Background(), state, r.ID, false, p)
	if p.dispatches != 1 {
		t.Fatal("resume redispatched")
	}
}
func TestRunBatchStopsAfterDispatchFailure(t *testing.T) {
	p := &fakeProvider{failAt: 2}
	_, e := Run(context.Background(), Options{Root: testRoot(t), Name: "n", StateDir: t.TempDir(), Definition: def(3), Provider: p})
	if e == nil || p.dispatches != 2 {
		t.Fatalf("dispatches=%d err=%v", p.dispatches, e)
	}
}
func TestRefreshRejectsMismatchedRun(t *testing.T) {
	p := &fakeProvider{}
	state := t.TempDir()
	r, e := Run(context.Background(), Options{Root: testRoot(t), Name: "n", StateDir: state, Definition: def(1), Provider: p})
	if e != nil {
		t.Fatal(e)
	}
	p.run = RemoteRun{ID: r.Runs[0].RunID, Repository: "wrong/repo", Workflow: "1", Event: "workflow_dispatch", Ref: "branch", HeadSHA: "abc", Conclusion: "success"}
	if _, e = refresh(context.Background(), filepath.Join("", state), r.ID, false, p); e == nil {
		t.Fatal("wrong state expected")
	}
}

func TestVerifyLeavesQueuedRunPending(t *testing.T) {
	p := &fakeProvider{}
	state := t.TempDir()
	d := def(1)
	d.Runs[0].Images = []Image{{Reference: "registry.example/acme/app:v1"}}
	r, err := Run(context.Background(), Options{Root: testRoot(t), Name: "n", StateDir: state, Definition: d, Provider: p})
	if err != nil {
		t.Fatal(err)
	}
	p.run = RemoteRun{ID: r.Runs[0].RunID, Repository: "acme/repo", Workflow: "1", Event: "workflow_dispatch", HeadSHA: "abc", Status: "queued"}
	out, err := refresh(context.Background(), state, r.ID, true, p)
	if err != nil {
		t.Fatal(err)
	}
	if out.Runs[0].Verified || out.Runs[0].Status != "queued" {
		t.Fatalf("queued run was verified: %#v", out.Runs[0])
	}
}

func TestPollReturnsCompletedFailure(t *testing.T) {
	p := &fakeProvider{}
	state := t.TempDir()
	r, err := Run(context.Background(), Options{Root: testRoot(t), Name: "n", StateDir: state, Definition: def(1), Provider: p})
	if err != nil {
		t.Fatal(err)
	}
	p.run = RemoteRun{ID: r.Runs[0].RunID, Repository: "acme/repo", Workflow: "1", Event: "workflow_dispatch", HeadSHA: "abc", Status: "completed", Conclusion: "failure"}
	if _, err := poll(context.Background(), state, r.ID, false, p, "1s"); err == nil {
		t.Fatal("completed failure passed")
	}
}

func TestVerifyRejectsNoImagesBeforeDispatch(t *testing.T) {
	p := &fakeProvider{}
	if _, err := Run(context.Background(), Options{Root: testRoot(t), Name: "n", StateDir: t.TempDir(), Definition: def(1), Verify: true, Provider: p}); err == nil || p.dispatches != 0 {
		t.Fatalf("err=%v dispatches=%d", err, p.dispatches)
	}
}

func TestGitHubProviderBoundsOversizedChild(t *testing.T) {
	if os.Getenv("JOBRUN_OVERSIZED_HELPER") == "1" {
		for range int(maxArtifactBytes/(1<<20)) + 2 {
			fmt.Fprint(os.Stdout, string(make([]byte, 1<<20)))
		}
		os.Exit(0)
	}
	p := GitHubProvider{Command: func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.CommandContext(ctx, os.Args[0], "-test.run=TestGitHubProviderBoundsOversizedChild")
	}}
	old := os.Getenv("JOBRUN_OVERSIZED_HELPER")
	t.Setenv("JOBRUN_OVERSIZED_HELPER", "1")
	_ = old
	done := make(chan error, 1)
	go func() { _, err := p.command(context.Background(), "api"); done <- err }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("oversized child passed")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("oversized child hung")
	}
}

func TestVerifyArtifactForwardsExactDeclaredImage(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("result.json")
	if err != nil {
		t.Fatal(err)
	}
	_, err = w.Write([]byte(`{"schema_version":1,"head_sha":"abc","images":[{"reference":"registry.example/acme/app:v1","digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	p := &fakeProvider{artifacts: []Artifact{{ID: 1, Name: "result", Size: int64(buf.Len())}}, zip: buf.Bytes()}
	old := verifyOCI
	defer func() { verifyOCI = old }()
	var got ociverify.Options
	verifyOCI = func(_ context.Context, o ociverify.Options) (ociverify.Result, error) {
		got = o
		return ociverify.Result{Reference: o.Reference, ExpectedDigest: o.ExpectedDigest, Verified: true}, nil
	}
	r := receipt{Repository: "acme/repo", HeadSHA: "abc"}
	x := receiptRun{RunID: 1, ResultArtifact: "result", Images: []Image{{Reference: "registry.example/acme/app:v1", Platforms: []string{"linux/amd64"}}}}
	result, err := verifyArtifact(context.Background(), p, r, x)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || got.Reference != "registry.example/acme/app:v1" || got.Platforms[0] != "linux/amd64" {
		t.Fatalf("unexpected verification: %#v %#v", got, result)
	}
}

func TestResolveRejectsInvalidTimeoutAndInputLimit(t *testing.T) {
	d := def(1)
	d.Timeout = "25h"
	if _, err := Resolve(d, nil); err == nil {
		t.Fatal("accepted overlong timeout")
	}
	d = def(1)
	d.Inputs = map[string]Input{}
	for i := range 26 {
		d.Inputs[fmt.Sprintf("x%d", i)] = Input{}
	}
	if _, err := Resolve(d, nil); err == nil {
		t.Fatal("accepted input overflow")
	}
}

func TestResolveRejectsUnsafeRepositoryAndOversizedSuppliedInput(t *testing.T) {
	d := def(1)
	d.Repository = "../repo"
	if _, err := Resolve(d, nil); err == nil {
		t.Fatal("accepted unsafe repository")
	}
	d = def(1)
	if _, err := Resolve(d, map[string]string{"tag": string(make([]byte, 4097))}); err == nil {
		t.Fatal("accepted oversized supplied input")
	}
}

func TestRefreshInvalidatesPriorProofWhenRunIsQueued(t *testing.T) {
	p := &fakeProvider{}
	state := t.TempDir()
	d := def(1)
	d.Runs[0].Images = []Image{{Reference: "registry.example/acme/app:v1"}}
	r, err := Run(context.Background(), Options{Root: testRoot(t), Name: "n", StateDir: state, Definition: d, Provider: p})
	if err != nil {
		t.Fatal(err)
	}
	stored, path, err := readReceipt(state, r.ID)
	if err != nil {
		t.Fatal(err)
	}
	stored.Runs[0].Verified = true
	stored.Runs[0].Verification = []ociverify.Result{{Verified: true}}
	if err := writeReceipt(path, stored); err != nil {
		t.Fatal(err)
	}
	p.run = RemoteRun{ID: r.Runs[0].RunID, Repository: "acme/repo", Workflow: "1", Event: "workflow_dispatch", HeadSHA: "abc", Status: "queued"}
	out, err := refresh(context.Background(), state, r.ID, false, p)
	if err != nil {
		t.Fatal(err)
	}
	if out.Runs[0].Verified || len(out.Runs[0].Images) != 0 {
		t.Fatalf("stale proof retained: %#v", out.Runs[0])
	}
}

func TestRefreshInvalidatesPriorProofOnChangedAttempt(t *testing.T) {
	p := &fakeProvider{}
	state := t.TempDir()
	d := def(1)
	d.Runs[0].Images = []Image{{Reference: "registry.example/acme/app:v1"}}
	r, err := Run(context.Background(), Options{Root: testRoot(t), Name: "n", StateDir: state, Definition: d, Provider: p})
	if err != nil {
		t.Fatal(err)
	}
	stored, path, err := readReceipt(state, r.ID)
	if err != nil {
		t.Fatal(err)
	}
	stored.Runs[0].Verified = true
	stored.Runs[0].Verification = []ociverify.Result{{Verified: true}}
	if err := writeReceipt(path, stored); err != nil {
		t.Fatal(err)
	}
	p.run = RemoteRun{ID: r.Runs[0].RunID, Attempt: 2, Repository: "acme/repo", Workflow: "1", Event: "workflow_dispatch", HeadSHA: "abc", Status: "completed", Conclusion: "success"}
	if _, err := refresh(context.Background(), state, r.ID, false, p); err == nil {
		t.Fatal("changed attempt passed")
	}
	stored, _, err = readReceipt(state, r.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Runs[0].Verified || len(stored.Runs[0].Verification) != 0 {
		t.Fatal("stale proof persisted")
	}
}

func TestWriteReceiptRejectsUnreadableOversize(t *testing.T) {
	if err := writeReceipt(filepath.Join(t.TempDir(), "r.json"), receipt{Name: string(make([]byte, maxReceiptBytes+1))}); err == nil {
		t.Fatal("oversized receipt was written")
	}
}
