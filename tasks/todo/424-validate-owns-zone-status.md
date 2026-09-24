---
id: TASK-424
title: "Make ce task validate own the zone status rule"
type: bug
priority: P2
effort: M
exec-tier: standard
status: todo
created: 2026-09-24
---

## Summary

[ISSUE-007](../issue/007-ce-task-validate-does-not-constrain-the-status-field.md)은
그대로다. `ce task validate`는 `status`가 어휘 밖이거나 존과 어긋나도 통과시킨다.
DVA의 `tools/doccheck` `checkCardStatus`가 그 검사를 대신한다. 상류 검사가
같은 입력을 거부하는 것이 보이기 전에 로컬 검사를 지우면 다른 저장소는
무방비가 된다. 소유는 갈린다. 상류는 ce-agent-kit#3.

방향: zone/status 제약을 `ce task validate`로 옮긴 뒤, 그 빌드가 설치된
다음에만 doccheck의 같은 검사를 제거한다. 상류가 `status` 필드 자체를
없애기로 하면 그것도 같은 완료다. 두 검사가 동시에 살아있는 상태는 그 전까지만
유지한다.

## Completion Criteria

- [ ] `ce task validate`가 어휘 밖 값, 빈 값, 존과 어긋난 status를 거부한다 | verify: human — 상류 테스트와 같은 fixture에 대한 doccheck 거부가 함께 링크된다
- [ ] 그 검사가 설치된 `ce`에서 관찰된 뒤에만 `checkCardStatus`와 zone 표를 제거한다 | verify: human — 제거 커밋이 링크되고, 존이 어긋난 fixture는 공유 게이트를 통해 `make doc-check`가 실패한다

## Out of scope

- 상류 검사 없이 doccheck sweep만 지우는 일.

## Sources

- [ISSUE-007](../issue/007-ce-task-validate-does-not-constrain-the-status-field.md)
