---
id: TASK-384
title: "Re-review TASK-344 and TASK-371 independently and issue their receipts"
type: chore
priority: P1
effort: M
exec-tier: strong
status: todo
created: 2026-09-14
source: "TASK-388이 durable 경로를 열자 보드의 남은 실패 2건의 사유가 경로가 아니라 receipt 부재로 좁혀졌다"
depends-on: [TASK-388]
---

## Summary

`ce task validate --all`의 남은 실패는 둘이고 사유가 같다.

```
tasks/done/344-harden-frontmatter-value-parsing-against-comments-and-repeats.md
  ❌ Done card blocks TASK-343 but declares no quality-review-receipt
tasks/done/371-align-planprogress-with-the-shared-progress-contract.md
  ❌ Done card blocks TASK-354 but declares no quality-review-receipt
```

[[TASK-388]] 이전에는 이 둘을 "durable 경로가 없어서 못 고친다"고 읽을 수 있었다. 이제
경로는 있다. 남은 것은 **검토 자체**다. 두 카드는 2026-09-10에 닫혔고 독립 리뷰 기록이
없다. 각 카드의 `verification-evidence`는 산문이며 파일을 가리키지 않는다.

## 하지 않을 것

**과거에 리뷰가 있었던 것처럼 receipt를 만들지 않는다.** validator가 기대 digest를 에러
메시지에 출력하므로 기계적으로는 지금 당장 통과시킬 수 있다. digest는 "이 카드 내용에
대해 검토했다"를 고정할 뿐 검토가 실재했는지는 말하지 않는다 — 그래서 통과시키는 것과
정직한 것은 별개다. 이 카드가 발급하는 receipt의 `reviewed-at`은 **오늘**이고
`reviewer`는 오늘 실제로 돈 리뷰어다. 2026-09-10을 소급해 적지 않는다.

## 범위

두 카드가 실제로 바꾼 코드를 지금 상태에서 재검토한다.

- **TASK-344** — 프론트매터 값 파서를 트레일링 주석·중복 키에 강화. `tools/doccheck`.
  후속 TASK-343(파일명 번호 충돌 가드)을 풀었다.
- **TASK-371** — `tools/planprogress`를 shared task progress contract와 맞춤. 후속
  TASK-354(보드 게이트 연결)를 푼다.

리뷰어는 이 카드를 여는 세션과 달라야 한다. 각 카드의 완료 기준 바인딩을 위임 요약이
아니라 직접 재실행하고, 결과를 receipt의 `criteria-results`에 남긴다.

## 이것이 닫히면

`ce task validate --all`이 0 invalid가 되고 `ce task gate`가 처음으로 초록이 된다.
그 다음이 [[TASK-354]]다 — 게이트를 `branch.readiness`에 연결하는 결정은
[[TASK-377]]이 이미 내려 두었고 남은 것은 `.gz-git.yaml` 3줄과 대화형 bootstrap
1회다. 즉 이 카드는 [[ISSUE-001]]의 P0를 걷어내는 마지막 장이다.

## Completion Criteria

- [ ] TASK-344가 오늘 날짜의 독립 리뷰 판정을 담은 추적 receipt를 가리킨다 | verify: `/usr/bin/grep -rq --include='344-*.md' 'quality-review-receipt: tasks/receipts/' tasks`
- [ ] TASK-371도 같다 | verify: `/usr/bin/grep -rq --include='371-*.md' 'quality-review-receipt: tasks/receipts/' tasks`
- [ ] 두 receipt의 `reviewed-at`이 소급 날짜가 아니다 | verify: `human — 리뷰어 세션 기록과 대조`
- [ ] 보드가 통과한다 | verify: `ce task validate --all`
- [ ] 공유 게이트가 초록이다 | verify: `ce task gate`
