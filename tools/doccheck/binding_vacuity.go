package main

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// grepDashLRe permits ordinary grep option words before -L.  It accepts a
	// whitespace boundary too because grep can be xargs' operand rather than a
	// pipeline command. -L reports files with no match, but its exit status
	// differs between BSD and GNU grep; it cannot serve as a portable absence
	// assertion in a binding.
	grepDashLRe = regexp.MustCompile(`(?:^|\s|[;&|])(?:/usr/bin/)?grep(?:\s+-[A-Za-z0-9_-]+)*\s+-[A-Za-z0-9_-]*L[A-Za-z0-9_-]*(?:\s|$)`)

	// grepTestNameRe is deliberately limited to the future-test idiom used by
	// cards.  A todo card naming a test that already exists proves no proposed
	// change; completed cards are outside the checked zone and preserve history.
	grepTestNameRe = regexp.MustCompile(`(?:^|\s|[;&|])(?:/usr/bin/)?grep(?:\s+-[A-Za-z0-9_-]+)*\s+['"]?func\s+(Test[A-Za-z0-9_]+)\(['"]?\s+([^\s;|&]+)`)
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
func checkBindingVacuity(from, body string, testDecls map[string][]string) (inverted, bareSuite, existingTodoTest int, msgs []string) {
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
		for _, match := range grepTestNameRe.FindAllStringSubmatch(binding.Span, -1) {
			if testDeclaredInCorpus(testDecls[match[1]], match[2]) {
				existingTodoTest++
				msgs = append(msgs, fmt.Sprintf("%s:%d: todo verify binding names already-declared test %s in %s; it must not pass before this card is completed", from, binding.Line, match[1], match[2]))
			}
		}
	}
	return inverted, bareSuite, existingTodoTest, msgs
}

func testDeclaredInCorpus(paths []string, corpus string) bool {
	corpus = strings.TrimPrefix(strings.TrimSuffix(corpus, "/"), "./")
	if corpus == "" || corpus == "." {
		return len(paths) > 0
	}
	for _, testPath := range paths {
		if testPath == corpus || strings.HasPrefix(testPath, corpus+"/") {
			return true
		}
	}
	return false
}

func isOpenTaskCardPath(from string) bool {
	return strings.HasPrefix(from, "tasks/todo/") || strings.HasPrefix(from, "tasks/doing/")
}
