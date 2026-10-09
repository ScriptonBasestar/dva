---
id: TASK-504
title: "Settle where accepted and open decisions live"
type: decision
priority: P3
effort: S
needs-human: true
execution-mode: decision
human-grade: human
status: Proposed
created: 2026-10-10
---

## Summary

결정 기록의 위치를 세 색인이 서로 다르게 설명한다. 미결 결정과 수락된 결정을 어디에
두는지 정해야 한다. 그래야 에이전트가 새 결정 카드를 올바른 곳에 만들 수 있다.

## Context

2026-10-10 문서 정합성 검토에서 확인했다.

- [tasks/README.md](../README.md)의 보드 구조는 `tasks/decision/`을 "미결 결정 카드와
  인덱스, 수락된 정본은 `decisions/`"라고 쓴다. 같은 문서의 현재 상태 절은
  "수락된 결정: decision/"이라고 링크한다.
- [tasks/decision/README.md](README.md)는 "미결 선택이다"라고 쓴다. 그러나 표의 두 행
  (TASK-487, TASK-491)은 모두 accepted다.
- [decisions/README.md](../../decisions/README.md)도 Proposed 카드가 인간 확정을 기다리는
  곳이라고 쓴다. 수락된 DECISION-001~004를 담고 있다.
- 두 색인의 계수기가 다르다. `decisions/`는 "로그 000, 다음 001"이고
  `tasks/decision/`은 "로그 491, 다음 492"다. ce의 decision index root가 경로마다 따로
  잡히는 결과로 보인다. 어느 쪽이 정본 계수기인지는 문서에 없다.
- 이 검토가 만든 TASK-501~504는 사용자 요청과 현행 tasks/README 규약대로 `tasks/decision/`에 두었다.

## Decision

- Status: Proposed

선택지:

1. **미결은 `tasks/decision/`, 수락 후 `decisions/`로 승격**: 수락된 TASK-487·491을
   `decisions/`로 옮긴다(또는 아카이브한다). tasks/README의 "수락된 결정" 링크를 고친다.
2. **모든 결정은 `decisions/`**: `tasks/decision/`을 비우고 새 카드도 `decisions/`에 만든다.
   계수기는 하나로 합친다.
3. **현행 유지, 문구만 정정**: `tasks/decision/`이 미결과 수락을 모두 담는다고 README를
   고친다. 계수기 두 개의 의미를 각 색인에 적는다.

## Rationale

검토자 권고는 1이다. 현행 tasks/README의 보드 구조 문구와 가장 가깝다. 수락된
결정을 이미 `decisions/`에 모아 온 관례와도 맞다. 보드 규약을 바꾸는 일이라 사람이 정한다.

## Consequences

- 1·2는 카드 이동이다. `make doc-check`의 카드 경로 해석과 `ce task validate`를 다시 돌려야 한다.
- 2는 ce의 `tasks/decision` 지원이 legacy라는 관찰과 맞는다. 대신 decision 카드에
  `TASK-` ID를 계속 쓸지도 정해야 한다.

## Resolution Criteria

- [ ] 사람이 세 위치 규약 중 하나를 고르고 계수기 정본을 적는다 | verify: human — choice recorded on this card before any decision card is moved
