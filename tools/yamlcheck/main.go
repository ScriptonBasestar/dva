// Command yamlcheck extracts fenced YAML blocks annotated with 'yaml dva.yml'
// from repository documentation (primarily USAGE.md) and validates them against
// the JSON schema and semantic warning rules (TASK-357).
//
// A marked example must pass with 0 hard errors and 0 warnings (including canonical
// section ordering). Runs with 0 annotated blocks exit 1 to prevent vacuous passes.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// ExampleBlock represents a fenced YAML snippet extracted from markdown.
type ExampleBlock struct {
	FilePath  string
	StartLine int
	EndLine   int
	Content   string
}

// ExtractExampleBlocks extracts all code fences with info string matching "yaml dva.yml"
// from the given markdown file.
func ExtractExampleBlocks(filePath string) ([]ExampleBlock, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var blocks []ExampleBlock
	scanner := bufio.NewScanner(f)
	lineNum := 0

	inFence := false
	fenceMarker := ""
	var currentLines []string
	startLine := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if !inFence {
			if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
				marker := trimmed[:3]
				info := strings.TrimSpace(trimmed[3:])
				// Match info string: "yaml dva.yml" exactly or prefixed
				if info == "yaml dva.yml" || strings.HasPrefix(info, "yaml dva.yml ") {
					inFence = true
					fenceMarker = marker
					startLine = lineNum
					currentLines = nil
				}
			}
		} else {
			if strings.HasPrefix(trimmed, fenceMarker) && strings.Trim(trimmed, fenceMarker) == "" {
				inFence = false
				blocks = append(blocks, ExampleBlock{
					FilePath:  filePath,
					StartLine: startLine,
					EndLine:   lineNum,
					Content:   strings.Join(currentLines, "\n"),
				})
				currentLines = nil
			} else {
				currentLines = append(currentLines, line)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if inFence {
		return nil, fmt.Errorf("%s:%d: unclosed code fence", filePath, startLine)
	}

	return blocks, nil
}

// CheckResult represents the outcome of checking all extracted blocks.
type CheckResult struct {
	TotalBlocks int
	Errors      []string
	Warnings    []string
}

// ValidateBlocks runs schema and semantic checks against all extracted blocks.
func ValidateBlocks(blocks []ExampleBlock) CheckResult {
	res := CheckResult{
		TotalBlocks: len(blocks),
	}

	for _, b := range blocks {
		data := []byte(b.Content)
		_, warns, err := config.ValidateConfigBytes(data)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("%s:%d: schema error:\n%v", b.FilePath, b.StartLine, err))
		}
		for _, w := range warns {
			res.Warnings = append(res.Warnings, fmt.Sprintf("%s:%d: semantic warning:\n%s", b.FilePath, b.StartLine, w))
		}
	}

	return res
}

func main() {
	targetFile := flag.String("file", "USAGE.md", "Markdown file to scan for annotated YAML examples")
	flag.Parse()

	// If a positional argument is provided, resolve repo root or target
	if flag.NArg() > 0 {
		arg := flag.Arg(0)
		fi, err := os.Stat(arg)
		if err == nil && fi.IsDir() {
			*targetFile = filepath.Join(arg, "USAGE.md")
		} else {
			*targetFile = arg
		}
	}

	blocks, err := ExtractExampleBlocks(*targetFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	if len(blocks) == 0 {
		fmt.Fprintf(os.Stderr, "ERROR: vacuous run — zero '```yaml dva.yml' blocks found in %s\n", *targetFile)
		os.Exit(1)
	}

	res := ValidateBlocks(blocks)
	fmt.Printf("yaml_examples_checked: %d\n", res.TotalBlocks)
	fmt.Printf("yaml_example_errors:   %d\n", len(res.Errors))
	fmt.Printf("yaml_example_warnings: %d\n", len(res.Warnings))

	for _, e := range res.Errors {
		fmt.Fprintf(os.Stderr, "  [error] %s\n", e)
	}
	for _, w := range res.Warnings {
		fmt.Fprintf(os.Stderr, "  [warn]  %s\n", w)
	}

	if len(res.Errors) > 0 || len(res.Warnings) > 0 {
		os.Exit(1)
	}
}
