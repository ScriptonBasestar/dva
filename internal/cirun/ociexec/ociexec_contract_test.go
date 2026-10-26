package ociexec

// These tests exercise the executor's process boundary, not Docker.  A symlink
// to the test binary is the only executable placed on PATH; TestMain turns it
// into a deliberately small Docker protocol fixture before the test suite runs.

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

const fakeID = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const fakeImage = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

type fakeDockerState struct {
	Scenario  string   `json:"scenario"`
	Created   bool     `json:"created"`
	Started   bool     `json:"started"`
	Create    []string `json:"create,omitempty"`
	BadConfig bool     `json:"bad_config"`
}

func TestMain(m *testing.M) {
	if os.Getenv("DVA_OCI_FAKE_DOCKER") == "1" {
		fakeDockerMain()
		return
	}
	os.Exit(m.Run())
}

func fakeDockerMain() {
	statePath := os.Getenv("DVA_OCI_FAKE_STATE")
	var state fakeDockerState
	b, _ := os.ReadFile(statePath)
	_ = json.Unmarshal(b, &state)
	args := os.Args[1:]
	for len(args) >= 2 && (args[0] == "--host" || args[0] == "--config") {
		if args[0] == "--host" && args[1] != os.Getenv("DVA_OCI_FAKE_SOCKET") {
			state.BadConfig = true
		}
		if args[0] == "--config" {
			config, err := os.ReadFile(filepath.Join(args[1], "config.json"))
			if err != nil || string(config) != "{}\n" || os.Getenv("HTTP_PROXY") == "ambient-proxy-sentinel" || os.Getenv("DOCKER_CONFIG") != args[1] || os.Getenv("DOCKER_CLI_HINTS") != "false" || os.Getenv("DOCKER_CLI_HOOKS") != "false" {
				state.BadConfig = true
			}
		}
		args = args[2:]
	}
	write := func(v any) { _ = json.NewEncoder(os.Stdout).Encode(v) }
	save := func() { b, _ := json.Marshal(state); _ = os.WriteFile(statePath, b, 0o600) }
	if state.BadConfig {
		save()
	}
	if len(args) == 0 {
		os.Exit(2)
	}
	if len(args) >= 2 && args[0] == "image" && args[1] == "inspect" {
		if state.Scenario == "metadata-timeout" {
			time.Sleep(2 * time.Second)
			return
		}
		if state.Scenario == "image-unavailable" {
			os.Exit(1)
		}
		image := imageInfo{ID: fakeImage}
		image.RepoDigests = []string{"example.test/ci@" + fakeImage}
		if state.Scenario == "image-env" {
			image.Config.Env = []string{"IMAGE_ONLY=image"}
		}
		if state.Scenario == "image-volume" {
			image.Config.Volumes = map[string]any{"/data": map[string]any{}}
		}
		write([]imageInfo{image})
		return
	}
	switch strings.Join(args[:min(3, len(args))], " ") {
	case "context inspect":
		write([]map[string]any{{"Endpoints": map[string]any{"docker": map[string]string{"Host": os.Getenv("DVA_OCI_FAKE_SOCKET")}}}})
	case "info --format {{json .}}":
		// A CLI info call would permit host Docker CLI plugins. The daemon fixture
		// intentionally has no such command so this test fails if it regresses.
		fmt.Fprint(os.Stderr, "docker info CLI must not be invoked")
		os.Exit(2)
	case "container create --name":
		state.Create, state.Created = append([]string(nil), args...), true
		save()
		if state.Scenario == "lost-create" {
			fmt.Fprint(os.Stderr, "lost response")
			os.Exit(1)
		}
		fmt.Fprint(os.Stdout, fakeID)
	case "container inspect dva-ci-oci-":
		// Not reachable: the random name is appended to the command.
		os.Exit(1)
	default:
		if len(args) >= 3 && args[0] == "container" && args[1] == "inspect" {
			if state.Scenario == "absence-rpc" && state.Started {
				os.Exit(1)
			}
			if !state.Created {
				os.Exit(1)
			}
			info := fakeContainer(state.Create, state.Scenario, state.Started)
			write([]containerInfo{info})
			return
		}
		if len(args) >= 3 && args[0] == "container" && args[1] == "start" {
			state.Started = true
			save()
			if state.Scenario == "timeout" {
				time.Sleep(2 * time.Second)
				return
			}
			nonce := strings.TrimPrefix(flagValue(state.Create, "--label", labelNonce+"="), labelNonce+"=")
			if state.Scenario == "bad-kernel" {
				fmt.Fprintln(os.Stdout, "DVA_OCI_LIMITS:"+nonce+":1:2:3:4:5")
				return
			}
			if state.Scenario == "missing-kernel" {
				return
			}
			if state.Scenario == "overflow" {
				fmt.Fprint(os.Stdout, strings.Repeat("x", 4096))
				return
			}
			fmt.Fprintf(os.Stdout, "DVA_OCI_LIMITS:%s:8:16777216:0:1000000000:1000000000\n", nonce)
			return
		}
		if len(args) >= 3 && args[0] == "container" && args[1] == "rm" {
			if state.Scenario == "cleanup-fail" {
				os.Exit(1)
			}
			state.Created = false
			save()
			return
		}
		if len(args) >= 3 && args[0] == "container" && args[1] == "ls" {
			if state.Scenario == "absence-rpc" {
				os.Exit(1)
			}
			if state.Created {
				fmt.Fprint(os.Stdout, fakeID)
			}
			return
		}
		fmt.Fprintln(os.Stderr, "unexpected fake docker args:", args)
		os.Exit(2)
	}
}

func startFakeDaemon(t *testing.T, statePath, scenario string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("Unix-domain Docker endpoint fixture")
	}
	dir, err := os.MkdirTemp("", "dvaoci-")
	if err != nil {
		t.Fatal(err)
	}
	socket := filepath.Join(dir, "d.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet || request.URL.Path != "/info" {
			http.NotFound(w, request)
			return
		}
		switch scenario {
		case "daemon-http-timeout":
			time.Sleep(2 * time.Second)
			return
		case "daemon-http-status":
			http.Error(w, "fixture status", http.StatusInternalServerError)
			return
		case "daemon-http-oversize":
			_, _ = fmt.Fprint(w, strings.Repeat("x", metadataBytes+1))
			return
		case "daemon-http-redirect":
			w.Header().Set("Location", "http://example.invalid/info")
			w.WriteHeader(http.StatusFound)
			return
		}
		data, readErr := os.ReadFile(statePath)
		if readErr != nil {
			http.Error(w, readErr.Error(), http.StatusInternalServerError)
			return
		}
		var state fakeDockerState
		if err := json.Unmarshal(data, &state); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		id := "daemon-a"
		if scenario == "daemon-change" && state.Created {
			id = "daemon-b"
		}
		info := daemonInfo{ID: id, OSType: "linux", CgroupVersion: "2", PidsLimit: true}
		if scenario == "controller-unsupported" {
			info.CgroupVersion = "1"
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(info); err != nil {
			return
		}
	})}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close(); _ = listener.Close(); _ = os.Remove(socket); _ = os.Remove(dir) })
	return "unix://" + socket
}

func fakeContainer(args []string, scenario string, started bool) containerInfo {
	name := flagValue(args, "--name", "")
	init := true
	i := containerInfo{ID: fakeID, Name: "/" + name, Image: fakeImage}
	i.Config.User, i.Config.Entrypoint = "0:0", []string{"/bin/sh"}
	i.Config.WorkingDir = flagValue(args, "--workdir", "")
	i.Config.Labels = map[string]string{labelRun: flagValue(args, "--label", labelRun+"="), labelStep: flagValue(args, "--label", labelStep+"="), labelNonce: flagValue(args, "--label", labelNonce+"=")}
	for n, v := range i.Config.Labels {
		i.Config.Labels[n] = strings.TrimPrefix(v, n+"=")
	}
	imageAt := -1
	for n, v := range args {
		if v == fakeImage {
			imageAt = n
			break
		}
	}
	if imageAt >= 0 {
		i.Config.Cmd = append([]string(nil), args[imageAt+1:]...)
	}
	i.Config.Healthcheck.Test = []string{"NONE"}
	for n := 0; n+1 < len(args); n++ {
		if args[n] == "--env" {
			i.Config.Env = append(i.Config.Env, args[n+1])
		}
	}
	if scenario == "image-env" {
		i.Config.Env = append(i.Config.Env, "IMAGE_ONLY=image")
	}
	i.HostConfig.Init = &init
	i.HostConfig.PidsLimit = 8
	i.HostConfig.Memory, i.HostConfig.MemorySwap = 16<<20, 16<<20
	i.HostConfig.NanoCPUs = 1_000_000_000
	i.HostConfig.NetworkMode, i.HostConfig.CgroupnsMode, i.HostConfig.IpcMode = "none", "private", "private"
	i.HostConfig.ReadonlyRootfs = true
	i.HostConfig.CapDrop = []string{"ALL"}
	i.HostConfig.SecurityOpt = []string{"no-new-privileges"}
	i.HostConfig.LogConfig.Type = "none"
	i.HostConfig.RestartPolicy.Name = "no"
	i.HostConfig.Tmpfs = map[string]string{"/tmp": "rw"}
	i.Mounts = append(i.Mounts, struct {
		Type, Source, Destination string
		RW                        bool
	}{Type: "bind", Source: mountSource(flagValue(args, "--mount", "")), Destination: "/workspace"})
	i.State.Status, i.State.ExitCode = "created", 0
	if started {
		i.State.Status = "exited"
	}
	if scenario == "renamed-owned" && started {
		i.Name = "/renamed-after-attach"
	}
	if scenario == "exit-seven" {
		i.State.ExitCode = 7
	}
	if scenario == "oom" {
		i.State.OOMKilled = true
	}
	if scenario == "policy-bad" {
		i.HostConfig.NetworkMode = "bridge"
	}
	if scenario == "labels-conflict" {
		i.Config.Labels[labelNonce] = "other"
	}
	return i
}

func flagValue(args []string, flag, prefix string) string {
	for n := 0; n+1 < len(args); n++ {
		if args[n] == flag && (prefix == "" || strings.HasPrefix(args[n+1], prefix)) {
			return args[n+1]
		}
	}
	return ""
}
func mountSource(value string) string {
	for part := range strings.SplitSeq(value, ",") {
		if strings.HasPrefix(part, "source=") {
			source, _ := strings.CutPrefix(part, "source=")
			return source
		}
	}
	return ""
}

func TestRunFakeDockerContracts(t *testing.T) {
	for _, tc := range []struct{ name, scenario, want, workdir string }{
		{"success", "", "", ""}, {"workload exit", "exit-seven", "workload exited 7", ""}, {"oom", "oom", "OOM=true", ""},
		{"image environment", "image-env", "", ""},
		{"declared workdir", "", "", "/workspace/nested"},
		{"renamed owned CID", "renamed-owned", "", ""},
		{"bad kernel", "bad-kernel", "no matching cgroup", ""}, {"policy", "policy-bad", "containment policy", ""},
		{"lost create", "lost-create", "create OCI container", ""}, {"daemon change", "daemon-change", "daemon identity changed", ""},
		{"labels conflict", "labels-conflict", "identity/labels differ", ""}, {"cleanup failure", "cleanup-fail", "container cleanup", ""},
		{"overflow", "overflow", "output limit", ""}, {"timeout", "timeout", "context deadline exceeded", ""}, {"image volume", "image-volume", "declaring VOLUME", ""},
		{"missing kernel", "missing-kernel", "no matching cgroup", ""}, {"absence RPC", "absence-rpc", "cannot establish whether", ""},
		{"image unavailable", "image-unavailable", "image must already", ""}, {"unsupported controller", "controller-unsupported", "Linux cgroup-v2", ""},
		{"metadata timeout", "metadata-timeout", "context deadline exceeded", ""},
		{"daemon HTTP timeout", "daemon-http-timeout", "context deadline exceeded", ""}, {"daemon HTTP status", "daemon-http-status", "HTTP status 500", ""},
		{"daemon HTTP oversize", "daemon-http-oversize", "metadata exceeded output limit", ""}, {"daemon HTTP redirect", "daemon-http-redirect", "redirects are forbidden", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.Mkdir(filepath.Join(root, "input"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(filepath.Join(root, "state"), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "input", "a.txt"), []byte("input"), 0o644); err != nil {
				t.Fatal(err)
			}
			state := filepath.Join(root, "state.json")
			b, _ := json.Marshal(fakeDockerState{Scenario: tc.scenario})
			if err := os.WriteFile(state, b, 0o600); err != nil {
				t.Fatal(err)
			}
			endpoint := startFakeDaemon(t, state, tc.scenario)
			bin := filepath.Join(root, "bin")
			if err := os.Mkdir(bin, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(os.Args[0], filepath.Join(bin, "docker")); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", bin)
			t.Setenv("DVA_OCI_FAKE_DOCKER", "1")
			t.Setenv("DVA_OCI_FAKE_STATE", state)
			t.Setenv("DVA_OCI_FAKE_SOCKET", endpoint)
			t.Setenv("HOME", "/inherited-secret-home")
			t.Setenv("HTTP_PROXY", "ambient-proxy-sentinel")
			t.Setenv("DOCKER_CONFIG", filepath.Join(root, "ambient-docker-config"))
			t.Setenv("DOCKER_HOST", "")
			t.Setenv("DOCKER_CONTEXT", "")
			// The helper is a re-executed `go test -race` binary. Its own race
			// runtime otherwise waits one second at exit, racing this test's exact
			// one-second workload deadline; this affects no production process.
			t.Setenv("GORACE", "atexit_sleep_ms=0")
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			cfg := config.CIOCI{Image: "example.test/ci@" + fakeImage, Argv: []string{"echo", "literal; $HOME"}, Source: "input", Workdir: tc.workdir, PidsLimit: 8, MemoryBytes: 16 << 20, CPUs: 1, OutputLimitBytes: 1024, Environment: map[string]string{"VISIBLE": "ok"}}
			report, err := Run(ctx, Options{Root: root, StateDir: filepath.Join(root, "state"), RunID: "run", Step: "step", Config: cfg})
			if tc.want == "" {
				if err != nil {
					t.Fatalf("Run: %v", err)
				}
				if report.Kernel == nil || report.ExitCode == nil || *report.ExitCode != 0 || report.Cleanup != "removed" {
					t.Fatalf("unexpected success report: %+v", report)
				}
				info, statErr := os.Stat(report.DockerConfigPath)
				configBytes, readErr := os.ReadFile(filepath.Join(report.DockerConfigPath, "config.json"))
				if statErr != nil || info.Mode().Perm() != 0o700 || readErr != nil || string(configBytes) != "{}\n" {
					t.Fatalf("private Docker config is not sealed: mode=%v stat=%v read=%v data=%q", info.Mode(), statErr, readErr, configBytes)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err=%v, report=%+v, want %q", err, report, tc.want)
			}
			if report.IntentPath != "" {
				data, readErr := os.ReadFile(report.IntentPath)
				if readErr != nil {
					t.Fatal(readErr)
				}
				if strings.Contains(string(data), "inherited-secret-home") || strings.Contains(string(data), "literal; $HOME") || strings.Contains(string(data), "VISIBLE=ok") {
					t.Fatalf("intent retained forbidden value: %s", data)
				}
			}
			var got fakeDockerState
			data, _ := os.ReadFile(state)
			_ = json.Unmarshal(data, &got)
			if got.BadConfig {
				t.Fatalf("daemon command inherited ambient Docker config or proxy")
			}
			if tc.scenario == "" {
				joined := strings.Join(got.Create, "\x00")
				for _, flag := range []string{"--network\x00none", "--log-driver\x00none", "--no-healthcheck", "--memory-swap\x0016777216", "--pids-limit\x008"} {
					if !strings.Contains(joined, flag) {
						t.Fatalf("create lacks %q: %q", flag, got.Create)
					}
				}
				if !strings.Contains(joined, "literal; $HOME") {
					t.Fatalf("argv not literal: %q", got.Create)
				}
				wantWorkdir := tc.workdir
				if wantWorkdir == "" {
					wantWorkdir = "/workspace"
				}
				if !strings.Contains(joined, "--workdir\x00"+wantWorkdir) {
					t.Fatalf("workdir %q missing: %q", wantWorkdir, got.Create)
				}
			}
		})
	}
}

func TestRunRejectsUnpinnedImageBeforeDockerPayload(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "input"), 0o755); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := Run(ctx, Options{Root: root, StateDir: filepath.Join(root, "state"), RunID: "run", Step: "step", Config: config.CIOCI{Image: "example:latest", Argv: []string{"true"}, Source: "input", PidsLimit: 8, MemoryBytes: 16 << 20, CPUs: 1, OutputLimitBytes: 1024}})
	if err == nil || !strings.Contains(err.Error(), "image must be an immutable") {
		t.Fatalf("err=%v", err)
	}
}

func TestSnapshotInputRejectsEscapeAndCopiesImmutably(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "input"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "input", "a"), []byte("before"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{"../outside", "linked"} {
		dst := filepath.Join(t.TempDir(), "snapshot")
		if err := os.Mkdir(dst, 0o700); err != nil {
			t.Fatal(err)
		}
		if _, err := snapshotInput(root, source, dst); err == nil {
			t.Fatalf("%s accepted", source)
		}
	}
	dst := filepath.Join(t.TempDir(), "snapshot")
	if err := os.Mkdir(dst, 0o700); err != nil {
		t.Fatal(err)
	}
	digest, err := snapshotInput(root, "input", dst)
	if err != nil || digest == "" {
		t.Fatalf("snapshot: %q %v", digest, err)
	}
	if err := os.WriteFile(filepath.Join(root, "input", "a"), []byte("after"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dst, "a"))
	if err != nil || string(got) != "before" {
		t.Fatalf("snapshot mutable: %q %v", got, err)
	}
}

func TestSnapshotInputLengthFramesAmbiguousRecords(t *testing.T) {
	// Old NUL-delimited hashing could parse one file's bytes as the next file's
	// pathname/mode/content record. The two trees below must never share a digest.
	makeTree := func(files map[string]string) string {
		t.Helper()
		root, dst := t.TempDir(), filepath.Join(t.TempDir(), "snapshot")
		if err := os.Mkdir(filepath.Join(root, "input"), 0o755); err != nil {
			t.Fatal(err)
		}
		for name, value := range files {
			if err := os.WriteFile(filepath.Join(root, "input", name), []byte(value), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.Mkdir(dst, 0o700); err != nil {
			t.Fatal(err)
		}
		digest, err := snapshotInput(root, "input", dst)
		if err != nil {
			t.Fatal(err)
		}
		return digest
	}
	first := makeTree(map[string]string{"a": "x\x00b\x00644\x00y"})
	second := makeTree(map[string]string{"a": "x", "b": "y"})
	if first == second {
		t.Fatalf("ambiguous source trees share digest %s", first)
	}
}
