package main

import (
	"reflect"
	"testing"
)

func TestCheckReportsForbiddenAndUndeclared(t *testing.T) {
	graph := parseGraph(modulePath + "internal/lifecycle fmt " + modulePath + "internal/runner\n" +
		modulePath + "internal/new\n")
	got := check(graph, map[string][]string{"internal/lifecycle": {"internal/config"}})
	want := []string{
		"internal/lifecycle: must not import internal/runner",
		"internal/new: not declared in allowedImports",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestCheckAcceptsAllowedImports(t *testing.T) {
	graph := map[string][]string{"internal/exec": {"internal/config"}}
	if got := check(graph, allowedImports); len(got) != 0 {
		t.Fatalf("unexpected violations: %v", got)
	}
}
