---
id: TASK-433
title: "Stamp the tool version on gate verdicts"
type: bug
priority: P3
effort: S
exec-tier: standard
status: todo
created: 2026-09-24
---

## Summary

[ISSUE-030](../issue/030-gate-verdicts-carry-no-tool-version-stamp.md)은
그대로다. 2026-09-24 `ce task gate --json`의 키는 `status`, `steps`,
`summary`뿐이다. 어느 `ce`가 READY를 냈는지 판정 안에 없다. 호스트마다
바이너리가 다르면 불일치를 나중에 조사해야 한다. 소유는 ce-agent-kit#8.

방향: gate, validate, run-finish의 기계 판정에 도구 버전을 싣는다.
사람이 읽는 문장에만 적고 JSON에서 빼지 않는다.

## Completion Criteria

- [ ] `ce task gate --json`과 신규 run-finish 영수증에 도구 버전 필드가 있다 | verify: human — 상류 반영 후 이 저장소 출력에 그 필드가 있다

## Out of scope

- DVA 게이트 로그에 버전을 따로 받아 적는 래퍼.

## Sources

- [ISSUE-030](../issue/030-gate-verdicts-carry-no-tool-version-stamp.md)
