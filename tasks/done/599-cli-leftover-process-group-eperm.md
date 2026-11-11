---
id: TASK-599
title: "cli 테스트 helper의 EPERM 불안정"
type: bug
priority: P3
effort: S
exec-tier: standard
status: done
quality-review: pass
quality-reviewed-at: 2026-11-11T05:00:00Z
quality-review-evidence: tasks/done/evidence/TASK-599/independent-review.json
quality-review-receipt: tasks/done/evidence/TASK-599/done-review-7873403a0b4dee1848cc46908f8f2c440e5c3071201ce3229c9b3905b02ff85e.json
created: 2026-11-11
---

## Summary

`internal/cli/process_group_kill_unix_test.go`의 `killLeftoverProcessGroup`은 `ESRCH`만 허용한다. TASK-598에서 macOS가 좀비만 남은 프로세스 그룹에 `ESRCH` 대신 `EPERM`을 돌려주는 것이 격리 재현으로 확인됐고(약 3.5%), `internal/lifecycle`의 같은 helper는 고쳤다. 이 helper는 pid 파일이 남아 있을 때만 도는 더 좁은 경로지만 같은 경합이 가능하다(추정, 재현 전). 재현해 같은 방식으로 고친다. TASK-598 리뷰 중 발견했다.

## Completion Criteria

- [x] killLeftoverProcessGroup이 이미 죽은(좀비만 남은) 프로세스 그룹의 EPERM을 오류로 보고하지 않는다 | verify: `go test ./internal/cli/ -run TestKillGroupTolerantlyToleratesOnlyGroupAlreadyGone -count=1 && go test ./internal/cli/ -run TestKillGroupTolerantlyAcceptsZombieOnlyGroups -count=1`

## Scope Notes

- 격리 재현(macOS, 2000회 반복): 리더만 죽이고 reap하지 않은 그룹(좀비만 남은 그룹)에 `kill(-pgid, SIGKILL)`을 한 번 더 보내면 13회(약 0.65%) `EPERM`이었다. 리더를 reap한 뒤 보내는 변형(`sh -c 'sleep & wait'` 그룹)은 2000회에서 0회였다. 즉 가설대로 좀비만 남은 그룹이 원인이며, pid 파일 경로(`composition_flags_exec_test.go`)도 같은 호출을 하므로 같은 경합이 가능하다.
- 수리: `killLeftoverProcessGroup`이 `killGroupTolerantly`(kill 함수를 주입받는다)를 거쳐 `ESRCH`와 `EPERM`을 "그룹이 이미 없음"으로 취급한다. `internal/lifecycle`의 `killProcessGroupInCleanup`과 같은 정책이며 `pid<=1`·자기 그룹 가드는 그대로다. 한계도 같다: 다른 uid의 살아 있는 그룹에 대한 진짜 권한 문제는 이 helper가 못 잡는다(pid가 우리 자식에서 오므로 감수).
- 바인딩: 주입 kill 함수로 오류별 허용·보고를 고정하는 `TestKillGroupTolerantlyToleratesOnlyGroupAlreadyGone`와, 실제 좀비 그룹 1500회를 도는 `TestKillGroupTolerantlyAcceptsZombieOnlyGroups`(0.65% 확률이라 1500회에서 놓칠 확률 약 0.006%; `-short`는 200회라 약 27% 확률로 놓칠 수 있어 약함).
- 돌연변이: EPERM 허용을 지운 복사본에서 두 테스트 모두 실패했다(후자는 `kill process group …: operation not permitted` 7건). 복원 후 통과.

## Review Attempts

### 1. 2026-11-11 — reviewer: claude-code-subagent (opus, 독립), executor: claude-sonnet-5-5

- 대상: head 263fd4e5 및 델타 0d11293f(카드 600 바인딩 한 줄). 판정: pass. 기준 검증 명령이 통과했고 EPERM 허용을 제거한 변이를 테스트가 잡았다. `go test compare, cli, lifecycle -count=3`이 통과했다.
- 비차단: 코드 주석과 카드가 EPERM 비율을 약 0.65%로 적었으나 리뷰어 실측은 부하에 따라 8~40%였다(후속 TASK-601). 다음 전환: done/.
