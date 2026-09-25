---
id: TASK-424
title: "Make ce task validate own the zone status rule"
type: bug
priority: P2
effort: M
exec-tier: standard
quality-review: pass
quality-reviewed-at: 2026-09-25
quality-review-evidence: "Independent fresh done-review PASS by /root/rereview_task424 at 2026-09-25T04:42:20Z; confirmed strict policy selection, durable paired evidence, intentional resolved-issue exception, retained identity/duplicate-ID/filename checks, and restored trailing-comment regression test."
status: done
created: 2026-09-24
---

## Summary

상류 `0f0a5f3c`는 live zone의 unknown, empty, zone-mismatch status를 거부한다.
독립 리뷰 결과, CE 기본 validator는 status 없는 카드, alias, frozen archive를 허용하는
호환 계약도 가진다. 이를 기본값에서 바꾸면 CE의 카드 생성기와 기존 저장소를 깨뜨리므로
ISSUE-041은 저장소가 명시적으로 선택하는 strict status policy를 추가하는 방향으로
조정했다.

CE 기본 동작과 opt-in strict 동작을 각각 검증했다. DVA가 opt-in을 선택하고 live/archive
status fixture와 카드 생성·이동·archive·issue resolve 경로가 왕복 통과한 뒤 DVA의
`checkCardStatus` 복사본을 제거했다. CE strict mode는 resolver가 만든 `issue/`의
`status: done` + nonempty `resolution:`도 canonical terminal state로 허용한다. 이전 DVA
복사본은 이 출력을 거부했으므로 이 차이는 evidence에 명시했다. 보드 readiness 판단은
공유 `ce task gate`가 소유하며, `make doc-check`는 DVA 저장소의 문서·소스 검사만 유지한다.

## Completion Criteria

- [x] Shared validator rejects unknown, empty, and zone-mismatched status values in live zones | verify: `ce-agent-kit` commit `0f0a5f3c`; `internal/usecase/task/validator_status_test.go` covers the three cases
- [x] CE default behavior keeps status-less cards, documented aliases, and frozen archives valid | verify: human — upstream compatibility tests
- [x] An explicit repository-owned strict policy rejects missing/non-canonical live status and invalid `archive/` plus `_archive/` status | verify: human — upstream policy tests and full archive-corpus validation
- [x] DVA selects that policy; `new`, `move`, archive, and issue-resolve producers round-trip through it; paired fixtures agree on ordinary live/archive zone literals | verify: human — CE/DVA fixtures, producer round trips, and the resolved-issue terminal-state ruling in `tasks/done/evidence/TASK-424/strict-status-parity-20260925.md.txt`
- [x] Remove DVA `checkCardStatus` only after the selected CE policy passes those checks; retain identity and duplicate-ID/filename checks | verify: human — see the linked parity evidence for focused tests, strict board validation, `ce task gate`, and `make doc-check`

## Out of scope

- Shared board readiness verdicts in `make doc-check`; `ce task gate` owns those verdicts.

## Sources

- [ISSUE-007](../_archive/issue/007-ce-task-validate-does-not-constrain-the-status-field.md)
- [ISSUE-041](../_archive/issue/041-upstream-status-validator-contract-parity.md)

## 2026-09-25 implementation review

CE task branch `dev/codex/mbp/fix/validator-status-parity` commit `cd196a22` adds
`card-dialect.strict-status: true`; default CE compatibility remains unchanged.
Independent review failed it: nested `archive/<zone>/...` and `_archive/<zone>/...`
cards are resolved as live workflow zones, and producer/CLI round-trip coverage is
incomplete. Keep the DVA guard until those findings are fixed and re-reviewed.

## 2026-09-25 follow-up review

Commit `5a42af9e` fixes archive precedence, nested archive fixtures, default alias/frozen
archive compatibility, strict `ValidateAll`, and strict `new` round trips. Re-review still
fails the agreed completion contract: strict validation is not run over actual `move` and
archive/resolve producer output; CLI tests cover strict validate only, not `TaskNew`,
`TaskMove`, and `TaskArchive` round trips. CE TASK-328 also lacks completed exact-HEAD CI
evidence. Keep `checkCardStatus` until those tests, CI, and independent review pass.

## 2026-09-25 producer/CLI re-review PASS

Commit `96bb3674` added strict CLI round trips for `new → move → archive`,
`new issue → resolve`, and backlog `new`, followed by `validate --all`. Independent review
confirmed these paths, prior archive/default compatibility fixes, and exact implementation
CI run `6cbd6430691d0f8010b035c3909afcdd` (7m26s PASS, unchanged attestation). The run
evidence is in CE commit `6274dd9f`. CE source commit `dba2348b` now contains the implementation
and review card on master/origin; its task worktree and branch were reclaimed by `run-finish`.
At that point DVA strict-policy selection and paired parity were still pending.

## 2026-09-25 DVA strict-policy parity

DVA now selects `card-dialect.strict-status: true` in root `ce-tasks.yaml`. CE source
`dba2348b` builds cleanly; strict status fixtures cover each live zone and both archive
spellings, reject missing/empty/alias/unknown/mismatched values, and producer CLI fixtures
round-trip `new → move → archive` and `new issue → resolve`. The pre-removal DVA status fixture
suite passed on the base source; paired evidence is linked above. The DVA doccheck status sweep
was removed while preserving identity-zone resolution and duplicate ID/filename coverage.
Strict validation accepts the full DVA board (492 valid, 0 invalid), lint is clean, and the
shared board gate is READY after resolved ISSUE-024, ISSUE-007, and ISSUE-041 were archived and
PLAN-011 counts were updated. Before ISSUE-024 resolution, the legacy local guard reported 488 checked cards and 0
status mismatches. After CE `task resolve` wrote the canonical `status: done` plus
`resolution: fixed` pair, the old DVA guard reported that one status mismatch; CE strict
validator intentionally accepts it, while CE lint classifies a resolved issue still in `issue/`
as stray and asks for archive. The issue was then archived and the board is clean. The initial
independent code review requested durable paired fixture evidence, which is now linked above.
A later review caught the missing trailing-comment duplicate-ID regression test; that test is
restored alongside the quoted-hash test, `go test ./tools/doccheck` passes, and fresh independent
re-review is pending.

## DVA review attempts

- 2026-09-25 independent code review (`/root/review_task424_doccheck`): **FAIL** because paired
  CE/DVA fixture results were not durably linked. Added
  `tasks/done/evidence/TASK-424/strict-status-parity-20260925.md.txt` with revisions, exact
  commands, corpus results, and the intended resolved-issue status difference.
- 2026-09-25 independent final review (`/root/final_review_task424`): **FIX NEEDED**. The review
  found that the existing trailing-comment duplicate-ID regression had been replaced by the
  quoted-hash check. Restored both as distinct tests and reran `go test ./tools/doccheck` (PASS,
  1.735s). Filed ISSUE-044 for the separate CE reference/test naming gap. Fresh independent
  re-review is pending.
- 2026-09-25 independent fresh done-review (`/root/rereview_task424`): **PASS** at
  `2026-09-25T04:42:20Z`. Confirmed all criteria, durable paired evidence, bounded resolver
  exception, identity/duplicate coverage, and restoration of the trailing-comment regression.
