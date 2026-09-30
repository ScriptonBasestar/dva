package main

import (
	"strings"
	"testing"
)

func TestBindingGateRecursion(t *testing.T) {
	for _, binding := range []string{
		"ce task gate",
		"ce task gate --json",
		"ce task gate --dir tasks",
		"make doc-check && ce task gate",
		"/usr/local/bin/ce task gate",
		"ce task gate; echo done",
		"timeout 300 ce task gate",
		"ce validate --all; ce task gate",
	} {
		res := portabilityFixture(t, "- [ ] gate | verify: `"+binding+"`\n")
		if res.OK || res.GateRecursionBindings != 1 || !containsAny(res.PortabilityDetail, "recurses without bound") {
			t.Fatalf("binding=%q gate=%d detail=%v ok=%v", binding, res.GateRecursionBindings, res.PortabilityDetail, res.OK)
		}
		if !strings.HasPrefix(res.PortabilityDetail[0], "tasks/todo/001-portability.md:1:") {
			t.Fatalf("detail %q does not carry task file and line", res.PortabilityDetail[0])
		}
	}
}

func TestBindingGateRecursionDoesNotMatchLookalikes(t *testing.T) {
	for _, binding := range []string{
		"make doc-check && ce task validate --all",
		"ce task validate",
		"echo 'ce task gate is forbidden'",
		"printf '%s\\n' \"run ce task gate later\"",
		"ce lint",
	} {
		res := portabilityFixture(t, "- [ ] safe | verify: `"+binding+"`\n")
		if res.GateRecursionBindings != 0 {
			t.Fatalf("binding=%q matched gate=%d detail=%v", binding, res.GateRecursionBindings, res.PortabilityDetail)
		}
	}
}

func TestBindingGateRecursionSkipsArchive(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docs/a.md", "# A\n\nSee [self](a.md).\n")
	writeFile(t, root, "tasks/_archive/2026-09/001-kept.md", "- [x] history | verify: `ce task gate`\n")
	inv := mustInventory(t, root, "docs/a.md", "tasks/_archive/2026-09/001-kept.md")
	res := Check(CheckInput{Root: root, Inventory: inv})
	if res.GateRecursionBindings != 0 {
		t.Fatalf("archive binding matched: gate=%d detail=%v", res.GateRecursionBindings, res.PortabilityDetail)
	}
}
