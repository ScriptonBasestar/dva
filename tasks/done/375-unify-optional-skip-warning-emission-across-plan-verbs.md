---
id: TASK-375
title: "unify optional-skip warning emission across the seven plan verbs"
type: refactor
priority: P3
effort: S
exec-tier: standard
created-at: 2026-09-12T16:00:00+09:00
source: "tasks/done/374-surface-optional-entry-skips-on-the-execution-path.md (review-374 Findings 7·8)"
status: done
---

# Task 375: optional-skip 경고의 방출 규칙을 일곱 동사에서 통일

## Summary

TASK-374 가 optional 엔트리 skip 경고를 실행 경로 전체에 연결했다. 연결 자체는
끝났고 회귀 가드도 있다. 남은 것은 **표현의 불균일**이다 — 일곱 방출 지점이
같은 경고를 서로 다른 순서와 서로 다른 정보량으로 낸다. 독립 리뷰(review-374)가
Finding 7·8 로 냈고, 한 장의 카드로 묶으라는 것이 리뷰어의 권고다: 둘 다
`plan_lifecycle.go` + `composition_flags.go` 에 떨어지고 `assertSkipWarned` 와
USAGE.md 의 같은 문단을 함께 건드리기 때문이다. 나눠서 처리하면 일주일 간격으로
같은 헬퍼와 같은 문단을 두 번 고치게 되고, 그게 둘이 어긋나는 경로다.

## Problem

**(1) 방출 순서가 동사마다 다르다 (주)**

`build.go:234`, `logs.go:154`, `plan_lifecycle.go:559`(`status`) 는 각자의 `report`
검사 **앞에서** 경고하고, `plan_lifecycle.go:350`(`up`) 은 **뒤에서** 경고한다.

그래서 환경이 불완전한 plan 은 세 동사에서 "경고를 낸 뒤 거절"하고 나머지 네
동사에서는 "조용히 거절"한다. 사용자가 보는 것은 거절 사유가 아닌 경고가 먼저
나오는 화면이다 — 틀린 정보는 아니지만 같은 입력에 대해 동사마다 다른 화면이다.

**(2) composition 경고가 출처 child 를 밝히지 않는다**

`printCompositionWarnings`(`plan_lifecycle.go:287-300`)는 각 child 의 경고를 그대로
흘린다. 여러 child 가 같은 이름의 optional 엔트리를 건너뛰면 출력이 구분되지
않아 어느 child 의 것인지 알 수 없다.

## Why this is a card and not a follow-up commit

일곱 자리 중 세 자리만 맞추면 불균일이 줄지 않고 모양만 바뀐다. 규칙을 하나
정하고("경고는 `report` 검사 뒤" 또는 "앞") 일곱 자리를 모두 그 규칙으로
옮겨야 한다. (2)는 메시지 포맷 변경이라 `assertSkipWarned` 의 리터럴과 USAGE.md
의 경고 예시 문구를 동시에 움직인다.

## Completion Criteria

- [x] 일곱 방출 지점이 각자의 `report`/거절 검사에 대해 동일한 순서 규칙을 따르고, 그 규칙이 코드 주석에 한 번 적혀 있다 | verify: human — `build.go`·`logs.go`·`composition_restart.go`·`composition_flags.go`·`plan_lifecycle.go` 의 방출 지점을 모두 열어 순서가 일치하는지 확인
- [x] composition 경고가 출처 child plan 이름을 포함한다 | verify: `go test ./internal/cli -run TestOptionalSkipIsReportedByComposition -count=1`
- [x] 순서 규칙을 고정하는 테스트가 있다 — 환경이 불완전한 plan 에서 경고와 거절의 상대 순서를 단언한다 | verify: `go test ./internal/cli -run TestOptionalSkip -count=1`
- [x] USAGE.md 의 경고 예시가 바뀐 메시지와 일치한다 | verify: `go test ./internal/cli -run TestUsageWarningExamplesMatchTheRenderedFormat`

## Notes

TASK-374 가 의도적으로 **제외한** 세 방출 지점(`hooks.go:144`,
`manifest_plans.go:93`, `compositionChildEnvironment:188`)은 이 카드에서도
제외 대상이다 — 리뷰어가 독립 확인했다. `compositionChildEnvironment` 는
`ResolveCompositionPlan` 이 이미 해석한 child 를 재해석하므로 여기서 방출하면
child 마다 2~5회 중복 출력된다(`composition_orchestrator.go:437-446` 이
`Up:448`·`WaitReady:463`·`Down:474`·`Stop:489`·`IsUp:501` 다섯 곳에서 호출).
`IsUp` 은 순수 probe 라 출력하면 안 된다.

## Verification Evidence

**결정한 규칙**: 경고는 **해석 성공 직후, 그 동사의 거절 판단 앞에서** 낸다.
`printPlanWarnings` 의 doc 에 `ORDERING RULE` 로 한 번 적었다
(`internal/cli/plan_lifecycle.go`). 근거는 두 줄이다 — 경고는 해석이 무엇을
찾았는지에 대한 진술이라 그 동사가 실행을 거절하든 말든 참이고, 거절된 실행이야말로
사용자가 원인을 찾아야 하는 실행이다.

앞이냐 뒤냐는 다수결로 고른 게 아니라 이 근거로 골랐지만, 결과적으로 이동량도
최소였다 — 일곱 자리 중 `build`·`logs`·`status` 와 composition 다섯 동사는 이미
앞이었고 `up`/`down`/`stop`/`restart` 넷만 뒤였다.

**이동 후 열한 자리 실측**(`build.go:234`, `logs.go:154`, `plan_lifecycle.go:360`
(up)·`426`(down)·`496`(stop)·`541`(restart)·`573`(status),
`composition_flags.go:345`(up)·`367`(down)·`403`(stop)·`497`(status),
`composition_restart.go:32`): 전부 해석 직후·거절 검사 앞.

**child 라벨**: `printCompositionWarnings` 가 `warning: [child: <plan>] …` 형태로
낸다. `assertSkipWarnedFrom` 이 child 이름에서 기대 prefix 를 만들기 때문에 라벨을
떼면 composition 테스트만 깨지고 leaf 테스트는 그대로 통과한다 — 라벨이 장식이
아니라 단언이다.

**범위 밖에서 하나 더 고쳤고, 그게 틀렸다** (review-375 F1·F2·F6 반영 후 재작성):
`runCompositionBuild`/`runCompositionLogs` 는 composition 레벨에서 한 번, 그리고
자식마다 `runPlanBuild`/`runPlanLogs` 안에서 또 한 번 같은 경고를 내고 있었다. 라벨이
붙으면서 "라벨 있는 줄 + 라벨 없는 줄"로 눈에 띄어 composition 레벨 호출을 제거했다.

두 가지가 틀렸다.

- **중복의 나이를 과장했다**(F6). "이 카드 이전부터"라고 적었지만
  `git log -S "printCompositionWarnings(comp)"` 는 `fe448b6` 와 `690c2ad` 둘만
  돌려준다 — 중복은 바로 앞 커밋인 TASK-374 에서, 같은 작성자가 만들었다. 오래된
  빚이 아니라 하루 전에 내가 판 구덩이였고, "pre-existing" 이라는 말이 그 사실을
  가렸다.
- **제거는 경고 중립이 아니었다**(F1·F2). 자식 자리의 방출은 `runCompositionBuild`
  기준 `validateCompositionFlagScope` **뒤에서** 도는 루프 안에 있고, 그 루프는 첫
  자식 실패에서 `return` 한다. 그래서 composition 레벨 호출을 없앤 순간 (a) 플래그가
  거절되면 아무 경고도 안 나오고 (b) 앞 자식이 실패하면 뒤 자식 경고가 사라졌다 —
  이 카드가 고치려던 결함을 같은 커밋에서 되살린 것이다. `6be1780` 에서 composition
  레벨 방출을 검사 앞으로 되돌리고, 중복은 **나중 방출**(루프 안)을 `suppressPlanWarnings`
  로 죽여서 없앴다. 방향이 중요하다 — 규칙을 지키는 쪽은 먼저 나오는 방출이다.

### Mutation testing

TASK-374 에서 기록한 휴리스틱 — *"카드가 말하는 버그 쪽으로 변이시켜라, 수정이
추가한 코드 쪽이 아니라"* — 을 그대로 적용했다. 두 변이 모두 **수정 전 상태를 정확히
재현**한다:

- **변이 A**: `runPlanUp` 에서만 방출을 거절 검사 뒤로 되돌림 →
  `TestOptionalSkipWarnsBeforeTheVerbRejects`, `TestOptionalSkipWarningPrecedesThePlanHeader`
  둘 다 FAIL. 나머지는 통과 — 즉 기존 테스트 전부가 이 버그에 대해 비어 있었다는
  뜻이고, 그게 이 카드가 존재하는 이유다.
- **변이 B**: `printCompositionWarnings` 에서 child 라벨 제거 →
  `TestOptionalSkipIsReportedByCompositionUp/Status` FAIL, leaf 테스트는 통과.

새 테스트 두 개가 규칙을 양쪽 끝에서 고정한다: 거절 경로에서는 "경고는 있고
`[plan: ...]` 헤더는 없다"(검사 앞에서만 낼 수 있는 조합), 실행 경로에서는
"경고 index < 헤더 index". 앞의 것만 있으면 방출을 헤더 아래로 옮겨도 통과한다 —
그 입력에서는 헤더가 아예 안 나오기 때문이다.

### Gates

`make build` 0, `make lint` 0, `make doc-check` 0, `make check-generate` 0,
`make test` 전체 통과(`internal/cli` coverage 82.0%).

### 독립 리뷰(review-375) 대응 — verdict conditional

**F1 (Major) — 범위 밖 수정이 카드 자신의 결함을 두 동사에 되살렸다. 수정함.**

`runCompositionBuild`/`runCompositionLogs` 에서 composition 레벨 방출을 제거한 것이
틀렸다. 나는 자식 순회가 항상 도는 것처럼 추론했는데, 순회는 **`validateCompositionFlagScope`
건너편**에서 돈다 — 이 커밋이 코드에 새겨 넣은 ORDERING RULE 이 "방출이 앞서야 할
검사"로 이름까지 지목한 바로 그 검사다. 리뷰어 실측:

- `690c2ad`: `dva build <composition> --no-cache` → 거절, stderr **빈 문자열**
- `690c2ad^`: 같은 호출 → 경고 출력됨
- `690c2ad`: `dva up <composition> --bogus` → 같은 부류의 거절, 경고 **출력됨**

즉 같은 입력에 `up` 은 경고하고 `build` 는 침묵한다 — 카드 Problem 절이 묘사한
"한 입력, 일곱 화면"을 고치는 커밋이 두 동사에 다시 만든 것이다.

**F2 (Major) — 앞선 자식이 실패하면 뒷자식 경고가 사라진다. 수정함.**

`runCompositionBuild` 는 첫 자식 오류에서 `return` 한다. 내가 근거로 적은
*"자식을 각자 헤더 아래 개별 제시하므로"* 는 **순회가 도달한 자식에 대해서만** 참이다.
`runCompositionLogs` 는 `errs` 에 모으고 계속하므로 영향 없고, `build` 만 해당된다.

**수정 내용**

1. `printCompositionWarnings(comp)` 를 두 동사에 복원 — `validateCompositionFlagScope`
   **앞**. 이제 열한 자리가 예외 없이 규칙을 따른다.
2. `suppressPlanWarnings` 패키지 플래그를 도입해 자식 순회 동안만 켠다. 이중 출력은
   composition 레벨을 지워서가 아니라 자식 쪽을 잠재워서 없앤다 — 순서 규칙을 지키는
   쪽(앞선 방출)을 남기는 것이 옳은 방향이기 때문이다. `runCompositionLogs` 가 같은
   자리에서 `forceSubprocess` 를 같은 이유로 켜고 있어 새 모양을 발명하지 않았다.
3. USAGE.md 의 "build/logs 는 라벨 없는 예외" 문단 삭제 — 예외가 없어졌다.

**회귀 테스트 3개, 변이로 검증**

- `TestOptionalSkipIsReportedWhenCompositionBuildRejectsItsFlags` (F1)
- `TestOptionalSkipIsReportedForChildrenAfterAFailingOne` (F2)
- `TestCompositionBuildWarnsAboutEachChildExactlyOnce` (억제가 두 번째 이중출력이
  되지 않도록 고정)

변이 C(= `690c2ad` 상태 정확 복원: 두 동사의 composition 레벨 방출 제거 + 억제 제거)
→ F1·F2 테스트 FAIL, 중복 테스트는 통과(그 상태엔 중복이 없으므로 — 올바른 결과).
변이 D(= composition 레벨 방출은 두고 억제만 제거) → 중복 테스트만 FAIL, `got 2`.
두 변이가 서로 다른 테스트를 잡는다는 것이 셋이 각기 다른 성질을 고정한다는 증거다.

**배운 것**: 규칙을 코드에 적어 놓고 같은 커밋에서 그 규칙을 어겼다. 규칙 문장이
`validateCompositionFlagScope` 를 명시적으로 지목하고 있었는데도 — **자기가 방금
쓴 규칙을 자기 변경분에 대해 검사하지 않았다.** 순회형 동사는 "검사 뒤에서 도는
루프"를 갖고 있어 규칙 위반이 특히 눈에 안 띈다.

**Gates(수정 후)**: `build` 0, `lint` 0, `doc-check` 0, `check-generate` 0,
`test` 전체 통과(`internal/cli` coverage 82.0%).

### review-375 조건 5·6 이행 (후속 커밋)

**조건 5 — 방출 지점 전수 고정 / composition 순서 단언.**
리뷰가 지적한 구멍: `runPlanDown`·`runPlanStop`·`runPlanRestart` 와
`runCompositionDown`·`runCompositionStop`·`runCompositionRestart` 의 방출을 지우고
`runCompositionUp` 의 방출을 플래그 검사 아래로 내리는 **일곱 개 동시 변이**를 해도
스위트가 전부 녹색이었다. 열한 개 방출 지점 중 실제로 고정돼 있던 건 build·status·
logs·composition up·composition status 뿐이었다.

- `TestOptionalSkipIsReportedByEveryRemainingVerb` — 나머지 여섯 동사를 각자
  **자기 러너로** 호출하는 테이블. 공용 헬퍼를 거쳐 도달하면 헬퍼를 고정하게 되는데,
  이 테스트의 존재 이유가 바로 "이들이 서로 다른 호출 지점"이라는 사실이다.
- `TestCompositionUpWarnsBeforeItRejectsItsFlags` — 존재가 아니라 **순서**를 단언한다.
  `--bogus-flag` 로 검사가 거절하는 입력을 주면, 방출이 검사 뒤로 내려간 순간 아무것도
  출력되지 않는다. 존재만 보는 단언은 다른 입력으로 통과해 버린다.

변이 E(위 여섯 방출 전부 삭제) → 새 테스트 3종 + 기존 5종이 모두 FAIL. 삭제 전에는
이 중 어느 것도 안 깨졌다.

**조건 5 부수 — 억제의 전제 고정.** `suppressPlanWarnings` 는 composition 레벨 해석과
자식별 재해석이 같은 경고를 낸다는 전제 위에 서 있다. 두 호출은 실제로 다르다 —
`ResolveCompositionPlan` 은 `ResolvePlan(owner, name, entry.Vars)`, `runPlanBuild` 는
`ResolvePlan(root, name, nil)`. 오늘은 어긋날 수 없다: lifecycle 패키지의 유일한
`warn()` 호출 지점(`resolver.go:429`)이 러너 선언 dir 와 owner file dir 만 보고,
vars 는 `resolved.EnvVars` 에만 병합되기 때문이다. "오늘은" 이라서 테스트로 박았다 —
`TestCompositionAndPerChildResolutionAgreeOnWarnings` 는 `composes[].vars` 가 실제로
설정된 fixture 로 두 결과를 비교하고, 자식이 경고를 하나도 안 내면 빈 슬라이스끼리
비교해 엉뚱한 이유로 통과하지 않도록 먼저 `t.Fatal` 한다.

**조건 6** — 위 "범위 밖에서 하나 더 고쳤고, 그게 틀렸다" 문단으로 재작성.

**F4 — 바인딩을 실제 검사로 교체**: 기준 4 의 원래 `verify:` 는
`make doc-check (regression-guard)` 였는데, `yamlcheck` 는 태그된 ```yaml 펜스만
검사하고 USAGE.md 의 경고 예시는 태그 없는 출력 펜스라 **아무것과도 대조되지 않았다.**
기준은 바인딩이 아니라 손검사로 참이었다.

리뷰어의 제안대로 리터럴을 grep 하지 않고 **렌더해서** 고정했다 —
`TestUsageWarningExamplesMatchTheRenderedFormat` 은 실제 `printPlanWarnings` /
`printCompositionWarnings` 출력을 캡처하고, 문서가 일부러 자리표시자를 쓰는 디렉토리
경로만 치환한 뒤 그 줄이 USAGE.md 안에 있는지 확인한다. 리터럴을 grep 하면 문자열
사본을 하나 더 만들어 그 사본에 문서를 묶게 되지만, 이 방식은 포맷 문자열 자체에
묶는다. 변이로 확인: `warning: [child: %s] %s` → `warning: (child %s) %s` 와
`resolver.go` 의 `— skipped, directory %q` → `— skipped; directory %q` 를 각각
넣으면 leaf·composition 서브테스트가 FAIL 한다.

**review-375 최종 verdict: `pass`** (조건 없음). 리뷰어가 F1·F2 를 자기 프로브로
`690c2ad^` / `690c2ad` / `6be1780` 3지점 비교해 재현했고, Q1(억제의 전제)을
**철회**했다 — 두 경로가 config(root vs owner)와 vars(CLI `--var` vs `composes[].vars`)
양축에서 실제로 다르지만 둘 다 경고에 닿지 못한다는 것을 구조적으로(`resolver.go:261`
의 owner 재유도, skip 검사 425 가 첫 var 병합 448 보다 위) 그리고 경험적으로
(`dir: vendor/${TARGET}` + 충돌하는 `composes[].vars` fixture 가 양쪽에서 바이트 동일한
경고 집합을 냄) 확인했다. 그 불변식을 `suppressPlanWarnings` doc 에 적어 두었고,
깨지면 `TestCompositionAndPerChildResolutionAgreeOnWarnings` 가 잡는다.

리뷰어가 추가한 변이 하나: 억제의 `defer` 리셋만 제거 → 6개 테스트 FAIL. 패키지 전역의
누수는 (설계가 아니라 테스트 순서 덕이지만) 막혀 있다.
