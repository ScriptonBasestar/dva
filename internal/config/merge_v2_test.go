package config

import "testing"

func TestMergeVars(t *testing.T) {
	base := &Config{Vars: map[string]string{"A": "1", "B": "2"}}
	other := &Config{Vars: map[string]string{"B": "3", "C": "4"}}

	if err := base.mergeFrom(other); err != nil {
		t.Fatal(err)
	}

	if base.Vars["A"] != "1" {
		t.Error("A should be preserved")
	}
	if base.Vars["B"] != "3" {
		t.Error("B should be overridden")
	}
	if base.Vars["C"] != "4" {
		t.Error("C should be added")
	}
}

func TestMergePlans(t *testing.T) {
	base := &Config{Plans: map[string]*PlanConfig{
		"dev": {
			Description:  "dev plan",
			Environment:  "dev",
			EndpointTags: []string{"infra"},
			Vars:         map[string]string{"A": "1"},
			Entries:      []PlanEntry{{Name: "db", Order: 10}},
		},
	}}
	other := &Config{Plans: map[string]*PlanConfig{
		"dev": {
			EndpointTags: []string{"app"},
			Vars:         map[string]string{"B": "2"},
			Entries:      []PlanEntry{{Name: "api", Order: 20}},
		},
	}}

	if err := base.mergeFrom(other); err != nil {
		t.Fatal(err)
	}

	plan := base.Plans["dev"]
	if plan.Description != "dev plan" {
		t.Error("description should be preserved")
	}
	if plan.Environment != "dev" {
		t.Error("environment should be preserved")
	}
	if len(plan.EndpointTags) != 1 || plan.EndpointTags[0] != "app" {
		t.Errorf("endpoint_tags should be replaced, got %v", plan.EndpointTags)
	}
	if plan.Vars["A"] != "1" {
		t.Error("var A should be preserved")
	}
	if plan.Vars["B"] != "2" {
		t.Error("var B should be added")
	}
	if len(plan.Entries) != 1 || plan.Entries[0].Name != "api" {
		t.Error("entries should be replaced")
	}
}

func TestMergeSites(t *testing.T) {
	base := &Config{Sites: map[string]*SiteConfig{
		"local": {Description: "local", Vars: map[string]string{"A": "1"}},
	}}
	other := &Config{Sites: map[string]*SiteConfig{
		"local": {
			Vars: map[string]string{"B": "2"},
			EntryOverrides: map[string]*SiteEntryOverride{
				"api": {Runner: "docker"},
			},
		},
	}}

	if err := base.mergeFrom(other); err != nil {
		t.Fatal(err)
	}

	site := base.Sites["local"]
	if site.Description != "local" {
		t.Error("description should be preserved")
	}
	if site.Vars["A"] != "1" {
		t.Error("var A should be preserved")
	}
	if site.Vars["B"] != "2" {
		t.Error("var B should be added")
	}
	if len(site.EntryOverrides) != 1 {
		t.Error("entry_overrides should be added")
	}
}

func TestMergeLifecycleEntryRunners(t *testing.T) {
	base := &LifecycleEntry{
		Name:          "api",
		DefaultRunner: "native",
		Runners: map[string]any{
			"native": &NativeRunnerConfig{
				Dir: "apps/api",
				Run: "go run ./cmd/api",
				Env: map[string]string{"BASE": "1"},
			},
		},
	}
	other := &LifecycleEntry{Runners: map[string]any{
		"native": &NativeRunnerConfig{
			Build: "go build ./cmd/api",
			Env:   map[string]string{"OVERRIDE": "1"},
		},
		"docker": map[string]any{"image": "myorg/api:dev"},
	}}

	merged, err := MergeLifecycleEntry(base, other)
	if err != nil {
		t.Fatal(err)
	}
	if merged.DefaultRunner != "native" {
		t.Error("default_runner should be preserved")
	}
	if len(merged.Runners) != 2 {
		t.Errorf("expected 2 runners, got %d", len(merged.Runners))
	}
	native, ok := merged.Runners["native"].(*NativeRunnerConfig)
	if !ok {
		t.Fatalf("native runner type = %T, want *NativeRunnerConfig", merged.Runners["native"])
	}
	if native.Dir != "apps/api" || native.Run != "go run ./cmd/api" || native.Build != "go build ./cmd/api" {
		t.Errorf("native runner = %#v, want inherited dir/run and overridden build", native)
	}
	if native.Env["BASE"] != "1" || native.Env["OVERRIDE"] != "1" {
		t.Errorf("native env = %#v, want key-level merge", native.Env)
	}
}

func TestMergeLifecycleEntryTypedRunnerConfigs(t *testing.T) {
	base := &LifecycleEntry{Runners: map[string]any{
		"process": &ProcessPluginConfig{Command: "serve", Dir: "app", ReadyTimeout: 10},
		"script":  &ScriptPluginConfig{Up: "up", Down: "down"},
		"docker":  &DockerPluginConfig{Image: "base", Ports: []string{"8080:80"}, Env: map[string]string{"A": "1"}},
		"helm":    &HelmPluginConfig{Chart: "chart", Values: []string{"base.yml"}, Set: map[string]string{"a": "1"}},
		"tilt":    &TiltPluginConfig{Dir: "base", Args: []string{"--base"}},
	}}
	override := &LifecycleEntry{Runners: map[string]any{
		"process": &ProcessPluginConfig{ReadyTimeout: 20},
		"script":  &ScriptPluginConfig{Stop: "stop"},
		"docker":  &DockerPluginConfig{Ports: []string{}, Env: map[string]string{"B": "2"}},
		"helm":    &HelmPluginConfig{Set: map[string]string{"b": "2"}},
		"tilt":    &TiltPluginConfig{Args: []string{"--override"}},
	}}

	merged, err := MergeLifecycleEntry(base, override)
	if err != nil {
		t.Fatal(err)
	}
	process := merged.Runners["process"].(*ProcessPluginConfig)
	if process.Command != "serve" || process.Dir != "app" || process.ReadyTimeout != 20 {
		t.Errorf("process runner = %#v", process)
	}
	script := merged.Runners["script"].(*ScriptPluginConfig)
	if script.Up != "up" || script.Down != "down" || script.Stop != "stop" {
		t.Errorf("script runner = %#v", script)
	}
	docker := merged.Runners["docker"].(*DockerPluginConfig)
	if docker.Image != "base" || docker.Ports == nil || len(docker.Ports) != 0 || docker.Env["A"] != "1" || docker.Env["B"] != "2" {
		t.Errorf("docker runner = %#v", docker)
	}
	helm := merged.Runners["helm"].(*HelmPluginConfig)
	if helm.Chart != "chart" || helm.Values[0] != "base.yml" || helm.Set["a"] != "1" || helm.Set["b"] != "2" {
		t.Errorf("helm runner = %#v", helm)
	}
	tilt := merged.Runners["tilt"].(*TiltPluginConfig)
	if tilt.Dir != "base" || len(tilt.Args) != 1 || tilt.Args[0] != "--override" {
		t.Errorf("tilt runner = %#v", tilt)
	}
}

func TestMergeLifecycleEntryComposeRunner(t *testing.T) {
	base := &LifecycleEntry{
		Name:          "infra",
		DefaultRunner: "compose",
		Runners: map[string]any{
			"compose": &ComposePluginConfig{
				Files:       []string{"compose.yml"},
				ProjectName: "myapp",
			},
		},
	}
	other := &LifecycleEntry{Runners: map[string]any{
		"compose": &ComposePluginConfig{
			Files: []string{"compose.yml", "compose.dev.yml"},
		},
	}}

	merged, err := MergeLifecycleEntry(base, other)
	if err != nil {
		t.Fatal(err)
	}
	composeCfg := merged.ComposeConfig()
	if composeCfg == nil {
		t.Fatal("expected compose runner config")
	}
	if composeCfg.ProjectName != "myapp" {
		t.Errorf("project_name = %q, want preserved", composeCfg.ProjectName)
	}
	if len(composeCfg.Files) != 2 || composeCfg.Files[1] != "compose.dev.yml" {
		t.Errorf("files = %v, want override files", composeCfg.Files)
	}
}
