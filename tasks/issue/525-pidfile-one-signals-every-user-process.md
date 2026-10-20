---
id: ISSUE-525
title: "A pidfile holding 1 makes dva stop signal every process the user owns"
type: bug
priority: P2
effort: S
exec-tier: standard
status: todo
severity: medium
ownership: local
discovered-in: "ISSUE-515 test cleanup fix review (2026-10-21)"
discovered-at: 2026-10-21
created: 2026-10-21
---

## Summary

`signalableProcessGroupPID`는 `pid > 0`만 확인한다(`internal/lifecycle/process_group_contract.go`).
그래서 pidfile 값이 1이면 그 값이 그대로 `terminateProcessGroup(pid)`에 전달되고, 이 함수는
`syscall.Kill(-1, SIGTERM)`을 보낸다. `kill(-1)`은 호출한 사용자가 신호를 보낼 수 있는 모든
프로세스에 신호를 보낸다. 손상되거나 손으로 고친 pidfile 하나 때문에 `dva down`/`stop`이
사용자의 다른 프로세스까지 종료시킬 수 있다. `TestSignalableProcessGroupPID`는 pid 1이 신호
대상이라는 동작을 오히려 고정하고 있다.

## 소유권 — 이 저장소다

pidfile을 해석하고 프로세스 그룹에 신호를 보내는 코드는 DVA `internal/lifecycle`이 소유한다.

## Reproduction

1. native mode health check나 tilt background 항목의 pidfile
   (`.dva/pids/<name>.pid`)에 `1`을 쓴다.
2. 그 항목을 멈추는 `dva` 명령을 실행한다. 이 명령은 `orchestrator_mode.go`의 health check
   정지 경로, 또는 `tilt.go`의 `stopBackgroundProcess`를 탄다.
3. 두 경로 모두 `signalableProcessGroupPID(1)`이 true라서 `terminateProcessGroup(1)`,
   곧 `kill(-1, SIGTERM)`까지 간다. 이 판단은 코드 읽기로만 확인했고 실행으로 재현하지는 않았다.
   재현할 때는 버려도 되는 사용자나 컨테이너에서 한다.

## Expected vs Actual

- Expected: pid ≤ 1이나 자기 프로세스 그룹은 신호 대상에서 거부하고, 오류로 드러낸다.
- Actual: pid 1을 신호 대상으로 받아들여 `kill(-1)`을 보낸다.

## 범위

`signalableProcessGroupPID`의 경계, 그 테스트, 두 호출 경로의 오류 처리를 고친다. ISSUE-515의
hosted 정지는 다른 원인으로 확인됐다(그 카드의 2026-10-20 그룹 신호 진단). 이 카드는 그
조사에서 기각된 가설의 실제 결함 부분만 다룬다. 테스트 cleanup의 같은 경계는
`internal/lifecycle/process_group_kill_unix_test.go`에 이미 들어갔다.
