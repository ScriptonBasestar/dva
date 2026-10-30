---
id: TASK-540
title: "Make installed skills carry a working initialization path"
type: bug
priority: P2
status: todo
created: 2026-10-31
---

## Summary

Installed skills gain a working `dva init` path that needs neither a source checkout nor `am`, and `dva skill status` reports installs that differ from the running binary.

## Completion Criteria

- [ ] dva skill status reports outdated installs | verify: `go test -count=1 -run TestStatusClassifiesBundleDrift ./internal/skillinstall`
- [ ] bundled skills are portable | verify: `go test -count=1 -run TestBundledSkillsAvoidRuntimeOnlyInjection ./internal/skillinstall`
- [ ] dva-config documents dva init | verify: `/usr/bin/grep -q "dva init --dry-run" skills/dva-config/SKILL.md`
