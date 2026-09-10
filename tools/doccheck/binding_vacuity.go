package main

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// grepTestNameRe is deliberately limited to the future-test idiom used by
	// cards.  A todo card naming a test that already exists proves no proposed
	// change. The optional leading ! is captured so an absence assertion is not
	// mistaken for a positive proof that the proposed test already exists.
	grepTestNameRe = regexp.MustCompile(`(?:^|[;&|])\s*(!\s*)?(?:/usr/bin/)?grep(?:\s+-[A-Za-z0-9_-]+)*\s+['"]?func\s+(Test[A-Za-z0-9_]+)\(['"]?\s+([^\s;|&]+)`)
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
		if bindingInvokesGrepDashL(binding.Span) {
			inverted++
			msgs = append(msgs, fmt.Sprintf("%s:%d: verify binding passes grep -L; BSD and GNU grep invert its success meaning, use ! /usr/bin/grep -q for absence", from, binding.Line))
		}

		if _, bare := bareSuiteTargets[strings.TrimSpace(binding.Span)]; bare && !binding.RegressionGuard {
			bareSuite++
			msgs = append(msgs, fmt.Sprintf("%s:%d: verify binding is a bare suite target %q; add a criterion-specific binding or (regression-guard)", from, binding.Line, binding.Span))
		}
		for _, match := range grepTestNameRe.FindAllStringSubmatch(binding.Span, -1) {
			if match[1] == "" && testDeclaredInCorpus(testDecls[match[2]], match[3]) {
				existingTodoTest++
				msgs = append(msgs, fmt.Sprintf("%s:%d: todo verify binding names already-declared test %s in %s; it must not pass before this card is completed", from, binding.Line, match[2], match[3]))
			}
		}
	}
	return inverted, bareSuite, existingTodoTest, msgs
}

type bindingShellToken struct {
	text   string
	quoted bool
	op     bool
}

// bindingInvokesGrepDashL finds a real grep command instead of scanning quoted
// text. It also recognizes xargs' command operand, which is how TASK-367 used
// grep -L. This is intentionally a narrow shell reader: it does not evaluate
// expansions, but it preserves enough command and quote boundaries for a
// portable documentation binding check.
func bindingInvokesGrepDashL(span string) bool {
	tokens := bindingShellTokens(span)
	for i := 0; i < len(tokens); i++ {
		if tokens[i].op || !isCommandPosition(tokens, i) {
			continue
		}
		command := bindingCommandName(tokens[i].text)
		switch command {
		case "grep":
			if grepArgsContainDashL(tokens, i+1) {
				return true
			}
		case "xargs":
			if operand := xargsCommandOperand(tokens, i+1); operand >= 0 && bindingCommandName(tokens[operand].text) == "grep" && grepArgsContainDashL(tokens, operand+1) {
				return true
			}
		}
	}
	return false
}

func bindingShellTokens(s string) []bindingShellToken {
	var tokens []bindingShellToken
	var word strings.Builder
	quoted := false
	flush := func() {
		if word.Len() > 0 {
			tokens = append(tokens, bindingShellToken{text: word.String(), quoted: quoted})
			word.Reset()
			quoted = false
		}
	}
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\'', '"':
			quote := s[i]
			quoted = true
			for i++; i < len(s) && s[i] != quote; i++ {
				word.WriteByte(s[i])
			}
		case '\\':
			if i+1 < len(s) {
				word.WriteByte(s[i+1])
				i++
			} else {
				word.WriteByte(s[i])
			}
		case ' ', '\t', '\r', '\n':
			flush()
		case ';', '|', '&':
			flush()
			tokens = append(tokens, bindingShellToken{text: string(s[i]), op: true})
			if i+1 < len(s) && s[i+1] == s[i] && (s[i] == '&' || s[i] == '|') {
				i++
			}
		default:
			word.WriteByte(s[i])
		}
	}
	flush()
	return tokens
}

func isCommandPosition(tokens []bindingShellToken, i int) bool {
	if i == 0 || tokens[i-1].op {
		return true
	}
	for j := i - 1; j >= 0 && !tokens[j].op; j-- {
		if tokens[j].text != "!" && !isShellAssignment(tokens[j].text) && !isCommandPrefix(tokens[j].text) {
			return false
		}
	}
	return true
}

func bindingCommandName(word string) string {
	if slash := strings.LastIndexByte(word, '/'); slash >= 0 {
		word = word[slash+1:]
	}
	return word
}

func grepArgsContainDashL(tokens []bindingShellToken, start int) bool {
	for i := start; i < len(tokens) && !tokens[i].op; i++ {
		arg := tokens[i]
		if arg.quoted || !strings.HasPrefix(arg.text, "-") || arg.text == "--" {
			return false
		}
		if strings.Contains(arg.text[1:], "L") {
			return true
		}
	}
	return false
}

func xargsCommandOperand(tokens []bindingShellToken, start int) int {
	for i := start; i < len(tokens) && !tokens[i].op; i++ {
		if tokens[i].text == "--" {
			if i+1 < len(tokens) && !tokens[i+1].op {
				return i + 1
			}
			return -1
		}
		if strings.HasPrefix(tokens[i].text, "-") {
			if xargsOptionNeedsValue(tokens[i].text) && i+1 < len(tokens) && !tokens[i+1].op {
				i++
			}
			continue
		}
		return i
	}
	return -1
}

func xargsOptionNeedsValue(option string) bool {
	switch option {
	case "-n", "-s", "-P", "-E", "-e", "-I", "-L", "-d", "--max-args", "--max-chars", "--max-procs", "--eof", "--replace", "--max-lines", "--delimiter":
		return true
	default:
		return false
	}
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
