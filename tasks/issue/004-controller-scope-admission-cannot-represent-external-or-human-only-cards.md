---
id: ISSUE-004
title: "Controller scope admission cannot represent external or human-only cards"
type: bug
status: todo
priority: P1
effort: S
exec-tier: standard
severity: medium
discovered-in: "2026-09-10 direct queue-run preflight"
discovered-at: 2026-09-10
ownership: upstream
created: 2026-09-10
upstream-ref: "ce-agent-kit#1"
---

## Summary

The direct queue preflight requires every non-decomposable `todo` and the selected P0 `issue` to
have non-empty `allowed-paths`. The controller rejects `tasks/` paths,
absolute paths, parent traversals, and glob patterns. This is correct for an
executor that modifies product files, but it cannot represent cards whose
truthful work is exclusively external, human-operated, or a controller-owned
state transition.

DVA currently has each shape:

- ISSUE-001 is a P0 external receipt-contract blocker owned by ce-agent-kit
  and ce-workbook/task_management.
- TASK-329 and TASK-348 require external project readiness or a real build.
- TASK-307, TASK-309, TASK-319, TASK-321, and TASK-351 await an explicit
  human design decision.

Adding a synthetic DVA file scope would authorize an implementation the card
does not own. Adding its task-card path is rejected because `tasks/` is
controller-owned. The queue therefore exits 65 after selecting the card but before provider/action
execution or a controller-owned route transition can occur.

## Reproduction

1. Run the bound `queue_preflight` command for this repository.
2. Observe `task-allowed-paths-missing` for ISSUE-001 and active todos.
3. Add a task-card path as a hypothetical scope and observe
   `task-allowed-path-invalid`: `tasks/` is forbidden by
   `entry_selection._allowed_paths()`.

## Expected vs Actual

- Expected: cards that have no repository implementation scope can receive a
  controller-owned external/manual disposition without granting a fake executor
  scope.
- Actual: preflight selects the P0 issue, then rejects it before provider/action
  execution and route transition.

## P1 Blocker

- `reason`: DVA cannot resume its direct lifecycle queue truthfully while the
  scope contract treats external/manual cards as executable implementation
  units.
- `owner`: ce-agent-kit owns queue preflight, selection, and allowed-path
  validation. DVA owns only its adopted card states after the shared contract
  supports them. This line read ce-workbook/task_management until 2026-09-15;
  the 소유권 section below carries the measurement that corrected it.
- `next_action`: define and test an explicit machine-readable disposition-only
  admission contract. It must name the authoritative discriminator, permitted
  controller-owned routes, and fail-closed behavior before it can classify an
  external or human-only card ahead of implementation-scope validation.
- `next_check`: bound `queue_preflight` exits 0 for DVA without permitting
  `tasks/`, absolute paths, parent traversals, or globbed executor scopes.

## 소유권 — 상류다 (2026-09-15 명시)

큐 preflight·선정·allowed-path 검증은 `ce-agent-kit` 소유다 — P1 Blocker의 `owner`
항목이 그대로 적는다. 세 기준 전부가 상류 selection 테스트와 회귀를 검증
대상으로 요구한다. 보고는 [[TASK-399]]가 `ce-agent-kit#1`로 수행했다.

**2026-09-15 정정 — 상류는 `ce-agent-kit`이다.** 이 절과 위 본문은 원래
`ce-workbook/task_management`를 지목했고, 그게 실측과 어긋났다([[ISSUE-027]]).
실행되는 `ce`의 빌드정보가 자기 모듈을 직접 증언하고, ce-workbook에는 추적되는
구현이 없다:

```
$ go version -m "$(command -v ce)"
	path	github.com/archmagece/ce-agent-kit/cmd/ce
	mod	github.com/archmagece/ce-agent-kit	v0.8.5-...

$ git -C ~/mywork/ce/ce-workbook ls-files | grep -i preflight | grep -v '^tasks/'
(없음)
```

등록 지점은 `cmd/ce/handlers_task.go`, 구현은
`internal/adapter/cli/commands/task_preflight.go`로 **같은 저장소 안에 함께** 있다.
ce-workbook의 `tmp/.../queue_preflight.py`는 `.gitignore`된 미추적 스크래치 파일이고,
`task_management/USAGE.md`의 "기존 Python preflight"는 그 문서가 같은 자리에서 "새
lifecycle 규칙은 ce-agent-kit에서만 작성"이라고 적는 별개 개념이다. 보고처는
`ssh://git@gitlab.polypia.net:2224/archmagece/ce-agent-kit.git`다.

## Resolution Criteria

- [ ] Preflight distinguishes executor-owned implementation cards from external
  and human-only disposition cards | verify: human — upstream selection tests
  cover P0 issue, external todo, and decision todo shapes
- [ ] A DVA queue run routes ISSUE-001 and the external/manual cards without a
  synthetic product-file scope | verify: human — fresh controller evidence is
  linked here
- [ ] Scope validation remains strict for real implementation cards | verify: human — upstream regression rejects `tasks/`, absolute, parent, and glob paths

## 후속 (2026-09-24)

2026-09-24 재측정에서도 상류 계약은 그대로다. 작업은
[TASK-421](../blocked/421-admit-external-and-human-only-cards.md)가 소유한다.
