---
id: TASK-542
title: "Reject unknown interaction runner names"
type: bug
priority: P3
exec-tier: standard
status: done
created: 2026-10-31
quality-review: pass
quality-reviewed-at: 2026-11-01
quality-review-evidence: tasks/done/evidence/TASK-542/independent-review.json
---

# TASK-542: Reject unknown interaction runner names

## Summary

An interaction `runner:` typo silently ran under compose. Validation and the run path now
refuse it; `compose` stays accepted as an explicit name.

## Completion Criteria

- [x] Unknown names fail in the runner and in validation | verify: `go test -count=1 -run 'TestNewRunnerRejectsUnknownName|TestValidateInteractionRunners' -v ./internal/runner ./internal/config 2>&1 | /usr/bin/grep -c -- '--- PASS' | /usr/bin/grep -qx 2`
