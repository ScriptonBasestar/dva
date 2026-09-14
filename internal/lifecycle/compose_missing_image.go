package lifecycle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	dvaexec "github.com/ScriptonBasestar/dva/internal/exec"
)

// missingImageProbeTimeout bounds the whole probe — one `compose config` plus one
// `docker image inspect` per image-only service — not each call separately. The budget
// belongs to the diagnosis because that is what must not outlive its usefulness: a
// per-call bound still lets a stalled daemon burn timeout x N before returning.
//
// It is longer than dockerDaemonProbeTimeout because it covers more work, and it exists
// for the same stated reason (docker_daemon.go): this runs on a path that has already
// failed and must not be able to turn a failed command into a hang. On expiry the probe
// yields nothing and the original error stands, exactly as when compose config fails.
// A var, not a const, so a test can shrink it and exercise the expiry branch without
// needing a process that actually stalls.
var missingImageProbeTimeout = 20 * time.Second

// MissingLocalImageError reports that a failed `compose up` left images that compose
// itself cannot produce: declared with `image:` and no `build:`, and still absent from
// the local image store after the failure.
//
// It is the third member of the family DockerDaemonError and ComposeConfigError started —
// lifecycle failures DVA answers at the failure point instead of relaying docker's raw
// stderr followed by a bare exit status. The condition it names is the one a user cannot
// read off that stderr: of 27 lines, one says `pull access denied` and 26 are the
// cascade compose produces after cancelling the remaining pulls.
//
// Images is a candidate list, not a verdict, and Error() says so. See
// missingLocalImages for why the boundary is drawn there.
type MissingLocalImageError struct {
	// Images are the image references that have no build: section and were not present
	// locally after the failure, sorted for a stable message.
	Images []string
	cause  error
}

func (e *MissingLocalImageError) Error() string {
	msg := fmt.Sprintf("%d image(s) have no build: section and are not present locally: %s",
		len(e.Images), strings.Join(e.Images, ", "))
	msg += "\n       → at least one of them could not be pulled; compose cannot build any of them, so whichever is the cause must be pulled or built and tagged outside compose"
	msg += "\n       → compose cancels the remaining pulls once one fails, so the others may only have been interrupted — this is a candidate list, not a verdict"
	msg += "\n       → re-run the same up to narrow it down: an interrupted pull succeeds on the retry, and what stays is the cause"
	return msg
}

func (e *MissingLocalImageError) Unwrap() error { return e.cause }

// diagnoseMissingLocalImages turns a failed `compose up` into a MissingLocalImageError
// when the post-hoc probe finds image-only services whose images are still absent.
// Returns runErr unchanged otherwise, so the caller can wrap it exactly once.
//
// DockerDaemonError wins: an unreachable daemon fails every pull and every inspect, so a
// list of "missing" images there would be an artefact of the outage, not a diagnosis of
// it. The probe also runs only on a failure — a successful up spawns no extra subprocess,
// the same principle DockerDaemonReachable's placement already established.
func (p *ComposePlugin) diagnoseMissingLocalImages(pctx *PluginContext, runErr error) error {
	if runErr == nil {
		return nil
	}
	if _, ok := errors.AsType[*DockerDaemonError](runErr); ok {
		return runErr
	}
	// Only docker's image store can answer for docker's pull. A runner the config
	// replaced via compose.command (podman-compose, nerdctl) keeps images somewhere
	// else, and `docker image inspect` would report every one of them missing.
	cmd, _, err := p.buildArgs(pctx, nil)
	if err != nil || filepath.Base(cmd) != "docker" {
		return runErr
	}

	missing := p.missingLocalImages(pctx)
	if len(missing) == 0 {
		return runErr
	}
	return &MissingLocalImageError{Images: missing, cause: runErr}
}

// composeConfigServices is the slice of `docker compose config --format json` this
// diagnosis needs: which services declare an image, and which of those can build it.
type composeConfigServices struct {
	Services map[string]struct {
		Image string          `json:"image"`
		Build json.RawMessage `json:"build"`
	} `json:"services"`
}

// missingLocalImages returns the images of `image:`-only services that are absent from
// the local image store, sorted. An empty result means the diagnosis does not apply.
//
// Why this is the boundary — and why the result is reported as candidates rather than as
// a verdict. Compose cancels every in-flight pull as soon as one is denied, so after the
// failure the causal image and its interrupted victims are indistinguishable by presence
// alone: all of them are absent, and all of them are image-only. The two ways to tell
// them apart are both worse than saying so:
//
//   - Re-querying the registry, or `compose pull --ignore-pull-failures`, costs a second
//     network round trip on a path that has already failed and can fail for new reasons
//     of its own.
//   - Name-shape heuristics cannot do it at all: nothing in `sigdock-idp:latest`
//     (local-only, never pullable) distinguishes it from `redis:7-alpine` (a Docker Hub
//     official image), and a registry host in the name proves neither.
//
// Reporting 13 interrupted victims as "13 images you must build" would be a worse answer
// than today's bare exit status. Reporting them as candidates, with the cancellation
// stated, is not — and a re-run narrows the list for free, because an interrupted pull
// succeeds the second time.
func (p *ComposePlugin) missingLocalImages(pctx *PluginContext) []string {
	ctx, cancel := context.WithTimeout(context.Background(), missingImageProbeTimeout)
	defer cancel()

	cfg, err := p.composeConfigJSON(ctx, pctx)
	if err != nil {
		// The probe is advisory: if the project cannot be described, the original
		// failure is still the honest answer.
		pctx.Logger.Debug("missing-image probe: compose config failed", "error", err)
		return nil
	}

	seen := make(map[string]bool)
	var missing []string
	for _, svc := range cfg.Services {
		if svc.Image == "" || hasBuild(svc.Build) {
			continue
		}
		if seen[svc.Image] {
			continue
		}
		seen[svc.Image] = true
		if !p.imagePresentLocally(ctx, pctx, svc.Image) {
			// A cancelled inspect also returns false. Absent-because-unanswered is
			// not absent, so the deadline ends the probe rather than contributing a
			// name to the list.
			if ctx.Err() != nil {
				pctx.Logger.Debug("missing-image probe: deadline reached", "error", ctx.Err())
				return nil
			}
			missing = append(missing, svc.Image)
		}
	}
	sort.Strings(missing)
	return missing
}

// hasBuild reports whether a service's build: key carries a value.
//
// Measured against Docker Compose 5.5.1: for a service that only declares an image the
// key is *omitted entirely*, not emitted as null — so the nil RawMessage is the shape
// that matters, and the null case is kept only because nothing in compose's output
// contract promises it will stay omitted. Short-form `build: .` is normalised to an
// object, so it reads as a value here.
func hasBuild(raw json.RawMessage) bool {
	t := strings.TrimSpace(string(raw))
	return t != "" && t != "null"
}

// composeConfigJSON runs `docker compose ... config --format json`, which parses and
// merges the compose file set without needing the daemon — the same call family
// preflightConfig already makes.
func (p *ComposePlugin) composeConfigJSON(ctx context.Context, pctx *PluginContext) (*composeConfigServices, error) {
	cmd, cmdArgs, err := p.buildArgs(pctx, []string{"config", "--format", "json"})
	if err != nil {
		return nil, err
	}
	// buildArgs appends mode-derived service names only to `up`, so config would
	// otherwise describe the whole project while the up that failed touched a subset.
	// Diagnosing a service the run never started names a confident wrong image in place
	// of the real failure — worse than the bare exit status this replaces. Scope the
	// probe to exactly what ran.
	if pctx.ComposeServices != nil && len(*pctx.ComposeServices) > 0 {
		cmdArgs = append(cmdArgs, *pctx.ComposeServices...)
	}
	pctx.Logger.Debug("missing-image probe", "command", cmd, "args", cmdArgs)
	out, err := dvaexec.ExecSubprocessCaptureInDirContext(ctx, pctx.Env, composeWorkdir(pctx), cmd, cmdArgs, false)
	if err != nil {
		return nil, fmt.Errorf("compose config: %w", err)
	}
	var cfg composeConfigServices
	if err := json.Unmarshal([]byte(out), &cfg); err != nil {
		return nil, fmt.Errorf("parse compose config: %w", err)
	}
	return &cfg, nil
}

// imagePresentLocally reports whether `docker image inspect` finds the reference. One
// call per image rather than one batched call: a batch exits non-zero when any member is
// missing and names the rest only in stderr prose, which is exactly the string matching
// this diagnosis exists to avoid. The daemon is known reachable here — runSubprocess
// consulted it before handing the failure on.
func (p *ComposePlugin) imagePresentLocally(ctx context.Context, pctx *PluginContext, image string) bool {
	_, err := dvaexec.ExecSubprocessCaptureInDirContext(ctx, pctx.Env, composeWorkdir(pctx), "docker",
		[]string{"image", "inspect", image}, false)
	return err == nil
}
