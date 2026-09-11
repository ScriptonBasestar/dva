package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractExampleBlocks(t *testing.T) {
	doc := `# Title

Some intro text.

` + "```yaml" + `
# Unannotated YAML
foo: bar
` + "```" + `

` + "```yaml dva.yml" + `
version: "0.1"
stack:
  api:
    default_runner: compose
` + "```" + `

Another section.

` + "~~~yaml dva.yml" + `
version: "0.1"
plans:
  dev:
    entries: []
` + "~~~" + `

` + "```go" + `
func main() {}
` + "```" + `
`

	tmpDir := t.TempDir()
	docPath := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(docPath, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}

	blocks, err := ExtractExampleBlocks(docPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}

	if blocks[0].StartLine != 10 {
		t.Errorf("block 0: expected startLine 10, got %d", blocks[0].StartLine)
	}
	if !strings.Contains(blocks[0].Content, "stack:") {
		t.Errorf("block 0: expected stack content, got %q", blocks[0].Content)
	}

	if blocks[1].StartLine != 19 {
		t.Errorf("block 1: expected startLine 19, got %d", blocks[1].StartLine)
	}
	if !strings.Contains(blocks[1].Content, "plans:") {
		t.Errorf("block 1: expected plans content, got %q", blocks[1].Content)
	}
}

func TestValidateBlocksCleanPass(t *testing.T) {
	validYAML := `version: "0.1"

vars:
  APP_PORT: "3000"

environment:
  LOG_LEVEL: debug

stack:
  api:
    default_runner: compose
    runners:
      compose:
        files: [docker-compose.yml]

plans:
  dev:
    description: "Local development"
    entries:
      - name: api
    environment: local
    site: laptop

default_plan: dev

environments:
  local:
    environment:
      LOG_LEVEL: debug

sites:
  laptop:
    vars:
      APP_PORT: "3001"
`

	blocks := []ExampleBlock{
		{
			FilePath:  "USAGE.md",
			StartLine: 865,
			EndLine:   900,
			Content:   validYAML,
		},
	}

	res := ValidateBlocks(blocks)
	if len(res.Errors) > 0 {
		t.Fatalf("expected 0 errors, got %d: %v", len(res.Errors), res.Errors)
	}
	if len(res.Warnings) > 0 {
		t.Fatalf("expected 0 warnings, got %d: %v", len(res.Warnings), res.Warnings)
	}
}

func TestValidateBlocksDetectsSchemaError(t *testing.T) {
	invalidYAML := `version: "0.1"
invalid_root_property: true
stack:
  api:
    runners:
      compose:
        unknown_runner_key: true
`
	blocks := []ExampleBlock{
		{
			FilePath:  "USAGE.md",
			StartLine: 10,
			EndLine:   20,
			Content:   invalidYAML,
		},
	}

	res := ValidateBlocks(blocks)
	if len(res.Errors) == 0 {
		t.Fatalf("expected schema errors for invalid property, got none")
	}
}

func TestValidateBlocksDetectsSemanticWarnings(t *testing.T) {
	// modes: triggers deprecation warning
	yamlWithModes := `version: "0.1"
modes:
  dev:
    compose_services: [api]
`
	blocks := []ExampleBlock{
		{
			FilePath:  "USAGE.md",
			StartLine: 50,
			EndLine:   60,
			Content:   yamlWithModes,
		},
	}

	res := ValidateBlocks(blocks)
	if len(res.Warnings) == 0 {
		t.Fatalf("expected semantic warning for deprecated 'modes', got none")
	}
}

func TestValidateBlocksDetectsSectionOrderWarning(t *testing.T) {
	// Out-of-order: plans before stack
	outOfOrderYAML := `version: "0.1"
plans:
  dev:
    entries:
      - name: api
stack:
  api:
    default_runner: compose
    runners:
      compose:
        files: [docker-compose.yml]
`
	blocks := []ExampleBlock{
		{
			FilePath:  "USAGE.md",
			StartLine: 70,
			EndLine:   85,
			Content:   outOfOrderYAML,
		},
	}

	res := ValidateBlocks(blocks)
	foundOrderWarn := false
	for _, w := range res.Warnings {
		if strings.Contains(w, "section order:") {
			foundOrderWarn = true
			break
		}
	}
	if !foundOrderWarn {
		t.Fatalf("expected section order warning, got %v", res.Warnings)
	}
}
