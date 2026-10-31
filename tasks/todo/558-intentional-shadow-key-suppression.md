---
id: TASK-558
title: "Declare intended shadow keys"
type: feat
priority: P3
exec-tier: standard
status: todo
created: 2026-11-01
---

# TASK-558: Declare intended shadow keys

## Summary

An intended shadow key has no suppression, so its warning stays under --strict (observed in funbricks-notifire). Source: DVA feedback plan of 2026-10-31, measured on 0.4.2 across 52 projects.

## Completion Criteria

- [ ] A declared intended shadow key no longer fails --strict, and an undeclared one still does | verify: human — bind a test when the design is settled
