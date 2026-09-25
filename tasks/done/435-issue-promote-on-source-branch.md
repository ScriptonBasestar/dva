---
id: TASK-435
title: "Let issue-promote finish on the source branch"
type: bug
priority: P1
effort: S
exec-tier: standard
status: done
created: 2026-09-24
quality-review: pass
quality-reviewed-at: 2026-09-25
quality-review-evidence: "Independent review (not implementer) PASS for source-branch apply guard, detached-HEAD refusal, and preserved cleanup refusal. The reviewer noted link assertions could be more explicit; focused behavior tests and full DVA commit CI passed."
---

## Summary

[ISSUE-032](../_archive/issue/032-issue-promote-apply-fails-at-trailing-cleanup-on-source-branch-master.md)의
현행 재현은 trailing cleanup이 아니다. `issue-promote` 단일·일괄 apply가 공용
`cleanupApplySourceGuard`를 재사용해 **승격 전에** source branch를 거부한다.
이 경로에는 `TaskPromote` 뒤 cleanup 호출이 없다. 따라서 오류는 보드 변경 전에
발생한다.

방향: issue-promote 전용 guard는 detached HEAD만 거부하고 source branch는
허용한다. 실제 destructive cleanup의 source-branch 보호는 유지한다.

## Completion Criteria

- [x] 단일 및 일괄 issue-promote apply가 source branch에서 성공하고 detached HEAD는 변경 전에 거부된다 | verify: human — 상류의 single/bulk source branch 성공 테스트와 detached HEAD 무변경 테스트를 확인
- [x] 일반 destructive cleanup은 source branch를 계속 거부한다 | verify: human — 기존 상류 cleanup source/detached apply 거부 테스트와 guard 구현을 확인

## Out of scope

- 이 보드의 열린 이슈를 실제 promote로 태스크화하는 일.

## Sources

- [ISSUE-032](../_archive/issue/032-issue-promote-apply-fails-at-trailing-cleanup-on-source-branch-master.md)

## Current evidence

`ce-agent-kit` master 통합 커밋 `05b0b4ae3c18517df2d7635f3b185da4ee3d9baf`에서
`issue-promote` apply guard가 detached HEAD만 거부하고, destructive cleanup guard는
그대로 source branch를 거부하는 것을 확인했다. Focused tests
`TestIssuePromote(RejectsDetachedHEAD|AllowsConfiguredSourceBranch|AllAllowsConfiguredSourceBranch)`와
`TestTaskCleanup_refusesSourceAndDetachedApply`가 통과했다. `dva ci status`의 commit
profile run `bcff08eca177a5e8913aa4dd5a335d62`는 succeeded였다.
