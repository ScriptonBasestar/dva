package lifecycle

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/ScriptonBasestar/dva/internal/config"
	dvaexec "github.com/ScriptonBasestar/dva/internal/exec"
)

// ComposePlugin manages services via Docker Compose.
type ComposePlugin struct{}

func (p *ComposePlugin) Name() string { return "compose" }

func (p *ComposePlugin) Up(ctx context.Context, pctx *PluginContext) (*Result, error) {
	if pctx.Entry.ComposeConfig() == nil {
		return &Result{}, nil
	}

	args := composeUpArgs(pctx)

	if pctx.DryRun {
		cmd, cmdArgs, err := p.buildArgs(pctx, args)
		if err != nil {
			return nil, err
		}
		pctx.Logger.Info("dry-run", "command", cmd, "args", cmdArgs)
		return &Result{}, nil
	}

	// Preflight: validate the compose file set resolves before `up`. This makes a
	// compose file whose -f or include: target is missing/invalid surface as a
	// dva-owned, actionable error instead of docker's raw stderr followed by a
	// bare exit status. `config` only parses/merges — it needs no daemon.
	if cfg := pctx.Entry.ComposeConfig(); len(cfg.Files) > 0 {
		if err := p.preflightConfig(pctx); err != nil {
			return nil, err
		}
	}

	if err := p.runSubprocess(pctx, args); err != nil {
		// A failed up is diagnosed a second time, for the condition compose's own
		// output buries: an image compose cannot build and could not pull. See
		// MissingLocalImageError. Returns err unchanged when it does not apply.
		return nil, fmt.Errorf("compose up: %w", p.diagnoseMissingLocalImages(pctx, err))
	}

	// Query service status after up
	services, _ := p.queryServices(pctx)

	return &Result{Services: services}, nil
}

func composeUpArgs(pctx *PluginContext) []string {
	cfg := pctx.Entry.ComposeConfig()
	upOpts := cfg.UpOptions
	if len(upOpts) == 0 {
		upOpts = []string{"-d", "--wait"}
	}
	if !pctx.Wait {
		var filtered []string
		for _, o := range upOpts {
			if o != "--wait" {
				filtered = append(filtered, o)
			}
		}
		upOpts = filtered
		if len(upOpts) == 0 {
			upOpts = []string{"-d"}
		}
	}
	if pctx.Force {
		hasForce := slices.Contains(upOpts, "--force-recreate")
		if !hasForce {
			upOpts = append(upOpts, "--force-recreate")
		}
	}
	return append([]string{"up"}, upOpts...)
}

func (p *ComposePlugin) Down(ctx context.Context, pctx *PluginContext) error {
	if pctx.Entry.ComposeConfig() == nil {
		return nil
	}

	args := composeDownArgs(pctx)
	if note := composeDownLeftovers(pctx, args); note != "" {
		fmt.Fprintf(os.Stderr, "[compose] %s: %s\n", pctx.Entry.Name, note)
	}

	if pctx.DryRun {
		cmd, cmdArgs, err := p.buildArgs(pctx, args)
		if err != nil {
			return err
		}
		pctx.Logger.Info("dry-run", "command", cmd, "args", cmdArgs)
		return nil
	}

	return p.runSubprocess(pctx, args)
}

// composeDownArgs picks between the two teardown shapes compose offers.
//
// A plan that selects services gets `rm --force --stop <services>`: `compose down` has no
// service filter, so it would also remove the services other plans of the same project
// still run. `rm` only reaches containers (and, with --volumes, their anonymous volumes);
// named volumes and the project network are project-scoped and stay. Purge is the one
// request where staying is wrong — it asks for a clean slate — so it widens to the
// project-wide `down` regardless of the selection (TASK-311).
func composeDownArgs(pctx *PluginContext) []string {
	selected := pctx.ComposeServices != nil && len(*pctx.ComposeServices) > 0
	if selected && !pctx.Purge {
		args := []string{"rm", "--force", "--stop"}
		if pctx.Volumes {
			args = append(args, "--volumes")
		}
		return append(args, *pctx.ComposeServices...)
	}
	args := []string{"down", "--remove-orphans"}
	if pctx.Volumes || pctx.Purge {
		args = append(args, "--volumes")
	}
	if pctx.RemoveImages || pctx.Purge {
		args = append(args, "--rmi", "local")
	}
	return args
}

// composeDownLeftovers names what a service-scoped `rm` cannot remove, so the operator
// learns it from the command rather than from `docker volume ls` afterwards.
func composeDownLeftovers(pctx *PluginContext, args []string) string {
	if len(args) == 0 || args[0] != "rm" {
		return ""
	}
	left := "the project network"
	if pctx.Volumes {
		left = "named volumes and the project network"
	}
	return fmt.Sprintf("removing selected services only; %s stay — 'dva down <plan> --purge' tears down the whole compose project", left)
}

func (p *ComposePlugin) Stop(ctx context.Context, pctx *PluginContext) error {
	if pctx.Entry.ComposeConfig() == nil {
		return nil
	}

	args := composeStopArgs(pctx)

	if pctx.DryRun {
		cmd, cmdArgs, err := p.buildArgs(pctx, args)
		if err != nil {
			return err
		}
		pctx.Logger.Info("dry-run", "command", cmd, "args", cmdArgs)
		return nil
	}

	return p.runSubprocess(pctx, args)
}

func composeStopArgs(pctx *PluginContext) []string {
	args := []string{"stop"}
	if pctx.ComposeServices != nil && len(*pctx.ComposeServices) > 0 {
		args = append(args, *pctx.ComposeServices...)
	}
	return args
}

func (p *ComposePlugin) Status(ctx context.Context, pctx *PluginContext) ([]ServiceStatus, error) {
	return p.queryServicesContext(ctx, pctx)
}

// buildArgs constructs the docker compose command and arguments from plugin config.
// Mode-derived profiles are injected before the subcommand; mode-derived services
// are appended after the subcommand args (only for "up").
func (p *ComposePlugin) buildArgs(pctx *PluginContext, extraArgs []string) (string, []string, error) {
	cfg := pctx.Entry.ComposeConfig()

	// For sourced entries (TASK-051), relative compose files resolve against the
	// fetched/referenced source dir. The command also runs with that dir as its
	// working directory (see composeWorkdir), matching the legacy
	// `cd <dir> && docker compose` behavior so default file discovery, .env,
	// relative build contexts and volumes resolve as the external author intended.
	baseDir := pctx.ConfigDir
	if wd := composeWorkdir(pctx); wd != "" {
		baseDir = wd
	}

	cmd, args, err := dvaexec.ComposeArgv(pctx.Env, instanceComposeConfig(pctx, cfg, baseDir), baseDir)
	if err != nil {
		return "", nil, fmt.Errorf("entry %q: %w", pctx.Entry.Name, err)
	}

	// Inject mode-derived --profile flags (before subcommand args)
	for _, profile := range pctx.ComposeProfiles {
		args = append(args, "--profile", profile)
	}

	args = append(args, extraArgs...)

	// Append mode-derived service names (only for "up" subcommand)
	if pctx.ComposeServices != nil && len(*pctx.ComposeServices) > 0 {
		if len(extraArgs) > 0 && extraArgs[0] == "up" {
			args = append(args, *pctx.ComposeServices...)
		}
	}

	return cmd, args, nil
}

// composeWorkdir returns the working directory a sourced compose entry runs in.
// Sourced entries (TASK-051) execute in their fetched/referenced source dir so
// relative compose files, default file discovery, .env and build contexts
// resolve as the external stack's author intended (matching the legacy
// `cd <dir> && docker compose` behavior). Returns "" for non-sourced entries,
// which then inherit the current working directory.
func composeWorkdir(pctx *PluginContext) string {
	if src := pctx.Entry.Source; src != nil {
		if d, err := config.SourceDir(src, pctx.Entry.Name, pctx.ConfigDir); err == nil {
			return d
		}
	}
	return ""
}

// runSubprocess executes a docker compose command as a subprocess.
//
// A failure is inspected before it is returned: when the daemon is the reason, the bare
// exit status is replaced by the diagnosis DVA already implements. Sitting here rather
// than in Up means up, down and stop all report the condition the same way — the failure
// is identical for all three, so the explanation should be too.
//
// The probe runs only after a failure, never before. A pre-flight daemon check would make
// every successful command pay a subprocess round-trip for a condition that is rare, and
// the preflight that does exist (preflightConfig) is justified precisely because it needs
// no daemon.
func (p *ComposePlugin) runSubprocess(pctx *PluginContext, args []string) error {
	cmd, cmdArgs, err := p.buildArgs(pctx, args)
	if err != nil {
		return err
	}
	pctx.Logger.Debug("compose subprocess", "command", cmd, "args", cmdArgs)
	runErr := dvaexec.ExecSubprocessInDir(pctx.Env, composeWorkdir(pctx), cmd, cmdArgs, false)
	if runErr == nil {
		return nil
	}

	// Only docker's daemon can answer for docker's failure. A runner the config replaced
	// via compose.command (podman-compose, nerdctl) has its own daemon story, and naming
	// Docker's there would be a guess dressed as a diagnosis. filepath.Base so an absolute
	// /usr/local/bin/docker is still recognised.
	if filepath.Base(cmd) != "docker" || DockerDaemonReachable(pctx.Env) {
		return runErr
	}
	op := ""
	if len(args) > 0 {
		op = args[0]
	}
	return &DockerDaemonError{Op: op, cause: runErr}
}

// preflightConfig runs `docker compose ... config --quiet` to validate the
// compose file set (including that every -f and include: target resolves)
// before `up`. Output is captured so a broken reference surfaces through a
// ComposeConfigError carrying docker's own diagnostic, rather than being
// streamed raw ahead of a bare exit status.
func (p *ComposePlugin) preflightConfig(pctx *PluginContext) error {
	cmd, cmdArgs, err := p.buildArgs(pctx, []string{"config", "--quiet"})
	if err != nil {
		return err
	}
	pctx.Logger.Debug("compose preflight", "command", cmd, "args", cmdArgs)
	out, err := dvaexec.ExecSubprocessCaptureInDir(pctx.Env, composeWorkdir(pctx), cmd, cmdArgs, false)
	if err != nil {
		return &ComposeConfigError{
			Files:  pctx.Entry.ComposeConfig().Files,
			Detail: strings.TrimSpace(out),
			cause:  err,
		}
	}
	return nil
}

// ComposeConfigError reports that the pre-up `docker compose config` preflight
// failed — typically a file referenced by -f or include: that does not resolve,
// or invalid/merge-conflicting YAML. It carries docker's own diagnostic (Detail)
// so the real cause surfaces instead of a bare exit status, plus a remediation
// hint. Mirrors ResolveError's cause-wrapping shape (Error/Unwrap).
type ComposeConfigError struct {
	Files  []string
	Detail string
	cause  error
}

func (e *ComposeConfigError) Error() string {
	msg := "compose config is invalid"
	if e.Detail != "" {
		msg += ": " + firstLine(e.Detail)
	}
	msg += "\n       → a compose file referenced by -f or include: does not resolve or is invalid"
	msg += "\n       → check compose.files in dva.yml and any include: paths, then run: docker compose config"
	return msg
}

func (e *ComposeConfigError) Unwrap() error { return e.cause }

// firstLine returns the first line of s (docker's diagnostics are typically a
// single line; guard against multi-line output leaking into the summary).
func firstLine(s string) string {
	if before, _, ok := strings.Cut(s, "\n"); ok {
		return strings.TrimSpace(before)
	}
	return s
}

// composeServiceInfo mirrors the parts of `compose ps --format json` that dva reads, from
// either compose plugin. docker compose fills Service and encodes Labels as a comma-separated
// string; podman-compose has no Service, keeps container names in Names and its labels in an
// object. Both shapes must parse, so Labels decides its own encoding.
type composeServiceInfo struct {
	Name       string             `json:"Name"`
	Service    string             `json:"Service"`
	Names      []string           `json:"Names"`
	Labels     composeLabels      `json:"Labels"`
	State      string             `json:"State"`
	ExitCode   int                `json:"ExitCode"`
	Health     string             `json:"Health"`
	Publishers []composePublisher `json:"Publishers"`
}

// composeLabels holds the two label encodings the compose plugins emit: an object
// (podman ps: {"io.podman.compose.service":"db"}) and docker compose's "k=v,k=v" string.
// An unrecognized shape is dropped rather than failing the whole container record: a name
// resolved from the other fields is better than losing the container from the status.
type composeLabels map[string]string

func (l *composeLabels) UnmarshalJSON(data []byte) error {
	s := strings.TrimSpace(string(data))
	switch {
	case s == "", s == "null":
		return nil
	case strings.HasPrefix(s, "{"):
		var m map[string]string
		if err := json.Unmarshal([]byte(s), &m); err == nil {
			*l = m
		}
		return nil
	}
	var raw string
	if err := json.Unmarshal([]byte(s), &raw); err != nil {
		return nil
	}
	labels := make(composeLabels)
	for pair := range strings.SplitSeq(raw, ",") {
		k, v, ok := strings.Cut(pair, "=")
		if !ok || k == "" {
			continue
		}
		labels[k] = v
	}
	*l = labels
	return nil
}

// serviceNameOf resolves the compose service a container belongs to. Service is authoritative
// when present; podman-compose leaves it empty, so the compose service label is read next
// (either plugin's key) and the container name is the last resort. Returning "" keeps an
// unresolved record unresolved instead of giving it another container's name.
func serviceNameOf(info composeServiceInfo) string {
	if name := strings.TrimSpace(info.Service); name != "" {
		return name
	}
	for _, key := range [...]string{"com.docker.compose.service", "io.podman.compose.service"} {
		if name := strings.TrimSpace(info.Labels[key]); name != "" {
			return name
		}
	}
	for _, name := range info.Names {
		if name = strings.TrimSpace(name); name != "" {
			return name
		}
	}
	return ""
}

type composePublisher struct {
	TargetPort    int `json:"TargetPort"`
	PublishedPort int `json:"PublishedPort"`
}

// Status queries list every container of the project, stopped ones included: a project whose
// containers all exited or crashed must report those services (and so read "not ready"), not an
// empty list that reads as "nothing to judge". docker compose ps hides stopped containers
// unless given -a. podman-compose ps already passes -a to `podman ps` itself and its argument
// parser has no -a (podman-compose 1.6.0 compose_ps_parse), so passing one there would make
// every podman-compose status query fail.
const (
	dockerComposePsAllFlag = "-a"
	podmanComposePsAllFlag = ""
)

// composePsArgs builds the `ps` arguments both compose plugins use for status queries: the
// tool's include-stopped flag (empty when the tool always includes them) and JSON output.
func composePsArgs(allFlag string) []string {
	args := []string{"ps"}
	if allFlag != "" {
		args = append(args, allFlag)
	}
	return append(args, "--format", "json")
}

// statusQueryArgs is the full docker compose ps command line of a status query.
func (p *ComposePlugin) statusQueryArgs(pctx *PluginContext) (string, []string, error) {
	return p.buildArgs(pctx, composePsArgs(dockerComposePsAllFlag))
}

// queryServices runs docker compose ps and returns parsed service statuses.
func (p *ComposePlugin) queryServices(pctx *PluginContext) ([]ServiceStatus, error) {
	return p.queryServicesContext(context.Background(), pctx)
}
func (p *ComposePlugin) queryServicesContext(ctx context.Context, pctx *PluginContext) ([]ServiceStatus, error) {
	if pctx.Entry.ComposeConfig() == nil {
		return nil, nil
	}

	cmd, cmdArgs, err := p.statusQueryArgs(pctx)
	if err != nil {
		return nil, err
	}
	c := exec.Command(cmd, cmdArgs...)
	c.Dir = composeWorkdir(pctx)
	out, err := runStatusQuery(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("compose ps: %w", err)
	}

	return parseComposeServicesJSON(out)
}

// parseComposeServicesJSON parses docker compose ps JSON output (array or JSON-lines)
// into ServiceStatus slice. Shared by ComposePlugin and PodmanComposePlugin.
func parseComposeServicesJSON(out []byte) ([]ServiceStatus, error) {
	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return nil, nil
	}

	var infos []composeServiceInfo
	if err := json.Unmarshal(out, &infos); err != nil {
		// Try JSON lines format
		for line := range strings.SplitSeq(trimmed, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			var info composeServiceInfo
			if err := json.Unmarshal([]byte(line), &info); err != nil {
				fmt.Fprintf(os.Stderr, "[warn] failed to parse compose service info: %v\n", err)
				continue
			}
			infos = append(infos, info)
		}
	}

	// A container that exited with code 0 while another container of the project still runs is
	// a one-shot that completed (an init or migration job). The query lists stopped containers,
	// but such a job is left out, as it was before the query asked for them, so it neither reads
	// as a stopped service nor turns a healthy project "not ready". When nothing runs, every
	// container is kept: an exited project is reported as one. A non-zero exit is always kept.
	anyRunning := false
	for _, info := range infos {
		anyRunning = anyRunning || serviceLooksRunning(info.State)
	}

	services := make([]ServiceStatus, 0, len(infos))
	for _, info := range infos {
		if anyRunning && info.ExitCode == 0 && strings.EqualFold(strings.TrimSpace(info.State), "exited") {
			continue
		}
		ports := make(map[int]int)
		for _, pub := range info.Publishers {
			if pub.PublishedPort > 0 {
				ports[pub.PublishedPort] = pub.TargetPort
			}
		}
		services = append(services, ServiceStatus{
			Name:   serviceNameOf(info),
			State:  info.State,
			Health: info.Health,
			Ports:  ports,
		})
	}

	return services, nil
}

// instanceComposeConfig gives an instance its own compose project: what compose would have
// named the ordinary run's project, with the instance appended. Without it, two instances in one
// config directory would be the same compose project and the second `up` would recreate the
// first one's containers. The ordinary run (no instance) is returned as is.
func instanceComposeConfig(pctx *PluginContext, cfg *config.ComposePluginConfig, baseDir string) *config.ComposePluginConfig {
	if pctx.Instance == "" {
		return cfg
	}
	var out config.ComposePluginConfig
	if cfg != nil {
		out = *cfg
	}
	out.ProjectName = instanceProjectName(pctx, out.ProjectName, out.Files, baseDir)
	return &out
}

// instanceProjectName is the compose project of an instance: the project the ordinary run would
// get, then "-" and the instance, normalized the way compose requires an explicit project name
// to be (lowercase letters, digits, - and _, starting with a letter or digit). Compose only
// normalizes a name it derives itself, so "my.app-A" would be rejected.
//
// The ordinary run's project is the declared project_name, else the top-level `name:` of the
// first compose file, else the base name of that file's directory (baseDir when there is no
// file). Following compose here, not inventing a scheme, keeps entries that share a project in
// a plain run (two files in one directory share a network) sharing one per instance, and
// entries that were separate projects separate.
func instanceProjectName(pctx *PluginContext, declared string, files []string, baseDir string) string {
	base := pctx.Env.Interpolate(declared)
	if base == "" {
		base = composeDerivedProject(pctx, files, baseDir)
	}
	name := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		case r >= 'A' && r <= 'Z':
			return r + ('a' - 'A')
		}
		return '-'
	}, base+"-"+pctx.Instance)
	return strings.TrimLeft(name, "-_")
}

// composeDerivedProject is the project name compose derives when none is given: the first
// file's top-level `name:` if it has one, else the base name of the file's directory.
func composeDerivedProject(pctx *PluginContext, files []string, baseDir string) string {
	dir := baseDir
	if len(files) > 0 {
		first := pctx.Env.Interpolate(files[0])
		if !filepath.IsAbs(first) {
			first = filepath.Join(baseDir, first)
		}
		dir = filepath.Dir(first)
		if data, err := os.ReadFile(first); err == nil {
			var doc struct {
				Name string `yaml:"name"`
			}
			if yaml.Unmarshal(data, &doc) == nil && doc.Name != "" && !strings.Contains(doc.Name, "$") {
				return doc.Name
			}
		}
	}
	return filepath.Base(dir)
}
