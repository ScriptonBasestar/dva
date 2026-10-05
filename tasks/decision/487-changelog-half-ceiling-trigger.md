---
id: TASK-487
title: "CHANGELOG half-ceiling trigger is already true"
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

DECISION-001의 수락된 마커와 ceiling은 유지한다. ceiling 절반 트리거가
2026-10-05 측정에서 이미 참이다. 이 카드는 그 미결 선택만 연다. 선택을
적용하지 않는다. 수락된 정본은 `decisions/DECISION-001`에 남는다. 이 카드는
DECISION-005가 아니다.

## Context

CHANGELOG.md 최상단 마커의 ceiling은 `87296 bytes, 1000 lines`다. 그 절반은
43648 bytes, 500 lines다. 2026-10-05 `wc` 측정은 67110 bytes, 744 lines다.
둘 다 절반을 넘고, 마커 주석의 44KB도 넘는다. 마커 문장과 승인된 ceiling은
바꾸지 않는다.

## Decision

- Status: Proposed

해결은 대기 중이다. 수락된 선택이 아니다. 선택지는 다음 세 가지다.

1. 절반 트리거를 유지하고 지금 v0.2.x 이하를 아카이브한다.
2. 트리거를 실제 ceiling으로 바꾼다. 권고다. 승인된 상한의 이유는 약 30%
   성장 여유였고, 절반 트리거는 그 여유보다 앞에서 이미 참이다.
3. 릴리스 경계에서 아카이브한다.

이 카드는 1, 2, 3 중 어느 것도 적용하지 않는다. 사람이 고를 때까지 Proposed다.

## Rationale

옵션 A로 수락된 DECISION-001은 면제 마커, ceiling, 절반 아카이브 단계를
함께 적었다. 현재 파일은 절반을 이미 넘었고 ceiling은 아직 아래다. 마커를
이 카드에서 고치면 수락된 ceiling과 히스토리 문장이 바뀐다. 트리거를 어디로
둘지만 사람에게 남긴다.

## Consequences

- 마커와 ceiling `87296 bytes, 1000 lines`는 그대로다.
- 아카이브는 이 선택이 수락되기 전에 실행하지 않는다.
- 수락 뒤의 실행은 별도 구현 카드다. 이 카드의 `allowed-paths`는 없다.

## Resolution Criteria

- [ ] 사람이 절반 유지·즉시 아카이브, 실제 ceiling, 릴리스 경계 중 하나를 고르고 적용 범위를 적는다 | verify: human — choice recorded on this card before any changelog archive or marker edit
