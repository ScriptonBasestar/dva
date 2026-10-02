package config

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// UnmarshalYAML supports runner-based entries plus legacy parser compatibility:
//
// Runner-based:
//
//	my-service:
//	  default_runner: compose
//	  runners:
//	    compose:
//	      files: [docker-compose.yml]
//
// Nested (legacy parser compatibility): plugin config under a named sub-key
//
//	compose:
//	  order: 10
//	  compose:
//	    files: [docker-compose.yml]
//
// Flat legacy plugin entries are still parsed for non-schema callers.
func (e *LifecycleEntry) UnmarshalYAML(node *yaml.Node) error {
	// Decode common fields + nested plugin configs
	var raw struct {
		Plugin        string                       `yaml:"plugin"`
		Order         int                          `yaml:"order"`
		Description   string                       `yaml:"description"`
		Tags          []string                     `yaml:"tags"`
		Vars          map[string]string            `yaml:"vars"`
		Exports       map[string]string            `yaml:"exports"`
		HealthChecks  map[string]HealthCheckConfig `yaml:"health_checks"`
		DefaultRunner string                       `yaml:"default_runner"`
		Source        *SourceConfig                `yaml:"source"`
		Tunnel        *TunnelConfig                `yaml:"tunnel"`
		Optional      bool                         `yaml:"optional"`
		Primary       bool                         `yaml:"primary"`

		// Nested format: plugin config under its type key
		Compose       *ComposePluginConfig       `yaml:"compose"`
		Process       *ProcessPluginConfig       `yaml:"process"`
		Script        *ScriptPluginConfig        `yaml:"script"`
		Docker        *DockerPluginConfig        `yaml:"docker"`
		Kubectl       *KubectlPluginConfig       `yaml:"kubectl"`
		Helm          *HelmPluginConfig          `yaml:"helm"`
		Kustomize     *KustomizePluginConfig     `yaml:"kustomize"`
		Tilt          *TiltPluginConfig          `yaml:"tilt"`
		Skaffold      *SkaffoldPluginConfig      `yaml:"skaffold"`
		PodmanCompose *PodmanComposePluginConfig `yaml:"podman_compose"`
		Vagrant       *VagrantPluginConfig       `yaml:"vagrant"`
		SAM           *SAMPluginConfig           `yaml:"sam"`
		Serverless    *ServerlessPluginConfig    `yaml:"serverless"`
		Multipass     *MultipassPluginConfig     `yaml:"multipass"`
	}
	if err := node.Decode(&raw); err != nil {
		return err
	}

	e.Order = raw.Order
	e.Description = raw.Description
	e.Tags = raw.Tags
	e.Vars = raw.Vars
	e.Exports = raw.Exports
	e.HealthChecks = raw.HealthChecks
	e.DefaultRunner = raw.DefaultRunner
	e.Source = raw.Source
	e.Tunnel = raw.Tunnel
	e.Optional = raw.Optional
	e.Primary = raw.Primary

	runners, err := decodeRunnersMap(node)
	if err != nil {
		return err
	}
	e.Runners = runners

	// Nested format: detect by checking which plugin sub-key is set
	switch {
	case raw.Compose != nil:
		e.Compose = raw.Compose
		e.Plugin = "compose"
		return nil
	case raw.Process != nil:
		e.Process = raw.Process
		e.Plugin = "process"
		return nil
	case raw.Script != nil:
		e.Script = raw.Script
		e.Plugin = "script"
		return nil
	case raw.Docker != nil:
		e.Docker = raw.Docker
		e.Plugin = "docker"
		return nil
	case raw.Kubectl != nil:
		e.Kubectl = raw.Kubectl
		e.Plugin = "kubectl"
		return nil
	case raw.Helm != nil:
		e.Helm = raw.Helm
		e.Plugin = "helm"
		return nil
	case raw.Kustomize != nil:
		e.Kustomize = raw.Kustomize
		e.Plugin = "kustomize"
		return nil
	case raw.Tilt != nil:
		e.Tilt = raw.Tilt
		e.Plugin = "tilt"
		return nil
	case raw.Skaffold != nil:
		e.Skaffold = raw.Skaffold
		e.Plugin = "skaffold"
		return nil
	case raw.PodmanCompose != nil:
		e.PodmanCompose = raw.PodmanCompose
		e.Plugin = "podman-compose"
		return nil
	case raw.Vagrant != nil:
		e.Vagrant = raw.Vagrant
		e.Plugin = "vagrant"
		return nil
	case raw.SAM != nil:
		e.SAM = raw.SAM
		e.Plugin = "sam"
		return nil
	case raw.Serverless != nil:
		e.Serverless = raw.Serverless
		e.Plugin = "serverless"
		return nil
	case raw.Multipass != nil:
		e.Multipass = raw.Multipass
		e.Plugin = "multipass"
		return nil
	}

	// Flat format: plugin type from explicit `plugin:` field
	if raw.Plugin != "" {
		e.Plugin = raw.Plugin
		return e.resolvePluginConfig(node)
	}

	// No plugin detected: store raw node for deferred resolution from entry name
	e.rawNode = node
	return nil
}

func decodeRunnersMap(entryNode *yaml.Node) (map[string]any, error) {
	runnersNode := findMapValueNode(entryNode, "runners")
	if runnersNode == nil {
		return nil, nil
	}
	if runnersNode.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("runners: expected mapping node")
	}

	runners := make(map[string]any)
	for i := 0; i+1 < len(runnersNode.Content); i += 2 {
		name := runnersNode.Content[i].Value
		cfgNode := runnersNode.Content[i+1]
		cfg, err := decodeRunnerNode(name, cfgNode)
		if err != nil {
			return nil, fmt.Errorf("runners.%s: %w", name, err)
		}
		runners[name] = cfg
	}
	return runners, nil
}

func findMapValueNode(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func decodeRunnerNode(name string, node *yaml.Node) (any, error) {
	normalized := normalizeRunnerName(name)
	switch normalized {
	case "native":
		cfg := &NativeRunnerConfig{}
		if err := node.Decode(cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	case "docker":
		// Map runners.docker to the docker lifecycle plugin (TASK-017 Option A).
		// Nested docker: already uses DockerPluginConfig; keep runners shape aligned.
		cfg := &DockerPluginConfig{}
		if err := node.Decode(cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	case "compose":
		cfg := &ComposePluginConfig{}
		if err := node.Decode(cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	case "process":
		cfg := &ProcessPluginConfig{}
		if err := node.Decode(cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	case "script":
		cfg := &ScriptPluginConfig{}
		if err := node.Decode(cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	case "kubectl":
		cfg := &KubectlPluginConfig{}
		if err := node.Decode(cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	case "helm":
		cfg := &HelmPluginConfig{}
		if err := node.Decode(cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	case "kustomize":
		cfg := &KustomizePluginConfig{}
		if err := node.Decode(cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	case "tilt":
		cfg := &TiltPluginConfig{}
		if err := node.Decode(cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	case "skaffold":
		cfg := &SkaffoldPluginConfig{}
		if err := node.Decode(cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	case "podman-compose":
		cfg := &PodmanComposePluginConfig{}
		if err := node.Decode(cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	case "vagrant":
		cfg := &VagrantPluginConfig{}
		if err := node.Decode(cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	case "sam":
		cfg := &SAMPluginConfig{}
		if err := node.Decode(cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	case "serverless":
		cfg := &ServerlessPluginConfig{}
		if err := node.Decode(cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	case "multipass":
		cfg := &MultipassPluginConfig{}
		if err := node.Decode(cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	default:
		m := make(map[string]any)
		if err := node.Decode(&m); err != nil {
			return nil, err
		}
		return m, nil
	}
}

func normalizeRunnerName(name string) string {
	if name == "podman_compose" {
		return "podman-compose"
	}
	if mapped, ok := knownPluginNames[name]; ok {
		return mapped
	}
	return name
}

// resolvePluginConfig decodes plugin-specific fields from a flat YAML node.
func (e *LifecycleEntry) resolvePluginConfig(node *yaml.Node) error {
	switch e.Plugin {
	case "compose":
		cfg := &ComposePluginConfig{}
		if err := node.Decode(cfg); err != nil {
			return fmt.Errorf("compose plugin: %w", err)
		}
		e.Compose = cfg
	case "process":
		cfg := &ProcessPluginConfig{}
		if err := node.Decode(cfg); err != nil {
			return fmt.Errorf("process plugin: %w", err)
		}
		e.Process = cfg
	case "script":
		cfg := &ScriptPluginConfig{}
		if err := node.Decode(cfg); err != nil {
			return fmt.Errorf("script plugin: %w", err)
		}
		e.Script = cfg
	case "docker":
		cfg := &DockerPluginConfig{}
		if err := node.Decode(cfg); err != nil {
			return fmt.Errorf("docker plugin: %w", err)
		}
		e.Docker = cfg
	case "kubectl":
		cfg := &KubectlPluginConfig{}
		if err := node.Decode(cfg); err != nil {
			return fmt.Errorf("kubectl plugin: %w", err)
		}
		e.Kubectl = cfg
	case "helm":
		cfg := &HelmPluginConfig{}
		if err := node.Decode(cfg); err != nil {
			return fmt.Errorf("helm plugin: %w", err)
		}
		e.Helm = cfg
	case "kustomize":
		cfg := &KustomizePluginConfig{}
		if err := node.Decode(cfg); err != nil {
			return fmt.Errorf("kustomize plugin: %w", err)
		}
		e.Kustomize = cfg
	case "tilt":
		cfg := &TiltPluginConfig{}
		if err := node.Decode(cfg); err != nil {
			return fmt.Errorf("tilt plugin: %w", err)
		}
		e.Tilt = cfg
	case "skaffold":
		cfg := &SkaffoldPluginConfig{}
		if err := node.Decode(cfg); err != nil {
			return fmt.Errorf("skaffold plugin: %w", err)
		}
		e.Skaffold = cfg
	case "podman-compose", "podman_compose":
		cfg := &PodmanComposePluginConfig{}
		if err := node.Decode(cfg); err != nil {
			return fmt.Errorf("podman-compose plugin: %w", err)
		}
		e.PodmanCompose = cfg
	case "vagrant":
		cfg := &VagrantPluginConfig{}
		if err := node.Decode(cfg); err != nil {
			return fmt.Errorf("vagrant plugin: %w", err)
		}
		e.Vagrant = cfg
	case "sam":
		cfg := &SAMPluginConfig{}
		if err := node.Decode(cfg); err != nil {
			return fmt.Errorf("sam plugin: %w", err)
		}
		e.SAM = cfg
	case "serverless":
		cfg := &ServerlessPluginConfig{}
		if err := node.Decode(cfg); err != nil {
			return fmt.Errorf("serverless plugin: %w", err)
		}
		e.Serverless = cfg
	case "multipass":
		cfg := &MultipassPluginConfig{}
		if err := node.Decode(cfg); err != nil {
			return fmt.Errorf("multipass plugin: %w", err)
		}
		e.Multipass = cfg
	default:
		return fmt.Errorf("unknown plugin %q", e.Plugin)
	}
	return nil
}
