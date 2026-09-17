---
id: ISSUE-012
title: "Card ids are allocated by scanning the board, so parallel worktrees collide"
type: bug
status: todo
priority: P2
severity: medium
ownership: local
created: 2026-09-14
discovered-in: "TASK-388 (dev/claude/mbp/fix/durable-receipt-path), 2026-09-14 리베이스 중"
discovered-at: 2026-09-14
promoted-to: [TASK-407]
---

## Summary

**두 세션이 같은 카드 번호 383을 각자 발급했다.**

- `276988b` (master) — `tasks/todo/383-diagnose-a-missing-image-that-compose-cannot-build.md`
- `dev/claude/mbp/fix/durable-receipt-path` — `383-move-review-receipts-to-a-tracked-durable-path.md`

둘 다 자기 시점의 보드에서 최대 id가 382임을 보고 383을 골랐다. 어느 쪽도 틀리게
행동하지 않았다 — **번호 발급이 "지금 보이는 것 중 최대+1"이고, 워크트리가 격리돼
있으므로 서로가 보이지 않는다.** 저장소가 병렬 작업을 권장하면서 병렬에 안전하지
않은 발급 규칙을 쓴다.

이번에는 리뷰어(review-387)가 base 낡음을 지적하다 우연히 발견했다. 리베이스가 두
파일을 **충돌 없이** 나란히 놓기 때문에 Git은 아무 경고도 하지 않는다.

## Evidence

```
$ git ls-tree -r --name-only origin/master -- tasks | grep -E '/38[0-9]-'
tasks/done/380-currentize-the-docs-that-task-379-left-behind.md
tasks/done/382-currentize-task-380-closing-record.md
tasks/todo/381-check-plan-prose-against-plan-frontmatter.md
tasks/todo/383-diagnose-a-missing-image-that-compose-cannot-build.md
```

### 정정 (2026-09-14) — 중복 id 검사는 있고, 실제로 잡는다

이 절은 원래 "리베이스 직후 383이 둘 있었는데 `ce task validate`도 `make doc-check`도
오류를 내지 않았다 — 중복 id를 재는 게이트가 없다"고 적었다. **틀렸다.**
`tools/doccheck/cardids.go:44`의 `checkDuplicateCardIDs`가 `check.go:244`에서 호출되고,
커밋 `c69653e` "fix(doccheck): report task ids claimed by more than one card"는
**2026-09-07**로 이 충돌보다 일주일 앞선다. 그 함수의 주석이 이번 시나리오를 그대로
적고 있다 — *"a worker numbering a new card from the highest id it could see picked ids
already taken on master."* 파일명 번호 중복 검사도 함께 있다.

직접 쟀다. `tasks/todo/`에 `id: TASK-388`인 스크래치 카드를 하나 떨구고 `make doc-check`:

```
card_ids:            401 (duplicate: 1)
filename_numbers:    397 (duplicate: 0)
  DUP-ID   TASK-388 is claimed by 2 cards: tasks/done/388-....md, tasks/todo/999-probe-duplicate-id.md
doc-check: FAIL
```

프로브를 지우면 rc=0으로 돌아온다. **게이트는 있고 작동한다.**

### 그러면 왜 초록불이었나 — 두 카드가 한 트리에 같이 있던 적이 없어서다

원래 서술의 관측("리베이스 직후 둘이 나란히 있었다")이 실제로 성립한 창은 없었거나
그 창에서 `make doc-check`이 돌지 않았다. 어느 쪽이든 결론은 같고, 이것이 이 이슈의
**진짜** 결함이다: 충돌은 두 브랜치의 **합집합에서만** 존재했다. 각 체크아웃은 자기
쪽 383만 보고, 둘 다 정당하게 초록불이다. **체크아웃 단위 게이트로는 원리적으로 잡을
수 없다** — 발급이 경쟁하는 순간과 두 파일이 한 트리에서 만나는 순간 사이에 시차가
있고, 뒤쪽이 오기 전에 사람이 개명하면 게이트는 아무것도 못 본다.

### 2026-09-15 재발 — 이번엔 번호 셋과 **작업 자체**가 겹쳤다

같은 형태가 다시 일어났고, 이번에는 규모가 다르다. 두 세션이 [[TASK-395]]를
**둘 다 착수해 둘 다 끝냈다.** 각자 독립 리뷰까지 받았고 둘 다 게이트 초록이었다.

| 겹친 것 | master (`95c17bb` 등) | `dev/claude/mbp/feat/task-395` |
|---|---|---|
| TASK-395 구현 | 산문 `## 소유권` 제목 매칭 | frontmatter `ownership:` 필드 |
| ISSUE-024 | done-finalize가 review-evidence의 산출물을 막는다 | 같은 상류 명령이 두 저장소에 귀속된다 |
| TASK-396 | 오버사이즈 카드 둘을 예산 안으로 | 상류 이슈를 열고 번호를 되적는다 |

번호 충돌은 이 카드가 이미 다루던 것이다. 새로 드러난 것은 **번호가 아니라 카드
자체의 중복 수행**이다 — 보드에는 "이 카드를 다른 브랜치가 지금 잡고 있다"를
표현할 자리가 없다. `status: todo`는 어느 워크트리에서도 todo다.

두 구현 중 하나를 버리지 않고 [[TASK-398]]로 축만 얹어 합쳤고, 중복된 두 카드는
ISSUE-027 · TASK-399로 재번호했다. 즉 **손실은 없었지만 그건 사람이 리베이스
충돌을 보고 알아챘기 때문이지 어떤 검사가 알려줘서가 아니다.** 충돌이 없는
파일들이었다면 조용히 둘 다 들어갔을 것이다.

## 왜 조용한가

중복 id 검사(위 §Evidence)는 **한 트리 안의** 중복만 잰다. 이 충돌은 두 브랜치에
하나씩 있었으므로 어느 체크아웃에서도 중복이 아니었다. 그리고 발견 즉시 개명했으므로
둘이 한 트리에서 만나는 순간도 오지 않았다 — 게이트는 정직하게 초록불이었다.

`[[TASK-NNN]]` wikilink는 별개로 전혀 검사되지 않는다. `tools/doccheck/markdown.go:15`가
추출하는 것은 인라인 링크 `[text](target)`와 참조 정의 `[label]: target` 둘뿐이고
doccheck 어디에도 `[[...]]`를 읽는 코드가 없다.

## 개명이 남긴 것 — 존재 검사로는 잡을 수 없는 오조준

충돌을 개명으로 해소하면 **옛 id를 가리키던 참조가 전부 다른 카드로 재바인딩된다.**
383은 살아 있는 번호이므로 이것은 깨진 링크가 아니라 **멀쩡히 풀리는 틀린 링크**다.

2026-09-14 기준 남은 인스턴스는 `tasks/done/386-quote-the-two-unbackticked-verify-bindings.md`
두 자리다.

| 위치 | 형태 | wikilink 검사로 잡히나 |
|---|---|---|
| `:14` | `verification-evidence:` 프론트매터 안의 맨 `TASK-383` | 아니오 — wikilink가 아니다 |
| `:74` | 본문의 `[[TASK-383]]` | 아니오 — `tasks/todo/383-diagnose-…md`로 정상 resolve된다 |

둘 다 고칠 수 없다. 그 카드는 receipt로 봉인돼 master에 통합됐다 — [[ISSUE-010]]이
소유한 "봉인된 카드에 정정 경로가 없다"의 인스턴스다. `tasks/plan/007`과 TASK-385
카드의 같은 오조준은 봉인 전이라 [[TASK-385]]에서 겨눴다.

**그러므로 dangling-wikilink 검사를 해법으로 적으면 안 된다.** 계기가 된 사례를 잡지
못하는 규칙이고, [[TASK-385]]가 [[TASK-381]]에 대해 기록한 것과 똑같은 함정이다.
잡을 수 있는 것은 게이트가 아니라 절차다 — **개명 시점에 in-tree의 옛 id 참조를
일괄로 훑어 재조준하거나 명시적으로 면제한다.**

## Reproduction

1. 워크트리 A에서 보드 최대 id가 N임을 보고 카드 N+1을 만든다.
2. 같은 시각 워크트리 B에서도 같은 것을 본다 — A의 카드는 아직 커밋되지 않았거나
   커밋됐어도 B의 base에 없다 — 그래서 B도 N+1을 만든다.
3. **각 워크트리에서 `ce task validate --all`과 `make doc-check`을 돌린다. 둘 다
   통과한다** — 각자 자기 쪽 N+1만 보이므로 중복이 아니다. 이것이 잡히지 않는 창이다.
4. 한쪽이 먼저 통합된다. 나머지 쪽은 리베이스하면 두 파일이 충돌 없이 나란히 놓인다.
5. 그 상태에서 `make doc-check`을 돌리면 **이제는 잡힌다**: `DUP-ID TASK-N+1 is
   claimed by 2 cards`, `doc-check: FAIL`. 위 §Evidence의 실측이 이 단계다.
6. 5단계 전에 사람이 개명하면 게이트는 끝까지 아무것도 보지 못한다 — 실제로 일어난
   경로가 이것이다.

## Expected vs Actual

- Expected: 발급이 애초에 경쟁하지 않는 방식(예약, 해시, 브랜치별 접두사 등)으로
  이뤄진다 — 두 브랜치가 같은 번호를 **고르지 않는다.**
- Actual: 둘 다 조용히 같은 번호를 고른다. 사후 검사(`DUP-ID`)는 두 파일이 한 트리에서
  만난 뒤에만 작동하고, 그 전에 개명하면 영영 작동하지 않는다. 검사는 안전망이지
  발급 규칙이 아니다.

## Impact

중간. 데이터 손실은 없지만 **참조가 가리키는 대상이 모호해진다** — 이 저장소가
카드 사이 의존(`depends-on:`, `blocks:`, `children:`, `[[TASK-NNN]]`)에 크게 기대므로,
모호한 id는 그 구조 전체의 신뢰도를 깎는다. 이번 인스턴스는 발견이 빨라
`TASK-388`로 개명해 해소했다(아래 Related). 늦게 발견됐다면 이미 통합된 참조
양쪽을 풀어야 했다.

## 소유권 — 이 저장소다 (2026-09-15 명시)

잡을 수 있는 것은 게이트가 아니라 절차다 — 열린 기준 2·3은 AGENTS.md의 발급 규칙
정립과 개명 절차 문서화, 즉 보드 관행이다. Related가 [[ISSUE-011]]과 대비해 이 카드는
상류 소유가 아님을 명시한다.

## Resolution Criteria

- [x] 중복 카드 id를 재는 검사가 `make doc-check`에 있다 — **이슈 작성 시점에 이미 충족돼 있었다**(`c69653e`, 2026-09-07) | verify: `/usr/bin/grep -q 'func checkDuplicateCardIDs' tools/doccheck/cardids.go`
- [ ] 발급 규칙이 병렬 워크트리에서 충돌하지 않도록 정해지고 `AGENTS.md`에 적혀 있다 | verify: `human — AGENTS.md의 발급 규칙을 읽고, 두 워크트리가 서로를 보지 못하는 상태에서도 같은 번호를 고르지 않는지 확인`
- [ ] 카드 개명 절차가 옛 id 참조 일괄 훑기를 포함한다 | verify: `human — 절차 문서를 읽고, 봉인된 카드에 도달하지 못하는 경우의 처리까지 적혀 있는지 확인`

## Related

- [[TASK-388]] — 충돌한 쪽. 이 이슈를 발견한 작업이며, 개명으로 인스턴스를 해소했다.
- [[ISSUE-011]] — 같은 계열(게이트가 재지 않아 조용한 결함)이지만 그쪽은 상류 소유다.
- [[ISSUE-010]] — 개명이 남긴 `done/386`의 오조준 둘을 고칠 수 없는 이유를 소유한다.
- [[TASK-385]] — 봉인 전이던 두 자리를 `[[TASK-388]]`로 겨눈 작업.
- [[TASK-389]] — 이 정정을 실은 작업.
