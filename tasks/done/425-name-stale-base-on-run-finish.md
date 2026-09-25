---
id: TASK-425
title: "Name a stale base when run-finish refuses to integrate"
type: bug
priority: P2
effort: S
exec-tier: standard
status: done
created: 2026-09-24
quality-review: pass
quality-reviewed-at: 2026-09-25
quality-review-evidence: "독립 done-review PASS: 격리 fixture의 실제 run-finish 출력이 BLOCKED, stale source-tip 사유, worktree rebase 조치를 기록하며 no-force-push SHA 증거와 링크가 있다."
---

## Summary

stale base를 이유로 명명하는 좁은 동작은 `ce-agent-kit#2`에서 구현됐다.
더 넓은 [ISSUE-008](../_archive/issue/008-run-finish-does-not-name-a-stale-base-as-the-reason-it-blocked.md)은
두 commit SHA 노출과 추가 재현 검증을 남기며, 이 후속 범위는 TASK-438에 둔다.

방향: 거부 이유가 stale base임을 말하고, 다음 행동으로 워크트리 안 rebase를
적는다. gz-git 진단으로 보내지 않는다.

## Completion Criteria

- [x] stale base 거부가 기준 커밋이 소스 팁이 아님을 말하고 rebase를 다음 행동으로 적는다 | verify: human — 그 문구가 있는 재현 로그가 이 카드에 링크된다

## Out of scope

- rebase를 `run-finish`가 대신 실행하는 일.

## Sources

- [ISSUE-008](../_archive/issue/008-run-finish-does-not-name-a-stale-base-as-the-reason-it-blocked.md)

## Current evidence

- Integrated implementation: `ce-agent-kit@8f2ed4514e010ec61cae35ec18d6fa8ea4ea85b8`, `internal/usecase/taskruntime/service_lifecycle.go:249-252`.
- Stale/current-base branch test: `internal/usecase/taskruntime/integration_provider_test.go:354`.
- Actual stale-base finish capture: [2026-09-25 fixture output](evidence/TASK-425/stale-base-run-finish-2026-09-25.txt), run with `ce-agent-kit@05b0b4ae3c18517df2d7635f3b185da4ee3d9baf` using `ce task run-finish task-425-stale-base --json`. The isolated fixture advanced only `master`; `run-finish` returned `BLOCKED`, named the source-tip mismatch, and directed a rebase. Independent done-review: PASS.

The broader ISSUE-008 follow-up TASK-438 will add the commit identities to the
verdict and strengthen the no-force-push fixture. TASK-425 only closes this
card's narrower wording and evidence requirement.
