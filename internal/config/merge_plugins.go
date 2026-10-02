package config

import (
	"fmt"
	"maps"
)

// MergeLifecycleEntry deep-merges other into base.
// Returns an error if a restricted field (plugin) is changed.
func MergeLifecycleEntry(base, other *LifecycleEntry) (*LifecycleEntry, error) {
	if base == nil {
		return other, nil
	}
	if other == nil {
		return base, nil
	}

	// Restricted field: plugin must not change
	if other.Plugin != "" && base.Plugin != "" && other.Plugin != base.Plugin {
		// stack_overrides entries often have empty Name (map key is the identity). Naming
		// "" as the entry is noise beside the outer `[warn] stack_override "api": …` that
		// already carries the key (TASK-157 option B — fix the message, do not backfill Name).
		if base.Name == "" {
			return nil, fmt.Errorf("cannot override plugin type: %q → %q (restricted field)", base.Plugin, other.Plugin)
		}
		return nil, fmt.Errorf("cannot override plugin type for stack entry %q: %q → %q (restricted field)", base.Name, base.Plugin, other.Plugin)
	}

	// Scalar replace (non-zero takes precedence)
	if other.Order != 0 {
		base.Order = other.Order
	}

	// List replace
	if other.Tags != nil {
		base.Tags = other.Tags
	}

	// Map merge
	base.Exports = mergeStringMap(base.Exports, other.Exports)

	// HealthChecks map merge
	if other.HealthChecks != nil {
		if base.HealthChecks == nil {
			base.HealthChecks = make(map[string]HealthCheckConfig)
		}
		maps.Copy(base.HealthChecks, other.HealthChecks)
	}

	// Plugin configs: merge if same type, set if base had none
	if other.Compose != nil {
		if base.Compose != nil {
			mergeComposeConfig(base.Compose, other.Compose)
		} else {
			base.Compose = other.Compose
		}
	}
	if other.Process != nil {
		if base.Process != nil {
			mergeProcessConfig(base.Process, other.Process)
		} else {
			base.Process = other.Process
		}
	}
	if other.Script != nil {
		if base.Script != nil {
			mergeScriptConfig(base.Script, other.Script)
		} else {
			base.Script = other.Script
		}
	}
	if other.Docker != nil {
		if base.Docker != nil {
			mergeDockerConfig(base.Docker, other.Docker)
		} else {
			base.Docker = other.Docker
		}
	}
	if other.Kubectl != nil {
		if base.Kubectl != nil {
			mergeKubectlConfig(base.Kubectl, other.Kubectl)
		} else {
			base.Kubectl = other.Kubectl
		}
	}
	if other.Helm != nil {
		if base.Helm != nil {
			mergeHelmConfig(base.Helm, other.Helm)
		} else {
			base.Helm = other.Helm
		}
	}
	if other.Kustomize != nil {
		if base.Kustomize != nil {
			mergeKustomizeConfig(base.Kustomize, other.Kustomize)
		} else {
			base.Kustomize = other.Kustomize
		}
	}
	// Tier 2/3 plugins: simple replace (less common, deep merge not needed yet)
	if other.Tilt != nil {
		base.Tilt = other.Tilt
	}
	if other.Skaffold != nil {
		base.Skaffold = other.Skaffold
	}
	if other.PodmanCompose != nil {
		base.PodmanCompose = other.PodmanCompose
	}
	if other.Vagrant != nil {
		base.Vagrant = other.Vagrant
	}
	if other.SAM != nil {
		base.SAM = other.SAM
	}
	if other.Serverless != nil {
		base.Serverless = other.Serverless
	}
	if other.Multipass != nil {
		base.Multipass = other.Multipass
	}

	if other.Description != "" {
		base.Description = other.Description
	}
	base.Vars = mergeStringMap(base.Vars, other.Vars)
	if other.DefaultRunner != "" {
		base.DefaultRunner = other.DefaultRunner
	}

	if other.Runners != nil {
		if base.Runners == nil {
			base.Runners = make(map[string]any)
		}
		for runnerName, runnerConfig := range other.Runners {
			if existing, ok := base.Runners[runnerName]; ok {
				base.Runners[runnerName] = mergeRunnerConfig(existing, runnerConfig)
			} else {
				base.Runners[runnerName] = runnerConfig
			}
		}
	}

	return base, nil
}

// --- Plugin config merge helpers ---
// Strategy: scalar replace (non-zero), list replace (non-nil), map merge

func mergeComposeConfig(base, other *ComposePluginConfig) {
	if other.ProjectName != "" {
		base.ProjectName = other.ProjectName
	}
	if other.Command != "" {
		base.Command = other.Command
	}
	if other.Method != "" {
		base.Method = other.Method
	}
	if other.Files != nil {
		base.Files = other.Files
	}
	if other.UpOptions != nil {
		base.UpOptions = other.UpOptions
	}
	if other.Tags != nil {
		base.Tags = other.Tags
	}
	if other.Services != nil {
		if base.Services == nil {
			base.Services = make(map[string]ServiceTagConfig)
		}
		maps.Copy(base.Services, other.Services)
	}
}

func mergeNativeRunnerConfig(base, other *NativeRunnerConfig) {
	if other.Dir != "" {
		base.Dir = other.Dir
	}
	if other.Build != "" {
		base.Build = other.Build
	}
	if other.PostBuild != "" {
		base.PostBuild = other.PostBuild
	}
	if other.Run != "" {
		base.Run = other.Run
	}
	base.Env = mergeStringMap(base.Env, other.Env)
}

func mergeProcessConfig(base, other *ProcessPluginConfig) {
	if other.Command != "" {
		base.Command = other.Command
	}
	if other.Dir != "" {
		base.Dir = other.Dir
	}
	if other.ReadyTimeout != 0 {
		base.ReadyTimeout = other.ReadyTimeout
	}
}

func mergeScriptConfig(base, other *ScriptPluginConfig) {
	if other.Up != "" {
		base.Up = other.Up
	}
	if other.Down != "" {
		base.Down = other.Down
	}
	if other.Stop != "" {
		base.Stop = other.Stop
	}
}

func mergeDockerConfig(base, other *DockerPluginConfig) {
	if other.Image != "" {
		base.Image = other.Image
	}
	if other.Name != "" {
		base.Name = other.Name
	}
	if other.Ports != nil {
		base.Ports = other.Ports
	}
	if other.Volumes != nil {
		base.Volumes = other.Volumes
	}
	if other.Options != nil {
		base.Options = other.Options
	}
	base.Env = mergeStringMap(base.Env, other.Env)
}

func mergeDockerRunnerConfig(base, other *DockerRunnerConfig) {
	if other.Image != "" {
		base.Image = other.Image
	}
	if other.Run != "" {
		base.Run = other.Run
	}
	if other.Build != "" {
		base.Build = other.Build
	}
	if other.Command != "" {
		base.Command = other.Command
	}
	if other.Ports != nil {
		base.Ports = other.Ports
	}
	if other.Volumes != nil {
		base.Volumes = other.Volumes
	}
	if other.Options != nil {
		base.Options = other.Options
	}
	base.Env = mergeStringMap(base.Env, other.Env)
}

func mergeKubectlConfig(base, other *KubectlPluginConfig) {
	if other.Namespace != "" {
		base.Namespace = other.Namespace
	}
	if other.Context != "" {
		base.Context = other.Context
	}
	if other.Kubeconfig != "" {
		base.Kubeconfig = other.Kubeconfig
	}
	if other.Manifests != nil {
		base.Manifests = other.Manifests
	}
}

func mergeHelmConfig(base, other *HelmPluginConfig) {
	if other.Chart != "" {
		base.Chart = other.Chart
	}
	if other.Release != "" {
		base.Release = other.Release
	}
	if other.Namespace != "" {
		base.Namespace = other.Namespace
	}
	if other.Context != "" {
		base.Context = other.Context
	}
	if other.Values != nil {
		base.Values = other.Values
	}
	base.Set = mergeStringMap(base.Set, other.Set)
}

func mergeKustomizeConfig(base, other *KustomizePluginConfig) {
	if other.Dir != "" {
		base.Dir = other.Dir
	}
	if other.Namespace != "" {
		base.Namespace = other.Namespace
	}
	if other.Context != "" {
		base.Context = other.Context
	}
}

func mergeTiltConfig(base, other *TiltPluginConfig) {
	if other.Dir != "" {
		base.Dir = other.Dir
	}
	if other.Args != nil {
		base.Args = other.Args
	}
}

func mergeSkaffoldConfig(base, other *SkaffoldPluginConfig) {
	if other.Config != "" {
		base.Config = other.Config
	}
	if other.Profile != "" {
		base.Profile = other.Profile
	}
	if other.Args != nil {
		base.Args = other.Args
	}
}

func mergePodmanComposeConfig(base, other *PodmanComposePluginConfig) {
	if other.Files != nil {
		base.Files = other.Files
	}
	if other.ProjectName != "" {
		base.ProjectName = other.ProjectName
	}
}

func mergeVagrantConfig(base, other *VagrantPluginConfig) {
	if other.Dir != "" {
		base.Dir = other.Dir
	}
	if other.Machine != "" {
		base.Machine = other.Machine
	}
}

func mergeSAMConfig(base, other *SAMPluginConfig) {
	if other.Template != "" {
		base.Template = other.Template
	}
	if other.Port != 0 {
		base.Port = other.Port
	}
	if other.Args != nil {
		base.Args = other.Args
	}
}

func mergeServerlessConfig(base, other *ServerlessPluginConfig) {
	if other.Dir != "" {
		base.Dir = other.Dir
	}
	if other.Port != 0 {
		base.Port = other.Port
	}
	if other.Args != nil {
		base.Args = other.Args
	}
}

func mergeMultipassConfig(base, other *MultipassPluginConfig) {
	if other.Name != "" {
		base.Name = other.Name
	}
	if other.Image != "" {
		base.Image = other.Image
	}
	if other.CPUs != 0 {
		base.CPUs = other.CPUs
	}
	if other.Memory != "" {
		base.Memory = other.Memory
	}
	if other.Disk != "" {
		base.Disk = other.Disk
	}
	if other.CloudInit != "" {
		base.CloudInit = other.CloudInit
	}
}

func mergeRunnerConfig(base, other any) any {
	switch baseConfig := base.(type) {
	case *NativeRunnerConfig:
		if otherConfig, ok := other.(*NativeRunnerConfig); ok {
			mergeNativeRunnerConfig(baseConfig, otherConfig)
			return baseConfig
		}
	case *ComposePluginConfig:
		if otherConfig, ok := other.(*ComposePluginConfig); ok {
			mergeComposeConfig(baseConfig, otherConfig)
			return baseConfig
		}
	case *ProcessPluginConfig:
		if otherConfig, ok := other.(*ProcessPluginConfig); ok {
			mergeProcessConfig(baseConfig, otherConfig)
			return baseConfig
		}
	case *ScriptPluginConfig:
		if otherConfig, ok := other.(*ScriptPluginConfig); ok {
			mergeScriptConfig(baseConfig, otherConfig)
			return baseConfig
		}
	case *DockerPluginConfig:
		if otherConfig, ok := other.(*DockerPluginConfig); ok {
			mergeDockerConfig(baseConfig, otherConfig)
			return baseConfig
		}
	case *DockerRunnerConfig:
		if otherConfig, ok := other.(*DockerRunnerConfig); ok {
			mergeDockerRunnerConfig(baseConfig, otherConfig)
			return baseConfig
		}
	case *KubectlPluginConfig:
		if otherConfig, ok := other.(*KubectlPluginConfig); ok {
			mergeKubectlConfig(baseConfig, otherConfig)
			return baseConfig
		}
	case *HelmPluginConfig:
		if otherConfig, ok := other.(*HelmPluginConfig); ok {
			mergeHelmConfig(baseConfig, otherConfig)
			return baseConfig
		}
	case *KustomizePluginConfig:
		if otherConfig, ok := other.(*KustomizePluginConfig); ok {
			mergeKustomizeConfig(baseConfig, otherConfig)
			return baseConfig
		}
	case *TiltPluginConfig:
		if otherConfig, ok := other.(*TiltPluginConfig); ok {
			mergeTiltConfig(baseConfig, otherConfig)
			return baseConfig
		}
	case *SkaffoldPluginConfig:
		if otherConfig, ok := other.(*SkaffoldPluginConfig); ok {
			mergeSkaffoldConfig(baseConfig, otherConfig)
			return baseConfig
		}
	case *PodmanComposePluginConfig:
		if otherConfig, ok := other.(*PodmanComposePluginConfig); ok {
			mergePodmanComposeConfig(baseConfig, otherConfig)
			return baseConfig
		}
	case *VagrantPluginConfig:
		if otherConfig, ok := other.(*VagrantPluginConfig); ok {
			mergeVagrantConfig(baseConfig, otherConfig)
			return baseConfig
		}
	case *SAMPluginConfig:
		if otherConfig, ok := other.(*SAMPluginConfig); ok {
			mergeSAMConfig(baseConfig, otherConfig)
			return baseConfig
		}
	case *ServerlessPluginConfig:
		if otherConfig, ok := other.(*ServerlessPluginConfig); ok {
			mergeServerlessConfig(baseConfig, otherConfig)
			return baseConfig
		}
	case *MultipassPluginConfig:
		if otherConfig, ok := other.(*MultipassPluginConfig); ok {
			mergeMultipassConfig(baseConfig, otherConfig)
			return baseConfig
		}
	}

	baseMap, baseOk := base.(map[string]any)
	otherMap, otherOk := other.(map[string]any)
	if baseOk && otherOk {
		result := make(map[string]any, len(baseMap))
		maps.Copy(result, baseMap)
		maps.Copy(result, otherMap)
		return result
	}
	return other
}
