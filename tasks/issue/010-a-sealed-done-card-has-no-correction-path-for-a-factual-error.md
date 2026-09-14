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

**`blocks:`를 선언한** done 카드에 한해, `quality-review-receipt`가 가리키는 파일이
읽히면 **카드 전체가 digest로 봉인된다.** 조건은 셋이고 전부 필요하다 —
`validator_receipt.go`의 조기 return 셋에 하나씩 대응한다. 첫째는 카드 파일이
**`tasks/done/` 아래에 있을 것**이다(`:38`은 `zoneFromPath(path)`를 보지
`status:` 필드를 보지 않는다 — `status: done`이어도 파일이 `tasks/doing/`에
있으면 검사는 실행되지 않는다). 나머지 둘이 아래다. `blocks:`가 없으면
`ce task validate`는 receipt 검사에 도달조차 하지 않는다 — `tasks/done/386-…md`는
무수정 상태에서 이미 pinned digest와 현재 바이트가 어긋나 있는데도, 본문에 한 줄을
덧붙여도 `✅ Valid`다. 근거는 이 저장소 안에 이미 있다: TASK-386 receipt의
`reviewed-card-sha256-algorithm`이 "declares no `blocks:`, so `ce task validate` never
reaches the quality-review-receipt check"라고 직접 적고, `tasks/done/382-…md`의
`quality-review-evidence`는 review-382가 그 조기 return을 `ce-agent-kit`
소스(`validator_receipt.go`)와 합성 카드 실험 양쪽으로 검증했다고 적는다.
`quality-review-receipt` 자신은 봉인에서 빠진다 — 아래 §Evidence의 프로브 둘이
그 한 필드에 대해 양방향으로 확인한 것이고, **다른 필드도 빠지는지는 재지 않았다.**
digest 범위의 정본은 이 저장소가 아니라 `ce` 쪽에 있고, 여기서 관측할 수 있는 것은
"이 필드는 빠진다"와 "**검사가 실행되는 카드에서는** 본문 한 줄을 더하면 깨진다"까지다.
그래서 검사가 실행되는 카드에 남은 사실관계 오류를 편집으로 정정하면 receipt가 깨지고
카드가 invalid가 된다. 검사가 실행되지 않는 카드는 반대 방향으로 나쁘다 — 고칠 수는
있지만 receipt가 아무것도 보증하지 않는다.

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

반대 방향도 쟀다. 같은 카드에서 `quality-review-receipt` **그 한 줄만** 다른 경로로
바꾸고(같은 바이트의 receipt를 `tmp/probe-376.json`에 복사해 두고 포인터를 그쪽으로
돌렸다) 다시 validate하면 `✅ Valid`다. 즉 본문 한 줄을 더하면 깨지는 봉인이 이
필드의 값 변경에는 반응하지 않는다 — `tasks/receipts/` 이관([[TASK-388]])이 18장의
포인터를 고치면서도 봉인을 깨지 않은 근거가 이것이다.

구체적 오류: TASK-376의 `quality-review-evidence`가 "Low 5건을 같은 브랜치에서 전부
수정했다"고 적는다. Low 하나(`:441/:446` 트랩 부재 + `:394` 대조군 캐시)는 두 지적을 한
줄에 담고 있었고 후반부만 닫혔다. 전반부는 [[TASK-387]]이 "적용하지 않는다"로 판정하고
그 근거를 스크립트 주석에 남겼다 — 카드에는 남길 수 없었다.

## 왜 재리뷰가 답이 아닌가 (오늘 기준)

원칙적으로는 재리뷰가 정답이다: 새 리뷰어가 카드를 다시 읽고 새 receipt를 발급한다.
하지만 그러려면 **이미 통합된 done 카드를 다시 리뷰 대상으로 올리는 절차**가 있어야
하고, 지금 보드에는 없다. [[TASK-384]]가 TASK-344/371에 대해 정확히 그 절차를 한 번
수행하도록 **계획돼 있다** — 2026-09-14 기준 그 카드는 아직 master에 없고
([[TASK-388]]의 브랜치에만 있다) 착수되지도 않았다. 따라서 "그 절차가 일반화 가능한
형태로 남는지"는 오늘 확인된 사실이 아니라 앞으로 확인할 것이고, 이 이슈는 그것을
기다린다. TASK-384가 끝나기 전에는 여기에 적을 절차가 없다.

## Reproduction

**선행 조건 둘.** 고른 카드가 (1) `blocks:`를 선언해야 검사가 실행되고,
(2) `quality-review-receipt` 경로가 현재 체크아웃에서 실제로 읽혀야 한다. 둘 중
하나라도 빠지면 3단계는 이 이슈의 결함이 아니라 **다른 것**을 보여 준다 — (1)이
빠지면 `✅ Valid`(검사에 도달조차 하지 않는다), (2)가 빠지면
`quality-review-receipt ... cannot be read`.

이 문단은 원래 "TASK-376의 포인터가 gitignore된 `tmp/` 아래라 재현이 안 된다"고
적었다. [[TASK-388]]이 receipt를 추적 경로로 옮긴 뒤로 **더는 그렇지 않다.** 아래
절차는 이제 있는 그대로 재현된다. 조건을 만족하는 카드 목록은 §Impact에 있다.

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

**봉인이 실제로 걸리는 카드는 소수다.** 모수가 두 번 좁혀진다. 먼저 `blocks:`가
있어야 검사가 실행되고(§Summary), 그다음 포인터가 현재 체크아웃에서 읽혀야 한다.

이 절은 처음 쓸 때 "오늘 걸리는 카드는 TASK-376 한 장뿐"이라고 적었다. 그때는
참이었다 — receipt 대부분이 gitignore된 `tmp/` 아래여서 뒤쪽 조건이 거의 아무 데서도
성립하지 않았다. [[TASK-388]]이 그 조건을 **닫았고**(과거형이다), 그래서 이 수는
바뀌었다. master `8b56802` 기준 done 카드 전수 조사:

| 카드 | `blocks:` | 포인터 읽힘 | 봉인 |
|---|---|---|---|
| TASK-344 | `[TASK-343]` | receipt 없음 | 아니오 |
| TASK-371 | `[TASK-354]` | receipt 없음 | 아니오 |
| TASK-376 | `[TASK-328, TASK-348]` | 예 | **예** |
| TASK-377 | `[TASK-354]` | 예 | **예** |
| TASK-378 | `[TASK-329]` | 예 | **예** |
| TASK-379 | `[TASK-328]` | 예 | **예** |

**넷이다.** 나머지 done 카드는 전부 `blocks:`를 선언하지 않아 receipt를 달고도 검사에
도달하지 않는다. 즉 위조를 막는 장치는 여전히 **대부분의 카드에서 켜져 있지 않고**,
그 앞쪽 조건은 카드 작성자가 `blocks:`를 쓰는지에 달려 있어 아무도 닫지 않는다.

(이 수를 세는 것 자체가 이 이슈의 사례다. 원래 문장은 쓴 날 참이었고 TASK-388이
통합되면서 조용히 틀려졌다 — 카드가 봉인되기 전이라 고칠 수 있었을 뿐이다.)

그 위에서, 낮다 — 오늘 알려진 **낡은 기록**은 세 자리이고, 두 종류로 갈린다.

- **정정 불가 — TASK-376 한 자리.** `blocks:`를 선언하고 receipt가 읽히므로 봉인돼
  있다. 본문을 고치면 digest가 움직여 `ce task validate`가 빨간불이 된다. 틀린 문장은
  완료 판정이 아니라 리뷰 요약의 범위 서술이다.
- **정정 가능하지만 무보증 — `tasks/done/386-…md`의 `TASK-383` 참조 둘**(→
  [[ISSUE-012]]). 386은 `depends-on: []`이고 `blocks:`가 없어 위 표에 없다. 재확인했다:
  본문에 한 줄을 덧붙여도 `✅ Valid`다. 지금 당장 편집으로 고칠 수 있다 — 대신
  그 카드의 receipt는 편집 전후 어느 쪽도 보증하지 않는다. 검사에 닿지 않기 때문이다.

두 종류가 §Summary의 양방향 논지를 그대로 실증한다. 봉인은 위조를 막는 대신 정정도
막고, 봉인이 없으면 정정은 자유롭지만 receipt가 아무것도 말하지 않는다. 다만 **보드가
커질수록 조용히 늘어나는 종류의 부채**다. 봉인된 카드의 수는 단조 증가하고, 그중
사실관계가 낡는 카드의 비율은 0이 아니다.

## Resolution Criteria

- [ ] 봉인된 done 카드의 사실관계 정정 절차가 문서로 존재한다 | verify: `human — 절차 문서를 읽고, 위조(리뷰 없이 digest 갱신)와 구분되는 지점이 명시돼 있는지 확인`
- [ ] TASK-376의 `quality-review-evidence`가 Low 한 건의 전반부 미적용을 반영한다 | verify: `human — 376의 quality-review-evidence를 읽고, '전부 수정했다'가 사라졌는지와 미적용 한 건이 어느 지적인지 문장 안에서 읽히는지 확인`

## Related

- [[TASK-387]] — 이 이슈를 발견한 카드
- [[TASK-384]] — done 카드 재리뷰 절차를 처음으로 실행하는 카드
- [[ISSUE-001]] — receipt 부채의 상위 이슈
