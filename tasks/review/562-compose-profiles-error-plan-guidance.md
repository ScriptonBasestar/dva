---
id: TASK-562
title: "Point compose profiles errors at plan entries"
type: bug
priority: P3
exec-tier: standard
status: review
created: 2026-11-01
---

# TASK-562: Point compose profiles errors at plan entries

## Summary

The stack runner compose `profiles` error does not tell the author to use a plan entry. Source: DVA feedback plan of 2026-10-31, measured on 0.4.2 across 52 projects.

## Completion Criteria

- [ ] The stack runner compose profiles error names the plan entry fix | verify: `go test ./internal/config -run '^TestValidateRejectsComposeRunnerProfilesWithPlanEntryGuidance$'`
