---
id: TASK-564
title: "Stop reporting parent-owned compose as drift"
type: bug
priority: P3
exec-tier: standard
status: review
created: 2026-11-01
---

# TASK-564: Stop reporting parent-owned compose as drift

## Summary

A compose file owned by a parent project is reported as drift in the child. Source: DVA feedback plan of 2026-10-31, measured on 0.4.2 across 52 projects.

## Completion Criteria

- [ ] Drift detection does not report a compose file the parent owns, and still reports a real orphan | verify: `go test ./internal/cli -run '^TestDetectConfigDriftWarnings_(Parent|AncestorWithout|UnloadableParent|UnsupportedParent|SymlinkedSubproject)'`
