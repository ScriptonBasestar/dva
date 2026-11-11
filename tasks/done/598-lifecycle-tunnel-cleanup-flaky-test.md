---
id: TASK-598
title: "lifecycle 터널 정리 테스트 불안정"
type: bug
priority: P3
effort: S
exec-tier: standard
status: done
quality-review: pass
quality-reviewed-at: 2026-11-11T02:00:00Z
quality-review-evidence: tasks/done/evidence/TASK-598/independent-review.json
quality-review-receipt: tasks/done/evidence/TASK-598/done-review-d0e1c6a72e861f356c27094044fd70dd9e0d8ed4dc135b0328384dd930caa7d0.json
created: 2026-11-09
---

## Summary

`internal/lifecycle`의 `TestStatusTunnelCleanupEscalatesOnlyItsOwnedProcessGroup`가 `-count=50`에서 `kill process group …: operation not permitted`(`status_owned_tunnel_unix_test.go:25`)로 간헐 실패한다. 원인을 찾아 테스트 또는 정리 로직을 고친다. TASK-592 리뷰 중 재현했다.

## Completion Criteria

- [x] TestStatusTunnelCleanupEscalatesOnlyItsOwnedProcessGroup이 반복 실행(-count=50)에서 실패하지 않는다 | verify: `go test ./internal/lifecycle/ -run TestStatusTunnelCleanupEscalatesOnlyItsOwnedProcessGroup -count=50`

## Scope Notes

- 원인(추정, 격리 재현은 못 함): 테스트 정리 helper `killProcessGroupInCleanup`가 SIGKILL을 보내는 시점에는 정리 대상 코드(`CloseStatus`)가 이미 그룹을 죽이고 리더를 reap한 뒤라 그룹에 좀비만 남는데, macOS는 이때 ESRCH 대신 EPERM을 돌려준다. 실패는 약 1%(count=100 세 번 실패, 실패 시각이 250ms 유예 직후와 일치)였다.
- 운영 코드는 이 오류를 이미 버리고(`tunnel.go`) 있어 영향이 없다. helper가 EPERM도 "그룹이 이미 없다"로 취급하도록 고쳤다. 수리 후 -count=500이 통과했다(통과율 1%에서 우연히 500회 통과할 확률은 약 0.7%).
- 한계: EPERM을 삼키므로, 우리 자식이 아닌 그룹에 대한 진짜 권한 문제는 이 helper가 못 잡는다. pid가 우리 자식에서 오고 `pid<=1` 가드가 남아 있어 감수한다.

## Review Attempts

### 1. 2026-11-11 — reviewer: claude-code-subagent (opus, 독립), executor: claude-sonnet-5-5

- 대상: head 724e418d. 판정: pass. 기준 PASS(-count=100 통과). 리뷰어가 원인을 격리 재현했다: 그룹을 죽이고 리더를 reap한 뒤 다음 `kill(-pgid)`가 200회 중 7회 EPERM을 돌려줬다(약 3.5%). 위 Scope Notes의 "격리 재현은 못 함"은 구현 시점의 서술이며, 이 리뷰로 재현이 확인됐다.
- EPERM을 삼켜도 pid<=1·자기 그룹 가드는 그대로이며, 운영 코드(`tunnel.go`)는 이 오류를 이미 버린다. 남는 맹점은 다른 uid로 바뀐 살아 있는 그룹 구성원뿐이며 호출부에는 없다.
- 비차단(후속): `internal/cli/process_group_kill_unix_test.go`의 `killLeftoverProcessGroup`도 ESRCH만 허용해 같은 불안정성이 있을 수 있다(추정). 다음 전환: done/.
