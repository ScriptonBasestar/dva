---
id: TASK-561
title: "Declare Makefile target and interaction aliases"
type: feature
priority: P3
exec-tier: standard
status: done
created: 2026-11-01
quality-review: pass
quality-reviewed-at: 2026-11-04T08:01:37Z
quality-review-evidence: tasks/done/evidence/TASK-561/independent-review.json
quality-review-receipt: tasks/done/evidence/TASK-561/done-review-edf414031e0325c35236fdbe11f3e46aa2e8cd07eb147577544ab18fe16dbb52.json
---

# TASK-561: Declare Makefile target and interaction aliases

## Summary

A Makefile target and an interaction doing the same work cannot be declared as aliases of each other. Source: DVA feedback plan of 2026-10-31, measured on 0.4.2 across 52 projects.

## Completion Criteria

- [x] An alias declaration links a Makefile target to an interaction | verify: `go test ./internal/cli -run '^TestSuggestionAliasCoverage'`

## Review Attempts

- 2026-11-04T08:01:37Z — first card review: reviewer `/root/doctor_flake_diagnosis` (strong), executor `/root/parent_compose_drift` (standard), integrator `/root` (strong). Criterion PASS after the declared guarded test ran with `-count=1 -v`; required named tests executed. No blocking findings; source `3f7930b953accd2943120cdba177ab8436c116d5`. Evidence: `tasks/done/evidence/TASK-561/independent-review.json`; review runtime `review-explicit-suppression` (review). Outcome: pass; next transition: done.
