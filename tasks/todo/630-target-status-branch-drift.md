---
id: TASK-630
title: "target status: flag when a worktree's branch or commit changed since up"
type: feature
priority: P3
effort: XS
exec-tier: standard
status: todo
created: 2026-11-22
allowed-paths: [internal/cli/, internal/lifecycle/, USAGE.md, CHANGELOG.md]
---

## Summary

`dva target <id> up` records branch and commit in `<pids>/<name>.identity.json`. In the
TASK-538 pilot, switching the worktree's branch while running left `dva target <id> status`
silent. Status should compare the identity file with the current worktree and flag drift.

## Completion Criteria

- [ ] `dva target <id> status` shows a drift marker with recorded vs current branch/commit when they differ, in text and `--json`. | verify: human — internal/cli 또는 internal/lifecycle에 추가한 TargetDrift 테스트 통과를 독립 리뷰가 확인
- [ ] No marker when branch and commit match. | verify: human — internal/cli 또는 internal/lifecycle에 추가한 TargetDrift 테스트 통과를 독립 리뷰가 확인
