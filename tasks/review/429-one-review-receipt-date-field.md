---
id: TASK-429
title: "Pin one date field name on review receipts"
type: bug
priority: P2
effort: S
exec-tier: standard
status: review
created: 2026-09-24
---

## Summary

[ISSUE-022](../issue/022-the-review-receipt-date-field-has-two-spellings-and-nothing-picks-one.md)는
그대로다. 리뷰 날짜가 `quality-reviewed-at`과 `quality-review-date`로 갈린다.
validator가 읽는 이름은 `quality-reviewed-at` 하나다. 다른 철자는 조용히
집계에서 빠진다. 보드의 과거 카드를 한 철자로 고쳐도 다음 카드가 다시 갈린다.
막는 것은 스키마다. 소유는 ce-agent-kit#3.

방향: 날짜 필드 이름을 상류 스키마에서 하나로 고정하고, 다른 철자는 거부하거나
읽어서 같은 필드로 받아들인다.

## Completion Criteria

- [x] 영수증 날짜 필드 이름이 상류 스키마에 하나로 고정된다 | verify: human — canonical validator가 그 필드를 허용 목록 또는 required로 다룬다

## Evidence

ce-agent-kit `5d70c9d8`. 정본 이름은 `quality-reviewed-at`이다. `quality-review-date`는 경고하고 검증을 실패시키지 않는다. 이미 그 철자로 닫힌 카드를 다시 쓰지 않기 위해서다. `TestReviewDateAliasWarnsWithoutFailing`이 통과했고, 이 저장소 `ce task gate`는 여전히 READY다.

## Out of scope

- 이미 한 철자로 맞춰 둔 이 보드의 과거 카드를 다시 쓰는 일.

## Sources

- [ISSUE-022](../issue/022-the-review-receipt-date-field-has-two-spellings-and-nothing-picks-one.md)
