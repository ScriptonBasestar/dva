---
id: TASK-548
title: "Mark am flows optional in the dva skill"
type: bug
priority: P3
status: done
created: 2026-10-31
quality-review: pass
quality-reviewed-at: 2026-11-04
quality-review-evidence: tasks/done/evidence/TASK-548/independent-review.json
---

## Summary

The `dva` skill states that `am` flows are optional, and `dva-config` names `dva config init` as the same command as `dva init`.

## Completion Criteria

- [x] dva skill states am is optional | verify: `/usr/bin/grep -q "these flows require" skills/dva/SKILL.md`
