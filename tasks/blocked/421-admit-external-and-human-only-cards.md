---
id: TASK-421
title: "Admit external and human-only cards without a fake path scope"
type: bug
priority: P1
effort: M
exec-tier: strong
status: blocked
created: 2026-09-24
---

## Summary

[ISSUE-004](../issue/004-controller-scope-admission-cannot-represent-external-or-human-only-cards.md)는
삭제된 Python controller를 현행 상류 동작으로 취급하고 있다. 워크북은
`task_management/engine`과 `/execution`이 TASK-045에서 삭제됐으며
`entry_selection.py`도 `cc65f521`에서 제거됐다고 기록한다. 현재 CE 카탈로그에는
`ce-agent-kit`만 있지만 이 controller의 후속 구현체는 선언되어 있지 않다.

따라서 기존 `ce-agent-kit#1` 소유 표기는 근거가 없다. 현행 `ce task preflight`는
criteria·verify·의존성만 평가하며, 새 selector/controller 경로를 만들려면 별도
제품 결정과 CLI 소유권 지정이 필요하다. 이 task는 그 결정 전까지 막아 둔다.

방향: 새 controller의 정본과 실행 진입점이 정해진 뒤 범위를 다시 잡는다. 가짜
제품 파일 범위를 만들어 통과시키지 않는다.

## Completion Criteria

- [ ] 외부 또는 사람 전용 카드가 제품 파일 범위 없이 큐에서 라우팅된다 | verify: human — 상류 선정 테스트가 P0 issue, external todo, decision todo를 구분한다
- [ ] 실제 구현 카드의 경로 검사는 `tasks/`, 절대 경로, 부모 탐색, 글로브를 계속 거부한다 | verify: human — 상류 회귀가 그 네 형태를 거부한다

## Out of scope

- DVA 카드에 합성 `allowed-paths`를 넣어 preflight를 통과시키는 일.

## Sources

- [ISSUE-004](../issue/004-controller-scope-admission-cannot-represent-external-or-human-only-cards.md)
- [ISSUE-040](../issue/040-locate-canonical-owner-of-controller-scope-admission.md)

## Blocker evidence (2026-09-25)

`ce-workbook/tasks/README.md` records that TASK-045 deleted `task_management/engine`
and `/execution`; it names `task_management/engine/operate/entry_selection.py` as
deleted by `cc65f5213595e804fcd79cc2a494b725ebbaccce`. The current shared CE catalog
names `ce-agent-kit` as the active CE source but names no successor for that Python
controller. `ce-agent-kit`'s current `ce task preflight` has no selector or
`_allowed_paths()` path. Implementing the desired queue disposition there would be a
new feature, not a continuation of the deleted owner.
