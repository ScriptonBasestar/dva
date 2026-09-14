---
id: TASK-389
title: "Correct ISSUE-012's duplicate-id claim and record the plain-sha256 pin drift"
type: docs
priority: P2
effort: S
exec-tier: standard
status: doing
created: 2026-09-14
source: "review-383의 TASK-388 사후 회신. ISSUE-012가 '중복 id를 재는 게이트가 없다'고 적었으나 doccheck의 checkDuplicateCardIDs가 c69653e(2026-09-07)부터 돌고 있다. 그리고 TASK-386·388 두 건에서 plain-sha256 pin이 카드를 닫는 행위 자체로 무효화된 것이 실측됐다"
depends-on: []
---

## Summary

[[TASK-388]]이 통합된 뒤 리뷰어 review-383이 388 자체는 건드리지 않는 정정 셋을
보냈다. 전부 이슈 카드 쪽이고, 둘은 **이미 적힌 문장이 틀렸다**는 지적이다.

1. [[ISSUE-012]]의 "중복 id를 재는 게이트가 없다"가 틀렸다.
2. ISSUE-012에 개명 부작용의 실현 인스턴스가 빠져 있고, 잘못된 해법(존재 검사)을
   적을 위험이 있다.
3. [[ISSUE-001]]에 pin 드리프트의 구조적 성격이 기록돼 있지 않다.

## 1. 중복 id 검사는 있다 — 직접 쟀다

`tools/doccheck/cardids.go:44`의 `checkDuplicateCardIDs`가 `check.go:244`에서
호출된다. 커밋 `c69653e`는 **2026-09-07**로 이 충돌보다 일주일 앞선다. 그 함수의
주석이 이번 시나리오를 그대로 적고 있다 — *"a worker numbering a new card from the
highest id it could see picked ids already taken on master."*

review-383은 이것을 스스로 돌리지 않고 (a) 게이트가 작동한다 / (b) 어떤 이유로
통과한다 중 어느 쪽인지 나에게 남겼다. 프로브를 이 워크트리에서 돌렸다 —
`tasks/todo/`에 `id: TASK-388`인 스크래치 카드를 떨구고 `make doc-check`:

```
card_ids:            401 (duplicate: 1)
filename_numbers:    397 (duplicate: 0)
  DUP-ID   TASK-388 is claimed by 2 cards: tasks/done/388-….md, tasks/todo/999-probe-duplicate-id.md
doc-check: FAIL
```

**(a)다.** 프로브를 지우면 rc=0으로 돌아온다.

그러면 실제 충돌 때 왜 초록불이었나 — **두 카드가 한 트리에 같이 있던 적이 없어서다.**
충돌은 두 브랜치의 합집합에서만 존재했고, 각 체크아웃은 자기 쪽 383만 봤다. 체크아웃
단위 게이트로는 원리적으로 잡을 수 없다. 그래서 ISSUE-012의 열린 질문은 "검사를
추가한다"가 아니라 **"발급이 애초에 경쟁하지 않게 한다"**로 바뀐다 — 검사는 안전망이지
발급 규칙이 아니다.

## 2. 개명이 남긴 오조준은 존재 검사로 잡히지 않는다

383 충돌을 개명으로 풀자 옛 `TASK-383` 참조가 전부 무관한
`tasks/todo/383-diagnose-…md`로 재바인딩됐다. 깨진 링크가 아니라 **멀쩡히 풀리는 틀린
링크**다. 남은 인스턴스는 `tasks/done/386-…md`의 `:14`(프론트매터 안의 맨 `TASK-383`)와
`:74`(`[[TASK-383]]`) 둘이고, 그 카드는 봉인돼 통합됐으므로 고칠 수 없다
([[ISSUE-010]]의 인스턴스). 봉인 전이던 `tasks/plan/007`과 385 카드의 같은 오조준은
[[TASK-385]]에서 겨눴다.

이 사실을 ISSUE-012에 적지 않으면 "dangling wikilink 검사를 추가한다"는 criterion이
생긴다. 그 규칙은 **계기가 된 두 사례를 하나도 잡지 못한다.** 잡을 수 있는 것은
게이트가 아니라 개명 시점의 일괄 훑기 절차다.

## 3. pin 드리프트는 우연이 아니라 절차의 구조다

카드를 닫는 행위(=`quality-review` 3줄 추가 + `done/` 이동)가 리뷰 시점 pin을
무효화한다. 두 건 실측이 ISSUE-001의 새 절에 있다. `blocks:` 없는 카드에서는
validator가 검사에 도달하지 않아 조용할 뿐이다.

## Completion Criteria

- [x] ISSUE-012가 중복 id 검사의 실재와 커밋을 기록한다 | verify: `/usr/bin/grep -rq --include='012-*.md' 'checkDuplicateCardIDs' tasks`
- [x] ISSUE-012의 Resolution Criteria가 존재 검사가 아니라 발급 규칙과 개명 절차를 요구한다 | verify: `human — §Resolution Criteria를 읽고, dangling wikilink 검사를 해법으로 적은 항목이 없는지 확인`
- [x] ISSUE-012가 봉인돼 못 고치는 두 자리를 기록한다 | verify: `/usr/bin/grep -rq --include='012-*.md' 'ISSUE-010' tasks`
- [x] ISSUE-001이 pin 드리프트 실측 두 건을 기록한다 | verify: `/usr/bin/grep -rq --include='001-*.md' '3edee27d' tasks`
- [x] 문서 게이트가 통과한다 | verify: `make doc-check` (regression-guard)
