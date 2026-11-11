---
id: TASK-598
title: "lifecycle 터널 정리 테스트 불안정"
type: bug
priority: P3
effort: S
exec-tier: standard
status: todo
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
