---
id: TASK-397
title: "Retarget the familybook harness step to the dev plan"
type: chore
priority: P2
effort: S
exec-tier: standard
status: todo
created: 2026-09-15
source: "TASK-328 실기동 회차(2026-09-15) — dogfood-run.sh의 steps_familybook이 존재하지 않는 plan hybrid를 가리킨다"
depends-on: []
---

## Summary

`tools/dogfoodrun/dogfood-run.sh`의 `steps_familybook`(`:219`)이 `up hybrid`(`:223`)와
`down hybrid --purge --project infra --force`(`:225`)를 실행하지만, familybook devbox의
plan은 `dev`로 개명됐다(devbox `6881c81` "rebuild around KMP"). 회차는 preview를
넘기지 못하고 만다 — 2026-09-15 TASK-328 실기동 회차에서 familybook 축이 미실행된
두 원인 중 하나다(나머지는 고아 볼륨 purge 동의, 사람 결정).

[[TASK-379]]가 primeno1에 대해 정확히 같은 수정을 한 선례다 — 이 카드는 familybook
축의 동일 수정이다. 스텝의 `--project infra` 스코프 지정이 개명된 plan 아래에서도
맞는지는 수정 시점에 함께 확인한다.

## Completion Criteria

- [ ] steps_familybook이 현재 plan 이름(`dev`)을 가리키고 하네스에 `hybrid` 참조가 남지 않는다 | verify: `! /usr/bin/grep -q 'hybrid' tools/dogfoodrun/dogfood-run.sh`
- [ ] familybook 축의 preview가 존재하지 않는 plan 오류 없이 통과한다 | verify: human — `./tools/dogfoodrun/dogfood-run.sh --preview familybook` 출력에서 plan 미조회 오류가 없는지 확인한다
