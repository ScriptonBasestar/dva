package ociexec

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

var containerIDPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)
var imageIDPattern = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)
var identityPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$`)

// Run creates only a fresh, labelled container, never an existing service.
// Its caller must supply a validated CIOCI and a finite context deadline.
func Run(ctx context.Context, opts Options) (result Report, retErr error) {
	result.Image = opts.Config.Image
	result.Requested = Limits{PidsLimit: opts.Config.PidsLimit, MemoryBytes: opts.Config.MemoryBytes, CPUs: opts.Config.CPUs, OutputLimitBytes: opts.Config.OutputLimitBytes}
	result.Cleanup = "not-created"
	deadline, bounded := ctx.Deadline()
	if !bounded || time.Until(deadline) <= 0 {
		return result, errors.New("OCI step requires a live finite deadline")
	}
	if !identityPattern.MatchString(opts.RunID) || !identityPattern.MatchString(opts.Step) || opts.StateDir == "" {
		return result, errors.New("OCI step requires a bounded CI run/step identity and state directory")
	}
	budget := time.Until(deadline).String()
	declaration := config.Config{CI: &config.CIConfig{Profiles: map[string]config.CIProfile{"oci": {
		Timeout: budget, MaxParallel: 1, Steps: []config.CIStep{{Name: opts.Step, OCI: &opts.Config, Timeout: budget}},
	}}}}
	if _, err := declaration.ResolveCIProfile("oci"); err != nil {
		return result, err
	}
	stateBase, err := secureStateDirectory(opts.StateDir)
	if err != nil {
		return result, err
	}
	state, err := secureStateDirectory(filepath.Join(stateBase, "oci"))
	if err != nil {
		return result, err
	}
	private, err := os.MkdirTemp(state, opts.RunID+"-"+opts.Step+"-")
	if err != nil {
		return result, err
	}
	privateRoot, err := os.OpenRoot(private)
	if err != nil {
		return result, err
	}
	defer func() { _ = privateRoot.Close() }()
	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		_ = os.Remove(private)
		return result, err
	}
	ownership := intent{Version: 1, RunID: opts.RunID, Step: opts.Step, Nonce: hex.EncodeToString(nonceBytes), Snapshot: filepath.Join(private, "input")}
	ownership.Name = "dva-ci-oci-" + ownership.Nonce
	result.IntentPath = filepath.Join(private, "intent.json")
	if err := privateRoot.Mkdir("input", 0o700); err != nil {
		return result, err
	}
	created, attached := false, false
	var docker *dockerClient
	defer func() {
		primary := retErr
		if primary != nil {
			result.ExecutionError = primary.Error()
		}
		// Cancellation must not cancel observation/removal of the owned object.
		finish, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		if created {
			if attached {
				observed, inspectErr := docker.inspect(finish, ownership.reference())
				if inspectErr == nil && ownsContainer(observed, ownership) {
					result.OOMKilled = observed.State.OOMKilled
					if observed.State.Status == "exited" && !observed.State.Running {
						code := observed.State.ExitCode
						result.ExitCode = &code
						if code != 0 || result.OOMKilled {
							primary = errors.Join(primary, fmt.Errorf("OCI workload exited %d (OOM=%t)", code, result.OOMKilled))
						}
					} else if primary == nil {
						primary = errors.New("OCI attach ended without an observed workload exit")
					}
				} else {
					primary = errors.Join(primary, errors.New("OCI workload outcome could not be observed with its bound identity"))
				}
			}
			result.Cleanup, result.CleanupError = docker.cleanup(finish, &ownership, result.IntentPath)
			result.ContainerID = ownership.ContainerID
		}
		if primary != nil {
			result.ExecutionError = primary.Error()
		}
		if result.CleanupError != "" {
			primary = errors.Join(primary, errors.New(result.CleanupError))
		}
		if !created || result.CleanupError == "" {
			if err := removeSnapshotAt(privateRoot); err != nil {
				result.CleanupError = "OCI snapshot cleanup: " + err.Error()
				primary = errors.Join(primary, errors.New(result.CleanupError))
			}
		}
		ownership.Result = result
		if err := persistIntent(result.IntentPath, ownership); err != nil {
			primary = errors.Join(primary, fmt.Errorf("persist OCI outcome: %w", err))
		}
		retErr = primary
	}()
	docker, err = newDocker(ctx, filepath.Join(private, "docker"))
	if err != nil {
		return result, err
	}
	result.DockerBinary, result.DockerSHA256, result.Endpoint, result.DaemonID = docker.binary, docker.binarySHA, docker.endpoint, docker.daemonID
	result.DockerConfigPath = docker.configDir
	ownership.Endpoint, ownership.DaemonID = docker.endpoint, docker.daemonID
	data, err := docker.metadata(ctx, "image", "inspect", opts.Config.Image)
	if err != nil {
		return result, fmt.Errorf("OCI image must already be available locally: %w", err)
	}
	var images []imageInfo
	if err := json.Unmarshal(data, &images); err != nil || len(images) != 1 || !imageIDPattern.MatchString(images[0].ID) || !imageMatches(images[0], opts.Config.Image) {
		return result, errors.New("OCI local image does not match its immutable digest")
	}
	if len(images[0].Config.Volumes) != 0 {
		return result, errors.New("OCI images declaring VOLUME are unsupported")
	}
	result.ImageID = images[0].ID
	result.InputSHA256, err = snapshotInput(opts.Root, opts.Config.Source, ownership.Snapshot)
	if err != nil {
		return result, err
	}
	argvJSON, err := json.Marshal(opts.Config.Argv)
	if err != nil {
		return result, err
	}
	argvDigest := sha256.Sum256(argvJSON)
	result.ArgvSHA256 = hex.EncodeToString(argvDigest[:])
	ownership.Result = result
	// This checkpoint precedes the first mutating Docker command.
	if err := persistIntent(result.IntentPath, ownership); err != nil {
		return result, err
	}
	if opts.Output != nil {
		_, _ = fmt.Fprintf(opts.Output, "ci OCI intent: %s\n", result.IntentPath)
	}
	if err := docker.sameDaemon(ctx); err != nil {
		return result, err
	}
	seconds := max(int64(1), int64(time.Until(deadline).Seconds())+1)
	args := createArgs(opts.Config, ownership, result.ImageID, seconds)
	created = true // a lost create response still needs identity-bound reconciliation
	result.Cleanup = "pending"
	data, err = docker.metadata(ctx, args...)
	if err != nil {
		return result, fmt.Errorf("create OCI container: %w", err)
	}
	ownership.ContainerID = strings.TrimSpace(string(data))
	if !containerIDPattern.MatchString(ownership.ContainerID) {
		return result, errors.New("docker create returned no full container ID")
	}
	result.ContainerID = ownership.ContainerID
	ownership.Result = result
	if err := persistIntent(result.IntentPath, ownership); err != nil {
		return result, err
	}
	observed, err := docker.inspect(ctx, ownership.reference())
	if err != nil {
		return result, err
	}
	if err := verifyPolicy(observed, ownership, images[0], opts.Config, args); err != nil {
		return result, err
	}
	result.Observed = &AppliedLimits{PidsLimit: observed.HostConfig.PidsLimit, MemoryBytes: observed.HostConfig.Memory, MemorySwapBytes: observed.HostConfig.MemorySwap, NanoCPUs: observed.HostConfig.NanoCPUs}
	if err := docker.sameDaemon(ctx); err != nil {
		return result, err
	}
	ownership.Result = result
	if err := persistIntent(result.IntentPath, ownership); err != nil {
		return result, err
	}
	startCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	stream := &capture{limit: opts.Config.OutputLimitBytes, cancel: cancel, writer: opts.Output}
	cmd := docker.command(startCtx, "container", "start", "--attach", ownership.ContainerID)
	cmd.Stdout, cmd.Stderr = stream, stream
	attached = true
	err = cmd.Run()
	payload, truncated := stream.data()
	result.OutputBytes, result.OutputTruncated = int64(len(payload)), truncated
	result.Kernel, retErr = observedKernel(payload, ownership.Nonce, opts.Config)
	if ctx.Err() != nil {
		retErr = errors.Join(retErr, ctx.Err())
	}
	if truncated {
		retErr = errors.Join(retErr, errors.New("OCI output limit exceeded"))
	}
	if err != nil {
		retErr = errors.Join(retErr, fmt.Errorf("docker OCI attach: %w", err))
	}
	return result, retErr
}

func (d *dockerClient) sameDaemon(ctx context.Context) error {
	sha, err := binaryDigest(d.binary)
	if err != nil || sha != d.binarySHA {
		return errors.New("OCI Docker executable identity changed")
	}
	info, err := d.info(ctx)
	if err != nil {
		return err
	}
	if info.ID != d.daemonID {
		return errors.New("OCI Docker daemon identity changed")
	}
	return nil
}

func (d *dockerClient) cleanup(ctx context.Context, ownership *intent, intentPath string) (string, string) {
	fail := func(err error) (string, string) { return "failed", "OCI container cleanup: " + err.Error() }
	if err := d.sameDaemon(ctx); err != nil {
		return fail(err)
	}
	info, err := d.inspect(ctx, ownership.reference())
	if err != nil {
		absent, listErr := d.absent(ctx, ownership.reference())
		if listErr == nil && absent && ownership.ContainerID != "" {
			return "absent", ""
		}
		if ownership.ContainerID == "" {
			return fail(errors.New("create result is ambiguous; absence cannot exclude a late create"))
		}
		return fail(errors.New("cannot establish whether the intended OCI container is absent"))
	}
	if !ownsContainer(info, *ownership) || !containerIDPattern.MatchString(info.ID) {
		return fail(errors.New("container identity/labels differ; no removal attempted"))
	}
	if ownership.ContainerID == "" {
		ownership.ContainerID = info.ID
		ownership.Result.ContainerID = info.ID
		if err := persistIntent(intentPath, *ownership); err != nil {
			return fail(err)
		}
	}
	if _, err := d.metadata(ctx, "container", "rm", "--force", "--volumes", info.ID); err != nil {
		return fail(err)
	}
	absent, err := d.absent(ctx, ownership.ContainerID)
	if err != nil || !absent {
		return fail(errors.New("container removal has no successful absence proof"))
	}
	return "removed", ""
}

func (i intent) reference() string {
	if i.ContainerID != "" {
		return i.ContainerID
	}
	return i.Name
}

var _ io.Writer = (*capture)(nil)
