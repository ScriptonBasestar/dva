package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestBindingPortabilityRejectsGrepDashL(t *testing.T) {
	for _, binding := range []string{
		"/usr/bin/grep -rq -L 'needle' docs",
		"! /usr/bin/find tasks/done -name '*.md' | /usr/bin/xargs -r /usr/bin/grep -L '^quality-review:' | /usr/bin/grep -q .",
	} {
		res := portabilityFixture(t, "- [ ] absence | verify: `"+binding+"`\n")
		if res.OK || res.InvertedGrepBindings != 1 || !containsAny(res.PortabilityDetail, "bsd and gnu", "grep -l") {
			t.Fatalf("binding=%q inverted=%d detail=%v ok=%v", binding, res.InvertedGrepBindings, res.PortabilityDetail, res.OK)
		}
	}
}

func TestBindingPortabilityDoesNotReadQuotedGrepDashL(t *testing.T) {
	res := portabilityFixture(t, "- [ ] quoted prose | verify: `printf '%s\\n' 'grep -L is not an invocation'`\n")
	if res.InvertedGrepBindings != 0 {
		t.Fatalf("quoted literal produced inverted=%d detail=%v", res.InvertedGrepBindings, res.PortabilityDetail)
	}
}

func TestBindingVacuityRejectsBareSuiteTarget(t *testing.T) {
	for _, target := range []string{"make test", "make lint", "make check", "make doc-check", "go test ./..."} {
		t.Run(target, func(t *testing.T) {
			res := portabilityFixture(t, "- [ ] suite | verify: `"+target+"`\n")
			if res.OK || res.BareSuiteBindings != 1 || !containsAny(res.PortabilityDetail, "bare suite target") {
				t.Fatalf("target=%q bare=%d detail=%v ok=%v", target, res.BareSuiteBindings, res.PortabilityDetail, res.OK)
			}
		})
	}
}

func TestBindingVacuityHonoursRegressionGuardMarkerPerLine(t *testing.T) {
	body := strings.Join([]string{
		"- [ ] guarded | verify: `make test` (regression-guard)",
		"- [ ] next line is independently checked | verify: `make test`",
	}, "\n")
	res := portabilityFixture(t, body)
	if res.BareSuiteBindings != 1 || !containsAny(res.PortabilityDetail, "next line") && !containsAny(res.PortabilityDetail, ":2:") {
		t.Fatalf("bare=%d detail=%v", res.BareSuiteBindings, res.PortabilityDetail)
	}
}

func TestBindingVacuityRejectsAlreadyMatchingTestName(t *testing.T) {
	res := portabilityFixture(t, "- [ ] future test | verify: `/usr/bin/grep -rq 'func TestSameStringSet(' pkg`\n")
	if res.OK || res.ExistingTodoTestNames != 1 || !containsAny(res.PortabilityDetail, "already-declared test testsamestringset") {
		t.Fatalf("existing=%d detail=%v ok=%v", res.ExistingTodoTestNames, res.PortabilityDetail, res.OK)
	}
}

func TestBindingVacuityAllowsNegatedExistingTestName(t *testing.T) {
	res := portabilityFixture(t, "- [ ] absent test is required | verify: `! /usr/bin/grep -rq 'func TestSameStringSet(' pkg`\n")
	if res.ExistingTodoTestNames != 0 {
		t.Fatalf("negated grep produced existing=%d detail=%v", res.ExistingTodoTestNames, res.PortabilityDetail)
	}
}

func TestBindingVacuityUsesGrepTestCorpus(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docs/a.md", "# A\n\nSee [self](a.md).\n")
	writeFile(t, root, "tasks/todo/001-portability.md", strings.Join([]string{
		"- [ ] different package | verify: `/usr/bin/grep -rq 'func TestSharedName(' pkg/right`",
		"- [ ] matching package after another command | verify: `test -d pkg/left && /usr/bin/grep -rq 'func TestSharedName(' pkg/left`",
	}, "\n"))
	writeFile(t, root, "pkg/left/thing_test.go", "package left\nimport \"testing\"\nfunc TestSharedName(t *testing.T) {}\n")
	writeFile(t, root, "pkg/right/thing_test.go", "package right\nimport \"testing\"\nfunc TestOtherName(t *testing.T) {}\n")
	inv := mustInventory(t, root, "docs/a.md", "tasks/todo/001-portability.md", "pkg/left/thing_test.go", "pkg/right/thing_test.go")
	res := Check(CheckInput{Root: root, Inventory: inv})
	if res.ExistingTodoTestNames != 1 || !containsAny(res.PortabilityDetail, "testsharedname in pkg/left") {
		t.Fatalf("existing=%d detail=%v", res.ExistingTodoTestNames, res.PortabilityDetail)
	}
}

func TestCheckReportsBindingInversionAndVacuityCounts(t *testing.T) {
	res := portabilityFixture(t, strings.Join([]string{
		"- [ ] inverse | verify: `/usr/bin/grep -L needle docs`",
		"- [ ] suite | verify: `make test`",
		"- [ ] old test | verify: `/usr/bin/grep -rq 'func TestSameStringSet(' pkg`",
	}, "\n"))
	if res.InvertedGrepBindings != 1 || res.BareSuiteBindings != 1 || res.ExistingTodoTestNames != 1 {
		t.Fatalf("counters inverted=%d bare=%d existing=%d", res.InvertedGrepBindings, res.BareSuiteBindings, res.ExistingTodoTestNames)
	}
	output := captureStdout(t, func() { printReport(res) })
	for _, line := range []string{"inverted_grep_bindings: 1", "bare_suite_bindings: 1", "existing_todo_test_bindings: 1"} {
		if !strings.Contains(output, line) {
			t.Fatalf("report misses %q:\n%s", line, output)
		}
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	previous := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	fn()
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stdout = previous
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	return string(data)
}
