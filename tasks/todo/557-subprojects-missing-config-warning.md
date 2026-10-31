---
id: TASK-557
title: "Warn when a subproject target has no dva.yml"
type: feat
priority: P3
exec-tier: standard
status: todo
created: 2026-11-01
---

# TASK-557: Warn when a subproject target has no dva.yml

## Summary

Validate says nothing when a `subprojects:` target has no dva.yml (observed in sigdock-idp). Source: DVA feedback plan of 2026-10-31, measured on 0.4.2 across 52 projects.

## Completion Criteria

- [ ] Validate warns and names the subproject whose target lacks dva.yml | verify: human — bind a test when the design is settled
