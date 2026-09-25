---
id: ISSUE-041
title: "Align upstream status validator with canonical card contract"
type: bug
status: todo
priority: P2
effort: S
severity: medium
ownership: upstream
upstream-ref: "ce-agent-kit#3"
discovered-in: "TASK-424 independent review of ce-agent-kit commit 0f0a5f3c"
discovered-at: 2026-09-25
created: 2026-09-25
---

## Summary

Commit `0f0a5f3c` adds live-zone status checks to the shared validator, but the
public validation path still skips archived cards, accepts a missing `status`,
and accepts aliases such as `pending`, `open`, or `completed`. DVA's current
`checkCardStatus` contract requires explicit canonical literals and checks its
archive zones. Removing that local check now would leave these cases without a
shared verdict. The shared validator is owned by ce-agent-kit#3.

## Reproduction

1. Validate a live-zone card with no `status` field; the shared validator accepts it.
2. Validate a card with an alias such as `status: pending`; the shared validator accepts the alias.
3. Put an invalid or statusless card under an archive directory and run public validation; the archive path is skipped.
4. Run DVA `make doc-check` against the same fixtures; its canonical status sweep rejects them.

## Expected vs Actual

- Expected: the shared validator applies the canonical contract to live and archived cards: `status` is required and only the exact status permitted by the zone is accepted.
- Actual: missing status is optional, aliases are normalized, and public validation skips archive paths.

## Resolution Criteria

- [ ] The shared validator rejects missing status, aliases, and archive mismatches under the canonical zone rules | verify: human — upstream regression tests cover each case
- [ ] An isolated CE build rejects the same fixtures as DVA's current status contract | verify: human — paired validator/doccheck fixture output is linked here
- [ ] TASK-424 removes the duplicated local board-status sweep only after shared parity is verified | verify: human — removal commit and passing repository doc checks are linked

## Recommendation

Complete the shared contract in ce-agent-kit#3, then remove the repository-local
copy through TASK-424. Until parity is demonstrated, preserve the existing
guard and keep TASK-424 blocked on this issue.

## Related

- [ISSUE-007](007-ce-task-validate-does-not-constrain-the-status-field.md)
- [TASK-424](../blocked/424-validate-owns-zone-status.md)

## 소유권 — 상류

`ce task validate`와 공유 readiness gate의 규칙은 ce-agent-kit#3이 소유한다.
DVA는 동일 입력에서 계약 parity를 검증하고, 상류가 전체 범위를 충족한 뒤
TASK-424를 통해 로컬 중복 검사를 제거한다.
