---
id: TASK-354
title: "Make the board pass ce task validate, then gate it so it cannot drift again"
type: chore
priority: P3
effort: S
exec-tier: cheap
status: todo
created: 2026-09-08
source: "TASK-352·353를 파일하다 `ce task validate`가 두 카드를 거부해 발견. 전수 측정으로 확대"
---

## Summary

`ce task validate`가 이 저장소 카드 **17개**를 거부한다. 그런데 저장소가 실제로 도는
게이트(`make doc-check`, `ce task preflight`)는 전부 통과다 — validate를 아무도 호출하지
않기 때문이다. 검사기가 있는데 연결돼 있지 않고, 그래서 아무도 모르는 사이에 벌어졌다.

측정: master `6db68de`, 2026-09-08, `find tasks/{todo,done,plan} -name '*.md' -exec ce task
validate {} \;`.

| 실패 사유 | 파일 수 | 대상 |
|---|---|---|
| `Missing Summary section` | 11 | todo 318·319·321·323, done 312·313·314·317·320·322·324 |
| `Invalid type "fix"` | 3 | todo 344·345·350 |
| `Invalid type "decision"` | 1 | done 334 |
| `Missing Completion Criteria` + `No criteria defined` | 2 | todo 350·351 |
| `Missing Children section` | 1 | plan 006 |

## 어느 쪽이 옳은가 — 추측이 아니라 셈으로

수정 방향을 정하려면 "validate가 맞나 보드가 맞나"를 알아야 한다. 보드 자신의 관례를 셌다
(todo+done 47장):

- `## Completion Criteria` **45장** vs `## Acceptance Criteria` **2장**
- `## Summary` 보유 36장 / 미보유 11장
- `type: bug` 10장 vs `type: fix` 3장

**45 대 2다.** 지배적 관례가 이미 validate와 같은 편이고, 어긋난 2장은 *이 세션이 오늘
파일한 TASK-350·351* 이다. 즉 드리프트가 이론이 아니라 실시간으로 유입되는 중이고,
그것이 이 카드가 게이트를 요구하는 이유다. 관례를 바꾸는 작업이 아니라 소수 이탈을
다수에 맞추는 작업이다.

`Missing Summary` 11장만 판단이 다르다 — 이들은 헤딩 없이 본문 산문으로 시작한다.
36 대 11이라 역시 다수가 validate 편이지만, 내용 변경 없이 기존 첫 문단 위에 `## Summary`
한 줄을 얹는 것으로 끝난다.

## 엔진 자기모순 — 기록만, 이 카드의 작업 대상 아님

같은 `ce` 바이너리의 두 하위 명령이 **같은 파일에 대해 정반대**를 말한다:

```
ce task validate tasks/todo/350-…md   → ❌ No criteria defined under Completion Criteria
ce task preflight --zone todo --json  → 350: {criteria: 6, bound: 6, runnable: true}
```

TASK-351도 같다(validate: 0개 / preflight: 4/4). TASK-321은 validate가
`Missing Summary`로 거부하는데 preflight는 `2/2 bound, runnable`이다. preflight는
`## Acceptance Criteria` 아래 기준을 정상 인식하고 validate는 못 한다.

이건 `ce-workbook/task_management` 엔진 소유이고 **여기에 카드를 만들지 않는다** —
PLAN-007 §External이 세운 선례 그대로다(이 저장소의 보드가 진척을 강제할 수 없는 항목은
영원히 열린 채 남는다). 이 카드는 DVA가 소유한 절반, 즉 보드 정규화와 게이트 연결만 한다.
엔진 쪽은 그 저장소에 보고한다.

## 작업

1. `type: fix` 3장 → `bug`. `type: decision` 1장(done 334) → 엔진이 받는 값으로. 334는
   아카이브 대상이므로 `ce task archive`가 거부하지 않는지 먼저 확인한다.
2. TASK-350·351의 `## Acceptance Criteria` → `## Completion Criteria`. 두 줄.
3. 11장에 `## Summary` 헤딩 추가. 본문은 건드리지 않는다 — 기존 첫 문단이 곧 요약이다.
4. PLAN-006에 `## Children` 절 추가 (PLAN-007에 이미 있는 형태를 따른다).
5. `make doc-check`에 전수 validate를 연결한다. 이것이 없으면 1~4는 다음 카드에서 다시
   무너진다 — 오늘 350·351이 그렇게 들어왔다.

`done/`·`plan/` 파일도 고치므로 통합 직전 origin/master로 rebase한다(PLAN-007 §Rules).

## Completion Criteria

- [ ] todo·done·plan 전 카드가 `ce task validate`를 통과한다 | verify: `! /usr/bin/find tasks/todo tasks/done tasks/plan -name '*.md' -exec ce task validate {} \; 2>&1 | /usr/bin/grep -q '❌'`
- [ ] `type: fix`와 `type: decision`이 보드에서 사라진다 | verify: `! /usr/bin/grep -rqE '^type: (fix|decision)$' tasks/todo tasks/done tasks/plan`
- [ ] `## Acceptance Criteria`가 보드에서 사라진다 | verify: `! /usr/bin/grep -rq '^## Acceptance Criteria$' tasks/todo tasks/done tasks/plan`
- [ ] doc-check가 카드 스키마를 검사하며, 위반 카드를 심으면 실패한다 | verify: `human — 위반 카드 하나를 심고 make doc-check가 그 경로를 지목하며 rc≠0인 것을 보인 기록이 카드에 있다`
- [ ] 게이트 통과 | verify: `make doc-check`

## Notes

- 오늘 노출은 0이다. `/task:run` Step 0은 preflight를 쓰고 `make doc-check`는 validate를
  부르지 않는다. 위험은 `ce task archive`처럼 validate 계열 검사를 쓰는 경로가 늘어날 때다.
- 우선순위를 P3으로 둔 이유가 그것이다. 다만 작업량이 작고(대부분 헤딩 한 줄) 5번 게이트가
  재발을 영구히 막으므로 비용 대비 이득은 좋다.
- 5번을 먼저 하면 1~4가 끝날 때까지 `make doc-check`가 빨간불이다. 1~4 → 5 순서를 지킨다.
