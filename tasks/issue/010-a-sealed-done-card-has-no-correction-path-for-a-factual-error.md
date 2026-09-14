---
id: ISSUE-010
title: "A receipt-sealed done card has no correction path for a factual error in its own record"
type: bug
status: todo
priority: P2
effort: M
exec-tier: strong
severity: low
discovered-in: "TASK-387 — TASK-376의 quality-review-evidence가 리뷰 지적 다섯 건을 '전부 수정했다'고 적었으나 한 건의 전반부가 남아 있었고, 그 문장을 고칠 수 없었다"
discovered-at: 2026-09-14
created: 2026-09-14
---

## Summary

`quality-review-receipt`를 단 done 카드는 **카드 전체가 digest로 봉인된다.** 봉인에서
빠지는 필드는 `quality-review-receipt` 자신 하나뿐이다. 그래서 카드에 남은 사실관계
오류를 편집으로 정정하면 receipt가 깨지고 카드가 invalid가 된다.

이것은 결함이 아니다 — 편집으로 고칠 수 있는 봉인은 봉인이 아니다. 결함은 **정정
경로가 없다는 것**이다. 오늘 선택지는 둘뿐이고 둘 다 나쁘다.

1. 카드를 고치고 receipt를 새 digest로 다시 쓴다 → 실제 리뷰 없이 리뷰가 있었던 것처럼
   만드는 위조다. `ce task validate`는 digest만 보므로 구분하지 못한다.
2. 틀린 문장을 그대로 둔다 → 보드가 틀린 기록을 정본으로 나른다.

## Evidence

2026-09-14 실측. `tasks/done/376-prepare-a-rehearsed-dogfood-run-harness-for-the-live-verification-round.md` 끝에 주석 한 줄을 덧붙이고 되돌렸다.

```
$ printf '\n<!-- digest probe -->\n' >> tasks/done/376-*.md
$ ce task validate tasks/done/376-*.md
❌ quality-review-receipt tasks/receipts/TASK-376/done-review-9515457848....json
   pins reviewed-card-sha256 9515457848... but this card digests to 296edf609b...:
   the card changed after it was reviewed
$ git checkout -- tasks/done/376-*.md && ce task validate tasks/done/376-*.md
✅ Valid (no errors or warnings)
```

구체적 오류: TASK-376의 `quality-review-evidence`가 "Low 5건을 같은 브랜치에서 전부
수정했다"고 적는다. Low 하나(`:441/:446` 트랩 부재 + `:394` 대조군 캐시)는 두 지적을 한
줄에 담고 있었고 후반부만 닫혔다. 전반부는 [[TASK-387]]이 "적용하지 않는다"로 판정하고
그 근거를 스크립트 주석에 남겼다 — 카드에는 남길 수 없었다.

## 왜 재리뷰가 답이 아닌가 (오늘 기준)

원칙적으로는 재리뷰가 정답이다: 새 리뷰어가 카드를 다시 읽고 새 receipt를 발급한다.
하지만 그러려면 **이미 통합된 done 카드를 다시 리뷰 대상으로 올리는 절차**가 있어야
하고, 지금 보드에는 없다. [[TASK-384]]가 TASK-344/371에 대해 정확히 그 절차를 한 번
수행하므로, 그 절차가 일반화 가능한 형태로 남는지가 이 이슈의 선행 조건이다.

## Reproduction

1. receipt를 단 done 카드를 고른다 — 예: `tasks/done/376-prepare-a-rehearsed-dogfood-run-harness-for-the-live-verification-round.md`.
2. 그 카드에 아무 문자나 한 줄 덧붙인다: `printf '\n<!-- probe -->\n' >> <card>`.
3. `ce task validate <card>`.
4. `git checkout -- <card>` 후 다시 `ce task validate <card>`.

## Expected vs Actual

**Expected.** 봉인된 카드의 사실관계를 정정하려는 사람이, 위조가 아닌 정식 경로를 밟아
카드를 고치고 새 receipt를 받을 수 있다.

**Actual.** 정정 경로가 없다. 편집은 즉시 invalid를 만들고, 유일하게 "동작하는" 우회는
리뷰 없이 digest만 새로 적는 것 — 즉 위조다. `ce task validate`는 digest 일치만 보므로
진짜 재리뷰와 위조를 구분하지 못한다. 구분은 도구가 아니라 사람의 규율에만 걸려 있다.

## Impact

낮다 — 오늘 알려진 인스턴스는 TASK-376 하나이고, 틀린 문장은 완료 판정이 아니라 리뷰
요약의 범위 서술이다. 다만 **보드가 커질수록 조용히 늘어나는 종류의 부채**다. 봉인된
카드의 수는 단조 증가하고, 그중 사실관계가 낡는 카드의 비율은 0이 아니다.

## Resolution Criteria

- [ ] 봉인된 done 카드의 사실관계 정정 절차가 문서로 존재한다 | verify: `human — 절차 문서를 읽고, 위조(리뷰 없이 digest 갱신)와 구분되는 지점이 명시돼 있는지 확인`
- [ ] TASK-376의 `quality-review-evidence`가 Low 한 건의 전반부 미적용을 반영한다 | verify: `/usr/bin/grep -rq --include='376-*.md' '미적용' tasks`

## Related

- [[TASK-387]] — 이 이슈를 발견한 카드
- [[TASK-384]] — done 카드 재리뷰 절차를 처음으로 실행하는 카드
- [[ISSUE-001]] — receipt 부채의 상위 이슈
