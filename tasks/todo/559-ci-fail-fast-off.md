---
id: TASK-559
title: "Allow turning off ci fail_fast"
type: feat
priority: P3
exec-tier: standard
status: todo
created: 2026-11-01
---

# TASK-559: Allow turning off ci fail_fast

## Summary

`dva ci` offers no way to turn fail_fast off (noted in the funbricks-notifire config comments). Source: DVA feedback plan of 2026-10-31, measured on 0.4.2 across 52 projects.

## Completion Criteria

- [ ] A ci profile can continue after a failed step and still exits non-zero | verify: human — bind a test when the design is settled
