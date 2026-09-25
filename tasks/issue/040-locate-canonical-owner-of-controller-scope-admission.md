---
id: ISSUE-040
title: "The historical controller scope contract has no current owner"
type: bug
status: todo
priority: P2
effort: S
severity: low
ownership: local
discovered-in: "TASK-421 upstream owner resolution"
discovered-at: 2026-09-25
created: 2026-09-25
---

## Summary

TASK-421 treats `entry_selection._allowed_paths()` as a live upstream implementation
owned by ce-agent-kit. The Python engine containing that function was deleted in
`cc65f521`; the current CE catalog names no successor. This leaves TASK-421 without a
repository or CLI entry point where its criteria can be implemented. Creating a new
controller in ce-agent-kit would be a new product feature and needs an explicit owner
and scope.

## Reproduction

1. Read `ce-workbook/tasks/README.md` at the TASK-045 history and deletion record.
2. Check the current shared CE catalog and `ce-agent-kit` preflight/action-admission
   code for a successor selector.
3. Confirm the catalog declares no successor for the deleted Python controller.

## Expected vs Actual

- Expected: TASK-421 names a live canonical implementation owner and entry point.
- Actual: `task_management/engine/operate/entry_selection.py` was deleted by
  `cc65f5213595e804fcd79cc2a494b725ebbaccce`; the current CE catalog identifies
  `ce-agent-kit` as active CE source but declares no successor for that controller.

## Evidence

- `ce-workbook/tasks/README.md:144-147,159-164` records TASK-045's deletion and names
  `entry_selection.py` as removed by `cc65f521`.
- The archived execution-contract-gap record in ce-workbook (lines 138–147) reports
  no production Python engine/execution references after the retirement.
- `ce-workbook/guidance/shared/ce/catalog-v1.yaml` lists `ce-agent-kit` as the active
  CE source without a successor mapping for `entry_selection`.
- Current `ce-agent-kit` preflight evaluates criteria, verify bindings, and
  dependencies; action admission consumes already-authored requests and does not
  provide a card selector or the historical `_allowed_paths()` API.

## Recommendation

Keep TASK-421 blocked until a product owner decides whether to create a new
controller, identifies its canonical repository and CLI route, and defines its
disposition contract. Do not make `tasks/` a synthetic file scope.

## 소유권 — 이 저장소다

이 이슈는 DVA 보드의 잘못된 소유 표기를 바로잡는 기록이며 이 로컬 기록만 이 저장소가
소유한다. controller의 구현 소유자는 정해져 있지 않으므로 특정 repository에 책임을
부여하지 않는다.
