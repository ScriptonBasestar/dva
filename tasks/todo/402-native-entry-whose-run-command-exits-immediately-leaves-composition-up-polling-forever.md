---
id: TASK-402
title: "Native entry whose run command exits immediately leaves composition up polling forever"
type: bug
priority: P1
effort: M
created: 2026-09-16
status: todo
---

## Summary

2026-09-16 familybook 실기동 회차 2차에서 실측: native 엔트리의 `run` 명령이 기동 직후
0으로 종료해도 `dva up`이 끝나지 않는다. 27분 무응답 후 사람이 프로세스를 종료해야
했다. 원인은 두 갭의 합:

1. **스폰 성공 ≠ 프로세스 생존.** `internal/lifecycle/process.go`의 `startLocalProcess`
   (`sh -c <run>`을 detached로 띄우고 pidfile·로그 파일만 남기고 돌아온다,
   process.go:234)은 즉사를 감지하지 못한다. 실측 사례: familybook-devbox
   `scripts/devbox.sh backend-run`이 `bin/familybook-server` 부재로 exit 3 즉사 —
   표준오차는 `.sb/dva/logs/<name>.log`에만 남는다(회차 재실행으로 파쇄돼 회차 당시엔
   회수 불가였다).
2. **웨이브 경계 readiness 대기에 데드라인이 없다.** composition 웨이브는
   `CompositionOrchestrator` → `PlanChildExecutor.WaitReady` →
   `Orchestrator.waitEntriesReady`(composition_orchestrator.go:527)로
   `WaitUntilReadyWithContext`를 부르는데, 전달되는 ctx에 deadline이 없으면
   `health.go:94`의 루프가 2초 간격으로 영원히 돈다. 엔트리의 `ready_timeout` 설정도
   이 경로에서는 읽히지 않는다. 프로세스 사망(pidfile 재검사)도 폴링 중에 확인되지
   않는다.

반면 단일 플랜 경로의 `startModeProcesses`(orchestrator.go:583)는 `opts.Wait` 시
`ReadyTimeout`을 적용해 `context.WithTimeout`을 쓴다 — 두 경로의 의미론이 다르다.

재현: `bin/familybook-server`를 지운 상태에서 `dva up dev`(familybook-devbox) →
up이 영원히 돌아간다. 로그엔 아무것도 출력되지 않는다.

수정 방향(후보, 구현은 소유자 판단): (a) detached 스폰 뒤 짧은 유예로 조기 종료·비정상
exit를 검출해 스폰 실패로 승격, (b) composition readiness 대기에 엔트리
`ready_timeout`(기본값 포함)을 강제, (c) 폴링 중 pidfile 라이브니스 재검사.

## Completion Criteria

- [ ] native 엔트리 run 명령이 기동 후 N초 내 비정상 종료하면 up이 그 오류로 실패한다 |
      verify: human — 구현과 함께 추가되는 조기 종료 검출 테스트(`go test ./internal/lifecycle/`)가
      그 오류 경로를 고정하는지 확인
- [ ] composition readiness 대기가 엔트리 ready_timeout(무설정 시 기본값)을 준수한다 |
      verify: human — 구현과 함께 추가되는 타임아웃 테스트가 대기 데드라인을 고정하는지 확인
- [ ] familybook-devbox 재현 시나리오에서 up이 유한 시간 안에 종료한다 | verify: human —
      바이너리 부재 상태에서 `dva up dev` 종료 확인
