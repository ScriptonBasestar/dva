---
id: ISSUE-007
title: "ce task validate does not constrain the status field"
type: bug
status: todo
priority: P2
effort: S
exec-tier: standard
severity: medium
discovered-in: "2026-09-13 board currentization review"
discovered-at: 2026-09-13
ownership: split
created: 2026-09-13
upstream-ref: "ce-agent-kit#3"
---

## Summary

`ce task validate` accepts any value in a card's `status` field — a value from
no vocabulary, or no value at all — and never compares it to the zone the card
sits in. A card in `tasks/done/` may claim `status: todo` and validate clean.

DVA is not currently exposed to that, because it carries its own gate:
`tools/doccheck`'s `checkCardStatus` holds the zone table TASK-287 froze
(`tasks/done/` → `done`, `tasks/todo/` and `tasks/issue/` → `todo`,
`tasks/_archive/` → `done|superseded`) and fails the build on a mismatch. That
local gate is what caught ISSUE-003's stale `todo` on 2026-09-13, three days
after the field went wrong. `ce task validate` had reported the same card
`✅ Valid` throughout.

The defect is therefore not that DVA is unprotected. It is that the protection
lives in the wrong repository. `tools/doccheck`'s status sweep is a
repository-local copy of a board judgement the shared gate should own — the
exact duplication that drifts the moment one copy is fixed, and the reason the
shared `ce task gate` exists.

Owner is external: `ce task validate` belongs to ce-workbook/task_management.
Filed here on the ISSUE-004 precedent.

## Reproduction

1. Take any valid card and set `status:` in turn to `done`, `resolved`,
   `closed`, `completed`, `banana`, and the empty string.
2. Run `ce task validate <card>` after each. All six report
   `✅ Valid (no errors or warnings)`.
3. Set `tasks/done/309-suppression-ergonomics.md` to `status: todo` and
   validate it: also `✅ Valid`, though its zone permits only `done`.
4. Run `make doc-check` on the same tree: `STATUS … permits status: done,
   found "todo"`, and the build fails.

Observed 2026-09-13. Both files were restored.

## Expected vs Actual

- Expected: the shared validator draws `status` from a declared vocabulary and
  rejects a value its zone refutes, so a repository does not need its own copy.
- Actual: every value passes the shared validator; only DVA's local doccheck
  sweep dissents.

## Impact

Two costs, and the second outlives the first.

Any repository adopting the CE task runtime without DVA's local sweep gets no
status checking at all — and will not know, because `ce task validate` reports
clean. DVA found this only by owning both gates and watching them disagree.

For DVA itself, the cost is a permanent maintenance obligation on a board rule
it should not own. The zone table has to be kept in step with a vocabulary
defined elsewhere, and the two will drift silently: doccheck's copy fails
loudly when a card is wrong, but nothing fails when the *table* is wrong.

A related gap sits beside this one: `ce task move` refuses a card in a kind zone
(`move failed: task is not in a zone (doing/todo/review/blocked/done)`), so
disposing an issue is a manual `git mv`. The runtime constrains neither the
field nor the transition that would keep it honest.

## Recommended Resolution

Move the zone/status constraint into `ce task validate`, then delete
`tools/doccheck`'s `checkCardStatus` sweep and its zone table rather than
keeping both. Two gates agreeing is not redundancy here — it is the drift
surface.

If upstream instead concludes that zone is authoritative and `status` is a
redundant copy with no reader, removing the field from the schema closes this
just as well. What cannot stand is an unvalidated field that each repository
polices on its own.

Sequencing matters: DVA must not delete its local sweep before the shared one
lands and is observed rejecting the same inputs. Until then the duplication is
the safe state.

## 소유권 — 갈린다 (2026-09-15 명시)

규칙 이전과 검사 제거가 짝이다. 기준 1·2(zone/status 제약의 `ce task validate` 이전)은
상류(`ce-workbook/task_management`) 소유고, 기준 3(`tools/doccheck`의 checkCardStatus
sweep 제거)은 이 저장소 소유다. 순서 제약은 카드가 이미 정한다 — 상류 검사가 착지하는
것을 관측하기 전까지 로컬 sweep을 지우지 않는다. 상류 절반의 보고 자리는 [[TASK-395]]가
만든다.

## Resolution Criteria

- [ ] `ce task validate` rejects a status value outside the declared vocabulary and one its zone refutes | verify: human — upstream tests cover an unknown value, an empty value, and a zone mismatch
- [ ] The shared validator rejects the same inputs DVA's doccheck sweep rejects today | verify: human — both gates are run on the same fixtures and the outputs are linked here
- [ ] `tools/doccheck`'s checkCardStatus sweep and zone table are removed once the shared gate covers them | verify: human — the removal commit is linked here and `make doc-check` still fails on a zone-mismatched fixture through the shared gate
