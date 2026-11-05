---
id: TASK-562
title: "Point compose profiles errors at plan entries"
type: bug
priority: P3
exec-tier: standard
status: done
created: 2026-11-01
quality-review: pass
quality-reviewed-at: 2026-11-04T07:09:40Z
quality-review-evidence: tasks/done/evidence/TASK-562/independent-review.json
quality-review-receipt: tasks/done/evidence/TASK-562/done-review-a0e42eb45f9eb94d07e2639fe3d7bccbd187397ad2a124f1efed49d00d5093bb.json
---

# TASK-562: Point compose profiles errors at plan entries

## Summary

The stack runner compose `profiles` error does not tell the author to use a plan entry. Source: DVA feedback plan of 2026-10-31, measured on 0.4.2 across 52 projects.

## Completion Criteria

- [x] The stack runner compose profiles error names the plan entry fix | verify: `go test ./internal/config -run '^TestValidateRejectsComposeRunnerProfilesWithPlanEntryGuidance$'`

## Review Attempts

- 2026-11-04T07:09:40Z — first card review: reviewer `/root/doctor_flake_diagnosis` (strong), executor `/root/config_missing_and_profiles` (standard). Criterion PASS after the declared guarded test ran with `-count=1 -v`; required named tests executed. No blocking findings; source `9962c14f51e3345fa9d472b19686f96b0882dd85`. Evidence: `tasks/done/evidence/TASK-562/independent-review.json`; review runtime `review-execution-stability-diagnostics` (review). Outcome: pass; next transition: done.
