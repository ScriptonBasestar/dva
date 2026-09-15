---
id: ISSUE-015
title: "The done-entry gate only asks cards that block a successor for a review"
type: bug
status: todo
priority: P2
severity: medium
effort: S
exec-tier: standard
ownership: local
created: 2026-09-14
discovered-in: "2026-09-14 board currentization — TASK-381 landed in done/ with no review verdict"
discovered-at: 2026-09-14
source: "PLAN-007 §2026-09-14 처리 방침이 '유입은 게이트가 막는다'로 재고를 닫았는데, 그 선언 이후 done에 들어온 첫 카드가 막히지 않고 통과했다"
---

## Summary

`ce task validate`가 done 카드에 리뷰 영수증을 요구하는 조건은 **그 카드가 `blocks:`로
후속 카드를 열어 줬는가** 하나뿐이다. 후속을 열지 않는 카드는 리뷰 판정이 없어도
`✅ Valid`로 통과한다.

[[PLAN-007]] §2026-09-14 처리 방침은 미리뷰 재고 39장을 셋으로 배분하면서
**"부채의 유입은 게이트가 막고, 재고는 선언으로 닫는다"**로 끝난다. 재고 쪽 판단은
그대로 유효하다. 문제는 유입 쪽이다 — 게이트가 막는 것은 **유입 전체가 아니라 후속을
가진 유입뿐**이고, 그 구분이 어디에도 적혀 있지 않다.

경계 자체는 설계로 읽힌다. 검증기의 오류 문언이 "the successors were unblocked on an
unrecorded review"이므로, 이 검사가 막으려는 것은 리뷰 부채 일반이 아니라 **기록되지
않은 리뷰를 근거로 다른 카드를 열어 준 것**이다. 좁은 목적에는 정확한 검사다. 결함은
그 좁은 검사를 **유입 차단 일반**으로 읽은 데서 생긴다.

## Evidence

[[TASK-381]]이 그 선언 이후 `done/`에 들어온 첫 카드이고, 리뷰 판정 없이 들어왔다.
`ce task lint`의 미리뷰 집계가 38장으로 하나 늘었고, 그 한 장은 PLAN-007이 만든 세
바구니(활성 plan 자식 22장 · 레거시 선언 17장 · 게이트가 막는 유입) 어디에도 없다.

같은 카드를 격리 사본에서 두 형태로 검증했다. 프론트매터에서 다른 것은 `blocks:` 한
줄뿐이고 본문과 영수증 상태는 동일하다:

```
$ ce task validate <done zone의 TASK-381 사본>
  ✅ Valid (no errors or warnings)

$ # 같은 파일에 blocks: [TASK-354] 한 줄만 추가
$ ce task validate <같은 사본>
  ❌ Done card blocks TASK-354 but declares no quality-review-receipt:
     the successors were unblocked on an unrecorded review
  ❌ Invalid: 1 error(s), 0 warning(s)
```

즉 리뷰 부채의 유무는 두 경우가 같고, 게이트의 판정만 갈린다.

대조군으로 TASK-344는 `blocks: [TASK-343]`을 선언하고 `quality-review: pass`와
`quality-review-receipt`를 함께 갖는다 — 검사가 걸리는 쪽은 이 형태를 요구했고 실제로
받아냈다. 검사가 하는 일 자체는 정상 동작한다.

## Reproduction

1. `blocks:`가 없는 done 카드 하나를 고른다. `quality-review` 계열 키는 없는 상태여야 한다.
2. `ce task validate <그 카드>` — `✅ Valid`.
3. 프론트매터에 `blocks: [<존재하는 카드 id>]` 한 줄을 추가한다.
4. `ce task validate <그 카드>` — `quality-review-receipt` 부재로 거부된다.

2와 4 사이에 리뷰 상태는 바뀌지 않았다.

## Expected vs Actual

- Expected: done 진입에 리뷰 판정을 요구할 것인지가 **후속 카드의 유무와 무관하게**
  하나의 규칙으로 정해지고, 그 규칙이 문서에 적힌 것과 같다.
- Actual: `blocks:`를 선언한 카드만 요구받는다. `blocks:` 없는 카드는 무제한으로 유입되고,
  `ce task lint`의 `UNFINALIZED DONE` 카운트만 조용히 자란다.

## Impact

중간. 조용히 틀리지는 않는다 — `ce task lint`가 매 게이트마다 미리뷰 수를 세어 내므로
부채는 보이는 곳에 쌓인다. 그러나 **보이는 것과 막히는 것은 다르다.** PLAN-007이
재고 17장을 레거시로 선언하며 치른 대가는 "다시는 이렇게 쌓이지 않는다"는 전제였고,
그 전제가 절반만 참이면 같은 선언을 나중에 다시 해야 한다. PLAN-007 문서 안에서 이미
한 번 반복된 실패 패턴("독립 6장은 TASK-373으로 분리한다"던 자리가 다시 9장으로 비었다)의
세 번째 회차가 된다.

비용의 방향도 적어 둔다. 이 검사를 모든 done 카드로 넓히면 후속 없는 작은 카드마다
독립 리뷰어 한 명이 필요해진다. 그것이 옳은 기본값인지는 이 카드가 결정하지 않는다 —
결정해야 한다는 사실만 기록한다.

## 소유권 — 이 저장소다 (2026-09-15 명시)

검사 자체는 "좁은 목적에는 정확한 검사"다 — 결함은 그 검사를 유입 차단 일반으로 읽은
[[PLAN-007]]의 지역 서술에서 생긴다. 기준 2(또는)가 PLAN-007 정정만으로 카드를 닫는
로컬 경로를 제공한다. 기준 1(게이트 확대)을 최종 결정으로 택하면 상류 소유로
재분류한다.

## Resolution Criteria

- [ ] done 진입 시 리뷰 판정을 요구하는 규칙이 `blocks:` 유무와 독립적으로 문서화되고,
      검증기의 동작이 그 문서와 일치한다 | verify: human — 상류 규칙 문서와
      `ce task validate`의 동작이 같은 조건을 말하는지 확인
- [ ] 또는 현재 경계가 의도된 설계임이 확인되고, [[PLAN-007]]의 "유입은 게이트가 막는다"가
      "후속을 여는 유입만 막는다"로 정정되어 남은 구멍의 크기가 기록된다
      | verify: human — PLAN-007 처리 방침 절에 정정이 반영됐는지 확인
- [ ] 회귀 관찰: `blocks:`를 선언한 done 카드는 영수증 없이 보드에 남지 않는다
      (오늘 이미 참이며 이 카드가 넓히려는 경계의 안쪽이다)
      | verify: `ce task gate`

## Related

- [[PLAN-007]] — §2026-09-14 처리 방침이 "유입은 게이트가 막는다"로 재고를 닫았다.
  이 카드는 그 문장의 범위를 정정한다.
- [[TASK-381]] — 선언 이후 유입된 첫 카드이자 이 결함의 재현 사례.
- [[TASK-388]] — 영수증을 둘 durable 경로를 열었다. 그 경로가 있어도 요구받지 않는
  카드는 영수증을 만들지 않는다.
- [[ISSUE-010]] — 봉인된 done 카드의 정정 경로 문제. 같은 계열(done 카드의 사후 기록)이지만
  그쪽은 이미 판정을 받은 카드를 다룬다.
