package ociexec

import (
	"encoding/csv"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/ScriptonBasestar/dva/internal/config"
)

const labelRun = "io.dva.ci.run"
const labelStep = "io.dva.ci.step"
const labelNonce = "io.dva.ci.nonce"

// Only a prepared, trusted image with /bin/sh and /usr/bin/timeout is supported.
// The PID namespace remains bounded even if the outer supervisor is killed.
// An external SIGKILL can leave a running or stopped container for recovery.
// The inner timeout is cooperative, not a deadline against a hostile payload.
const bootstrap = `set -eu
nonce=$1; expected_pids=$2; expected_memory=$3; expected_cpus=$4; seconds=$5
shift 5
test -f /sys/fs/cgroup/cgroup.controllers
read -r pids < /sys/fs/cgroup/pids.max
read -r memory < /sys/fs/cgroup/memory.max
read -r swap < /sys/fs/cgroup/memory.swap.max
read -r quota period < /sys/fs/cgroup/cpu.max
test "$pids" = "$expected_pids"
test "$memory" = "$expected_memory"
test "$swap" = 0
case "$quota:$period" in *[!0-9:]*|'') exit 125;; esac
test "$period" -gt 0
test "$quota" -eq "$((period * expected_cpus))"
test -x /usr/bin/timeout
printf 'DVA_OCI_LIMITS:%s:%s:%s:%s:%s:%s\n' "$nonce" "$pids" "$memory" "$swap" "$quota" "$period"
mkdir -p /tmp/home /tmp/config /tmp/data /tmp/cache /tmp/state /tmp/runtime
exec /usr/bin/timeout -s TERM -k 1 "$seconds" "$@"
`

var isolatedEnv = map[string]string{
	"HOME": "/tmp/home", "XDG_CONFIG_HOME": "/tmp/config", "XDG_DATA_HOME": "/tmp/data",
	"XDG_CACHE_HOME": "/tmp/cache", "XDG_STATE_HOME": "/tmp/state", "XDG_RUNTIME_DIR": "/tmp/runtime", "TMPDIR": "/tmp",
}

func imageMatches(image imageInfo, requested string) bool {
	if !strings.Contains(requested, "@") {
		return image.ID == requested
	}
	_, digest, _ := strings.Cut(requested, "@")
	for _, recorded := range image.RepoDigests {
		if strings.HasSuffix(recorded, "@"+digest) {
			return true
		}
	}
	return false
}

func createArgs(cfg config.CIOCI, ownership intent, imageID string, seconds int64) []string {
	var mount strings.Builder
	csvWriter := csv.NewWriter(&mount)
	_ = csvWriter.Write([]string{"type=bind", "source=" + ownership.Snapshot, "target=/workspace", "readonly"})
	csvWriter.Flush()
	workdir := cfg.Workdir
	if workdir == "" {
		workdir = "/workspace"
	}
	args := []string{"container", "create", "--name", ownership.Name,
		"--pull=never", "--restart=no", "--init", "--user", "0:0", "--network", "none",
		"--cgroupns", "private", "--ipc", "private", "--read-only", "--cap-drop", "ALL",
		"--security-opt", "no-new-privileges", "--no-healthcheck", "--log-driver", "none",
		"--pids-limit", strconv.FormatInt(cfg.PidsLimit, 10), "--memory", strconv.FormatInt(cfg.MemoryBytes, 10),
		"--memory-swap", strconv.FormatInt(cfg.MemoryBytes, 10), "--cpus", strconv.Itoa(cfg.CPUs),
		"--tmpfs", "/tmp:rw,nosuid,nodev,exec,size=" + strconv.FormatInt(cfg.MemoryBytes/2, 10) + ",mode=1777",
		"--mount", strings.TrimSuffix(mount.String(), "\n"), "--workdir", workdir,
		"--label", labelRun + "=" + ownership.RunID, "--label", labelStep + "=" + ownership.Step,
		"--label", labelNonce + "=" + ownership.Nonce, "--entrypoint", "/bin/sh"}
	values := map[string]string{}
	maps.Copy(values, cfg.Environment)
	maps.Copy(values, isolatedEnv)
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		args = append(args, "--env", key+"="+values[key])
	}
	args = append(args, imageID, "-eu", "-c", bootstrap, "dva-ci-oci", ownership.Nonce,
		strconv.FormatInt(cfg.PidsLimit, 10), strconv.FormatInt(cfg.MemoryBytes, 10), strconv.Itoa(cfg.CPUs), strconv.FormatInt(seconds, 10))
	return append(args, cfg.Argv...)
}

func ownsContainer(info containerInfo, expected intent) bool {
	identity := info.Name == "/"+expected.Name
	if expected.ContainerID != "" {
		identity = info.ID == expected.ContainerID
	}
	return identity && info.Config.Labels[labelRun] == expected.RunID &&
		info.Config.Labels[labelStep] == expected.Step && info.Config.Labels[labelNonce] == expected.Nonce &&
		(expected.ContainerID == "" || info.ID == expected.ContainerID)
}

func verifyPolicy(info containerInfo, expected intent, image imageInfo, cfg config.CIOCI, args []string) error {
	imageID := image.ID
	host := info.HostConfig
	if !ownsContainer(info, expected) || info.Image != imageID || info.State.Running || info.State.Status != "created" {
		return errors.New("created OCI container identity/image/state differs from intent")
	}
	if host.PidsLimit != cfg.PidsLimit || host.Memory != cfg.MemoryBytes || host.MemorySwap != cfg.MemoryBytes ||
		host.NanoCPUs != int64(cfg.CPUs)*1_000_000_000 || host.NetworkMode != "none" || host.CgroupnsMode != "private" ||
		!host.ReadonlyRootfs || host.Privileged || host.AutoRemove || host.PidMode != "" || host.IpcMode != "private" ||
		host.Init == nil || !*host.Init || host.LogConfig.Type != "none" || host.RestartPolicy.Name != "no" ||
		!slices.Equal(host.CapDrop, []string{"ALL"}) || len(host.CapAdd) != 0 ||
		(!slices.Contains(host.SecurityOpt, "no-new-privileges") && !slices.Contains(host.SecurityOpt, "no-new-privileges=true")) {
		return errors.New("docker did not apply the requested OCI containment policy")
	}
	index := slices.Index(args, imageID)
	workdir := cfg.Workdir
	if workdir == "" {
		workdir = "/workspace"
	}
	if info.Config.User != "0:0" || !slices.Equal(info.Config.Entrypoint, []string{"/bin/sh"}) || index < 0 ||
		info.Config.WorkingDir != workdir || !slices.Equal(info.Config.Cmd, args[index+1:]) || !slices.Equal(info.Config.Healthcheck.Test, []string{"NONE"}) {
		return errors.New("docker did not apply the trusted OCI entrypoint")
	}
	env := map[string]string{}
	for _, entry := range info.Config.Env {
		key, value, _ := strings.Cut(entry, "=")
		env[key] = value
	}
	for key, value := range isolatedEnv {
		if env[key] != value {
			return fmt.Errorf("docker did not isolate OCI %s", key)
		}
	}
	expectedEnv := map[string]string{}
	for _, entry := range image.Config.Env {
		key, value, _ := strings.Cut(entry, "=")
		expectedEnv[key] = value
	}
	maps.Copy(expectedEnv, cfg.Environment)
	maps.Copy(expectedEnv, isolatedEnv)
	if len(env) != len(expectedEnv) {
		return errors.New("docker injected undeclared OCI environment")
	}
	for key, value := range expectedEnv {
		if env[key] != value {
			return errors.New("docker OCI environment differs from declared/image inputs")
		}
	}
	binds := 0
	for _, mount := range info.Mounts {
		if mount.Type == "bind" && mount.Source == expected.Snapshot && mount.Destination == "/workspace" && !mount.RW {
			binds++
			continue
		}
		if mount.Type == "tmpfs" && mount.Destination == "/tmp" {
			continue
		}
		return errors.New("OCI container has an unexpected mount")
	}
	if binds != 1 || len(host.Binds) != 0 || len(host.Tmpfs) != 1 || host.Tmpfs["/tmp"] == "" {
		return errors.New("OCI container input/tmpfs mounts differ from policy")
	}
	return nil
}

func observedKernel(data []byte, nonce string, cfg config.CIOCI) (*Kernel, error) {
	prefix := "DVA_OCI_LIMITS:" + nonce + ":"
	for line := range strings.SplitSeq(string(data), "\n") {
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		fields := strings.Split(strings.TrimPrefix(line, prefix), ":")
		if len(fields) != 5 {
			break
		}
		values := make([]int64, 5)
		for index, field := range fields {
			value, err := strconv.ParseInt(field, 10, 64)
			if err != nil {
				return nil, errors.New("invalid OCI kernel evidence")
			}
			values[index] = value
		}
		k := &Kernel{PidsMax: values[0], MemoryMax: values[1], SwapMax: values[2], CPUQuota: values[3], CPUPeriod: values[4]}
		if k.PidsMax == cfg.PidsLimit && k.MemoryMax == cfg.MemoryBytes && k.SwapMax == 0 &&
			k.CPUPeriod > 0 && k.CPUQuota > 0 && k.CPUQuota/k.CPUPeriod == int64(cfg.CPUs) && k.CPUQuota%k.CPUPeriod == 0 {
			return k, nil
		}
		break
	}
	return nil, errors.New("OCI workload has no matching cgroup-v2 limit evidence")
}
