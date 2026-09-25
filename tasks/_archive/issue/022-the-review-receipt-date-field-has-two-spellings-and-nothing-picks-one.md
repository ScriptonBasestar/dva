---
id: ISSUE-022
title: "The review-receipt date field has two spellings and nothing picks one"
type: bug
status: done
priority: P2
severity: low
ownership: upstream
created: 2026-09-14
discovered-at: 2026-09-14
discovered-in: "TASK-393 (2026-09-14)"
upstream-ref: "ce-agent-kit#3"
resolution: fixed
resolved-at: 2026-09-25
resolution-summary: "ce-agent-kit 5d70c9d8 makes quality-reviewed-at canonical and warns/ignores quality-review-date; current tasks/done has no legacy aliases."
---

## Summary

Resolved in ce-agent-kit master `5d70c9d8`; the incident measurements below are
historical. `quality-reviewed-at` is canonical, and the legacy alias warns and is
ignored.

독립 done 리뷰 영수증의 날짜 필드가 이 보드에 **두 이름으로 존재한다.**
`quality-reviewed-at`와 `quality-review-date`다. 어느 검사도 둘 중 하나를 고르지
않는다 — `ce task validate`도, `ce task lint`도, `make doc-check`도 통과한다.

## Evidence

TASK-393 착수 직전 done 존의 실측:

```
$ /usr/bin/grep -rhoE '^quality-review(ed-at|-date):' tasks/done | sort | uniq -c
  10 quality-review-date:
  56 quality-reviewed-at:
```

`ce task lint`는 이 66장을 모두 "영수증 있음"으로 읽는다 — `UNFINALIZED DONE` 줄에
하나도 올라오지 않는다. lint가 실제로 보는 것은 `quality-review` 판정 필드뿐이고,
날짜 필드는 이름이 무엇이든 보지 않는다.

## Reproduction

1. done 존 카드 하나에 `quality-review: pass`를 쓰고 날짜 필드를 `quality-review-date`로 적는다.
2. 같은 존의 다른 카드에는 `quality-reviewed-at`으로 적는다.
3. `ce task validate --all`, `ce task lint`, `make doc-check`를 차례로 돌린다.

셋 다 초록이다. 두 카드의 영수증이 같은 스키마를 따른다고 주장할 근거가 어디에도
생기지 않는다.

## Expected vs Actual

| | |
|---|---|
| 기대 | 영수증 스키마가 필드 이름 하나를 정하고, 다른 이름은 거부되거나 최소한 경고가 뜬다 |
| 실제 | 두 이름이 공존하고 어느 게이트도 차이를 보지 못한다. 세 번째 이름을 지금 써도 똑같이 초록이다 |

## Impact

낮다 — **지금은** 사람이 읽어서 날짜를 찾을 수 있다. 위험은 나중에 온다.
영수증을 기계로 집계하려는 순간(예: "리뷰 없이 90일 지난 카드") 두 이름 중 하나만
보는 쿼리가 조용히 절반을 놓친다. 그리고 그 쿼리는 **초록으로 놓친다** — 이 보드가
이미 여러 번 만난 모양이다.

이름이 갈린 10장은 2026-09-14 이전에 생겼다. 즉 드리프트는 관측되기 전에 이미
진행 중이었고, 관측된 것은 여덟 장을 손으로 찍다가 우연히였다.

## 소유권 — 상류다

정본 스키마는 `ce-agent-kit`의 canonical validator에 있다. 이 저장소가 필드 이름을
통일해도(TASK-393이 여덟 장에 대해 그렇게 했다) 다음 카드가 다시 갈리는 것을 막지
못한다 — 막는 것은 스키마뿐이다. 보고는 [[TASK-399]]가 `ce-agent-kit#3`으로 수행했다.

## Resolution Criteria

- [x] 영수증 날짜 필드 이름이 상류 스키마에 하나로 고정된다 | verify: human — canonical validator가 그 필드를 enum/required로 다루는지 읽고 확인한다
- [x] 이 보드의 done 존에 이름이 하나만 남는다 | verify: `! /usr/bin/grep -rqE '^quality-review-date:' tasks/done`
- [x] 보드 게이트 통과 | verify: `ce task gate` (regression-guard)

## Resolution summary

ce-agent-kit `5d70c9d8`는 `quality-reviewed-at`을 정본으로 고정하고 legacy
`quality-review-date`를 경고 후 무시한다. `TestReviewDateAliasWarnsWithoutFailing`
통과. 현행 `tasks/done/`에는 `quality-review-date` 필드가 없다. commit CI
`70534773fe86bae9bc7227252b4f77cb`와 TASK-429 독립 done-review가 PASS다.
