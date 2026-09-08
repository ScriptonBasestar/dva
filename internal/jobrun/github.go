package jobrun

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"time"
)

const apiVersion = "2026-03-10"

// GitHubProvider invokes gh without a shell. Callers may use Provider fakes in tests.
type GitHubProvider struct {
	Command func(context.Context, string, ...string) *exec.Cmd
}

func (p GitHubProvider) command(ctx context.Context, args ...string) ([]byte, error) {
	requestCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	c := p.Command
	if c == nil {
		c = exec.CommandContext
	}
	cmd := c(requestCtx, "gh", args...)
	cmd.Env = withoutDebug(os.Environ())
	cmd.WaitDelay = time.Second
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("GitHub API request failed")
	}
	stopClose := context.AfterFunc(requestCtx, func() { _ = stdout.Close() })
	out, readErr := io.ReadAll(io.LimitReader(stdout, maxArtifactBytes+1))
	if int64(len(out)) > maxArtifactBytes {
		cancel()
	}
	waitErr := cmd.Wait()
	stopClose()
	if readErr != nil || int64(len(out)) > maxArtifactBytes || waitErr != nil {
		return nil, fmt.Errorf("GitHub API request failed")
	}
	return out, nil
}
func withoutDebug(env []string) []string {
	out := make([]string, 0, len(env))
	for _, x := range env {
		if len(x) >= 9 && x[:9] == "GH_DEBUG=" {
			continue
		}
		out = append(out, x)
	}
	return out
}
func apiArgs(method, path string) []string {
	return []string{"api", "--hostname", "github.com", "-H", "X-GitHub-Api-Version: " + apiVersion, "-X", method, path}
}
func (p GitHubProvider) ResolveWorkflow(ctx context.Context, repo, workflow string) (string, error) {
	a := apiArgs("GET", "repos/"+repo+"/actions/workflows/"+url.PathEscape(workflow))
	out, err := p.command(ctx, a...)
	if err != nil {
		return "", err
	}
	var x struct {
		TotalCount int   `json:"total_count"`
		ID         int64 `json:"id"`
	}
	if err := json.Unmarshal(out, &x); err != nil || x.ID <= 0 {
		return "", fmt.Errorf("invalid GitHub workflow response")
	}
	return strconv.FormatInt(x.ID, 10), nil
}
func (p GitHubProvider) HeadSHA(ctx context.Context, repo, ref string) (string, error) {
	a := apiArgs("GET", "repos/"+repo+"/commits/"+url.PathEscape(ref))
	out, err := p.command(ctx, a...)
	if err != nil {
		return "", err
	}
	var v struct {
		SHA string `json:"sha"`
	}
	if err = json.Unmarshal(out, &v); err != nil || v.SHA == "" {
		return "", fmt.Errorf("invalid GitHub commit response")
	}
	return v.SHA, nil
}
func (p GitHubProvider) Dispatch(ctx context.Context, r DispatchRequest) (DispatchResponse, error) {
	a := apiArgs("POST", "repos/"+r.Repository+"/actions/workflows/"+r.Workflow+"/dispatches")
	a = append(a, "-f", "ref="+r.Ref)
	for k, v := range r.Inputs {
		a = append(a, "-f", "inputs["+k+"]="+v)
	}
	out, err := p.command(ctx, a...)
	if err != nil {
		return DispatchResponse{}, err
	}
	var x DispatchResponse
	if err = json.Unmarshal(out, &x); err != nil || x.RunID <= 0 {
		return DispatchResponse{}, fmt.Errorf("invalid GitHub dispatch response")
	}
	return x, nil
}
func (p GitHubProvider) GetRun(ctx context.Context, repo string, id int64) (RemoteRun, error) {
	a := apiArgs("GET", "repos/"+repo+"/actions/runs/"+strconv.FormatInt(id, 10))
	out, err := p.command(ctx, a...)
	if err != nil {
		return RemoteRun{}, err
	}
	var x struct {
		ID         int64  `json:"id"`
		Event      string `json:"event"`
		HeadSHA    string `json:"head_sha"`
		Status     string `json:"status"`
		Conclusion string `json:"conclusion"`
		HTMLURL    string `json:"html_url"`
		HeadBranch string `json:"head_branch"`
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
		WorkflowID int64 `json:"workflow_id"`
		RunAttempt int   `json:"run_attempt"`
	}
	if err = json.Unmarshal(out, &x); err != nil {
		return RemoteRun{}, fmt.Errorf("invalid GitHub run response")
	}
	return RemoteRun{ID: x.ID, Repository: x.Repository.FullName, Workflow: strconv.FormatInt(x.WorkflowID, 10), Event: x.Event, Ref: x.HeadBranch, HeadSHA: x.HeadSHA, Status: x.Status, Conclusion: x.Conclusion, HTMLURL: x.HTMLURL, Attempt: x.RunAttempt}, nil
}
func (p GitHubProvider) ListArtifacts(ctx context.Context, repo string, id int64) ([]Artifact, error) {
	a := apiArgs("GET", "repos/"+repo+"/actions/runs/"+strconv.FormatInt(id, 10)+"/artifacts?per_page=100")
	out, err := p.command(ctx, a...)
	if err != nil {
		return nil, err
	}
	var x struct {
		TotalCount int `json:"total_count"`
		Artifacts  []struct {
			ID      int64  `json:"id"`
			Name    string `json:"name"`
			Expired bool   `json:"expired"`
			Size    int64  `json:"size_in_bytes"`
		} `json:"artifacts"`
	}
	if err = json.Unmarshal(out, &x); err != nil {
		return nil, fmt.Errorf("invalid GitHub artifacts response")
	}
	if x.TotalCount != len(x.Artifacts) {
		return nil, fmt.Errorf("GitHub artifacts response is paginated")
	}
	o := make([]Artifact, len(x.Artifacts))
	for i, a := range x.Artifacts {
		o[i] = Artifact{a.ID, a.Name, a.Expired, a.Size}
	}
	return o, nil
}
func (p GitHubProvider) DownloadArtifact(ctx context.Context, repo string, id int64) ([]byte, error) {
	a := apiArgs("GET", "repos/"+repo+"/actions/artifacts/"+strconv.FormatInt(id, 10)+"/zip")
	return p.command(ctx, a...)
}
