---
id: TASK-491
title: "Clarify queue acceptance scope and current evidence"
type: decision
priority: P2
effort: S
needs-human: true
execution-mode: decision
human-grade: human
status: Proposed
created: 2026-10-05
---

## Summary

ISSUE-004·006의 과거 카드·자동 루프 기준을 현재 소비자 계약으로 현행화할지 결정한다.
읽기 전용 조회를 ISSUE-453의 공개 pin 승인에 묶는 대기를 줄이는 제안이다.
이 결정은 산출물 공개나 pin 활성화 승인이 아니다.

## Context

- [ISSUE-004](../issue/004-controller-scope-admission-cannot-represent-external-or-human-only-cards.md)의
  기준 2는 과거 ISSUE-001의 “routes”를 요구한다. 분류인지 종료 전이인지 불명확하다.
- [ISSUE-006](../issue/006-preflight-reports-needs-human-cards-as-runnable.md)의
  기준 2는 더 이상 없는 “현재 네 카드”, 기준 3은 구현되지 않은 run-all loop를 가리킨다.
- `internal/taskqueue/taskqueue.go:57`의 Verdict는 읽기 전용이다. `:84`의 Start만
  승인 pin을 요구한다. 사람 terminal/rollback은 이 소비자의 구현 기능이 아니다.
- 현재 실제 조회는 [ISSUE-490](../issue/490-selected-taskchain-rejects-native-decision-directory.md)의
  native decision 디렉터리 호환성 실패로 막힌다. 공개 승인과 다른 차단 원인이다.
- `tasks/decision/487-changelog-half-ceiling-trigger.md`의 실제 사용자 승인과 Accepted
  전이는 현행 결정 카드 증거로 사용할 후보다. 합성 카드 전이를 실제 사용자 승인으로 쓰지 않는다.

## Decision

- Status: Proposed

1. **현재 읽기 전용 계약으로 현행화 (권장)**: 004는 외부·결정 분류와 엄격한
   구현 범위, 006은 agent/human 집합과 `human_required`/`empty`에 의한 종료
   판정으로 한정한다. 실제 현재 보드와 회귀 fixture를 구분해 기계 검증한다.
   terminal/rollback 제품 기능이 필요하면 별도 카드를 만든다. 공개 pin과 실제
   CE start 양성·음성 검증은 ISSUE-453에 남긴다. 기존 기준을 말없이 체크하지 않는다.
2. 기존 역사적 기준 유지: 과거 카드 복원 및 실제 run-all/terminal 제품 계약을
   먼저 정한다. 범위와 비용이 커지며 현재 큐 분류 완료와 별개의 작업이다.

ISSUE-453의 읽기 전용 provenance 안내는 현재 문서 계약을 유지하는 것을 권장한다.
읽기 전용 명령은 PATH 바이너리를 검증하지 않는다고 명시하고, 관찰 SHA/소스는
증거에 남긴다. 모든 읽기 전용 조회를 승인 pin으로 제한하거나 producer identity를
새 출력 필드로 제공하려면 별도의 제품 계약 결정이다.

## Rationale

원래 분류 결함을 닫기 위해 구현되지 않은 사람 terminal 기능까지 추가하는 것은
범위 확장이다. 읽기 전용 분류·verdict 검증에는 CE 변경 권한이 필요 없다.
실제 시작만 공개 artifact 승인을 기다리면 불필요한 직렬 의존을 제거할 수 있다.

## Consequences

- 선택 전 ISSUE-004·006 종료 기준과 사람 플래그는 유지한다.
- 옵션 1 수락 뒤 자동 구현자는 004·006의 기준을 기계 verify 명령으로 현행화하고,
  ISSUE-490 해결 후 실제 보드 출력 및 종료 판정 증거를 수집한다.
- producer canonical 경로 확인과 공개 산출물 승인 입력은 여전히 필요하다.
- 새 daemon/run-all 엔진이나 두 번째 보드 grader는 만들지 않는다.

## Resolution Criteria

- [ ] 옵션 1 또는 2와 읽기 전용 provenance 범위를 확정한다 | verify: human — record explicit acceptance scope; this does not authorize publishing or pin activation
