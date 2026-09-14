---
id: TASK-390
title: "Delegate the unreviewed-done inventory to ce task lint and downgrade ISSUE-001"
type: docs
priority: P2
effort: S
exec-tier: standard
status: todo
created: 2026-09-14
source: "PLAN-007의 2026-09-14 재고표는 master 8b56802(done 70장 / 미리뷰 39장)에 묶여 있고 6fd352a(74장 / 37장)에서 이미 낡았다. 같은 수를 ce task lint가 매 실행마다 센다"
depends-on: []
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
**P2로 내리되 닫지 않는다** — PLAN-007 §External이 정한 대로, 진척을 강제할 수 없는
항목을 닫는 것과 낮추는 것은 다르다.

## 이 카드가 하지 않는 것

- 미리뷰 37장을 소급 리뷰하지 않는다. TASK-385가 이미 정한 방침이고 그대로다.
- ISSUE-001을 닫지 않는다. criterion 1·2는 상류가 소유한다.
- `ce task lint` 출력을 저장소 문서에 복사해 두지 않는다 — 그렇게 하는 순간
  이 카드가 고치는 것과 같은 손유지 스냅샷이 하나 더 생긴다.

## Completion Criteria

- [x] PLAN-007의 재고표가 자기 수치를 `8b56802` 스냅샷으로 한정하고 현재 값의 출처로 `ce task lint`를 지목한다 | verify: `/usr/bin/grep -rq --include='007-done-backlog-triage.md' 'ce task lint' tasks`
- [x] ISSUE-001의 어느 완료 기준 줄도 controller 발급을 주장하지 않는다 | verify: `! /usr/bin/grep -rqE --include='001-task-runtime-cannot-review-legacy-done-cards-without-verification-evidence.md' '^- \[.\].*controller-produced' tasks`
- [x] 철회된 문구와 실제 발급 주체가 근거로 남는다 — 지운 자리를 비워 두지 않는다 | verify: `/usr/bin/grep -rq --include='001-task-runtime-cannot-review-legacy-done-cards-without-verification-evidence.md' '독립 리뷰어와 저자가 손으로' tasks`
- [x] ISSUE-001이 P2이고 여전히 열려 있다 | verify: `/usr/bin/grep -rq --include='001-task-runtime-cannot-review-legacy-done-cards-without-verification-evidence.md' '^priority: P2' tasks`
- [x] 보드 게이트가 통과한다 | verify: `ce task gate`
- [x] 문서 게이트가 통과한다 | verify: `make doc-check` (regression-guard)
