---
id: TASK-434
title: "Report list-form quality-review-evidence honestly"
type: bug
priority: P3
effort: S
exec-tier: standard
status: done
created: 2026-09-24
quality-review: pass
quality-reviewed-at: 2026-09-25
quality-review-evidence: "독립 done-review PASS: 리스트형 evidence는 필드 부재가 아니라 single-line scalar 형식 오류로 보고한다. TestListFormReviewEvidenceNamesTheType 및 전체 commit CI 70534773fe86bae9bc7227252b4f77cb 통과."
---

## Summary

`quality-review-evidence`가 리스트일 때 과거 검증기는 필드가 없다고 잘못
보고했다. `ce-agent-kit#9` 상류 수정은 형식 오류를 정확히 보고하며,
[ISSUE-031](../_archive/issue/031-list-form-quality-review-evidence-is-misreported-as-missing.md)은
닫혔다.

방향: 리스트를 만나면 한 줄 스칼라여야 한다고 말하거나, 리스트를 수용한다.
어느 쪽이든 "필드가 없다"고 말하지 않는다.

## Completion Criteria

- [x] 리스트형 `quality-review-evidence`가 형식 오류로 보고되거나 수용된다 | verify: human — 상류 반영 후 리스트형 카드의 validate 출력이 필드 부재가 아니다

## Evidence

ce-agent-kit `5d70c9d8`. 리스트형 필드는 `quality-review-evidence must be a single-line scalar`로 보고된다. `TestListFormReviewEvidenceNamesTheType`이 그 문장을 고정한다.

## Out of scope

- 이미 스칼라로 고친 이 보드 카드의 내용 변경.

## Sources

- [ISSUE-031](../_archive/issue/031-list-form-quality-review-evidence-is-misreported-as-missing.md)

## Current evidence

- `validator_review_stage.go` reports list-form values as a type error: `quality-review-evidence must be a single-line scalar`.
- Regression test: `TestListFormReviewEvidenceNamesTheType` in `validator_review_stage_test.go:34-44`.
- Integrated source commit: `ce-agent-kit@5d70c9d8`; full commit CI run `70534773fe86bae9bc7227252b4f77cb` passed.
- Independent done-review: PASS.
