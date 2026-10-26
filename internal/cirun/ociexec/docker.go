package ociexec

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

const metadataBytes = 64 << 10

type dockerClient struct {
	binary, binarySHA, endpoint, daemonID, configDir string
	env                                              []string
}

// capture cancels the producer on overflow while retaining a bounded prefix.
// Returning len(p) avoids changing the primary error into an io copy error.
type capture struct {
	mu        sync.Mutex
	buf       bytes.Buffer
	limit     int64
	truncated bool
	cancel    context.CancelFunc
	writer    io.Writer
}

func (c *capture) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := len(p)
	left := c.limit - int64(c.buf.Len())
	if int64(len(p)) > left {
		p = p[:max(0, left)]
		c.truncated = true
		if c.cancel != nil {
			c.cancel()
		}
	}
	_, _ = c.buf.Write(p)
	if c.writer != nil {
		if _, err := c.writer.Write(p); err != nil {
			return 0, err
		}
	}
	return n, nil
}

func (c *capture) data() ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]byte(nil), c.buf.Bytes()...), c.truncated
}

func binaryDigest(path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Size() > 512<<20 {
		return "", errors.New("OCI Docker executable must be a regular file no larger than 512 MiB")
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return "", errors.New("OCI Docker executable changed during identity capture")
	}
	h := sha256.New()
	if n, err := io.Copy(h, io.LimitReader(f, 512<<20+1)); err != nil || n > 512<<20 {
		if err == nil {
			err = errors.New("OCI Docker executable exceeded identity capture limit")
		}
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func newDocker(ctx context.Context, privateConfig string) (*dockerClient, error) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		return nil, errors.New("OCI CI requires a local Unix Docker endpoint on Linux or macOS")
	}
	path, err := exec.LookPath("docker")
	if err != nil {
		return nil, err
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return nil, err
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if filepath.Base(path) == "mise" || strings.Contains(filepath.ToSlash(path), "/mise/shims/") {
		return nil, errors.New("OCI Docker executable must not resolve through a tool-manager shim")
	}
	sha, err := binaryDigest(path)
	if err != nil {
		return nil, err
	}
	d := &dockerClient{binary: path, binarySHA: sha}
	for _, value := range os.Environ() {
		key, _, _ := strings.Cut(value, "=")
		switch key {
		case "DOCKER_HOST", "DOCKER_CONTEXT", "DOCKER_TLS_VERIFY", "DOCKER_CERT_PATH", "DOCKER_API_VERSION", "DOCKER_CLI_HOOKS", "DOCKER_CLI_HINTS":
		default:
			d.env = append(d.env, value)
		}
	}
	d.env = append(d.env, "DOCKER_CLI_HOOKS=false", "DOCKER_CLI_HINTS=false")
	host := os.Getenv("DOCKER_HOST")
	selected := os.Getenv("DOCKER_CONTEXT")
	if selected != "" || host == "" {
		args := []string{"context", "inspect"}
		if selected != "" {
			if !regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`).MatchString(selected) {
				return nil, errors.New("invalid Docker context name")
			}
			args = append(args, selected)
		}
		data, err := d.metadata(ctx, args...)
		if err != nil {
			return nil, err
		}
		var contexts []struct {
			Endpoints map[string]struct{ Host string }
		}
		if err := json.Unmarshal(data, &contexts); err != nil || len(contexts) != 1 {
			return nil, errors.New("docker context must resolve to exactly one endpoint")
		}
		host = contexts[0].Endpoints["docker"].Host
	}
	endpoint, err := url.Parse(host)
	if err != nil || endpoint.Scheme != "unix" || endpoint.Host != "" || endpoint.User != nil ||
		endpoint.RawQuery != "" || endpoint.Fragment != "" || !filepath.IsAbs(endpoint.Path) || strings.ContainsAny(host, "\x00\n\r") {
		return nil, errors.New("OCI CI supports only a local absolute unix:// Docker endpoint")
	}
	d.endpoint = host
	if err := os.Mkdir(privateConfig, 0o700); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(privateConfig, "config.json"), []byte("{}\n"), 0o600); err != nil {
		return nil, err
	}
	d.configDir = privateConfig
	// Resolve the caller's context once, then stop reading ambient client
	// credentials/proxies/plugins for all daemon and create/cleanup commands.
	isolated := make([]string, 0, len(d.env)+1)
	for _, value := range d.env {
		key, _, _ := strings.Cut(value, "=")
		switch strings.ToUpper(key) {
		case "DOCKER_CONFIG", "HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "NO_PROXY":
		default:
			isolated = append(isolated, value)
		}
	}
	d.env = append(isolated, "DOCKER_CONFIG="+privateConfig)
	info, err := d.info(ctx)
	if err != nil {
		return nil, err
	}
	if info.ID == "" || info.OSType != "linux" || info.CgroupVersion != "2" || !info.PidsLimit {
		return nil, errors.New("OCI CI requires a Linux cgroup-v2 Docker daemon with PID limits")
	}
	d.daemonID = info.ID
	return d, nil
}

func (d *dockerClient) command(ctx context.Context, args ...string) *exec.Cmd {
	if d.endpoint != "" {
		args = append([]string{"--config", d.configDir, "--host", d.endpoint}, args...)
	}
	cmd := exec.CommandContext(ctx, d.binary, args...)
	for _, arg := range args {
		if arg == "__complete" || arg == "__completeNoDesc" {
			cmd.Err = errors.New("reserved Docker transport completion token")
			break
		}
	}
	cmd.Env = d.env
	cmd.Stdin = nil
	cmd.WaitDelay = time.Second
	return cmd
}

func (d *dockerClient) metadata(parent context.Context, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	out, stderr := &capture{limit: metadataBytes, cancel: cancel}, &capture{limit: metadataBytes, cancel: cancel}
	cmd := d.command(ctx, args...)
	cmd.Stdout, cmd.Stderr = out, stderr
	err := cmd.Run()
	data, truncated := out.data()
	_, errorTruncated := stderr.data()
	if truncated || errorTruncated {
		return nil, errors.New("docker metadata exceeded output limit")
	}
	if err != nil {
		return nil, errors.Join(fmt.Errorf("docker %s: %w", args[0], err), ctx.Err())
	}
	return data, nil
}

func (d *dockerClient) info(ctx context.Context) (daemonInfo, error) {
	var info daemonInfo
	endpoint, err := url.Parse(d.endpoint)
	if err != nil || endpoint.Scheme != "unix" || endpoint.Host != "" || !filepath.IsAbs(endpoint.Path) {
		return info, errors.New("daemon metadata requires a pinned local Unix endpoint")
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", endpoint.Path)
		},
		DisableKeepAlives:      true,
		MaxResponseHeaderBytes: metadataBytes,
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
		return errors.New("daemon metadata redirects are forbidden")
	}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://docker/info", nil)
	if err != nil {
		return info, err
	}
	response, err := client.Do(req)
	if err != nil {
		return info, err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return info, fmt.Errorf("daemon metadata HTTP status %d", response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, metadataBytes+1))
	if err != nil {
		return info, err
	}
	if len(data) > metadataBytes {
		return info, errors.New("daemon metadata exceeded output limit")
	}
	if err := json.Unmarshal(data, &info); err != nil {
		return info, err
	}
	return info, nil
}

func (d *dockerClient) inspect(ctx context.Context, name string) (containerInfo, error) {
	var entries []containerInfo
	data, err := d.metadata(ctx, "container", "inspect", name)
	if err != nil {
		return containerInfo{}, err
	}
	if err := json.Unmarshal(data, &entries); err != nil || len(entries) != 1 {
		return containerInfo{}, errors.New("docker inspect did not return exactly one container")
	}
	return entries[0], nil
}

func (d *dockerClient) absent(ctx context.Context, reference string) (bool, error) {
	filter := "name=^/" + reference + "$"
	if containerIDPattern.MatchString(reference) {
		filter = "id=" + reference
	}
	data, err := d.metadata(ctx, "container", "ls", "--all", "--no-trunc", "--filter", filter, "--format", "{{.ID}}")
	return len(bytes.TrimSpace(data)) == 0, err
}
