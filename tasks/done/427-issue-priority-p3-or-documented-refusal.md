---
id: TASK-427
title: "Give issue cards a P3 priority or a documented refusal"
type: bug
priority: P3
effort: S
exec-tier: standard
status: done
created: 2026-09-24
quality-review: pass
quality-reviewed-at: 2026-09-25
quality-review-evidence: "독립 done-review PASS: 이슈 검증기와 신규 issue 생성 경로가 P0–P3를 공유하고 기본값 P1을 유지한다. TestIssuePriorityAllowsP3 및 commit CI 70534773fe86bae9bc7227252b4f77cb 통과."
---

## Summary

과거에는 issue의 `priority`가 P0–P2만 허용돼 task 어휘와 달랐다.
`ce-agent-kit#3`의 상류 수정으로 이 제한이 해소되어 [ISSUE-019](../_archive/issue/019-issue-cards-cannot-express-p3-while-task-cards-can.md)는
닫혔다.

방향: issue도 P3를 허용하거나, 거부 이유를 스키마 문서에 한 줄로 고정한다.
이 저장소에 priority 우회 검사를 만들지 않는다.

## Completion Criteria

- [x] issue 카드가 P3를 허용하거나, 거부 이유가 상류 스키마 문서에 있다 | verify: human — `canonical_validator.go`의 issue priority enum 또는 그 문서가 이 카드에 링크된다

## Evidence

ce-agent-kit `5d70c9d8`가 issue priority에 P3를 넣었다. `ce task new issue --priority P3`도 같은 목록을 쓴다. 기본값은 P1 그대로다. `TestIssuePriorityAllowsP3`가 통과했다.

## Out of scope

- 이 보드의 기존 이슈를 P3로 일괄 변경하는 일. 어휘가 열린 다음에 한다.

## Current evidence

- Issue priority enum and `ce task new issue` use the same P0–P3 set; default remains P1.
- Regression test: `TestIssuePriorityAllowsP3` in ce-agent-kit, integrated at `5d70c9d8`.
- Full commit CI run `70534773fe86bae9bc7227252b4f77cb` passed after this source commit.
- Independent done-review: PASS; reviewer confirmed the enum, creation path, and default behavior.

## Sources

- [ISSUE-019](../_archive/issue/019-issue-cards-cannot-express-p3-while-task-cards-can.md)
