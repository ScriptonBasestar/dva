// Package ociexec owns disposable, resource-limited Docker CI containers.
// It does not manage development services or execute native host diagnostics.
package ociexec

import (
	"io"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// Options binds one container to an existing CI execution and private state.
type Options struct {
	Root, StateDir, RunID, Step string
	Config                      config.CIOCI
	Output                      io.Writer
}

// Limits are requests; Observed and Kernel separately record their verification.
type Limits struct {
	PidsLimit        int64 `json:"pids_limit"`
	MemoryBytes      int64 `json:"memory_bytes"`
	CPUs             int   `json:"cpus"`
	OutputLimitBytes int64 `json:"output_limit_bytes"`
}

// Kernel records values read inside the container before its workload starts.
// pids includes kernel tasks/threads, not just operating-system processes.
type Kernel struct {
	PidsMax   int64 `json:"pids_max"`
	MemoryMax int64 `json:"memory_max"`
	SwapMax   int64 `json:"swap_max"`
	CPUQuota  int64 `json:"cpu_quota"`
	CPUPeriod int64 `json:"cpu_period"`
}

type AppliedLimits struct {
	PidsLimit       int64 `json:"pids_limit"`
	MemoryBytes     int64 `json:"memory_bytes"`
	MemorySwapBytes int64 `json:"memory_swap_bytes"`
	NanoCPUs        int64 `json:"nano_cpus"`
}

// Report keeps execution failure distinct from cleanup failure. Environment
// values and workload arguments are deliberately excluded from stored evidence.
type Report struct {
	DockerBinary     string         `json:"docker_binary,omitempty"`
	DockerSHA256     string         `json:"docker_sha256,omitempty"`
	DockerConfigPath string         `json:"docker_config_path,omitempty"`
	Endpoint         string         `json:"endpoint,omitempty"`
	DaemonID         string         `json:"daemon_id,omitempty"`
	Image            string         `json:"image"`
	ImageID          string         `json:"image_id,omitempty"`
	InputSHA256      string         `json:"input_sha256,omitempty"`
	ArgvSHA256       string         `json:"argv_sha256,omitempty"`
	IntentPath       string         `json:"intent_path,omitempty"`
	ContainerID      string         `json:"container_id,omitempty"`
	Requested        Limits         `json:"requested"`
	Observed         *AppliedLimits `json:"observed,omitempty"`
	Kernel           *Kernel        `json:"kernel,omitempty"`
	ExitCode         *int           `json:"exit_code,omitempty"`
	OOMKilled        bool           `json:"oom_killed"`
	OutputTruncated  bool           `json:"output_truncated"`
	OutputBytes      int64          `json:"output_bytes"`
	ExecutionError   string         `json:"execution_error,omitempty"`
	Cleanup          string         `json:"cleanup"`
	CleanupError     string         `json:"cleanup_error,omitempty"`
}

type intent struct {
	Version                  int `json:"version"`
	RunID, Step, Nonce, Name string
	Endpoint, DaemonID       string
	ContainerID, Snapshot    string
	Result                   Report
}

type daemonInfo struct {
	ID, OSType, CgroupVersion string
	PidsLimit                 bool
}

type imageInfo struct {
	ID          string `json:"Id"`
	RepoDigests []string
	Config      struct {
		Volumes map[string]any
		Env     []string
	}
}

type containerInfo struct {
	ID, Name, Image string
	Config          struct {
		User            string
		WorkingDir      string
		Labels          map[string]string
		Entrypoint, Cmd []string
		Env             []string
		Healthcheck     struct{ Test []string }
	}
	HostConfig struct {
		Init                                        *bool
		PidsLimit, Memory, MemorySwap, NanoCPUs     int64
		NetworkMode, CgroupnsMode, PidMode, IpcMode string
		ReadonlyRootfs, Privileged, AutoRemove      bool
		CapDrop, CapAdd, SecurityOpt, Binds         []string
		Tmpfs                                       map[string]string
		LogConfig                                   struct{ Type string }
		RestartPolicy                               struct{ Name string }
	}
	Mounts []struct {
		Type, Source, Destination string
		RW                        bool
	}
	State struct {
		Status             string
		Running, OOMKilled bool
		ExitCode           int
	}
}
