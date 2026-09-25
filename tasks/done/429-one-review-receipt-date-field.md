---
id: TASK-429
title: "Pin one date field name on review receipts"
type: bug
priority: P2
effort: S
exec-tier: standard
status: done
created: 2026-09-24
quality-review: pass
quality-reviewed-at: 2026-09-25
quality-review-evidence: "독립 done-review PASS: quality-reviewed-at 정본과 quality-review-date 경고 후 무시 정책이 구현·테스트와 일치한다. 전체 commit CI 70534773fe86bae9bc7227252b4f77cb 통과."
---

## Summary

[ISSUE-022](../_archive/issue/022-the-review-receipt-date-field-has-two-spellings-and-nothing-picks-one.md)의
호환 가능한 정본 규칙이다. 정본은 `quality-reviewed-at`이고, 알려진 과거 철자
`quality-review-date`는 경고 후 무시한다. 두 필드의 값을 합치거나 옛 카드를
다시 쓰지 않는다.

방향: 날짜 집계와 digest는 정본 이름만 사용하고, 알려진 별칭을 침묵 속에 받아들이지
않도록 진단한다. legacy 카드는 깨뜨리지 않는 경고 정책을 유지한다.

## Completion Criteria

- [x] `quality-reviewed-at`이 정본으로 사용되고 `quality-review-date`는 경고 후 무시된다 | verify: human — 상류 경고 테스트와 canonical digest field list가 이 카드에 링크된다

## Evidence

ce-agent-kit `5d70c9d8`. 정본 이름은 `quality-reviewed-at`이다. `quality-review-date`는 경고하고 검증을 실패시키지 않는다. 이미 그 철자로 닫힌 카드를 다시 쓰지 않기 위해서다. `TestReviewDateAliasWarnsWithoutFailing`이 통과했고, 이 저장소 `ce task gate`는 여전히 READY다.

## Out of scope

- 이미 한 철자로 맞춰 둔 이 보드의 과거 카드를 다시 쓰는 일.

## Sources

- [ISSUE-022](../_archive/issue/022-the-review-receipt-date-field-has-two-spellings-and-nothing-picks-one.md)

## Current evidence

- Alias 진단: `internal/usecase/task/validator_review_stage.go:18-20`.
- 정본 날짜 field 읽기: `internal/usecase/task/validator_review_stage.go:27-31`.
- 회귀 테스트: `internal/usecase/task/validator_review_stage_test.go:59-72`, `TestReviewDateAliasWarnsWithoutFailing`.
- Digest에서 정본만 제외: `internal/usecase/task/card_digest.go:22-28`.
- Integrated source commit: `ce-agent-kit@5d70c9d8`.
- Full commit CI run `70534773fe86bae9bc7227252b4f77cb` passed after this source commit.
- Independent done-review: PASS; the warning-and-ignore legacy policy matches implementation.
