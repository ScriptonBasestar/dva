---
id: TASK-375
title: "unify optional-skip warning emission across the seven plan verbs"
type: refactor
priority: P3
effort: S
exec-tier: standard
created-at: 2026-09-12T16:00:00+09:00
source: "tasks/done/374-surface-optional-entry-skips-on-the-execution-path.md (review-374 Findings 7·8)"
status: todo
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

- [ ] 일곱 방출 지점이 각자의 `report`/거절 검사에 대해 동일한 순서 규칙을 따르고, 그 규칙이 코드 주석에 한 번 적혀 있다 | verify: human — `build.go`·`logs.go`·`composition_restart.go`·`composition_flags.go`·`plan_lifecycle.go` 의 방출 지점을 모두 열어 순서가 일치하는지 확인
- [ ] composition 경고가 출처 child plan 이름을 포함한다 | verify: `go test ./internal/cli -run TestOptionalSkipIsReportedByComposition -count=1`
- [ ] 순서 규칙을 고정하는 테스트가 있다 — 환경이 불완전한 plan 에서 경고와 거절의 상대 순서를 단언한다 | verify: `go test ./internal/cli -run TestOptionalSkip -count=1`
- [ ] USAGE.md 의 경고 예시가 바뀐 메시지와 일치한다 | verify: `make doc-check` (regression-guard)

## Notes

TASK-374 가 의도적으로 **제외한** 세 방출 지점(`hooks.go:144`,
`manifest_plans.go:93`, `compositionChildEnvironment:188`)은 이 카드에서도
제외 대상이다 — 리뷰어가 독립 확인했다. `compositionChildEnvironment` 는
`ResolveCompositionPlan` 이 이미 해석한 child 를 재해석하므로 여기서 방출하면
child 마다 2~5회 중복 출력된다(`composition_orchestrator.go:437-446` 이
`Up:448`·`WaitReady:463`·`Down:474`·`Stop:489`·`IsUp:501` 다섯 곳에서 호출).
`IsUp` 은 순수 probe 라 출력하면 안 된다.
