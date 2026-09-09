---
id: TASK-354
title: "Make the board pass ce task validate, then gate it so it cannot drift again"
type: chore
priority: P2
effort: M
exec-tier: standard
status: todo
created: 2026-09-08
source: "TASK-352·353를 파일하다 `ce task validate`가 두 카드를 거부해 발견. 전수 측정으로 확대. 2026-09-09 재측정에서 노출이 0이 아님이 드러나 P3→P2, S→M"
needs-human: true
---

## Summary

`ce task validate`가 이 저장소 카드 **26장**을 거부한다(58장 중, 오류 인스턴스 32건).
그리고 그 26장 때문에 **`ce task gate`가 오늘 `not-ready`로 exit 1 한다** —
`{"status":"not-ready","summary":"task_validate_failed","failed_step":"validate"}`.

`ce task gate`는 보드 전체에 대한 단일 판정(validate + lint + preflight)이고, 개인 정책이
"보드 판정은 통합 러너가 공유 `ce task gate`에 위임한다"고 지정한 바로 그 명령이다.
저장소가 실제로 도는 게이트(`make doc-check`, `ce task preflight`)는 전부 통과다 —
둘 다 validate를 부르지 않기 때문이다.

측정: 2026-09-09, 이 브랜치.
`for f in tasks/{todo,done,plan}/*.md; do ce task validate "$f"; done` 전수 + `ce task gate --json`.

| 실패 사유 | 인스턴스 | 분포 |
|---|---|---|
| `Missing Summary section` | 20 | todo 11 · done 9 |
| `Invalid type "fix" for task` | 10 | todo 10 (전부) |
| `Invalid type "decision" for task` | 1 | done 334 |
| `Missing Children section` | 1 | plan 006 |

**2026-09-08 판 대비 무엇이 바뀌었나** (이 카드 자신의 옛 표는 폐기한다):

- 실패 파일 17 → **26**. 늘어난 것은 이 저장소가 그 사이 카드를 9장 더 파일했기 때문이다.
  드리프트가 멈춘 적이 없다는 뜻이고, 이 카드가 게이트를 요구하는 이유 그 자체다.
- `type: fix` 3 → **10**. 아래 §어느 쪽이 옳은가 참조 — 이 숫자는 원래 논거를 뒤집는다.
- `Missing Completion Criteria` 2 → **0**. TASK-350·351이 정규화됐다.
- `Missing Summary` 11 → 20이지만 **todo 쪽은 11로 같다**. done 쪽 9장이 새로 보인 것은
  당시 표가 done을 부분만 세었기 때문이다.

## 어느 쪽이 옳은가 — 셈이 두 항목에서 서로 다른 답을 낸다

이 카드의 2026-09-08 판은 "45 대 2, 소수 이탈을 다수에 맞춘다"는 하나의 논거로 세 항목을
한꺼번에 정당화했다. 재측정 결과 **그 논거는 항목마다 다르게 성립한다.** 그대로 두면 틀린
이유로 옳은 일을 하게 되므로 항목별로 분리해 다시 쓴다.

**1. `## Acceptance Criteria` — 이미 끝났다.** 보드에 0장, `## Completion Criteria` 55장.
당시 어긋나 있던 TASK-350·351은 그 뒤 정규화됐다. 아래 수용기준에 남기되
`(regression-guard)`로 표시한다 — 지금 통과하는 것이 확인이 아니라 재발 방지라는 뜻이다.

**2. `type: fix` vs `bug` — 다수결 논거가 뒤집혔다.** 살아 있는 zone만 세면
`fix` **10** 대 `bug` **8**이고, `fix` 10장은 **전부 todo**다. 즉 최근에 쓴 결함 카드는
예외 없이 `fix`를 골랐고 `bug`는 닫힌 카드 쪽에만 남아 있다. 살아 있는 관례는 `fix`다.

그런데 `_archive`까지 포함하면 `bug`가 **128장**이다. 다수결로는 어느 쪽도 결론이 아니다 —
"최근 관례" 대 "역사적 코퍼스"로 답이 갈린다.

그래서 이 항목의 근거는 다수결이 아니라 **스키마 권한**이다. `type`의 허용값은
`ce-workbook/task_management` 엔진이 소유하고, 보드는 그 어휘를 포크할 권한이 없다.
엔진이 `fix`를 거부하는 한 `fix`로 쓴 카드는 `ce task gate`를 영구히 빨간불로 만든다.
128 대 10은 그저 `bug`로 옮기는 쪽이 되돌리기 쉽다는 부수적 사실이다.

**어휘 자체에 이의가 있다면 그것은 엔진 저장소에 낼 안건이지 이 카드가 아니다.**
`fix`가 더 나은 이름이라고 판단되면 엔진 스키마에 `fix`를 추가하는 것이 옳은 순서이고,
그때까지 보드는 엔진이 받는 값을 쓴다. 이 카드는 후자만 한다.

**3. `Missing Summary` 20장 — 판단이 셋 중 가장 얕다.** 헤딩 없이 산문으로 시작할 뿐
첫 문단이 이미 요약이다. 내용 변경 없이 `## Summary` 한 줄을 얹는다.

## PLAN-008과 겹친다 — 순서가 있다

거부되는 26장 중 **8장이 PLAN-008의 자식**(TASK-358·359·360·361·362·363·364·365)이다.
`type: fix` 10장 중 6장, `Missing Summary` todo 11장 중 8장이 그 묶음이다.

PLAN-008은 그 여덟 장을 **한 번에 한 장씩 직렬로** 연다. 이 카드의 frontmatter 일괄 수정이
그 사이에 끼면 각 카드가 열릴 때마다 rebase 충돌 표면이 생긴다. 반대로 **PLAN-008이
1번 카드를 열기 전에 이 sweep을 끝내면 충돌 표면은 0이다** — 이 카드는 frontmatter와
헤딩만 만지고 PLAN-008은 본문과 소스를 만진다.

**결론: 이 카드를 PLAN-008 착수보다 먼저 넣는다.** P3이었다면 이 순서가 성립하지 않았을
것이고, 그것이 P2로 올린 두 번째 이유다(첫째는 `ce task gate` 실패).

## 작업

1. `type: fix` **10장** → `bug` (344·345·350·355·358·359·360·361·363·364).
   `type: decision` 1장(done 334) → 엔진이 받는 값으로. 334는 아카이브 대상이므로
   `ce task archive`가 거부하지 않는지 먼저 확인한다.
2. `Missing Summary` **20장**에 `## Summary` 헤딩 추가. 본문은 건드리지 않는다 —
   기존 첫 문단이 곧 요약이다.
3. PLAN-006에 `## Children` 절 추가. PLAN-008이 2026-09-09에 같은 결함으로 걸렸고
   `260ed63`이 고쳤다 — **그 커밋의 형태를 그대로 따른다**. frontmatter `children:`는
   planprogress가 세고 `## Children` 절은 validate가 요구한다. 둘 다 있어야 한다.
4. `## Acceptance Criteria`: 할 일 없음. 이미 0장.

## 게이트 연결 — 검사기를 새로 쓰지 않는다

2026-09-08 판의 5번은 "`make doc-check`에 전수 validate를 연결한다"였다. **그대로 하면 안
된다.** 개인 정책이 명시적으로 금지한다 — 보드 판정을 저장소마다 다시 구현하면 하나를
고치는 순간 갈라지고, 공유 게이트는 정확히 그것을 막으려고 존재한다. `tools/` 밑에
validate를 재구현하는 Go 도구를 만드는 것은 이 카드의 범위가 아니라 이 카드가 피해야 할
일이다.

옳은 형태는 **공유 게이트를 부르는 것**이다: `ce task gate`가 판정하고 저장소는 그 종료
코드를 그대로 쓴다.

붙일 자리에 결정이 하나 남아 있고, **이것이 이 카드의 `needs-human` 항목이다**:

- **`make doc-check`에 붙인다** — 가장 단순하고 지금 도는 게이트에 즉시 들어간다.
  대가: doc-check가 `ce` 바이너리 존재에 의존하게 되고, CI 러너에 `ce`가 없으면
  이 저장소의 문서 게이트 전체가 `ce` 설치 여부에 묶인다. **붙이기 전에 CI에서 `ce`가
  해결되는지 확인할 것.**
- **통합 러너에 붙인다** — 정책이 지정한 자리다. 그런데 이 저장소는 통합 도구에 readiness
  러너를 선언한 적이 없다(`.gz-git.yaml` 부재, Makefile에 해당 타깃 없음). 그 선언을
  만드는 것은 이 카드보다 큰 작업이다.

두 번째가 정본이지만 첫 번째가 오늘 실행 가능하다. **사람이 고르고 근거를 아래
`## 결정 기록`에 남긴다.** 고르지 않은 쪽은 별도 카드로 남긴다.

## 결정 기록

<!-- 게이트를 어디에 붙였는지와 그 근거를 여기에 적는다. CI에서 `ce`가 해결되는지
     확인한 결과도 함께 남긴다. -->

## Completion Criteria

- [ ] `ce task gate`가 보드 전체에 대해 ready로 종료한다 | verify: `ce task gate`
- [ ] `type: fix`와 `type: decision`이 보드에서 사라진다 | verify: `! /usr/bin/grep -rqE '^type: (fix|decision)$' tasks/todo tasks/done tasks/plan`
- [ ] `## Acceptance Criteria`가 보드에서 사라진다 (regression-guard — 착수 시점에 이미 0장) | verify: `! /usr/bin/grep -rq '^## Acceptance Criteria$' tasks/todo tasks/done tasks/plan`
- [ ] 저장소 게이트가 `ce task gate`를 호출한다 — validate를 재구현하지 않는다 | verify: `/usr/bin/grep -rq 'ce task gate' Makefile`
- [ ] 게이트를 어디에 붙였는지와 CI의 `ce` 해결 여부가 근거와 함께 기록됐다 | verify: human — 이 카드 `## 결정 기록` 절에 선택과 그 근거, 그리고 CI에서 `ce`가 해결되는지 확인한 결과가 적혀 있는지 확인
- [ ] 위반 카드를 심으면 게이트가 그 경로를 지목하며 실패한다 | verify: human — `type: fix` 카드 하나를 심고 저장소 게이트가 rc≠0으로 그 경로를 출력한 기록이 `## 결정 기록`에 있다
- [ ] 기존 게이트 통과 | verify: `make doc-check`

## Notes

- **노출은 더 이상 0이 아니다.** 2026-09-08 판은 "오늘 노출은 0"이라고 적었고 그것은
  `/task:run` Step 0이 preflight를 쓴다는 관찰에 근거했다. 맞는 관찰이지만 `ce task gate`를
  세지 않았다. 그 명령은 오늘 exit 1 한다.
- `ce task archive`는 `quality-review` 없는 카드를 거부한다(PLAN-007 §Starting state).
  validate 계열 검사를 archive가 어디까지 쓰는지는 1번 항목 착수 시 실측한다 — 추정하지
  않는다.
- 3번(PLAN-006 `## Children`)은 나머지와 독립이다. 다른 항목이 막히면 이것만 먼저 넣어도
  실패 파일이 26 → 25로 줄어든다.
- 엔진 자기모순은 여전히 기록만 한다: `ce task validate`는 `## Acceptance Criteria` 아래
  기준을 못 읽고 `preflight`는 읽는다. `ce-workbook/task_management` 소유이고
  PLAN-007 §External의 선례대로 **여기에 카드를 만들지 않는다** — 이 저장소의 보드가
  진척을 강제할 수 없는 항목은 영원히 열린 채 남기 때문이다. 엔진 쪽은 그 저장소에 보고한다.
- `done/`·`plan/` 파일도 고치므로 통합 직전 origin/master로 rebase한다(PLAN-007 §Rules).
