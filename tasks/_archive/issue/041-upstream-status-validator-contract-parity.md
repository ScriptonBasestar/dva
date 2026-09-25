---
id: ISSUE-041
title: "Add opt-in repository status policy to upstream validator"
type: bug
status: done
priority: P2
effort: S
severity: medium
ownership: upstream
upstream-ref: "ce-agent-kit#3"
discovered-in: "TASK-424 independent review of ce-agent-kit commit 0f0a5f3c"
discovered-at: 2026-09-25
created: 2026-09-25
resolution: fixed
resolved-at: 2026-09-25T04:45:56Z
resolution-summary: "Resolved as fixed by TASK-424."
---

## Summary

Commit `0f0a5f3c` adds live-zone status checks to the shared validator. Independent
review found that CE's default dialect intentionally permits missing status and
documented aliases, while canonical frozen archives are skipped. Those defaults are
used by existing repositories and CE's own card producers; changing them globally
would break compatibility. The shared validator is owned by ce-agent-kit#3.

Add a repository-owned opt-in strict status policy. Keep the CE default behavior
unchanged; when selected, require exact zone status literals, validate both archive
directory spellings, and ensure card producers round-trip through that policy. DVA selected
and verified this policy in TASK-424, then removed its local status sweep. CE also deliberately
accepts its resolver-produced `issue/ + status: done + nonempty resolution:` terminal state;
the old DVA table allowed only `todo` under `issue/`, so it rejected that canonical output.
TASK-424 records the behavior and explains that CE lint asks to archive the resolved issue.

## Reproduction

1. Validate a status-less live card and a documented alias using CE's default dialect; both remain compatible.
2. Select the repository-owned strict policy and validate the same fixtures; missing status and aliases are rejected.
3. Under strict policy, validate archived cards in both `archive/` and `_archive/`; invalid statuses are rejected, and producers round-trip successfully.

## Expected vs Actual

- Expected: CE default compatibility remains intact; repositories that opt in get exact live/archive status validation and producer parity.
- Actual when filed: CE master did not expose the opt-in contract; the initial task branch
  had archive classification and producer/CLI round-trip gaps. Those gaps were corrected,
  independently reviewed, and integrated into CE master `dba2348b`.

## Resolution Criteria

- [x] Default-dialect regression tests preserve status-less cards, aliases, and frozen archive behavior | verify: human — upstream tests
- [x] Opt-in strict policy rejects missing/non-canonical live status and validates `archive/` plus `_archive/` corpus | verify: human — upstream CLI and corpus tests
- [x] `new`, `move`, and archive/resolve producers round-trip under opt-in policy; paired DVA fixtures agree on ordinary live/archive zone literals, and the resolver-owned terminal issue exception is documented | verify: human — CE producer/status fixture suites and TASK-424 paired evidence
- [x] TASK-424 removes the local sweep only after DVA opts in and parity passes | verify: human — root `ce-tasks.yaml`, removed `checkCardStatus`, strict DVA board validation, and repository doc checks

## Recommendation

Keep strictness as an explicit ce-agent-kit repository policy while preserving the
default dialect. DVA has opted in and moved the board verdict to the shared gate; it no longer
keeps a duplicate local status table.

## Related

- [ISSUE-007](../../issue/007-ce-task-validate-does-not-constrain-the-status-field.md)
- [TASK-424](../../review/424-validate-owns-zone-status.md)

## 소유권 — 상류

`ce task validate`와 공유 readiness gate의 규칙은 ce-agent-kit#3이 소유한다.
DVA는 동일 입력에서 계약 parity를 검증하고, 상류가 전체 범위를 충족한 뒤
TASK-424를 통해 로컬 중복 검사를 제거한다.

## 2026-09-25 독립 재검토 — producer round trip 미완료

재검토에서 `5a42af9e`의 archive precedence와 default compatibility 수정은 통과했지만,
완료 기준은 아직 충족되지 않았다. mover/archive 테스트는 status를 기록하는지만
확인하고 그 결과를 strict validator로 다시 읽지 않는다. strict CLI coverage도 validate
경로에 한정되어 `TaskNew`, `TaskMove`, `TaskArchive` 실제 경로의 왕복을 확인하지 않는다.
TASK-328에도 exact-HEAD CI 완료 증거가 없다. 상류에 이 검증을 추가 요청했고, DVA의
`checkCardStatus`는 producer parity와 CI, 독립 재검토가 끝날 때까지 유지한다.

## 2026-09-25 producer/CLI 보완 및 재리뷰 PASS

CE `96bb3674`는 strict 설정을 실제 CLI로 읽어 `new → move → archive`,
`new issue → resolve`, backlog `new`를 수행한 뒤 `validate --all`로 결과를 다시
검증한다. resolved issue는 resolution metadata가 있을 때만 `status: done`으로
strict validation을 통과한다. 독립 리뷰는 nested archive 분류, `archive/`와
`_archive/` corpus, 기본 status-less/alias/frozen archive 호환성도 재확인해 PASS했다.
Exact implementation CI run `6cbd6430691d0f8010b035c3909afcdd`는 7m26s PASS이며
attestation before/after가 같다. 증거는 CE TASK-328에 기록했다. CE source commit
`dba2348b`가 implementation과 review card를 master/origin에 통합·push했고 해당 task
worktree와 branch 회수도 완료했다.

상류 구현 계약은 채워졌고, DVA는 TASK-424에서 strict policy를 선택해 paired fixture와
전체 보드 검증을 마쳤다. 이전 DVA guard는 일반 zone literal에서는 일치하지만 CE
resolver의 terminal issue 출력은 거부한다. TASK-424는 전체 old/new corpus가 일치한다고
주장하는 대신 이 차이를 paired evidence에 기록한다. 마지막 `make doc-check`와 독립
TASK-424 done-review PASS를 확인해 이슈를 resolve한다.
