---
id: TASK-630
title: "target status: flag when a worktree's branch or commit changed since up"
type: feature
priority: P3
effort: XS
exec-tier: standard
status: done
delivered-by: 03d693ae
quality-review: pass
quality-review-evidence: "ce review-evidence b6c03defdba2a21ecd0ffb86b304447109948c3fda24cdcf514f36670c366f65"
created: 2026-11-22
allowed-paths: [internal/cli/, internal/lifecycle/, USAGE.md, CHANGELOG.md]
---

## Summary

`dva target <id> up` records branch and commit in `<pids>/<name>.identity.json`. In the
TASK-538 pilot, switching the worktree's branch while running left `dva target <id> status`
silent. Status should compare the identity file with the current worktree and flag drift.

## Completion Criteria

- [x] `dva target <id> status` shows a drift marker with recorded vs current branch/commit when they differ, in text and `--json`. | verify: `go test ./internal/lifecycle -run TargetDrift -count=1`
- [x] No marker when branch and commit match. | verify: `go test ./internal/lifecycle -run TargetDriftNone -count=1`

## Decision (2026-11-22, user)

표시만 하고 `down`·`stop`은 막지 않는다(작업 계획 1번, 추천안 승인).

## Outcome

`ProcessPlugin.Status`가 실행 중인 엔트리의 identity(branch·commit)를 현재 checkout과 비교해
`ServiceStatus.Drift`를 채운다. `dva target <id> status`는 그 디렉터리에서 `dva status`를 실행하므로
같은 표시를 받는다. 텍스트는 서비스 표 아래 `!` 줄, JSON은 `drift`. 기록 identity가 없거나 git 밖이면 표시하지
않는다. 테스트는 `drift_unix_test.go`(변경 없음·브랜치+커밋·커밋만·텍스트/JSON). Status 연결을 끄면
변경 케이스 2개가 실패하는 것을 확인했다.

## Review Attempts

### 1. 2026-11-22 — reviewer: 독립 opus 서브에이전트(읽기 전용), executor: 구현 실행(03d693ae)

- 기준 전부 pass. 근거: `ce review-evidence b6c03def…`.
- dva target status와 dva status(text·--json)가 같은 ProcessPlugin.Status를 거친다. 비차단: end-to-end target status 테스트 없음.
- 다음 전환: done.
