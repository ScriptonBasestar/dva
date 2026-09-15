---
id: ISSUE-013
title: "ce task move does not sync the frontmatter status field"
type: bug
status: todo
priority: P2
severity: medium
effort: S
ownership: upstream
created: 2026-09-14
discovered-in: "TASK-381 (check-plan-prose-against-plan-frontmatter)"
discovered-at: 2026-09-14
source: "TASK-381 종료 중 ce task move가 status: todo를 done으로 옮기며 frontmatter를 갱신하지 않는 것을 발견"
upstream-ref: ""
---

## Summary

`ce task move <card> done`은 카드 파일을 `tasks/done/`으로 옮기지만
**frontmatter의 `status:` 필드는 건드리지 않는다.** 이동이 결정한 값을 이동 자신이
쓰지 않는다.

`move`가 실제로 동기화하는 것은 카드 본문의 `**Status**` 마크다운 테이블 셀
하나뿐이다. 그런 셀이 없는 카드(DVA 카드 대부분이 이 경우다)에서는
`(no **Status** cell found to sync)`만 찍고 끝난다. `status:` 필드는 이동 전 값
그대로 남는다.

## Evidence

TASK-381 종료 중 실제로 관찰됨 (2026-09-14).

1. `ce task move 381-check-plan-prose-against-plan-frontmatter done` 실행 →
   파일이 `tasks/done/381-check-plan-prose-against-plan-frontmatter.md`로 이동,
   출력에 `(no **Status** cell found to sync)`.
2. 바로 다음 게이트인 `make doc-check` 실행 → exit 2:
   ```
   STATUS tasks/done/381-check-plan-prose-against-plan-frontmatter.md: zone tasks/done/ permits status: done, found "todo"
   ```
3. `status: todo`를 `status: done`으로 손으로 고친 뒤 재실행하니 통과.

DVA 카드 대부분은 `**Status**` 테이블 셀을 두지 않으므로 이것은 예외 경로가 아니라
정상 경로다 — 에이전트가 `move`로 카드를 닫을 때마다 빌드가 빨갛게 되고, 그 직후
사람이나 에이전트가 손으로 고쳐야 한다.

## Reproduction

1. `**Status**` 테이블 셀이 없는 `tasks/todo/` 카드를 하나 고른다.
2. `ce task move <card> done`을 실행한다 — 파일은 옮겨지고
   `(no **Status** cell found to sync)`가 출력된다.
3. 이동된 파일의 frontmatter `status:`를 확인한다 — 여전히 `todo`다.
4. `make doc-check`을 실행한다 — zone/status 불일치로 실패한다.

## Expected vs Actual

- Expected: `move`가 결정한 존이 곧 `status:`의 값이므로, 이동이 성공하면
  frontmatter의 `status:`도 그 존이 허용하는 값으로 갱신된다.
- Actual: 본문의 `**Status**` 테이블 셀만(있을 때만) 동기화되고, `status:` 필드는
  이동 전 값에 머문다.

## Impact

빌드를 조용히 깨뜨리지는 않는다 — `make doc-check`이 즉시 잡아낸다. 비용은
**모든** 에이전트 주도 카드 종료 뒤에 반복되는 수리 왕복이다: `move`가 성공을
보고하고, 다음 게이트가 실패하고, 누군가 frontmatter를 손으로 고친다. `move`가
방금 스스로 결정한 존을 스스로 쓰기만 하면 사라지는 왕복이다.

[[ISSUE-007]]과는 다른 결함이다. ISSUE-007은 `ce task validate`가 `status:` 값을
존과 대조하지 않는다는 것 — 검사가 없다는 결함이다. 이 카드는 `ce task move`가
값을 쓰지 않는다는 것 — 다른 커맨드, 다른 결함이다. 둘은 나쁘게 조합된다: mover가
틀린 값을 남기고 공유 validator는 그것을 알아채지 못하므로, 지금 이것을 잡아내는
것은 DVA 저장소 로컬의 `tools/doccheck` `checkCardStatus`뿐이다 — 그런데
ISSUE-007은 바로 그 검사가 잘못된 저장소에 있다고 주장한다.

소유는 외부다 — `ce task move`는 ce-workbook/task_management 소속이다.
[[ISSUE-004]] 선례에 따라 여기에 기록한다: DVA에서 관찰된 상류 런타임의 결함을
잃지 않기 위해 DVA 카드로 남기고, 수정은 상류에서 이루어진다.

## 소유권 — 상류다 (2026-09-15 명시)

`ce task move`는 `ce-workbook/task_management` 소속이다 — Impact가 "소유는 외부다 …
수정은 상류에서 이루어진다"로 적는다([[ISSUE-004]] 선례). 보고 자리는 [[TASK-395]]가
만든다.

## Resolution Criteria

- [ ] `ce task move`가 카드를 이동시킬 때 frontmatter `status:`를 도착 존이
      허용하는 값으로 갱신한다 | verify: human — upstream 테스트가 `**Status**`
      테이블 셀이 없는 카드에 대해서도 frontmatter 갱신을 확인한다
- [ ] DVA 저장소의 live 카드 중 `status: todo`인 채로 `tasks/done/`에 있는 카드가
      없다 (회귀 감시 — mover가 실제로 고쳐졌음을 증명하지는 못한다. `tools/doccheck`가
      이미 개별 카드 단위로 이를 막고 있으므로, 이 검사는 그 사이를 뚫고 들어온
      다건을 한 번에 잡는 보조 신호일 뿐이다) | verify: `! /usr/bin/grep -rq '^status: todo$' tasks/done`
- [ ] 이 카드가 닫히는 시점의 `ce task move` 실행 결과(또는 upstream 픽스 링크)가
      본 카드에 남는다 | verify: human — 수정 커밋 또는 재현 실패 로그가 이 카드에
      링크되었는지 확인

## Related

- [[TASK-381]] — 이 결함을 처음 밟은 작업.
- [[ISSUE-007]] — 같은 필드를 다루지만 다른 커맨드의 다른 결함 (validate가
  검사하지 않음 vs move가 쓰지 않음).
- [[ISSUE-004]] — 외부 소유 결함을 DVA 카드로 남기는 선례.
