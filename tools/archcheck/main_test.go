package main

import (
	"reflect"
	"testing"
)

func TestCheckReportsForbiddenAndUndeclared(t *testing.T) {
	graph := parseGraph(modulePath + "internal/lifecycle fmt " + modulePath + "internal/runner\n" +
		modulePath + "internal/new\n")
	got := check(graph, map[string][]string{"internal/lifecycle": {"internal/config"}, "internal/gone": {}})
	want := []string{
		"internal/gone: declared in allowedImports but no such package",
		"internal/lifecycle: must not import internal/runner",
		"internal/new: not declared in allowedImports",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestCheckAcceptsAllowedImports(t *testing.T) {
	graph := map[string][]string{"internal/exec": {"internal/config"}}
	allowed := map[string][]string{"internal/exec": {"internal/config"}}
	if got := check(graph, allowed); len(got) != 0 {
		t.Fatalf("unexpected violations: %v", got)
	}
}
