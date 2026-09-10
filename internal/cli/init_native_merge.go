package cli

import (
	"fmt"
	"sort"

	"gopkg.in/yaml.v3"
)

// mergeInitDiscovery overlays declarations that came from repository-owned
// manifests onto the existing Compose template. The Compose stack declaration
// remains authoritative for its own name. Direct Makefile interactions replace
// same-name language-template guesses, such as npm test.
func mergeInitDiscovery(base, discovered string) (string, error) {
	var baseDoc, discoveredDoc yaml.Node
	if err := yaml.Unmarshal([]byte(base), &baseDoc); err != nil {
		return "", fmt.Errorf("parse generated Compose scaffold: %w", err)
	}
	if err := yaml.Unmarshal([]byte(discovered), &discoveredDoc); err != nil {
		return "", fmt.Errorf("parse generated native scaffold: %w", err)
	}
	baseRoot, discoveredRoot := yamlDocumentRoot(&baseDoc), yamlDocumentRoot(&discoveredDoc)
	if baseRoot == nil || discoveredRoot == nil || baseRoot.Kind != yaml.MappingNode || discoveredRoot.Kind != yaml.MappingNode {
		return "", fmt.Errorf("generated init scaffold is not a YAML mapping")
	}
	for i := 0; i+1 < len(discoveredRoot.Content); i += 2 {
		key, value := discoveredRoot.Content[i], discoveredRoot.Content[i+1]
		if key.Value == "version" {
			continue
		}
		baseValue := yamlMappingValue(baseRoot, key.Value)
		if baseValue == nil {
			baseRoot.Content = append(baseRoot.Content, key, value)
			continue
		}
		if baseValue.Kind == yaml.MappingNode && value.Kind == yaml.MappingNode {
			mergeMappingKeys(baseValue, value, key.Value == "interaction")
		}
	}
	ensureComposeInGeneratedDevPlan(baseRoot)
	sortInitTopLevelKeys(baseRoot)
	data, err := yaml.Marshal(&baseDoc)
	if err != nil {
		return "", fmt.Errorf("render combined init scaffold: %w", err)
	}
	return string(data), nil
}

func mergeMappingKeys(base, overlay *yaml.Node, replace bool) {
	for i := 0; i+1 < len(overlay.Content); i += 2 {
		key, value := overlay.Content[i], overlay.Content[i+1]
		if index := yamlMappingKeyIndex(base, key.Value); index >= 0 {
			if replace {
				base.Content[index+1] = value
			}
		} else {
			base.Content = append(base.Content, key, value)
		}
	}
}

func yamlMappingKeyIndex(node *yaml.Node, key string) int {
	if node == nil || node.Kind != yaml.MappingNode {
		return -1
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return i
		}
	}
	return -1
}

func ensureComposeInGeneratedDevPlan(root *yaml.Node) {
	stack := yamlMappingValue(root, "stack")
	if yamlMappingValue(stack, "compose") == nil {
		return
	}
	plans := yamlMappingValue(root, "plans")
	dev := yamlMappingValue(plans, "dev")
	entries := yamlMappingValue(dev, "entries")
	if entries == nil || entries.Kind != yaml.SequenceNode {
		return
	}
	for _, entry := range entries.Content {
		if name := yamlMappingValue(entry, "name"); name != nil && name.Value == "compose" {
			setYAMLMappingScalar(entry, "runner", "!!str", "compose")
			setYAMLMappingScalar(entry, "order", "!!int", "0")
			return
		}
	}
	composeEntry := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "name"},
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "compose"},
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "runner"},
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "compose"},
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "order"},
		{Kind: yaml.ScalarNode, Tag: "!!int", Value: "0"},
	}}
	entries.Content = append([]*yaml.Node{composeEntry}, entries.Content...)
}

func setYAMLMappingScalar(mapping *yaml.Node, key, tag, value string) {
	if index := yamlMappingKeyIndex(mapping, key); index >= 0 {
		mapping.Content[index+1] = &yaml.Node{Kind: yaml.ScalarNode, Tag: tag, Value: value}
		return
	}
	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: tag, Value: value},
	)
}

func sortInitTopLevelKeys(root *yaml.Node) {
	order := map[string]int{}
	for i, key := range []string{
		"version", "vars", "environment", "env_file", "stack", "plans", "default_plan",
		"environments", "sites", "checks", "default_mode", "suggestion_ignore", "modes",
		"health_checks", "interaction", "provision", "modules", "subprojects", "endpoints",
		"infra", "ssh", "devcontainer",
	} {
		order[key] = i
	}
	type pair struct{ key, value *yaml.Node }
	pairs := make([]pair, 0, len(root.Content)/2)
	for i := 0; i+1 < len(root.Content); i += 2 {
		pairs = append(pairs, pair{root.Content[i], root.Content[i+1]})
	}
	sort.SliceStable(pairs, func(i, j int) bool {
		left, leftKnown := order[pairs[i].key.Value]
		right, rightKnown := order[pairs[j].key.Value]
		switch {
		case leftKnown && rightKnown:
			return left < right
		case leftKnown:
			return true
		case rightKnown:
			return false
		default:
			return pairs[i].key.Value < pairs[j].key.Value
		}
	})
	root.Content = root.Content[:0]
	for _, pair := range pairs {
		root.Content = append(root.Content, pair.key, pair.value)
	}
}
