---
id: ISSUE-526
title: "TestLockRejectsAnotherProcess fails intermittently on hosted runners"
type: bug
priority: P3
effort: XS
exec-tier: standard
status: done
severity: low
ownership: local
discovered-in: "ISSUE-515 hosted bisection and fix verification"
discovered-at: 2026-10-21
created: 2026-10-21
resolution: fixed
resolved-at: 2026-10-21T03:21:44Z
resolution-summary: "Resolved as fixed."
---

## Summary

`internal/cirun` TestLockRejectsAnotherProcess는 자식 test binary가 lock을 잡을 때까지 부모가
1초만 기다린다(`internal/cirun/cirun_test.go`의 `deadline := time.Now().Add(time.Second)`).
hosted ubuntu runner에서 CI와 같은 `GOMAXPROCS=1`로 돌리면 자식이 시작해 lock을 잡기 전에 기한이
지나 `child never held lock: <nil>`로 실패할 때가 있다. 그러면 정상 코드에서도 `make test`가
빨간색이 된다.

## 소유권 — 이 저장소다

테스트와 cirun lock은 DVA `internal/cirun`이 소유한다.

## Reproduction

1. GitHub hosted `ubuntu-latest`에서 build→dogfood 뒤
   `GOMAXPROCS=1 sh scripts/without-dva.sh make test --silent`를 실행한다.
2. 관측된 실패:
   - ISSUE-515 1회차 run 39444449630의 `rest` job
   - ISSUE-515 수정 검증 run 39513943508의 job 118941445932(표본 g, 8개 중 1개)
   원본 로그는 dva primary checkout의 무시된 `tmp/hosted-followup-20261020/verify/`에 있다.
3. 로컬에서는 재현을 시도하지 않았다.

## Expected vs Actual

- Expected: 자식이 lock을 잡은 뒤 부모가 `ErrBusy`를 받고 통과한다. 결과가 runner 부하와 무관하다.
- Actual: 자식 시작이 1초를 넘기면 부모가 기한을 넘겨 실패한다.

## 범위

자식이 lock을 잡았다는 신호를 시간 기한이 아닌 동기화로 기다리도록 테스트를 고친다. 예를 들면
자식이 lock을 잡은 뒤 파일이나 pipe로 알리는 방식이다. 테스트 생략과 단순한 기한 연장은
범위 밖이다.

## 2026-10-21 해결

- 기한 대신 pipe로 동기화한다. 자식이 lock을 잡은 뒤 stdout에 `held`를 쓰고, 부모가 stdin을 닫을 때까지 lock을 유지한다. 부모는 그 줄을 읽은 뒤 `acquire`를 한 번만 호출해 `ErrBusy`를 확인하고, stdin을 닫아 자식 종료를 `Wait`로 확인한다.
- 자식이 lock을 얻지 못하면 원인을 stderr로 남기고, 부모의 실패 메시지가 그 stderr를 포함한다.
- 검증: `GOMAXPROCS=1 go test -run TestLock -count=30` 통과. 자식 acquire 앞에 2초 지연을 넣은 임시 변형에서도 통과했다(이전 테스트는 1초 기한에서 실패). make gate 전부, `GOOS=windows go vet`.
