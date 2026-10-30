---
id: TASK-540
title: "Make installed skills carry a working initialization path"
type: bug
priority: P2
status: todo
created: 2026-10-31
---

## Summary

<!-- One paragraph: what changes and why. -->

## Completion Criteria

- [ ] dva skill status reports outdated installs | verify: `go test -count=1 -run "^TestStatusReportsOutdatedBundle$" -v ./internal/skillinstall | /usr/bin/grep -q "^--- PASS"`
- [ ] bundled skills are portable | verify: `go test -count=1 -run "^TestBundledSkillsAvoidRuntimeOnlyInjection$" -v ./internal/skillinstall | /usr/bin/grep -q "^--- PASS"`
- [ ] dva-config documents dva init | verify: `/usr/bin/grep -q "dva init --dry-run" skills/dva-config/SKILL.md`
