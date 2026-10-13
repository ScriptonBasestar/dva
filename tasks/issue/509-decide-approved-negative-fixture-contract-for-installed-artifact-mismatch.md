---
id: ISSUE-509
title: "Decide approved negative-fixture contract for installed artifact mismatch"
type: bug
priority: P2
severity: medium
effort: S
needs-human: true
execution-mode: external
human-grade: human
status: todo
ownership: local
agent: human
discovered-at: 2026-10-13
discovered-in: "TASK-103 S2 current installed-DVA verification"
created: 2026-10-13
---

## Summary

S2를 측정할 승인되고 범위가 한정된 음성 픽스처 계약을 사람이 고르는 결정만 다룬다. 구체적인 기술 수단은 정하지 않는다. 사람이 계약을 고른 뒤의 구현은 별도의 DVA 작업이 필요하다.

## 소유권 — 이 저장소다

이 간극의 소유는 DVA다. 카탈로그 저장소 `task-manager-devbox`의 TASK-103은 차단된 채로 남으며, 그 카드의 인수 갱신은 여기 범위가 아니다. 참조는 canonical id `task-manager-devbox`만 쓰고 로컬 경로는 쓰지 않는다.

살아있는 DVA 이슈 506, 507, 508은 이 간극을 소유하지 않는다. 아카이브된 ISSUE-453은 해결된 출처 이력이다.

## 확인된 관찰

- 설치본 `/Users/archmagece/go/bin/dva`는 v0.3.0, 커밋 `1aa8cd6205fc1fa840eb5dc118f7b8e0bd1e0033`이다. 실제 `task-queue-start` 도움말에는 help와 전역 `debug`/`dry-run`/`json`만 있고, 지원되는 pin/path/hash/mismatch 주입 입력은 없다.
- 현재 DVA 소스 `68cdf136e182af227350fb1ec69ea3c19b651279`에서 `internal/cli/task_queue_start.go`는 `taskqueue.Start`를 호출한다. `internal/taskqueue/taskqueue.go`는 항상 `embeddedPins`를 넘긴다. `internal/taskqueue/pin.go`는 `taskchain-pins.json`을 임베드한다. `startWithPins(..., pins)`는 비공개 Go 테스트 봉합점일 뿐이다.
- 카탈로그 저장소 `task-manager-devbox`의 TASK-103은 계속 차단이다. S2 artifact-mismatch 입력이 지원되지 않고, S1–S7 전체 매트릭스는 인수되지 않았다.

## Reproduction

재현은 읽기 전용이다. 설치된 `dva`의 `task-queue-start` 도움말을 읽고, 위 소스 커밋의 `internal/cli/task_queue_start.go`, `internal/taskqueue/taskqueue.go`, `internal/taskqueue/pin.go`를 검사한다.

## Expected vs Actual

지원되는 설치본 mismatch 입력이 없다. S2와 S1–S7 전체 매트릭스는 차단된 채로 남는다.

## 범위

범위는 S2 측정을 위한 승인되고 범위가 한정된 음성 픽스처 계약에 대한 사람의 결정뿐이다. 구체적인 기술 수단은 정하지 않는다. 사람이 계약을 선택한 뒤의 구현은 별도의 DVA 작업이 필요하다.

## 제외

다음은 하지 않는다.

- CLI 플래그 구현
- 숨은 환경 변수
- 공개 테스트 주입
- 소스 pin 변경
- 프로덕션 pin 활성화
- PATH override/shadow
- 설치본 또는 전역 바이너리 변경
- CE start
- TASK-103 인수 갱신

## 기준선 문서 진단

ISSUE-506은 소유권이 미분류이고, ISSUE-507은 기존 report-link 실패 두 건을 기록한다. 이 등록은 그 진단을 악화시키지 않아야 한다.

완료 기준과 검증 명령은 이 이슈에 두지 않는다.
