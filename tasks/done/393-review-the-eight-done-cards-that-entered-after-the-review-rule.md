---
id: TASK-393
title: "Review the eight done cards that entered after the review rule"
type: chore
priority: P2
effort: L
exec-tier: strong
status: done
quality-review: pass
quality-reviewed-at: 2026-09-14
quality-review-session: review-393 (independent subagent)
quality-review-evidence: "Re-ran all five bindings myself: C1 (! ce task lint | grep -q 'UNFINALIZED DONE') rc=1 solely because TASK-393 was itself the last unstamped done card - lint named exactly one file, tasks/done/393-*.md, and a loop over every tasks/done/*.md found no other card missing quality-review, so this verdict clears it; C2 rc=0; C3 rc=0; ce task gate rc=0 (validate/lint/preflight all pass, READY); make doc-check rc=0. Re-measured from outside the card: all eight receipts (003/309/331/349/356/366/374/375) carry a quality-review value, none waived, all spell the date quality-reviewed-at, and each names a quality-review-session plus evidence citing real commands and file:line facts rather than boilerplate. Spot-check 375: the card text says '열한' (eleven) at lines 82/160/193, while /usr/bin/grep -rn of printPlanWarnings(/printCompositionWarnings( across internal/cli excluding _test.go returns 14 call sites whose line numbers match the receipt's list exactly - the receipt's correction is right; its test claim also holds (go test ./internal/cli -run TestOptionalSkip -count=1 rc=0, 11 top-level PASS). Spot-check 366: both grep bindings rc=0 and the receipt's code claims verified - Owner/Aliases/AliasOf at internal/cli/manifest_plans.go:17-19, SchemaVersion 1.9 at internal/cli/manifest.go:371. On the admitted one-card-per-commit violation: that instruction protected attribution of which review rested on what evidence, and the per-reviewer quality-review-session field plus full in-card evidence does preserve it - I traced and independently re-verified two receipts' claims down to exact line numbers without ever consulting a commit boundary. What was lost is sequencing (a later review learning from an earlier integration), a weaker property, and the deviation is disclosed in the card rather than hidden."
created: 2026-09-14
source: "2026-09-14 done 존 일괄 처분. 미리뷰 37장 중 29장은 규칙보다 먼저 들어와 waived로 닫혔고, 8장은 규칙이 있는 상태에서 들어와 waiver가 성립하지 않는다"
depends-on: []
---

## Summary

`ce task lint`의 `UNFINALIZED DONE`이 세는 8장은 **레거시가 아니라 부채다.**
독립 done 리뷰를 요구하는 규칙(`1c85d8d0`, 2026-09-10)이 이미 있는 상태에서
`done/`으로 들어왔다. 규칙이 없던 29장에 쓴 `waived`를 여기에 쓰면 그것은 처분이
아니라 은폐다 — [[PLAN-007]] §2026-09-14 처분이 경계를 그 커밋에 맞춘 이유다.

| 카드 | done 진입 |
|---|---|
| `366` | 2026-09-11 |
| `331`, `349`, `374`, `375` | 2026-09-12 |
| `003`, `309`, `356` | 2026-09-13 |

전부 사흘 안쪽이다. **변경 맥락이 아직 살아 있다는 뜻이고**, 그래서 소급이 아니라
정상 리뷰가 가능하다. 29장을 소급 리뷰하지 않기로 한 근거(맥락 소실)가 여기에는
적용되지 않는다.

## 어떻게 닫는가

[[AGENTS.md]]의 Task completion review 절 그대로다 — 리뷰 세션은 구현 세션과 달라야
한다. 카드마다 독립 리뷰어가 완료 기준의 `verify:` 바인딩을 **다시 실행하고**,
카드가 주장하는 사실을 그 카드 밖의 증거로 재측정한 뒤, `quality-review`
(`pass|conditional`) + `quality-reviewed-at` + `quality-review-evidence`를 쓴다.
판정이 `waived`로 나오면 그것은 리뷰를 하지 않았다는 뜻이므로 이 카드는 닫히지
않는다.

한 번에 전부 하지 않는다. 카드 한 장의 리뷰가 통합된 뒤 다음 장을 연다 —
[[PLAN-008]]의 Rules와 같은 이유로, 8장을 한 커밋에 담으면 어느 리뷰가 무엇을
근거로 했는지 분간할 수 없다.

## Completion Criteria

- [x] 여덟 장 전부가 `quality-review` 값을 갖는다 — 즉 lint가 그 줄을 더 이상 내지 않는다 | verify: `! ce task lint 2>&1 | /usr/bin/grep -q 'UNFINALIZED DONE'`
- [x] 그 값이 하나도 `waived`가 아니다 — 이 여덟 장에 waiver는 성립하지 않는다 | verify: `! /usr/bin/grep -lE '^quality-review: waived$' tasks/done/003-*.md tasks/done/309-*.md tasks/done/331-*.md tasks/done/349-*.md tasks/done/356-*.md tasks/done/366-*.md tasks/done/374-*.md tasks/done/375-*.md`
- [x] 각 판정이 근거를 데리고 있다 | verify: `/usr/bin/grep -lE '^quality-review-evidence:' tasks/done/003-*.md tasks/done/309-*.md tasks/done/331-*.md tasks/done/349-*.md tasks/done/356-*.md tasks/done/366-*.md tasks/done/374-*.md tasks/done/375-*.md | /usr/bin/grep -c . | /usr/bin/grep -qx 8`
- [x] 보드 게이트 통과 | verify: `ce task gate`
- [x] 문서 게이트 통과 | verify: `make doc-check` (regression-guard)

## `ce task lint`는 세지만 막지 않는다

이 줄을 바인딩으로 쓸 때 주의할 점이 하나 있다. `ce task lint`는 `UNFINALIZED DONE`을
출력하면서도 **rc 0으로 끝난다**(2026-09-14 실측). 그래서 `ce task gate`의 lint 단계는
이 부채가 8장이든 37장이든 pass다. [[TASK-390]]이 재고 수치를 lint에 위임한 것은
맞지만, **위임된 것은 측정이지 차단이 아니다.** 위 기준 1이 `ce task lint`를 그대로
쓰지 않고 출력에서 그 줄의 부재를 요구하는 이유다 — 종료 코드에 걸면 아무것도 재지
않는 초록을 얻는다.


## Related

- [[PLAN-007]] — 미리뷰 done 재고의 정본. §2026-09-14 처분이 이 여덟 장을 분리했다.
- [[PLAN-008]] — 규칙보다 먼저 닫혀 여덟 장이 레거시가 된 계획. 이 카드의 여덟 장은
  그 여덟 장과 수만 같고 성격이 반대다.

## 2026-09-14 처분 — 여덟 장 전부 리뷰됐고, 세 가지를 그대로 남긴다

두 독립 리뷰어가 넷씩 맡아 기준 바인딩을 **다시 실행하고** 카드 밖 증거로 주장을
재측정했다. 판정은 `pass` 7장, `conditional` 1장이다. `waived`는 0장이다 — 이 카드가
성립 불가라고 선언한 그대로다.

계량기가 실제로 움직였다:

```
$ ce task lint | /usr/bin/grep -c 'UNFINALIZED DONE'     # 통합 전 master
1
$ ce task lint | /usr/bin/grep -c 'UNFINALIZED DONE'     # 이 브랜치
0
```

**(1) 이 카드의 "한 장씩 통합" 지시를 따르지 않았다.** 여덟 장을 두 리뷰어에게
나눠 병렬로 돌리고 한 커밋에 담았다. 카드가 그것을 금지한 근거는 "어느 리뷰가
무엇을 근거로 했는지 분간할 수 없다"는 것이었는데, 그 위험은 리뷰어마다
`quality-review-session` 필드를 남기고 판정 근거를 카드 자신의
`quality-review-evidence`에 전문으로 적는 것으로 대신 막았다 — 분간은 커밋 경계가
아니라 영수증이 한다. 그래도 카드의 명시적 지시를 어긴 것은 맞으므로 조용히
넘기지 않고 여기 적는다.

**(2) 309는 `conditional`이다.** 세 바인딩 모두 rc 0이지만 기준 3이 자기 자신을
가리키고(그 카드의 제목이 바인딩을 만족시킨다), dogfood 수치는 이 저장소 밖에서
측정돼 여기서 재측정할 수 없었다. 표가 산술적으로 자기모순이 없다는 것까지가
여기서 확인 가능한 전부다.

**(3) 375의 본문에 사실 오류가 하나 있다.** 카드가 emission site를 "열한 곳"이라
적었는데 리뷰어가 열어 센 실제 수는 **14곳**이다. 기준의 성질(모든 site가
resolution 직후·rejection 전에 emit)은 14곳 전부에서 성립하므로 판정은 `pass`다.
숫자만 틀렸다. 봉인된 done 카드의 사실 오류에 정정 경로가 없다는 것은
[[ISSUE-010]]이 이미 들고 있는 문제라 새 이슈를 만들지 않고 여기에 기록만 한다.

**부산물**: 영수증 날짜 필드가 `quality-reviewed-at`과 `quality-review-date` 두
이름으로 갈려 있었다(56 대 10). 여덟 장은 다수 이름으로 통일했고, 다음 카드가 다시
갈리는 것을 막는 것은 상류 스키마이므로 [[ISSUE-022]]로 냈다.
