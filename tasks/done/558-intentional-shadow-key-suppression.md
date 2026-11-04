---
id: TASK-558
title: "Declare intended shadow keys"
type: feature
priority: P3
exec-tier: standard
status: done
created: 2026-11-01
quality-review: pass
quality-reviewed-at: 2026-11-04T08:01:37Z
quality-review-evidence: tasks/done/evidence/TASK-558/independent-review.json
quality-review-receipt: tasks/done/evidence/TASK-558/done-review-92ab4c1e3c471b7381c24d17c72e6575d93eecbfed3a55b306770c0edf64f23e.json
---

# TASK-558: Declare intended shadow keys

## Summary

An intended shadow key has no suppression, so its warning stays under --strict (observed in funbricks-notifire). Source: DVA feedback plan of 2026-10-31, measured on 0.4.2 across 52 projects.

## Completion Criteria

- [x] A declared intended shadow key no longer fails --strict, and an undeclared one still does | verify: `go test ./internal/cli -run 'Test(ValidateStrictIntentionalShadow|ValidateIntentionalShadowJSONRemainsVisible)$'`

## Review Attempts

- 2026-11-04T08:01:37Z — first card review: reviewer `/root/doctor_flake_diagnosis` (strong), executor `/root` (strong), integrator `/root` (strong). Criterion PASS after the declared guarded test ran with `-count=1 -v`; required named tests executed. No blocking findings; source `3f7930b953accd2943120cdba177ab8436c116d5`. Evidence: `tasks/done/evidence/TASK-558/independent-review.json`; review runtime `review-explicit-suppression` (review). Outcome: pass; next transition: done.
