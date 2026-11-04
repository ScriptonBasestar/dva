---
id: TASK-547
title: "Keep the no-plans hint from failing strict validation"
type: bug
priority: P2
status: done
created: 2026-10-31
quality-review: pass
quality-reviewed-at: 2026-11-04
quality-review-evidence: tasks/done/evidence/TASK-547/independent-review.json
---

## Summary

<!-- One paragraph: what changes and why. -->

## Completion Criteria

- [x] informational hints do not fail strict | verify: `go test -count=1 -run TestStrictIgnoresInformationalHints ./internal/cli`
