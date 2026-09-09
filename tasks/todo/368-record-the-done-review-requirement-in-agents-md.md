---
id: TASK-368
title: "Record the done-review requirement in AGENTS.md"
type: docs
priority: P3
effort: XS
exec-tier: cheap
status: todo
needs-human: true
created: 2026-09-09
source: "PLAN-007 §External이 '이 계획이 끝난 뒤' 하기로 적어 둔 DVA 쪽 절반. 소유 카드가 없었다"
---

## Summary

PLAN-007 §External 마지막 문단이 이렇게 적어 뒀다: "DVA 쪽 문서는 이 계획이 끝난 뒤
`AGENTS.md`에 '카드를 닫으려면 별도 세션의 done-review가 필요하다' 한 줄과 정책 링크만
둔다. 규칙 본문을 복제하지 않는다."

**그 한 줄은 없다.** `AGENTS.md`(291줄)에 `done-review`도 `quality-review`도 등장하지
않는다. 그리고 그 문장을 실행할 카드도 없었다 — PLAN-007이 할 일을 서술만 하고
소유자를 만들지 않았다.

## 왜 지금인가 — 강제가 없으니 문서가 유일한 표면이다

PLAN-007 §External 1번(`move --to done` 전제조건에 `quality-review` 요구)은 엔진 소유라
이 저장소가 진척을 강제할 수 없고, 그래서 카드를 만들지 않기로 했다. 그 결정은 여전히
옳다.

그 결과가 TASK-367이 측정한 것이다 — `tasks/done/` 19장, `quality-review` 0장. 기계
강제가 없는 동안 카드를 닫는 사람에게 요구를 전달하는 표면은 **문서 한 곳뿐**이고,
지금은 그 한 곳조차 비어 있다.

이 카드가 강제를 만들 수는 없다. 만들 수 있는 것은 **닫는 사람이 읽는 자리에 요구가
적혀 있는 상태**다. 그것이 XS이고 P3인 이유이며, 동시에 미루면 안 되는 이유다.

## 작업

`AGENTS.md`에 한 문단. 다음을 담는다:

- 카드를 `done`으로 옮기려면 구현 세션과 **다른 세션**의 done-review가 필요하다는 사실.
- `quality-review: pass | conditional | waived` 중 하나가 frontmatter에 남아야 하고,
  `waived`는 `quality-review-evidence`를 함께 요구한다는 사실(2026-09-09 `ce task archive
  --help`로 확인. PLAN-007 집필 시점에는 `waived`가 없었다).
- 규칙 본문이 아니라 canonical 위치로의 링크. 엔진 정책은 `ce-workbook/task_management`
  소유다.

**복제하지 않는다.** `CLAUDE.md` §Documentation이 요구하는 그대로 — canonical을 갱신하고
다른 문서에서는 요약과 링크만 쓴다. 여기서 규칙 전문을 다시 쓰면 엔진이 값을 하나 더
받는 날 두 문서가 갈라진다. `waived`가 정확히 그렇게 추가됐다.

## Completion Criteria

- [ ] AGENTS.md가 카드를 닫으려면 별도 세션의 done-review가 필요하다고 기술한다 | verify: `/usr/bin/grep -q "done-review" AGENTS.md`
- [ ] 규칙 본문을 복제하지 않고 링크만 둔다 | verify: human — 추가된 문단이 정책 규칙을 다시 서술하지 않고 canonical 위치를 가리키는지 확인
- [ ] 게이트 통과 | verify: `make doc-check`

## Notes

- TASK-367과 짝이지만 의존하지 않는다. 367은 이미 쌓인 19장을 처분하고, 이 카드는 다음
  19장이 같은 방식으로 쌓이지 않게 한다. 순서는 무관하고 둘 다 필요하다.
- 첫 수용기준이 `grep done-review`인 것은 약한 바인딩이다 — 단어가 있으면 통과한다.
  두 번째 사람 기준이 그 약함을 받는다. 더 강한 기계 기준을 쓰려면 문단의 형태를 미리
  고정해야 하는데, 그러면 이 카드가 금지하려는 복제를 카드 자신이 하게 된다.
