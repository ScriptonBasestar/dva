package config

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
)

//go:embed schema.json
var embeddedSchema embed.FS

// removedSchemaKeys maps keys DVA used to accept to whatever took over their
// job. Every one of them was real schema once, emitted by DVA's own templates
// and AI flows, so configs generated against an older version still carry them
// and "Additional property X is not allowed" alone leaves those users with a
// rejection and no next step.
//
// Keyed by property name alone, not by path: the guidance has to read correctly
// wherever the key turns up, since a stale config can carry it anywhere.
//
// This is also the list TestRemovedKeysAbsentFromGeneratorCorpus holds the AI
// generator corpus to — a key is only really gone once nothing teaches it.
var removedSchemaKeys = map[string]string{
	// TASK-036: services metadata; the reader (compose_status.go) is gone.
	"hint":    "removed: 'services' is tags-only — human-readable hints live in health_checks.<name>.start_hint",
	"related": "removed: group services with 'tags', or list them explicitly in modes.<name>.compose_services",
	// ee8ac8e: port metadata moved wholesale to the endpoints: section.
	"ports": "removed: DVA reads ports from the compose files — declare user-facing ones under the top-level 'endpoints:'",
	// TASK-035: both validated green and were never read.
	"interpolate": "removed: env_file values are always interpolated, with no way to opt out",
	"priority":    "removed: precedence is fixed — environment: < env_file < OS environment",
	// docs/43: the command-surface restructure took `dva app` and the AppManager
	// runtime with it. Keyed by name alone rather than as a root-only removal
	// because the nested `modes.<name>.applications` went in the same change —
	// the name is now valid at no depth, so the guidance reads correctly wherever
	// a stale config puts it.
	"applications": "removed: declare each app as a stack entry — stack.<name>.default_runner: native + runners.native; run `dva config migrate` to convert",
}

// rootField is the path gojsonschema reports for an error on the document root.
const rootField = "(root)"

// removedRootKeys is the same idea as removedSchemaKeys for keys 17a74b9 folded
// out of the document root, and it is separate for one reason: their names are
// still valid elsewhere. 'compose' names a stack entry and a runner, 'kubectl' a
// runner. Keying those by name alone would append "removed" guidance to the very
// shape that replaced them, and would make the corpus test reject the correct
// examples that teach it.
//
// Guidance from here is only appended when the error is on the root itself.
var removedRootKeys = map[string]string{
	"compose":   "removed from the root: declare it as a stack entry — stack.<name>.default_runner: compose + runners.compose",
	"kubectl":   "removed from the root: declare it as a stack entry — stack.<name>.default_runner: kubectl + runners.kubectl",
	"profiles":  "removed: renamed — use 'modes:'",
	"lifecycle": "removed: renamed — use 'stack:'",
}

// removedInteractionKeys is path-scoped: the property name is still valid at the
// document root (`env_file:`), so it must not join removedSchemaKeys. Guidance
// attaches only when the schema error names an interaction command node.
var removedInteractionKeys = map[string]string{
	"env_file": "removed from interaction: declare shared inputs in the top-level 'env_file:', or inline command-local values under this command's 'environment:'",
}

// isInteractionCommandField reports whether field is an interaction command node
// (`interaction.<name>` or nested `.subcommands.<name>`), not a child property.
func isInteractionCommandField(field string) bool {
	if field == "" || field == rootField {
		return false
	}
	parts := strings.Split(field, ".")
	if len(parts) < 2 || parts[0] != "interaction" {
		return false
	}
	for i := 2; i < len(parts); i += 2 {
		if parts[i-1] != "subcommands" {
			return false
		}
	}
	return len(parts)%2 == 0
}

// validateYAMLSchema checks raw config bytes against the embedded JSON schema.
//
// Shared by Config.Validate (on-disk path) and VerifyMigrated (in-memory rewrite):
// migration used to only decode into structs, which silently drops unknown fields,
// so a dead key could pass the gate and land in the user's dva.yml (TASK-182).
func validateYAMLSchema(yamlBytes []byte) error {
	schemaBytes, err := embeddedSchema.ReadFile("schema.json")
	if err != nil {
		schemaBytes, err = os.ReadFile("schema.json")
		if err != nil {
			return fmt.Errorf("schema file not found: %w", err)
		}
	}

	var yamlData any
	if err := yaml.Unmarshal(yamlBytes, &yamlData); err != nil {
		return fmt.Errorf("invalid YAML syntax: %w", err)
	}

	jsonData := convertYAMLToJSON(yamlData)
	jsonBytes, err := json.Marshal(jsonData)
	if err != nil {
		return fmt.Errorf("converting config to JSON: %w", err)
	}

	schemaLoader := gojsonschema.NewBytesLoader(schemaBytes)
	docLoader := gojsonschema.NewBytesLoader(jsonBytes)

	result, err := gojsonschema.Validate(schemaLoader, docLoader)
	if err != nil {
		return fmt.Errorf("schema validation error: %w", err)
	}

	if !result.Valid() {
		var errs []string
		for _, desc := range result.Errors() {
			line := fmt.Sprintf("  - %s: %s", desc.Field(), desc.Description())
			if desc.Type() == "additional_property_not_allowed" {
				if prop, ok := desc.Details()["property"].(string); ok {
					guidance, removed := removedSchemaKeys[prop]
					if !removed && desc.Field() == rootField {
						guidance, removed = removedRootKeys[prop]
					}
					if !removed && isInteractionCommandField(desc.Field()) {
						guidance, removed = removedInteractionKeys[prop]
					}
					if removed {
						line += "\n      " + guidance
					}
				}
			}
			errs = append(errs, line)
		}
		return fmt.Errorf("schema validation failed in dva.yml:\n%s", strings.Join(errs, "\n"))
	}
	return nil
}

// convertYAMLToJSON recursively converts YAML-decoded data to JSON-compatible types.
// YAML maps decode to map[string]any but sometimes keys are non-string.
func convertYAMLToJSON(v any) any {
	switch val := v.(type) {
	case map[string]any:
		result := make(map[string]any, len(val))
		for k, v := range val {
			result[k] = convertYAMLToJSON(v)
		}
		return result
	case map[any]any:
		result := make(map[string]any, len(val))
		for k, v := range val {
			result[fmt.Sprintf("%v", k)] = convertYAMLToJSON(v)
		}
		return result
	case []any:
		result := make([]any, len(val))
		for i, v := range val {
			result[i] = convertYAMLToJSON(v)
		}
		return result
	default:
		return v
	}
}

// ValidateConfigBytes decodes and validates a dva.yml configuration supplied as raw bytes.
// It enforces the JSON schema, structural constraints, and collects all semantic warnings
// (including canonical section ordering).
func ValidateConfigBytes(data []byte) (*Config, []string, error) {
	var hardErrs []error
	if err := validateYAMLSchema(data); err != nil {
		hardErrs = append(hardErrs, err)
	}

	cfg, err := decodeConfig(data)
	if err != nil {
		hardErrs = append(hardErrs, err)
		return nil, nil, errors.Join(hardErrs...)
	}

	if _, err := finalizeLoadedConfig(cfg); err != nil {
		hardErrs = append(hardErrs, err)
	}

	if err := cfg.validateRemoteDeclarations(); err != nil {
		hardErrs = append(hardErrs, err)
	}
	if err := cfg.validateCIProfiles(); err != nil {
		hardErrs = append(hardErrs, err)
	}
	if conflicts := ValidateReservedCommands(cfg.Interaction); len(conflicts) > 0 {
		var lines []string
		for _, conflict := range conflicts {
			lines = append(lines, fmt.Sprintf("  - interaction.%s: %s", conflict.Name, ConflictAdvice(conflict.Name)))
		}
		hardErrs = append(hardErrs, fmt.Errorf("reserved command conflict in this config:\n%s", strings.Join(lines, "\n")))
	}
	if names := ReservedSubprojectNames(cfg.Subprojects); len(names) > 0 {
		var lines []string
		for _, name := range names {
			lines = append(lines, fmt.Sprintf("  - subprojects.%s: %s", name, SubprojectConflictAdvice(name)))
		}
		hardErrs = append(hardErrs, fmt.Errorf("reserved subproject name in this config:\n%s", strings.Join(lines, "\n")))
	}

	warnings := cfg.ValidateWarnings()
	warnings = append(warnings, validateCanonicalOrderFromBytes(data)...)

	if len(hardErrs) > 0 {
		return cfg, warnings, errors.Join(hardErrs...)
	}
	return cfg, warnings, nil
}
