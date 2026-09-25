---
id: TASK-428
title: "Count Resolution Criteria in preflight"
type: bug
priority: P2
effort: S
exec-tier: standard
status: done
created: 2026-09-24
quality-review: pass
quality-reviewed-at: 2026-09-25
quality-review-evidence: "독립 done-review PASS: issue Resolution Criteria 집계와 runnable 처리의 경계가 구현·출력과 일치한다. 설치본 preflight 수치와 commit CI 70534773fe86bae9bc7227252b4f77cb 통과."
---

## Summary

과거 `ce task preflight --zone issue`는 Resolution Criteria만 있는 이슈를
`criteria: 0`으로 셌다. `ce-agent-kit#1` 상류 수정으로 기준 집계가 고쳐져
[ISSUE-020](../_archive/issue/020-preflight-counts-zero-criteria-on-a-card-whose-criteria-live-under-resolution-criteria.md)은
닫혔다.

방향: preflight가 issue 서식이 쓰는 기준 제목을 센다. 카드에 Completion
Criteria를 한 벌 더 붙여 검사를 통과시키지 않는다.

## Completion Criteria

- [x] Resolution Criteria만 있는 issue 카드의 preflight 기준 수가 0보다 크다 | verify: human — 이 저장소 issue 존 preflight JSON에서 그 카드의 criteria가 0이 아니다

## Evidence

ce-agent-kit `5d70c9d8`. 설치된 `ce`(`5d70c9d8`)로 2026-09-24에 `ce task preflight --zone issue --json`을 돌렸다. Resolution Criteria가 있는 카드는 기준 수가 1 이상이다. 예: ISSUE-020은 2, ISSUE-004는 3. 기준이 없는 ISSUE-032·039는 0이고, issue는 work-card 게이트로 막지 않아 그대로 runnable이다.

## Out of scope

- 이슈 본문에 `## Completion Criteria`를 추가하는 일.

## Current evidence

- `preflight.go` recognizes `Resolution Criteria` for issue cards.
- Installed `ce` at `5d70c9d8` reports ISSUE-020 criteria 2 and ISSUE-004 criteria 3; cards without criteria remain at 0.
- Full commit CI run `70534773fe86bae9bc7227252b4f77cb` passed after this source commit.
- Independent done-review: PASS; reviewer confirmed issue criteria are counted without applying the work-card runnable gate.

## Sources

- [ISSUE-020](../_archive/issue/020-preflight-counts-zero-criteria-on-a-card-whose-criteria-live-under-resolution-criteria.md)
