// Command archcheck: hold internal package imports to the declared dependency direction.
//
// TASK-539: ARCHITECTURE.md described the boundaries, but nothing failed when code crossed
// one. The table below is the machine-checked owner of that direction. A package missing
// from it fails too, so a new package has to state its dependencies.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"sort"
	"strings"
)

const modulePath = "github.com/ScriptonBasestar/dva/"

// allowedImports maps each package to the module packages it may import.
var allowedImports = map[string][]string{
	"cmd/dva":                {"internal/cli"},
	"internal/agentdeny":     {"internal/config", "internal/runner", "internal/skillinstall"},
	"internal/cirun":         {"internal/cirun/ociexec", "internal/config"},
	"internal/cirun/ociexec": {"internal/config"},
	"internal/cli": {
		"internal/agentdeny", "internal/cirun", "internal/config", "internal/exec",
		"internal/jobrun", "internal/lifecycle", "internal/logger", "internal/output",
		"internal/remotetarget", "internal/runner", "internal/secretpush", "internal/skillinstall",
	},
	"internal/config":       {},
	"internal/exec":         {"internal/config"},
	"internal/jobrun":       {"internal/ociverify", "internal/remotetarget"},
	"internal/lifecycle":    {"internal/config", "internal/exec"},
	"internal/logger":       {},
	"internal/ociverify":    {},
	"internal/output":       {},
	"internal/remotetarget": {},
	"internal/runner":       {"internal/config", "internal/exec", "internal/output"},
	"internal/secretpush":   {"internal/remotetarget"},
	"internal/skillclaim":   {},
	"internal/skillinstall": {"internal/skillclaim", "skills"},
}

// targetOSes covers every OS with build-tagged files; a host-only listing misses the rest.
var targetOSes = []string{"darwin", "linux", "windows"}

func main() {
	graph := map[string][]string{}
	for _, goos := range targetOSes {
		cmd := exec.Command("go", "list", "-f", "{{.ImportPath}} {{join .Imports \" \"}}", "./cmd/...", "./internal/...")
		cmd.Env = append(os.Environ(), "GOOS="+goos)
		out, err := cmd.Output()
		if err != nil {
			if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
				fmt.Fprintf(os.Stderr, "%s", exitErr.Stderr)
			}
			fmt.Fprintf(os.Stderr, "archcheck: go list (GOOS=%s): %v\n", goos, err)
			os.Exit(2)
		}
		for pkg, imports := range parseGraph(string(out)) {
			graph[pkg] = append(graph[pkg], imports...)
		}
	}
	violations := check(graph, allowedImports)
	report(os.Stdout, violations)
	if len(violations) > 0 {
		os.Exit(1)
	}
}

// parseGraph reads `path import...` lines into module-relative package imports.
// Test-only packages have no production imports and never enter this graph.
func parseGraph(listing string) map[string][]string {
	graph := map[string][]string{}
	for line := range strings.SplitSeq(strings.TrimSpace(listing), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		pkg := strings.TrimPrefix(fields[0], modulePath)
		imports := []string{}
		for _, imported := range fields[1:] {
			if rel, ok := strings.CutPrefix(imported, modulePath); ok {
				imports = append(imports, rel)
			}
		}
		graph[pkg] = imports
	}
	return graph
}

func check(graph, allowed map[string][]string) []string {
	var violations []string
	for pkg, imports := range graph {
		permitted, known := allowed[pkg]
		if !known {
			violations = append(violations, fmt.Sprintf("%s: not declared in allowedImports", pkg))
			continue
		}
		set := map[string]bool{}
		for _, p := range permitted {
			set[p] = true
		}
		for _, imported := range imports {
			if !set[imported] {
				violations = append(violations, fmt.Sprintf("%s: must not import %s", pkg, imported))
			}
		}
	}
	for pkg := range allowed {
		if _, exists := graph[pkg]; !exists {
			violations = append(violations, fmt.Sprintf("%s: declared in allowedImports but no such package", pkg))
		}
	}
	sort.Strings(violations)
	return slices.Compact(violations)
}

func report(w io.Writer, violations []string) {
	for _, v := range violations {
		_, _ = fmt.Fprintf(w, "ARCH  %s\n", v)
	}
	_, _ = fmt.Fprintf(w, "archcheck: %d violation(s)\n", len(violations))
}
