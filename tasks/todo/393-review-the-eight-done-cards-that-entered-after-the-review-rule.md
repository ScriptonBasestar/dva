---
id: TASK-393
title: "Review the eight done cards that entered after the review rule"
type: chore
priority: P2
effort: L
exec-tier: strong
status: todo
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

- [ ] 여덟 장 전부가 `quality-review` 값을 갖는다 — 즉 lint가 그 줄을 더 이상 내지 않는다 | verify: `! ce task lint 2>&1 | /usr/bin/grep -q 'UNFINALIZED DONE'`
- [ ] 그 값이 하나도 `waived`가 아니다 — 이 여덟 장에 waiver는 성립하지 않는다 | verify: `! /usr/bin/grep -lE '^quality-review: waived$' tasks/done/003-*.md tasks/done/309-*.md tasks/done/331-*.md tasks/done/349-*.md tasks/done/356-*.md tasks/done/366-*.md tasks/done/374-*.md tasks/done/375-*.md`
- [ ] 각 판정이 근거를 데리고 있다 | verify: `/usr/bin/grep -lE '^quality-review-evidence:' tasks/done/003-*.md tasks/done/309-*.md tasks/done/331-*.md tasks/done/349-*.md tasks/done/356-*.md tasks/done/366-*.md tasks/done/374-*.md tasks/done/375-*.md | /usr/bin/grep -c . | /usr/bin/grep -qx 8`
- [ ] 보드 게이트 통과 | verify: `ce task gate`
- [ ] 문서 게이트 통과 | verify: `make doc-check` (regression-guard)

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
