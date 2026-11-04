---
id: TASK-559
title: "Allow turning off ci fail_fast"
type: feature
priority: P3
exec-tier: standard
status: review
created: 2026-11-01
---

# TASK-559: Allow turning off ci fail_fast

## Summary

`dva ci` offers no way to turn fail_fast off (noted in the funbricks-notifire config comments). Source: DVA feedback plan of 2026-10-31, measured on 0.4.2 across 52 projects.

## Completion Criteria

- [ ] A ci profile can continue after a failed step and still exits non-zero | verify: `go test ./internal/cirun -run 'Test(RunFailFastFalseContinuesIndependentAndSkipsFailureDescendants|RunDefaultFailFastCancelsIndependentSibling|FailFastFalseStillHonorsExternalCancellationAndDeadline|FailFastFalseCheckpointFailureStillAborts|FailFastFalseExternalCancellationStopsRunningStep|FailFastFalseProfileDeadlineStopsRunningStep)$'`
