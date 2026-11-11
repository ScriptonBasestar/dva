---
id: TASK-596
title: "compare shared 항목의 보고와 down 범위"
type: bug
priority: P2
effort: M
exec-tier: strong
status: review
created: 2026-11-09
---

## Summary

`compare up`은 이미 떠 있던 shared 항목도 `started`로 보고하고, 인자 없는 `compare down`은 다른 비교가 쓰는 shared 항목까지 내릴 수 있다(`compare_down.go`). shared `up`·`status`는 vars를 넘기지 않아 shared 결과의 per_variant endpoint URL도 비교안별 값으로 치환되지 않는다. TASK-592 리뷰 중 발견했다.

## Completion Criteria

- [x] 이미 떠 있던 shared 항목은 started가 아니라 이미 실행 중으로 보고된다 | verify: `go test ./internal/cli/ -run TestCompareUpReportsAlreadyRunningSharedAsSkipped -count=1`
- [x] 인자 없는 compare down이 다른 비교가 쓰는 shared 항목을 내리지 않거나 내리기 전에 알린다 | verify: `go test ./internal/cli/ -run TestCompareDownKeepsSharedEntriesAnotherComparisonUses -count=1 && go test ./internal/cli/ -run TestCompareDownKeepsSharedWhenAnotherRecordIsUnreadable -count=1`
- [x] shared 결과의 per_variant endpoint URL이 비교안별 값으로 치환된다 | verify: `go test ./internal/cli/ -run TestCompareSharedEndpointsAreSubstitutedPerVariant -count=1 && go test ./internal/cli/ -run TestLifecycleRunnerEndpointsUseVarsWithoutLeakingThem -count=1`

## Scope Notes

- 기준 1: `up`은 shared를 시작하기 전에 shared 항목의 status를 읽고, 모두 running이면 `Up`을 부르지 않고 `skipped`로 보고한다(비교안의 `skipped`와 같은 "이미 실행 중" 의미, 새 결과 값을 만들지 않음). status를 읽지 못하거나 일부만 떠 있으면 지금처럼 `Up`(멱등)을 부르고 `started`로 보고한다. 판정은 비교안의 `variantRunning`과 같은 `entriesRunning`을 쓴다.
- 기준 2: "내리지 않는다"를 택했다. `down`이 shared를 내릴 차례에, 같은 `dva.yml`에 선언된 다른 비교 중 같은 shared 엔트리를 선언하고 기록(`.sb/dva/compare/<이름>.json`)이 남은 비교가 있으면 그 엔트리는 멈추지 않는다. 결과에 `shared.kept`(엔트리 → 쓰는 비교 이름)를 싣고, 모두 남기면 shared 결과 값은 새 값 `kept`, 일부만 멈추면 `stopped`이다. 이 비교의 기록은 그대로 지워지므로 남은 비교의 마지막 `down`이 엔트리를 멈춘다. 읽을 수 없는 다른 기록은 쓰는 중으로 본다(보수적). 이름을 준 `down`이 마지막 비교안을 내릴 때도 같은 규칙이다. 선언에서 사라진 비교의 기록이나 compare 밖의 `dva up <plan>`으로 띄운 엔트리는 보지 않는다(카드 범위 "다른 비교").
- 기준 3: shared 항목은 vars 없이 한 번 뜨므로 vars를 넘기는 것으로는 풀리지 않는다(어느 비교안의 값인지 정할 수 없음). 대신 runner에 `Endpoints`(plan 엔드포인트를 주어진 vars로 보간만 함)를 더하고, shared 결과의 엔드포인트 중 어느 비교안의 vars(`variantVars`: 포트, config 모드의 `DVA_VARIANT`)로 보간하면 URL이 달라지는 것을 비교안마다 한 벌씩 `variant` 필드와 함께 그 값으로 바꾼다. 비교안과 무관한 엔드포인트는 한 번만 남는다. `up`은 이번 호출에서 started·skipped인 비교안, `status`는 기록의 모든 비교안을 쓴다. 호출한 체크아웃의 환경은 명령 전체가 공유하므로 `Endpoints`는 그 복사본에만 vars를 합친다.
- 범위 밖(발견): config 모드 비교안의 `Up`/`Status`/`Down`은 호출한 체크아웃의 캐시된 환경(`rootEnvLoad`)에 그 비교안의 vars를 직접 합친다(`compareConfigAt` → `resolvePlanRuntime`의 `MergeVars`). 같은 명령 안에서 뒤에 오는 호출이 앞 비교안의 포트 var를 볼 수 있다. 이 카드는 새 `Endpoints` 경로만 복사본을 쓰게 했고 기존 경로는 바꾸지 않았다.
