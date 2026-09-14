package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// Environment the image shim reads, so one fake `docker` can model all four calls this
// diagnosis makes: the compose command that failed, `docker info`, `compose config
// --format json`, and `docker image inspect`.
const (
	shimConfigJSONVar    = "DVA_TEST_SHIM_CONFIG_JSON"
	shimConfigExitVar    = "DVA_TEST_SHIM_CONFIG_EXIT"
	shimMissingImagesVar = "DVA_TEST_SHIM_MISSING_IMAGES"
	shimLogVar           = "DVA_TEST_SHIM_LOG"
	shimInspectFifoVar   = "DVA_TEST_SHIM_INSPECT_FIFO"
)

// installImageShim replaces PATH with a single directory holding a `docker` that answers
// every call this probe makes from environment variables, and appends each invocation to
// a log file so a test can assert which subprocesses ran — including that none did.
//
// PATH is replaced rather than prepended, and the body uses only shell builtins, for the
// same reason installShims does: a real docker on the machine must not be able to decide
// the result.
func installImageShim(t *testing.T) (logPath string) {
	t.Helper()

	dir := t.TempDir()
	logPath = filepath.Join(dir, "calls.log")
	script := "#!/bin/sh\n" +
		"echo \"$*\" >> \"$" + shimLogVar + "\"\n" +
		"if [ \"$1\" = \"info\" ]; then exit \"${" + shimInfoExitVar + ":-0}\"; fi\n" +
		"if [ \"$1\" = \"image\" ] && [ \"$2\" = \"inspect\" ]; then\n" +
		"  if [ -n \"${" + shimInspectFifoVar + ":-}\" ]; then read x < \"$" + shimInspectFifoVar + "\"; fi\n" +
		"  for m in ${" + shimMissingImagesVar + ":-}; do\n" +
		"    if [ \"$m\" = \"$3\" ]; then exit 1; fi\n" +
		"  done\n" +
		"  exit 0\n" +
		"fi\n" +
		"sub=\"\"\n" +
		"for a in \"$@\"; do\n" +
		"  case \"$a\" in\n" +
		"    config|up|down|stop|rm|ps) if [ -z \"$sub\" ]; then sub=\"$a\"; fi ;;\n" +
		"  esac\n" +
		"done\n" +
		"if [ \"$sub\" = \"config\" ]; then\n" +
		"  printf '%s' \"${" + shimConfigJSONVar + ":-}\"\n" +
		"  exit \"${" + shimConfigExitVar + ":-0}\"\n" +
		"fi\n" +
		"exit \"${" + shimComposeExitVar + ":-0}\"\n"

	if err := os.WriteFile(filepath.Join(dir, "docker"), []byte(script), 0o755); err != nil {
		t.Fatalf("write docker shim: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "podman-compose"), []byte(script), 0o755); err != nil {
		t.Fatalf("write podman-compose shim: %v", err)
	}
	if err := os.WriteFile(logPath, nil, 0o644); err != nil {
		t.Fatalf("create shim log: %v", err)
	}
	t.Setenv("PATH", dir)
	t.Setenv(shimLogVar, logPath)
	return logPath
}

func shimCalls(t *testing.T, logPath string) []string {
	t.Helper()
	raw, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read shim log: %v", err)
	}
	var calls []string
	for l := range strings.SplitSeq(strings.TrimSpace(string(raw)), "\n") {
		if strings.TrimSpace(l) != "" {
			calls = append(calls, l)
		}
	}
	return calls
}

// The measured shape from the card: one image-only service that cannot be pulled, twelve
// more whose pulls compose cancelled, and a service that builds its own image.
const fixtureMixedProject = `{
  "services": {
    "web-svelte": {"image": "gizzahub/web-svelte:latest"},
    "grafana":    {"image": "grafana/grafana:10.2.0"},
    "api":        {"image": "gizzahub/api:latest", "build": {"context": "."}},
    "postgres":   {"image": "postgres:17-alpine"}
  }
}`

func upFailureContext(t *testing.T, command string) (*ComposePlugin, *PluginContext) {
	t.Helper()
	p, pctx := shimmedCompose(t, command)
	return p, pctx
}

// Given a failed up whose project has an image-only service still absent locally, When up
// returns, Then the error names that image instead of relaying the exit status.
func TestComposeUp_MissingLocalImage_IsDiagnosed(t *testing.T) {
	installImageShim(t)
	t.Setenv(shimComposeExitVar, "1")
	t.Setenv(shimInfoExitVar, "0")
	t.Setenv(shimConfigJSONVar, fixtureMixedProject)
	t.Setenv(shimMissingImagesVar, "gizzahub/web-svelte:latest")

	p, pctx := upFailureContext(t, "")

	_, err := p.Up(context.Background(), pctx)

	var missErr *MissingLocalImageError
	if !errors.As(err, &missErr) {
		t.Fatalf("Up() error = %v, want a *MissingLocalImageError", err)
	}
	if got, want := missErr.Images, []string{"gizzahub/web-svelte:latest"}; len(got) != 1 || got[0] != want[0] {
		t.Errorf("Images = %v, want %v", got, want)
	}
	if !strings.Contains(err.Error(), "compose up") {
		t.Errorf("error lost its compose context: %q", err.Error())
	}
}

// Given a service that declares build:, When its image is absent too, Then it is not part
// of the diagnosis — compose builds it, so a missing image is not this condition.
func TestComposeUp_BuildableService_IsNotDiagnosed(t *testing.T) {
	installImageShim(t)
	t.Setenv(shimComposeExitVar, "1")
	t.Setenv(shimInfoExitVar, "0")
	t.Setenv(shimConfigJSONVar, fixtureMixedProject)
	// Only the buildable service's image is absent.
	t.Setenv(shimMissingImagesVar, "gizzahub/api:latest")

	p, pctx := upFailureContext(t, "")

	_, err := p.Up(context.Background(), pctx)

	if err == nil {
		t.Fatal("Up() error = nil, want the compose failure")
	}
	if _, ok := errors.AsType[*MissingLocalImageError](err); ok {
		t.Fatalf("Up() blamed a service that builds its own image: %v", err)
	}
	for _, call := range shimCalls(t, os.Getenv(shimLogVar)) {
		if strings.HasPrefix(call, "image inspect gizzahub/api:latest") {
			t.Errorf("probe inspected a buildable service's image: %q", call)
		}
	}
}

// Given the cascade the card measured — one denied pull and several Interrupted ones —
// When the diagnosis runs, Then every absent image-only reference is reported, as
// candidates, with the cancellation stated.
//
// This is the boundary decision made for this card: after the failure the causal image
// and the interrupted victims are indistinguishable by presence, so the message must not
// claim all of them must be built. Reporting the victims as a verdict would be worse than
// the bare exit status it replaces; reporting them as candidates is not.
func TestComposeUp_InterruptedPulls_ReportedAsCandidates(t *testing.T) {
	installImageShim(t)
	t.Setenv(shimComposeExitVar, "1")
	t.Setenv(shimInfoExitVar, "0")
	t.Setenv(shimConfigJSONVar, fixtureMixedProject)
	// The causal image plus two whose pull compose cancelled.
	t.Setenv(shimMissingImagesVar, "gizzahub/web-svelte:latest grafana/grafana:10.2.0 postgres:17-alpine")

	p, pctx := upFailureContext(t, "")

	_, err := p.Up(context.Background(), pctx)

	var missErr *MissingLocalImageError
	if !errors.As(err, &missErr) {
		t.Fatalf("Up() error = %v, want a *MissingLocalImageError", err)
	}
	want := []string{"gizzahub/web-svelte:latest", "grafana/grafana:10.2.0", "postgres:17-alpine"}
	if len(missErr.Images) != len(want) {
		t.Fatalf("Images = %v, want %v", missErr.Images, want)
	}
	for _, w := range want {
		if !strings.Contains(err.Error(), w) {
			t.Errorf("error does not name %q: %q", w, err.Error())
		}
	}
	// The honesty clause: at least one, not all, and a way to narrow it.
	for _, want := range []string{"at least one", "interrupted", "candidate list", "re-run"} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Errorf("error does not state %q: %q", want, err.Error())
		}
	}
}

// Given a successful up, When it returns, Then the probe never runs — no `config
// --format json`, no `image inspect`.
//
// The principle DockerDaemonReachable's placement already set: a diagnosis that costs a
// subprocess round trip is paid for only by the path that has already failed.
func TestComposeUp_Success_RunsNoProbe(t *testing.T) {
	logPath := installImageShim(t)
	t.Setenv(shimComposeExitVar, "0")
	t.Setenv(shimConfigJSONVar, fixtureMixedProject)
	t.Setenv(shimMissingImagesVar, "gizzahub/web-svelte:latest")

	p, pctx := upFailureContext(t, "")

	if _, err := p.Up(context.Background(), pctx); err != nil {
		t.Fatalf("Up() error = %v, want nil", err)
	}

	for _, call := range shimCalls(t, logPath) {
		if strings.Contains(call, "config --format json") || strings.HasPrefix(call, "image inspect") {
			t.Errorf("successful up ran a diagnosis subprocess: %q", call)
		}
	}
}

// Given an unreachable daemon, When up fails with images absent, Then DockerDaemonError
// still wins — every pull and every inspect fails during an outage, so a missing-image
// list there would describe the outage, not a cause.
func TestComposeUp_DaemonUnreachable_BeatsMissingImage(t *testing.T) {
	installImageShim(t)
	t.Setenv(shimComposeExitVar, "1")
	t.Setenv(shimInfoExitVar, "1")
	t.Setenv(shimConfigJSONVar, fixtureMixedProject)
	t.Setenv(shimMissingImagesVar, "gizzahub/web-svelte:latest grafana/grafana:10.2.0")

	p, pctx := upFailureContext(t, "")

	_, err := p.Up(context.Background(), pctx)

	var daemonErr *DockerDaemonError
	if !errors.As(err, &daemonErr) {
		t.Fatalf("Up() error = %v, want a *DockerDaemonError", err)
	}
	if _, ok := errors.AsType[*MissingLocalImageError](err); ok {
		t.Fatalf("missing-image diagnosis masked the daemon outage: %v", err)
	}
}

// Given a replaced compose runner, When up fails, Then docker's image store is not
// consulted — podman-compose keeps its images elsewhere, so every reference would look
// absent.
func TestComposeUp_ReplacedRunner_SkipsImageProbe(t *testing.T) {
	logPath := installImageShim(t)
	t.Setenv(shimComposeExitVar, "1")
	t.Setenv(shimInfoExitVar, "0")
	t.Setenv(shimConfigJSONVar, fixtureMixedProject)
	t.Setenv(shimMissingImagesVar, "gizzahub/web-svelte:latest")

	p, pctx := upFailureContext(t, "podman-compose")

	_, err := p.Up(context.Background(), pctx)

	if err == nil {
		t.Fatal("Up() error = nil, want the podman-compose failure")
	}
	if _, ok := errors.AsType[*MissingLocalImageError](err); ok {
		t.Fatalf("Up() used docker's image store for a podman-compose project: %v", err)
	}
	for _, call := range shimCalls(t, logPath) {
		if strings.HasPrefix(call, "image inspect") {
			t.Errorf("probe ran for a replaced runner: %q", call)
		}
	}
}

// Given the project cannot be described (a compose config that fails), When up fails,
// Then the original failure survives — the probe is advisory and must not replace a real
// error with its own.
func TestComposeUp_ProbeConfigFails_KeepsOriginalError(t *testing.T) {
	installImageShim(t)
	t.Setenv(shimComposeExitVar, "1")
	t.Setenv(shimInfoExitVar, "0")
	t.Setenv(shimConfigExitVar, "1")

	p, pctx := upFailureContext(t, "")

	_, err := p.Up(context.Background(), pctx)

	if err == nil {
		t.Fatal("Up() error = nil, want the compose failure")
	}
	if _, ok := errors.AsType[*MissingLocalImageError](err); ok {
		t.Fatalf("probe invented a diagnosis from an unreadable project: %v", err)
	}
}

// Locks the message shape DockerDaemonError and ComposeConfigError already use: a cause
// line, then `→` direction lines.
func TestMissingLocalImageError_Error(t *testing.T) {
	e := &MissingLocalImageError{
		Images: []string{"gizzahub/web-svelte:latest", "grafana/grafana:10.2.0"},
		cause:  errors.New("exit status 1"),
	}

	msg := e.Error()

	summary, rest, ok := strings.Cut(msg, "\n")
	if !ok {
		t.Fatalf("Error() has no direction lines: %q", msg)
	}
	if !strings.Contains(summary, "gizzahub/web-svelte:latest") || !strings.Contains(summary, "build:") {
		t.Errorf("summary does not name the condition and the images: %q", summary)
	}
	if !strings.Contains(rest, "→") {
		t.Errorf("Error() lacks a `→` direction line: %q", msg)
	}
	// It must not claim every listed image has to be built.
	for _, forbidden := range []string{"must build all", "all of them must"} {
		if strings.Contains(strings.ToLower(msg), forbidden) {
			t.Errorf("Error() overclaims: %q", msg)
		}
	}
}

func TestMissingLocalImageError_Unwrap(t *testing.T) {
	cause := errors.New("boom")
	e := &MissingLocalImageError{Images: []string{"x"}, cause: cause}

	if !errors.Is(e, cause) {
		t.Errorf("errors.Is(e, cause) = false, want true")
	}

	var target *MissingLocalImageError
	if !errors.As(fmt.Errorf("wrapped: %w", e), &target) {
		t.Errorf("errors.As did not recover *MissingLocalImageError through a wrap")
	}
}

func TestHasBuild(t *testing.T) {
	cases := map[string]bool{
		``:                  false,
		`null`:              false,
		`{"context":"."}`:   true,
		`"."`:               true,
		`{}`:                true,
		"  null  ":          false,
		`{"context":"api"}`: true,
	}
	for in, want := range cases {
		if got := hasBuild([]byte(in)); got != want {
			t.Errorf("hasBuild(%q) = %v, want %v", in, got, want)
		}
	}
}

// A compose project whose services all build their own images yields nothing to report,
// even with every image absent.
func TestMissingLocalImages_AllBuildable_ReturnsNothing(t *testing.T) {
	installImageShim(t)
	t.Setenv(shimConfigJSONVar, `{"services":{"a":{"image":"a:1","build":{"context":"."}},"b":{"image":"b:1","build":"."}}}`)
	t.Setenv(shimMissingImagesVar, "a:1 b:1")

	dir := t.TempDir()
	p := &ComposePlugin{}
	pctx := &PluginContext{
		Entry:     &config.LifecycleEntry{Name: "compose", Compose: &config.ComposePluginConfig{}},
		Env:       config.NewEnvironment(nil, dir, dir),
		ConfigDir: dir,
		Logger:    slog.Default(),
	}

	if got := p.missingLocalImages(pctx); len(got) != 0 {
		t.Errorf("missingLocalImages() = %v, want empty", got)
	}
}

// Given a mode that narrows the up to a subset of services, When the up fails, Then the
// probe describes only those services — diagnosing one the run never started would name
// a confident wrong image in place of the real failure.
func TestComposeUp_ScopedServices_ProbesOnlyThatSubset(t *testing.T) {
	logPath := installImageShim(t)
	t.Setenv(shimComposeExitVar, "1")
	t.Setenv(shimInfoExitVar, "0")
	t.Setenv(shimConfigJSONVar, fixtureMixedProject)
	t.Setenv(shimMissingImagesVar, "gizzahub/web-svelte:latest")

	p, pctx := upFailureContext(t, "")
	scoped := []string{"web-svelte", "grafana"}
	pctx.ComposeServices = &scoped

	_, _ = p.Up(context.Background(), pctx)

	var configCall string
	for _, call := range shimCalls(t, logPath) {
		if strings.Contains(call, "config") {
			configCall = call
		}
	}
	if configCall == "" {
		t.Fatal("probe made no config call")
	}
	for _, want := range scoped {
		if !strings.Contains(configCall, " "+want) {
			t.Errorf("config call %q does not scope to %q", configCall, want)
		}
	}
	if strings.Contains(configCall, "postgres") {
		t.Errorf("config call %q names a service the up never started", configCall)
	}
}
