---
id: TASK-384
archived-at: 2026-09-17
verified-at: 2026-09-17
verification-summary: |
  Receipt pointers for TASK-344/371 present (grep rc 0); receipts reviewed-at
  2026-09-14, not backdated; ce task validate --all 86 valid/0 invalid;
  ce task gate READY (validate/lint/preflight pass); make doc-check exit 0.
title: "Re-review TASK-344 and TASK-371 independently and issue their receipts"
type: chore
priority: P1
effort: M
exec-tier: strong
status: done
created: 2026-09-14
source: "TASK-388이 durable 경로를 열자 보드의 남은 실패 2건의 사유가 경로가 아니라 receipt 부재로 좁혀졌다"
depends-on: [TASK-388]
completion-summary: "TASK-344와 TASK-371 두 카드를 오늘 날짜의 독립 리뷰로 재검토하고 각각 추적 경로에 receipt를 발급했다. 리뷰어가 두 카드 모두에서 같은 결함을 찾았다 — 두 카드의 verification-evidence가 산출물 없는 소급 독립 리뷰를 주장하고 있었고, 봉인 전에 둘 다 철회했다. 그 결과 ce task validate --all이 94 valid / 0 invalid, ce task gate가 READY로 처음 초록이 됐다"
verification-status: verified
verification-evidence: "완료 기준 5개: grep 바인딩 둘 rc=0, ce task validate --all rc=0(94 valid / 0 invalid), ce task gate rc=0(validate·lint·preflight 셋 다 pass, READY — task_board_ready). 소급 아님 확인은 커밋 토폴로지로 했다 — TASK-371은 36594bb2(12:33:49)와 5fc87d79(12:49:49)가 16분 간격 동일 저자이고 tasks/done/evidence/TASK-371이 없었다. make doc-check 0, make lint 0"
quality-review: waived
quality-reviewed-at: 2026-09-14
quality-review-evidence: "이 카드 자체는 독립 리뷰를 받지 않았고 waived로 정직하게 남긴다. 이 카드의 산출물은 receipt 둘이며 그 둘은 각각 독립 리뷰어(review-344, review-371, 둘 다 저자 아님)가 판정해 발급했다 — 즉 실질 내용은 리뷰됐고 리뷰되지 않은 것은 이 카드의 닫는 기록뿐이다. 여기에 pass를 찍으면 이 카드가 두 카드에서 철회시킨 바로 그 주장(근거 없는 독립 리뷰 주장)을 스스로 반복하게 된다"
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

## 결과 (2026-09-14)

두 카드 모두 receipt를 얻었고 보드가 처음으로 초록이 됐다.

| 카드 | 판정 | pin 종류 | receipt |
|---|---|---|---|
| TASK-344 | `pass` | CE canonical digest | `29e2ab55…` |
| TASK-371 | `conditional` | CE canonical digest | `1b0bc79f…` |

둘 다 canonical digest다 — 두 카드가 `blocks:`를 선언하므로 validator가 그 값을 실제로
비교한다. `blocks:` 없는 카드에 붙는 plain sha256 pin([[ISSUE-012]])과 달리 이 둘은
검사에 닿는다.

**두 리뷰어가 독립적으로 같은 결함을 찾았다.** 두 카드의 `verification-evidence`가
모두 산출물 없는 소급 독립 리뷰를 주장하고 있었다. 344는 "independent parser review",
371은 "independent review confirmed shared truncation contract"다. 어느 쪽도 receipt가
없고, 371은 카드를 연 커밋과 닫은 커밋이 16분 간격 동일 저자이며 그 문장이 구현자
자신의 닫는 커밋에서 추가됐다.

이 카드의 §하지 않을 것이 금지한 것은 "리뷰가 있었던 것처럼 receipt를 만드는 일"인데,
문장을 남긴 채 receipt를 붙이면 **새 receipt가 그 주장을 추인하는 모양**이 되어 같은
금지를 우회로 어기게 된다. 그래서 철회와 receipt를 같은 커밋에 실었다 — 봉인 이후에는
그 문장을 더 이상 고칠 수 없기 때문이다([[ISSUE-010]]).

## Completion Criteria

- [x] TASK-344가 오늘 날짜의 독립 리뷰 판정을 담은 추적 receipt를 가리킨다 | verify: `/usr/bin/grep -rq --include='344-*.md' 'quality-review-receipt: tasks/done/evidence/' tasks`
- [x] TASK-371도 같다 | verify: `/usr/bin/grep -rq --include='371-*.md' 'quality-review-receipt: tasks/done/evidence/' tasks`
- [x] 두 receipt의 `reviewed-at`이 소급 날짜가 아니다 | verify: `human — 리뷰어 세션 기록과 대조`
- [x] 보드가 통과한다 | verify: `ce task validate --all`
- [x] 공유 게이트가 초록이다 | verify: `ce task gate`
