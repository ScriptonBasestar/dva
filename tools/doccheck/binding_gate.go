package main

import "fmt"

// checkBindingGateRecursion rejects verify bindings that run `ce task gate`.
// The gate's bindings stage re-executes every checked machine binding, so a
// binding that names the gate makes the gate re-spawn itself; the stage's 30s
// timeout kills only the direct child and the grandchildren reparent to PID 1
// (ISSUE-454: 335 orphaned processes in 29 minutes on 2026-09-30). The
// upstream runtime guard is ce-agent-kit's; this is the authoring-time
// defense. Archived bindings are not re-executed by the gate — observed on a
// READY board carrying several such archive bindings — and archived cards are
// historical evidence, so callers exclude tasks/_archive instead of this
// function.
func checkBindingGateRecursion(from, body string) (count int, msgs []string) {
	for _, binding := range extractVerifyBindings(body) {
		if !bindingInvokesCeTaskGate(binding.Span) {
			continue
		}
		count++
		msgs = append(msgs, fmt.Sprintf("%s:%d: verify binding invokes ce task gate; the gate's bindings stage re-executes checked bindings and recurses without bound — use make doc-check or ce task validate --all", from, binding.Line))
	}
	return count, msgs
}

// bindingInvokesCeTaskGate reports whether the token span runs the `ce`
// command with `task gate` as its first two non-flag arguments, in any
// command position of the span (`ce task gate`, `make doc-check && ce task
// gate --json`, `/path/to/ce task gate`). Quoted words are data, not
// commands, and never match. Substantive flags such as --dir may sit after
// the subcommands; they do not affect the match.
func bindingInvokesCeTaskGate(span string) bool {
	tokens := bindingShellTokens(span)
	for i := range tokens {
		if tokens[i].op || tokens[i].quoted || !gateCommandPosition(tokens, i) {
			continue
		}
		if bindingCommandName(tokens[i].text) != "ce" {
			continue
		}
		args := 0
		matched := false
	walk:
		for j := i + 1; j < len(tokens) && !tokens[j].op; j++ {
			if tokens[j].quoted || len(tokens[j].text) > 0 && tokens[j].text[0] == '-' {
				continue
			}
			switch args {
			case 0:
				if tokens[j].text != "task" {
					break walk
				}
			case 1:
				matched = tokens[j].text == "gate"
				break walk
			}
			args++
		}
		if matched {
			return true
		}
	}
	return false
}

// gateCommandPosition is isCommandPosition with transparent wrappers widened:
// a landmine binding may arrive as `timeout 300 ce task gate` or `env … ce
// task gate`, whose numeric and wrapper words sit between the prefix and the
// command. The grep -L check keeps the narrower isCommandPosition; widening it
// there is a separate portability question this check does not own.
func gateCommandPosition(tokens []bindingShellToken, i int) bool {
	if i == 0 || tokens[i-1].op {
		return true
	}
	for j := i - 1; j >= 0 && !tokens[j].op; j-- {
		w := tokens[j].text
		if w == "!" || isShellAssignment(w) || isCommandPrefix(w) || isTransparentWrapper(w) {
			continue
		}
		return false
	}
	return true
}

func isTransparentWrapper(w string) bool {
	switch w {
	case "timeout", "nice", "nohup", "env", "stdbuf":
		return true
	}
	if w == "" {
		return false
	}
	for _, r := range w {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
