package main

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

var (
	// grepDashLRe permits ordinary grep option words before -L.  -L reports
	// files with no match, but its exit status differs between BSD and GNU grep;
	// it therefore cannot serve as a portable absence assertion in a binding.
	grepDashLRe = regexp.MustCompile(`(?:^|[;&|]\s*)(?:/usr/bin/)?grep(?:\s+-[A-Za-z0-9_-]+)*\s+-[A-Za-z0-9_-]*L[A-Za-z0-9_-]*(?:\s|$)`)

	// grepTestNameRe is deliberately limited to the future-test idiom used by
	// cards.  A todo card naming a test that already exists proves no proposed
	// change; completed cards are outside the checked zone and preserve history.
	grepTestNameRe = regexp.MustCompile(`\bfunc\s+(Test[A-Za-z0-9_]+)\s*\(`)
)

var bareSuiteTargets = map[string]struct{}{
	"make test":      {},
	"make lint":      {},
	"make check":     {},
	"make doc-check": {},
	"go test ./...":  {},
}

// checkBindingVacuity rejects bindings that can pass while proving nothing.
// It applies these prevention checks only to open task cards; completed records
// remain historical evidence and must not be rewritten.
func checkBindingVacuity(from, body string, testNames []string) (inverted, bareSuite, existingTodoTest int, msgs []string) {
	if !isOpenTaskCardPath(from) {
		return 0, 0, 0, nil
	}
	for _, binding := range extractVerifyBindings(body) {
		if grepDashLRe.MatchString(binding.Span) {
			inverted++
			msgs = append(msgs, fmt.Sprintf("%s:%d: verify binding passes grep -L; BSD and GNU grep invert its success meaning, use ! /usr/bin/grep -q for absence", from, binding.Line))
		}

		if _, bare := bareSuiteTargets[strings.TrimSpace(binding.Span)]; bare && !binding.RegressionGuard {
			bareSuite++
			msgs = append(msgs, fmt.Sprintf("%s:%d: verify binding is a bare suite target %q; add a criterion-specific binding or (regression-guard)", from, binding.Line, binding.Span))
		}
		if match := grepTestNameRe.FindStringSubmatch(binding.Span); match != nil && slices.Contains(testNames, match[1]) {
			existingTodoTest++
			msgs = append(msgs, fmt.Sprintf("%s:%d: todo verify binding names already-declared test %s; it must not pass before this card is completed", from, binding.Line, match[1]))
		}
	}
	return inverted, bareSuite, existingTodoTest, msgs
}

func isOpenTaskCardPath(from string) bool {
	return strings.HasPrefix(from, "tasks/todo/") || strings.HasPrefix(from, "tasks/doing/")
}
