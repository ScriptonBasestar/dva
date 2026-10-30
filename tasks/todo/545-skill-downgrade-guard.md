---
id: TASK-545
title: "Refuse skill downgrades and finish am-optional guidance"
type: bug
priority: P2
status: todo
created: 2026-10-31
---

## Summary

<!-- One paragraph: what changes and why. -->

## Completion Criteria

- [ ] status separates outdated, newer and differs | verify: `go test -count=1 -run TestStatusClassifiesBundleDrift ./internal/skillinstall`
- [ ] install refuses a provable downgrade | verify: `go test -count=1 -run TestInstallRefusesDowngradeUnlessAllowed ./internal/skillinstall`
- [ ] Agent Mesh keeps sibling links resolvable | verify: `go test -count=1 -run TestAgentMeshRewritesCrossSkillLinks ./internal/skillinstall`
