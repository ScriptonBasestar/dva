package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestBindingPortabilityRejectsGrepDashL(t *testing.T) {
	res := portabilityFixture(t, "- [ ] absence | verify: `/usr/bin/grep -rq -L 'needle' docs`\n")
	if res.OK || res.InvertedGrepBindings != 1 || !containsAny(res.PortabilityDetail, "bsd and gnu", "grep -l") {
		t.Fatalf("inverted=%d detail=%v ok=%v", res.InvertedGrepBindings, res.PortabilityDetail, res.OK)
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
