package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

func TestGeneratedReservedCommandCorpus(t *testing.T) {
	files := []string{
		"agent-mesh-flows/shared/library/shared-guardrails.md",
		"agent-mesh-flows/dva-diagnose.yaml",
		"agent-mesh-flows/dva-improve-guided/00-analyze.yaml",
		"agent-mesh-flows/dva-improve-guided/30-configure.yaml",
		"agent-mesh-flows/dva-improve.yaml",
		"internal/cli/library_reference.txt",
	}
	start, end := "<!-- AUTOGEN:reserved_commands:start -->", "<!-- AUTOGEN:reserved_commands:end -->"
	name := regexp.MustCompile("`([^`]+)`")
	var want []string
	for command := range ReservedCommands() {
		want = append(want, command)
	}
	sort.Strings(want)
	total := 0
	for _, path := range files {
		data, err := os.ReadFile(filepath.Join("../..", path))
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		starts, ends := strings.Count(text, start), strings.Count(text, end)
		if starts == 0 || starts != ends {
			t.Errorf("%s: unpaired or missing blocks: %d starts, %d ends", path, starts, ends)
			continue
		}
		if path == "agent-mesh-flows/dva-improve.yaml" && starts != 3 {
			t.Errorf("%s: got %d blocks, want 3", path, starts)
		}
		total += starts
		for i := 0; i < starts; i++ {
			before, rest, ok := strings.Cut(text, start)
			if !ok || strings.Contains(before, end) {
				t.Fatalf("%s: malformed block order", path)
			}
			block, rest, ok := strings.Cut(rest, end)
			if !ok || strings.Contains(block, start) {
				t.Fatalf("%s: malformed block order", path)
			}
			line := strings.Split(strings.TrimSpace(block), "\n")[0]
			matches := name.FindAllStringSubmatch(line, -1)
			got := make([]string, 0, len(matches))
			for _, m := range matches {
				got = append(got, m[1])
			}
			if !strings.Contains(line, fmt.Sprintf("Reserved (%d,", len(want))) || strings.Join(got, ",") != strings.Join(want, ",") {
				t.Errorf("%s block %d: stale reserved commands: %s", path, i+1, line)
			}
			text = rest
		}
		if strings.Contains(text, end) {
			t.Errorf("%s: extra end marker", path)
		}
	}
	if total != 8 {
		t.Errorf("got %d blocks, want 8", total)
	}
}
