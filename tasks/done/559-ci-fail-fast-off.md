---
id: TASK-559
title: "Allow turning off ci fail_fast"
type: feature
priority: P3
exec-tier: standard
status: done
created: 2026-11-01
quality-review: pass
quality-reviewed-at: 2026-11-04T08:01:37Z
quality-review-evidence: tasks/done/evidence/TASK-559/independent-review.json
quality-review-receipt: tasks/done/evidence/TASK-559/done-review-e5b40a561a4e52a926f87527dae7d48d8e038aafe89b6e919042ce9fa9cf5263.json
---

# TASK-559: Allow turning off ci fail_fast

## Summary

`dva ci` offers no way to turn fail_fast off (noted in the funbricks-notifire config comments). Source: DVA feedback plan of 2026-10-31, measured on 0.4.2 across 52 projects.

## Completion Criteria

- [x] A ci profile can continue after a failed step and still exits non-zero | verify: `go test ./internal/cirun -run 'Test(RunFailFastFalseContinuesIndependentAndSkipsFailureDescendants|RunDefaultFailFastCancelsIndependentSibling|FailFastFalseStillHonorsExternalCancellationAndDeadline|FailFastFalseCheckpointFailureStillAborts|FailFastFalseExternalCancellationStopsRunningStep|FailFastFalseProfileDeadlineStopsRunningStep)$'`

## Review Attempts

- 2026-11-04T08:01:37Z — first card review: reviewer `/root/doctor_flake_diagnosis` (strong), executor `/root/config_missing_and_profiles` (standard), integrator `/root` (strong). Criterion PASS after the declared guarded test ran with `-count=1 -v`; required named tests executed. No blocking findings; source `3f7930b953accd2943120cdba177ab8436c116d5`. Evidence: `tasks/done/evidence/TASK-559/independent-review.json`; review runtime `review-explicit-suppression` (review). Outcome: pass; next transition: done.
