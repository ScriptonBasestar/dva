---
id: TASK-332
title: "Remove the dead interaction env_file deprecation warning"
type: chore
priority: P3
effort: S
exec-tier: cheap
status: todo
created: 2026-09-07
source: "tasks/done/259 done-review (PLAN-007 Tier A batch 1)"
---

## Summary

`InteractionEnvFileMessage` (`internal/config/config.go:591-605`) has no caller. The
deprecation it announces — interaction-level `env_file:` being rejected in 0.1.49 — already
landed, so the constant and its doc comment are dead weight that reads as a live warning.
`internal/config/migrate_report.go:248` carries a doc comment cross-referencing it and must
lose that reference in the same change.

Nothing behavioural changes: the string is never printed today.

## Completion Criteria

- [ ] `InteractionEnvFileMessage` and its doc comment are gone from `internal/config/config.go` | verify: `! /usr/bin/grep -q InteractionEnvFileMessage internal/config/config.go`
- [ ] the cross-reference in `internal/config/migrate_report.go` is gone | verify: `! /usr/bin/grep -q InteractionEnvFileMessage internal/config/migrate_report.go`
- [ ] the build and tests stay green | verify: `make test`
