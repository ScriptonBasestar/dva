package cli

import (
	"regexp"
	"strings"
)

type makeVariable struct {
	value     string
	recursive bool
}

var makeVariableReference = regexp.MustCompile(`\$\(([^()]+)\)|\$\{([^{}]+)\}`)

// parseSimpleMakeVariables preserves GNU Make's two relevant variable flavors:
// := expands at assignment time, while = and ?= expand when a recipe is read.
func parseSimpleMakeVariables(lines []string) map[string]makeVariable {
	variables := map[string]makeVariable{}
	for _, line := range lines {
		if line == "" || strings.HasPrefix(line, "\t") || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		for _, operator := range []string{":=", "?=", "="} {
			name, value, ok := strings.Cut(line, operator)
			name = strings.TrimSpace(name)
			if !ok || name == "" || strings.ContainsAny(name, " :$+") {
				continue
			}
			if operator == "?=" {
				if _, exists := variables[name]; exists {
					break
				}
			}
			value = strings.TrimSpace(value)
			if operator == ":=" {
				value = expandMakeVariables(value, variables, map[string]bool{})
			}
			variables[name] = makeVariable{value: value, recursive: operator != ":="}
			break
		}
	}
	return variables
}

func expandMakeVariables(value string, variables map[string]makeVariable, visiting map[string]bool) string {
	matches := makeVariableReference.FindAllStringSubmatchIndex(value, -1)
	if len(matches) == 0 {
		return value
	}
	var b strings.Builder
	last := 0
	for _, match := range matches {
		b.WriteString(value[last:match[0]])
		nameStart, nameEnd := match[2], match[3]
		if nameStart < 0 {
			nameStart, nameEnd = match[4], match[5]
		}
		name := strings.TrimSpace(value[nameStart:nameEnd])
		variable, found := variables[name]
		if !found || visiting[name] {
			b.WriteString(value[match[0]:match[1]])
		} else if variable.recursive {
			visiting[name] = true
			b.WriteString(expandMakeVariables(variable.value, variables, visiting))
			delete(visiting, name)
		} else {
			// A := value was frozen when assigned. Any reference left in it
			// was unresolved then and must not bind to a later assignment.
			b.WriteString(variable.value)
		}
		last = match[1]
	}
	b.WriteString(value[last:])
	return b.String()
}

func firstMakeVariableReference(value string) string {
	if match := makeVariableReference.FindString(value); match != "" {
		return match
	}
	return ""
}

// normalizeMakeRecipeDollars preserves the shell command Make would execute.
// Make turns $$ into one literal dollar before invoking the shell. Every other
// dollar form is a Make reference here; named references should already have
// expanded, and automatic/single-character references have no safe value when
// init moves the recipe outside Make.
func normalizeMakeRecipeDollars(value string) (string, string) {
	var b strings.Builder
	for i := 0; i < len(value); i++ {
		if value[i] != '$' {
			b.WriteByte(value[i])
			continue
		}
		if i+1 >= len(value) {
			return "", "$"
		}
		next := value[i+1]
		if next == '$' {
			b.WriteByte('$')
			i++
			continue
		}
		if next == '(' || next == '{' {
			if reference := firstMakeVariableReference(value[i:]); reference != "" {
				return "", reference
			}
		}
		return "", value[i : i+2]
	}
	return b.String(), ""
}
