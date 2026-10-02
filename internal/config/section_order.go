package config

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// migrationGuideURL is printed in validate warnings, so it is the link users are most
// likely to click. Repo dva on branch master — not "main", which this repo has never had.
// It named dev-virtual-auto until the repo was renamed (TASK-060); that name still resolves
// through GitHub's rename redirect, but the redirect dies if the old name is ever reused.
const migrationGuideURL = "https://github.com/ScriptonBasestar/dva/blob/master/docs/42-migration-and-compatibility.md#11-migration"

// canonicalSectionOrder defines the recommended top-level key order for dva.yml.
var canonicalSectionOrder = []string{
	"version", "vars", "environment", "env_file", "stack", "plans",
	"default_plan", "environments", "sites",
	// Legacy sections retain a deterministic position during migration.
	// `applications` is absent by removal, not by oversight (docs/43): the key no
	// longer validates, so a file carrying it is rejected before this order check
	// ever runs — listing it here would only describe where a key that cannot
	// appear would have gone.
	"checks", "default_mode", "suggestion_ignore", "suggestions", "drift_ignore", "modes",
	"health_checks", "interaction", "provision", "modules", "subprojects",
	"endpoints", "infra", "ssh", "devcontainer",
}

// canonicalOrderIndex maps section name to its position in canonical order.
var canonicalOrderIndex map[string]int

func init() {
	canonicalOrderIndex = make(map[string]int, len(canonicalSectionOrder))
	for i, name := range canonicalSectionOrder {
		canonicalOrderIndex[name] = i
	}
}

// CanonicalSectionOrder returns a copy of the recommended top-level dva.yml key order.
// This is the canonical source for the section-order list embedded in
// agent-mesh-flows/shared/library/shared-guardrails.md by tools/libgen.
func CanonicalSectionOrder() []string {
	return slices.Clone(canonicalSectionOrder)
}

// validateCanonicalOrder reads the YAML file and checks that top-level keys
// follow the canonical section order. Returns warnings for out-of-order keys.
func validateCanonicalOrder(filePath string) []string {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}
	return validateCanonicalOrderFromBytes(data)
}

// validateCanonicalOrderFromBytes checks that top-level keys in raw YAML data
// follow the canonical section order. Returns warnings for out-of-order keys.
func validateCanonicalOrderFromBytes(data []byte) []string {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil
	}

	// doc is a DocumentNode wrapping a MappingNode
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil
	}

	// Extract top-level keys in file order, filtering to canonical-only keys
	var fileKeys []string
	for i := 0; i < len(root.Content)-1; i += 2 {
		key := root.Content[i].Value
		if _, ok := canonicalOrderIndex[key]; ok {
			fileKeys = append(fileKeys, key)
		}
	}

	if len(fileKeys) < 2 {
		return nil
	}

	// Check if file keys are in canonical order
	inOrder := true
	for i := 1; i < len(fileKeys); i++ {
		if canonicalOrderIndex[fileKeys[i]] < canonicalOrderIndex[fileKeys[i-1]] {
			inOrder = false
			break
		}
	}

	if inOrder {
		return nil
	}

	// Build the expected order for present keys
	var expected []string
	for _, s := range canonicalSectionOrder {
		if slices.Contains(fileKeys, s) {
			expected = append(expected, s)
		}
	}

	return []string{
		fmt.Sprintf("section order: found [%s] but canonical order is [%s]; consider reordering",
			strings.Join(fileKeys, " → "), strings.Join(expected, " → ")),
	}
}
