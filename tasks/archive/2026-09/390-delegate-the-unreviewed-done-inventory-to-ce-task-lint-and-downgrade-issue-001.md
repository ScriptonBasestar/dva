---
id: TASK-390
title: "Delegate the unreviewed-done inventory to ce task lint and downgrade ISSUE-001"
type: docs
priority: P2
effort: S
exec-tier: standard
status: done
archived-at: 2026-09-17
verified-at: 2026-09-17
verification-summary: "Re-verified 2026-09-17. PLAN-007 names ce task lint as count source and 8b56802 snapshot scope; ISSUE-001 carries no controller-produced claim, records hand authorship, priority P2; ce task gate READY."
created: 2026-09-14
source: "PLAN-007의 2026-09-14 재고표는 master 8b56802(done 70장 / 미리뷰 39장)에 묶여 있고 6fd352a(74장 / 37장)에서 이미 낡았다. 같은 수를 ce task lint가 매 실행마다 센다"
depends-on: []
completion-summary: "손으로 유지되던 미리뷰-done 재고표를 `ce task lint`에 위임하고(수치는 게이트가 세고 PLAN-007은 배분만 소유한다), ISSUE-001을 P0에서 P2로 내리되 닫지 않았다. ISSUE-001 criterion 4의 'controller-produced receipts' 주장은 철회하고 실제 발급 주체(TASK-384의 독립 리뷰어와 저자)를 근거로 남겼다"
verification-status: verified
verification-evidence: "완료 기준 7개의 verify 바인딩 전부 bare 실행 exit 0. ce task gate 0 (READY — task_board_ready), make doc-check 0, make build/test/lint 0. 재고 수치는 리뷰어가 카드의 awk로 blob에서 독립 재도출해 8b56802=70/39, 6fd352a=74/37로 일치"
quality-review: conditional
quality-reviewed-at: 2026-09-14
quality-review-evidence: "독립 리뷰 review-390(Claude Opus 5, 저자 아님)이 5889fc3을 판정했다. 모든 수치·인용·종료코드가 재현됐고 지적 넷은 전부 산문 귀속 결함이었다(§리뷰 지적 넷). 넷 다 같은 브랜치에서 접었고 조건은 충족됐다 — verdict 값은 리뷰어가 낸 그대로 남긴다. 리뷰어는 tasks/done/344·371에 임시 한 줄을 넣어 digest 핀 강제를 실증한 뒤 되돌렸다(트리 clean)"
quality-review-receipt: tasks/done/evidence/TASK-390/done-review-9beb5af3bfc17605bd2337a42a8b72cd0b8d645e0257fa896eaff8a4876287d7.json
---

## Summary

기록 둘이 사실과 어긋난다. 둘 다 문서이고, 둘 다 **손으로 유지되는 수치**가 원인이다.

**1. PLAN-007의 재고표는 태어난 순간부터 낡는다.** `### 2026-09-14 재측정`은
master `8b56802` 기준 done 70장 / `quality-review` 없는 카드 39장을 표로 적고,
그 아래에 그 수를 내는 `awk` 한 줄을 적어 둔다. 표 자체가 "기계가 아니라 사람의
재측정으로만 유지된다"고 쓰고 있는데, 이 카드를 여는 시점의 master `6fd352a`에서는
이미 **74장 / 37장**이다. 나흘이 아니라 같은 날 안에서 벌어진 드리프트다.

그런데 그 수를 세는 기계가 이미 있다. `ce task lint`가 매 실행마다 낸다:

```
UNFINALIZED DONE (37 card(s) in done/ with no quality-review verdict)
  tasks/done/... no quality-review verdict
  → review each, then stamp quality-review: pass|conditional|waived
```

그리고 `ce task lint`는 `ce task gate`의 세 단계 중 하나다(validate + lint +
preflight). 즉 이 수는 **통합 게이트가 돌 때마다 자동으로 재측정된다.** PLAN-007이
손으로 적은 표는 그 출력의 열등한 스냅샷이다.

표를 지우지는 않는다 — 2026-09-14의 소유권 배분(어느 plan이 몇 장을 갖는지)은
`ce task lint`가 내지 않는 정보이고, 그것이 PLAN-007이 실제로 소유한 판단이다.
**수는 게이트에 위임하고 배분만 남긴다.** 표에는 그것이 `8b56802`의 스냅샷이며
현재 값은 `ce task lint`가 낸다고 명시한다.

**2. ISSUE-001의 criterion 4가 오늘 참이 아닌 것을 주장한다.** 문장은
"`TASK-344` and `TASK-371` carry **genuine controller-produced** review receipts"인데,
바인딩은 `ce task gate --json` 하나다. 오늘 그 바인딩은 rc=0이고 두 카드는 실제로
`tasks/receipts/` 아래 CE 정규 digest로 고정된 receipt를 갖는다 — 그러나 그 receipt는
**controller가 발급한 것이 아니라 독립 리뷰어와 저자가 손으로 쓴 것이다**(TASK-384).
바인딩이 통과하는 동안 산문이 거짓이 되는 형태이고, 이것은 이 저장소가 TASK-371과
TASK-344에서 방금 철회시킨 바로 그 결함 — 근거 없는 주장 — 과 같은 계열이다.

문장을 사실로 고친다. controller 발급은 criterion 1·2가 소유한 상류 작업으로 남는다.

**3. 그래서 ISSUE-001은 P0이 아니다.** P0의 근거였던 것 — DVA 안에서 receipt를 만들
수 없다 — 은 두 번 반증됐다. 정규 digest는 저장소 안에서 probe로 얻을 수 있고
(TASK-384), durable 경로는 TASK-388이 닫았다(criterion 3, 이미 `[x]`). 남은 둘은
ce-agent-kit / ce-workbook 쪽 상류 작업이라 이 보드가 진척을 강제할 수 없다.
**P2로 내리되 닫지 않는다.** 진척을 강제할 수 없다는 것은 PLAN-007 §External이 그쪽
두 항목에 대해 이미 적은 사실이지만, 닫기와 낮추기를 가르는 규칙은 그 절에 없다 —
그 선택은 ISSUE-001이 자기 §Priority에서 근거와 함께 내린다.

## 이 카드가 하지 않는 것

- 미리뷰 37장을 소급 리뷰하지 않는다. TASK-385가 이미 정한 방침이고 그대로다.
- ISSUE-001을 닫지 않는다. criterion 1·2는 상류가 소유한다.
- `ce task lint` 출력을 저장소 문서에 복사해 두지 않는다 — 그렇게 하는 순간
  이 카드가 고치는 것과 같은 손유지 스냅샷이 하나 더 생긴다.

## 리뷰 지적 넷 (conditional — 같은 브랜치에서 접었다)

독립 리뷰 review-390이 `5889fc3`을 판정했다. 수치·인용·종료코드는 전부 재현됐고,
결함은 **산문의 귀속**과 이 커밋이 지운 한 문장에 있었다. 넷 다 여기서 고쳤다.

- **F1 (medium) — §External에 없는 규칙을 §External에 귀속했다.** "PLAN-007 §External이
  정한 대로 그런 항목은 닫지 않고 낮춘다"고 적었으나 그 절은 닫기와 낮추기를 가르지
  않는다. 실제로 적힌 것은 반대 방향의 조치다 — "둘 다 카드가 없고, 만들지 않는다."
  뒷받침되는 절반(이 보드가 상류 진척을 강제할 수 없다)만 남기고 귀속을 뗐다.
  낮추고 열어 두는 선택은 ISSUE-001이 자기 §Priority에서 내린다.
- **F2 (medium) — 위 절의 재고를 낡음 표시 없이 남겼다.** §2026-09-10은 지금도 현재형으로
  "`tasks/done/`은 **46장**"이라 적는데, 그 수치를 무효화하던 한 문장을 이 커밋이 지웠다.
  그 절의 자기 한정 문장은 19장을 한정할 뿐 46/9를 한정하지 않는다. 보드가 이미 쓰는
  `> **낡음 (날짜)**` 형식으로 표시를 붙였다.
- **F3 (low) — 이름을 바꾼 절을 옛 이름으로 가리켰다.** `§P0 Blocker`는 이 커밋이
  `## Priority — P0에서 P2로 (2026-09-14)`로 바꾼 제목이다. `make doc-check`는 절 참조를
  검사하지 않으므로 기계가 잡지 못한다. 옛 이름을 괄호로 병기해 고쳤다.
  (`tasks/receipts/TASK-388/`의 두 건은 digest 핀 아래라 건드리지 않는다 — 고치면
  보드가 빨개진다.)
- **F4 (low) — 완료 기준 1이 두 가지를 주장하고 하나만 묶었다.** 바인딩은 `ce task lint`
  리터럴만 봐서, 스냅샷 한정이 지워져도 통과한다. 주장 하나에 바인딩 하나가 되도록
  기준을 둘로 쪼갰다. 기준 2의 같은 틈(문구를 바꾼 주장은 통과한다)은 남긴다 —
  리터럴을 늘리면 그때마다 또 우회 가능하고, 그 계열은 [[TASK-350]]이 소유한다.

리뷰어가 `tasks/done/344`·`371`에 한 줄을 덧붙여 digest 핀이 실제로 강제되는지 확인한 뒤
되돌렸다. 두 카드 모두 불일치 메시지를 냈고, 원상 복구 후 다시 통과한다 — 핀이 CE의
정규 digest와 실제로 일치한다는 증거다.

## Completion Criteria

- [x] PLAN-007의 재고표가 현재 값의 출처로 `ce task lint`를 지목한다 | verify: `/usr/bin/grep -rq --include='007-done-backlog-triage.md' 'ce task lint' tasks`
- [x] 그 표가 자기 수치를 `8b56802` 스냅샷으로 한정한다 — 수치만 남고 한정이 사라지면 안 된다 | verify: `/usr/bin/grep -rq --include='007-done-backlog-triage.md' '8b56802.*스냅샷' tasks`
- [x] ISSUE-001의 어느 완료 기준 줄도 controller 발급을 주장하지 않는다 | verify: `! /usr/bin/grep -rqE --include='001-task-runtime-cannot-review-legacy-done-cards-without-verification-evidence.md' '^- \[.\].*controller-produced' tasks`
- [x] 철회된 문구와 실제 발급 주체가 근거로 남는다 — 지운 자리를 비워 두지 않는다 | verify: `/usr/bin/grep -rq --include='001-task-runtime-cannot-review-legacy-done-cards-without-verification-evidence.md' '독립 리뷰어와 저자가 손으로' tasks`
- [x] ISSUE-001이 P2이고 여전히 열려 있다 | verify: `/usr/bin/grep -rq --include='001-task-runtime-cannot-review-legacy-done-cards-without-verification-evidence.md' '^priority: P2' tasks`
- [x] 보드 게이트가 통과한다 | verify: `ce task gate`
- [x] 문서 게이트가 통과한다 | verify: `make doc-check` (regression-guard)
