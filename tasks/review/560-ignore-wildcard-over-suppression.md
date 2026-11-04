---
id: TASK-560
title: "Diagnose over-broad ignores and record ignore reasons"
type: feature
priority: P3
exec-tier: standard
status: review
created: 2026-11-01
---

# TASK-560: Diagnose over-broad ignores and record ignore reasons

## Summary

`suggestion_ignore` wildcards can hide more than intended, and ignore entries have no reason field. Source: DVA feedback plan of 2026-10-31, measured on 0.4.2 across 52 projects.

## Completion Criteria

- [ ] Validate diagnoses a wildcard that over-suppresses, and an ignore entry can carry a reason | verify: `go test ./internal/cli -run 'Test(SuggestionIgnoreObject.*|ValidateSuggestionIgnoreBoundRendersReasonAndFailsOnlyUnderStrict)$'`
