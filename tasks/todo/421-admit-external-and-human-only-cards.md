---
id: TASK-421
title: "Admit external and human-only cards without a fake path scope"
type: bug
priority: P1
effort: M
exec-tier: strong
status: todo
created: 2026-09-24
---

## Summary

[ISSUE-004](../issue/004-controller-scope-admission-cannot-represent-external-or-human-only-cards.md)는
그대로다. preflight는 구현 범위가 없는 외부·사람 전용 카드를 `allowed-paths`
부재로 거절하고, `tasks/` 경로를 범위로 주면 그것도 거절한다. 큐는 카드를
고른 뒤 실행도 상태 전이도 못 한 채 선다. 소유는 ce-agent-kit#1.

방향: 실행할 파일 범위와 처분 전용 카드를 구분하는 기계가 읽는 표시를 상류에
둔다. 가짜 제품 파일 범위를 만들어 통과시키지 않는다.

## Completion Criteria

- [ ] 외부 또는 사람 전용 카드가 제품 파일 범위 없이 큐에서 라우팅된다 | verify: human — 상류 선정 테스트가 P0 issue, external todo, decision todo를 구분한다
- [ ] 실제 구현 카드의 경로 검사는 `tasks/`, 절대 경로, 부모 탐색, 글로브를 계속 거부한다 | verify: human — 상류 회귀가 그 네 형태를 거부한다

## Out of scope

- DVA 카드에 합성 `allowed-paths`를 넣어 preflight를 통과시키는 일.

## Sources

- [ISSUE-004](../issue/004-controller-scope-admission-cannot-represent-external-or-human-only-cards.md)
