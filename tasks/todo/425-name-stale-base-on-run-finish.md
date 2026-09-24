---
id: TASK-425
title: "Name a stale base when run-finish refuses to integrate"
type: bug
priority: P2
effort: S
exec-tier: standard
status: todo
created: 2026-09-24
---

## Summary

[ISSUE-008](../issue/008-run-finish-does-not-name-a-stale-base-as-the-reason-it-blocked.md)은
그대로다. `run-finish`는 태스크 브랜치의 기준이 소스 팁이 아니면 통합을
거부하면서, 메시지는 recovery incomplete와 `resolve gz-git diagnostics and
retry`만 말한다. 거부에 쓴 비교 결과는 메시지에 없다. 소유는 ce-agent-kit#2.

방향: 거부 이유가 stale base임을 말하고, 다음 행동으로 워크트리 안 rebase를
적는다. gz-git 진단으로 보내지 않는다.

## Completion Criteria

- [ ] stale base 거부가 기준 커밋이 소스 팁이 아님을 말하고 rebase를 다음 행동으로 적는다 | verify: human — 그 문구가 있는 재현 로그가 이 카드에 링크된다

## Out of scope

- rebase를 `run-finish`가 대신 실행하는 일.

## Sources

- [ISSUE-008](../issue/008-run-finish-does-not-name-a-stale-base-as-the-reason-it-blocked.md)
