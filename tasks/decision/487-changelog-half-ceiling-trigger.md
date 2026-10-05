---
id: TASK-487
title: "CHANGELOG half-ceiling trigger is already true"
type: decision
priority: P2
effort: S
needs-human: true
execution-mode: decision
human-grade: human
status: Accepted
created: 2026-10-05
---

## Summary

DECISION-001의 수락된 ceiling `87296 bytes, 1000 lines`는 유지한다.
2026-10-05에 사용자가 옵션 2를 골랐다. 아카이브 문장은 실제 ceiling
트리거로 바뀐다. 지금 측정은 두 상한 아래라 아카이브하지 않는다. 이 카드는
DECISION-005가 아니다.

## Context

CHANGELOG.md 최상단 마커의 ceiling은 `87296 bytes, 1000 lines`다. 그 절반은
43648 bytes, 500 lines다. 2026-10-05 `wc` 측정은 67110 bytes, 744 lines다.
둘 다 절반을 넘고 ceiling은 넘지 않았다. 그 측정이 옵션 2의 근거다.

## Decision

- Status: Accepted

2026-10-05에 사용자가 옵션 2를 골랐다. 트리거는 실제 ceiling이다.
릴리스에서 바이트가 87296 이상이거나 물리 줄이 1000 이상이면 v0.2.x 이하를
아카이브한다. 이 비교는 문서화된 릴리스 트리거다. CE 파서가 그 비교를
기계적으로 집행한다고 말하지 않는다. 지금 측정은 두 상한 아래라
아카이브하지 않는다. 마커 문장 적용은
[TASK-489](../todo/489-apply-approved-changelog-ceiling.md)다.

기록으로 남긴 선택지:

1. 절반 트리거를 유지하고 지금 v0.2.x 이하를 아카이브한다. 고르지 않았다.
2. 트리거를 실제 ceiling으로 바꾼다. 사용자가 2026-10-05에 골랐다.
3. 릴리스 경계에서 아카이브한다. 고르지 않았다.

## Rationale

옵션 A로 수락된 DECISION-001은 면제 마커, ceiling, 절반 아카이브 단계를
함께 적었다. 당시 파일은 절반을 이미 넘었고 ceiling은 아래였다. 사용자는
절반 문장만 실제 ceiling 트리거로 바꾸기로 했다. ceiling 숫자는 유지한다.

## Consequences

- ceiling `87296 bytes, 1000 lines`는 그대로다.
- 아카이브는 지금 실행하지 않는다. 측정이 두 상한 아래다.
- 마커 문장은 TASK-489가 적용한다. 이 카드의 `allowed-paths`는 없다.

## Resolution Criteria

- [x] 사람이 2026-10-05에 옵션 2(실제 ceiling)를 골랐고 이 카드에 적었다. 아카이브는 하지 않는다 | verify: human — choice recorded on this card before any changelog archive or marker edit
