---
id: TASK-324
title: "duplicate-plan warning ignores composes and misfires on composition plans"
type: bug
priority: P2
effort: S
exec-tier: standard
created-at: 2026-09-05T14:00:00+09:00
source: "docs/dogfood/flow-taskchain.md"
status: done
---

# Task 324: composition plan 쌍이 항상 "equal declaration" warning

## Summary

composes로 구성된 plan은 entries가 비어 있는데, `plansHaveEqualDeclaration`이 TASK-244
D6 필드만 비교하고 `Composes`(TASK-260)를 보지 않아 실제로는 서로 다른 compose 구성을
가진 plan 쌍도 항상 "declare equal" warning을 오탐했다. `Composes`(plan, order,
depends_on, vars)를 비교 대상에 추가하고 메시지에 composes를 언급하도록 고쳐,
flow-taskchain-devbox에서 해당 warning이 0건이 되는 것을 확인했다.

## Repro

flow-taskchain-devbox `dva validate`:
`local-dev`(composes: local-infra, engine/dev, mcp/dev)와 `local-full`(composes: local-infra-full + 4 child)은
composes가 다른데 `plans "local-dev" and "local-full" declare equal environment, site, vars, endpoint_tags, and entries` warning이 뜬다.

## Cause

`internal/config/validate_warnings.go` `plansHaveEqualDeclaration`이 TASK-244 D6 필드만 비교하고
`Composes`(TASK-260)를 보지 않는다. composition plan은 entries가 비어 있어 두 개만 있어도 항상 동일 판정.

## Completion Criteria

- [x] `Composes`(plan, order, depends_on, vars)를 비교에 포함하고, 메시지에 composes를 언급 | verify: `make test` (2026-09-05 exit 0)
- [x] composes가 다른 plan 쌍은 warning 없음 / 같은 쌍은 warning 유지 테스트 | verify: `go test ./internal/config/ -run Duplicate` (구현 되돌리면 plan.composes·composes.* 케이스 FAIL 확인)
- [x] flow-taskchain-devbox `dva validate` "declare equal" warning 0 | verify: human — 수정 빌드로 `grep -c "declare equal"` = 0 (남은 warn 4건은 범위 밖 Makefile suggestion)
