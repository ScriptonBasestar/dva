---
id: TASK-560
title: "Diagnose over-broad ignores and record ignore reasons"
type: feature
priority: P3
exec-tier: standard
status: done
created: 2026-11-01
quality-review: pass
quality-reviewed-at: 2026-11-04T08:01:37Z
quality-review-evidence: tasks/done/evidence/TASK-560/independent-review.json
quality-review-receipt: tasks/done/evidence/TASK-560/done-review-8bec2d032a9e4ce07e99cff06a6f8d442bd501311a13809385badd7dc983b01b.json
---

# TASK-560: Diagnose over-broad ignores and record ignore reasons

## Summary

`suggestion_ignore` wildcards can hide more than intended, and ignore entries have no reason field. Source: DVA feedback plan of 2026-10-31, measured on 0.4.2 across 52 projects.

## Completion Criteria

- [x] Validate diagnoses a wildcard that over-suppresses, and an ignore entry can carry a reason | verify: `go test ./internal/cli -run 'Test(SuggestionIgnoreObject.*|ValidateSuggestionIgnoreBoundRendersReasonAndFailsOnlyUnderStrict)$'`

## Review Attempts

- 2026-11-04T08:01:37Z — first card review: reviewer `/root/doctor_flake_diagnosis` (strong), executor `/root/parent_compose_drift` (standard), integrator `/root` (strong). Criterion PASS after the declared guarded test ran with `-count=1 -v`; required named tests executed. No blocking findings; source `3f7930b953accd2943120cdba177ab8436c116d5`. Evidence: `tasks/done/evidence/TASK-560/independent-review.json`; review runtime `review-explicit-suppression` (review). Outcome: pass; next transition: done.
