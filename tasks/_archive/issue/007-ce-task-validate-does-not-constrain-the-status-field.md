---
id: ISSUE-007
title: "ce task validate does not constrain the status field"
type: bug
status: done
priority: P2
effort: S
exec-tier: standard
severity: medium
discovered-in: "2026-09-13 board currentization review"
discovered-at: 2026-09-13
ownership: split
created: 2026-09-13
upstream-ref: "ce-agent-kit#3"
resolution: fixed
resolved-at: 2026-09-25T04:45:47Z
resolution-summary: "Resolved as fixed by TASK-424."
---

## Summary

The CE default dialect accepts any value in a card's `status` field — a value from
no vocabulary, or no value at all — and does not compare it to the zone the card
sits in. A card in `tasks/done/` may claim `status: todo` and validate clean.

DVA used to avoid the gap through `tools/doccheck`'s copied `checkCardStatus` zone table,
which caught ISSUE-003's stale `todo` on 2026-09-13 while `ce task validate` reported the card
valid. TASK-424 has now selected CE's strict repository dialect, verified its status contract,
and removed that local copy.

At discovery, the defect was not that DVA was unprotected. It was that the protection
lived in the wrong repository. `tools/doccheck`'s status sweep is a
repository-local copy of a board judgement the shared gate should own — the
exact duplication that drifts the moment one copy is fixed, and the reason the
shared `ce task gate` exists.

Owner is external: `ce task validate` belongs to ce-agent-kit. Filed here on
the ISSUE-004 precedent — including, until 2026-09-15, that card's
misattribution, corrected in the 소유권 section below.

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

- Expected: CE's default dialect keeps existing repository compatibility; a repository that
  opts into strict mode gets zone/status validation from the shared validator and does not need
  a local copy.
- Actual at discovery (2026-09-13): every value passed the default validator; only DVA's local
  doccheck sweep dissented. CE now provides an opt-in strict dialect. In strict mode the
  resolver's canonical `issue/ + status: done + resolution:` output is allowed; the old DVA
  copy did not model that terminal exception, as recorded in TASK-424 evidence.

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

The selected resolution is an opt-in strict dialect in `ce task validate`, preserving
CE's default compatibility while allowing repositories to delegate the board status rule
to the shared validator. DVA selected strict mode, removed `checkCardStatus`, and retained
identity and duplicate-ID/filename checks in `tools/doccheck`.

If upstream instead concludes that zone is authoritative and `status` is a
redundant copy with no reader, removing the field from the schema closes this
just as well. What cannot stand is an unvalidated field that each repository
polices on its own.

Sequencing mattered: DVA kept its local sweep until CE strict mode, producer round-trips,
paired fixtures, whole-board validation, and independent TASK-424 review passed. The selected
resolver terminal-issue exception is documented in TASK-424 evidence.

## 소유권 — 갈린다 (2026-09-15 명시)

규칙 이전과 검사 제거가 짝이다. 기준 1·2(zone/status 제약의 `ce task validate` 이전)은
상류(`ce-agent-kit`) 소유고, 기준 3(`tools/doccheck`의 checkCardStatus sweep 제거)은 이
저장소 소유다. 순서 제약은 카드가 이미 정한다 — 상류 검사가 착지하는 것을 관측하기
전까지 로컬 sweep을 지우지 않는다. 상류 절반은 [[TASK-399]]가 `ce-agent-kit#3`으로
보고했다.

> **2026-09-15 정정.** 이 카드는 위 저장소를 `ce-workbook/task_management`로 적고
> 있었다. 측정한 결과 틀렸다 — `ce`는 `ce-agent-kit`에서 빌드되고(`go version -m
> $(which ce)` → `mod github.com/archmagece/ce-agent-kit`), 해당 서브커맨드는
> `cmd/ce/handlers_task.go`에 등록되고 `internal/adapter/cli/commands/`에 구현돼
> 있다. `ce-workbook`에 이름이 등장하는 것은 그 저장소가 `ce task *`를 **호출하는
> 계약 테스트**를 갖고 있기 때문이며, 소비자이지 소유자가 아니다. 오귀속은
> [[ISSUE-004]]의 `owner:` 줄에서 시작해 "ISSUE-004 선례"라는 인용을 타고 이 카드까지
> 번졌다 — 측정 없이 선례가 근거를 대신했다. 경위는 [[ISSUE-027]]이 기록한다.


## Resolution Criteria

- [x] `ce task validate` rejects a status value outside the declared vocabulary and one its zone refutes | verify: human — strict CE fixtures reject unknown, empty, missing, alias, and zone-mismatched values
- [x] The shared validator rejects the same invalid ordinary live/archive inputs DVA's sweep rejected; resolver terminal issue behavior is separately documented | verify: human — paired CE/DVA zone/archive fixtures and resolved-issue evidence in TASK-424
- [x] `tools/doccheck`'s `checkCardStatus` sweep and status table are removed after the shared gate covers them | verify: human — TASK-424 selected strict policy, removed the local sweep, `ce task validate --all` and `ce task gate` pass, and `make doc-check` passes as the repository-owned documentation/source gate

## 후속 (2026-09-24)

상류 live-zone 검사는 `0f0a5f3c`에 들어갔지만 archive·누락 status·alias 차이가
남아 [ISSUE-041](../../issue/041-upstream-status-validator-contract-parity.md)에 기록했다. CE
`dba2348b`에 repository-owned strict dialect가 통합된 뒤 TASK-424가 DVA에서 선택하고
paired status fixture, producer round-trip, 전체 보드 gate를 검증했다. 후속은
[TASK-424](../../done/424-validate-owns-zone-status.md)가 소유한다.
