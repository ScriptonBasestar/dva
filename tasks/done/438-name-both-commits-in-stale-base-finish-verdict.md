---
id: TASK-438
title: "Name both commits in stale-base run-finish verdicts"
type: bug
priority: P2
effort: M
exec-tier: standard
status: done
created: 2026-09-25
depends-on: [TASK-425, TASK-433]
quality-review: pass
quality-reviewed-at: 2026-09-25
quality-review-evidence: "Independent done-review PASS by /root/review_433_434: verified both fixture verdicts and identity evidence, regression coverage for stale Check/cleanup exit 3/no automatic rebase, exact-HEAD CI for 71b1d169, master/origin integration, and complete task worktree/branch reclaim. The separate ISSUE-043 recovery gap was closed by TASK-439."
---

## Summary

TASK-425 now names a stale source tip and directs the operator to rebase in the
task worktree. ISSUE-008 still requires the verdict to identify the exact merge
base and current source-tip commits. This card owns that remaining upstream
runtime change after TASK-425's stale-base contract and TASK-433's receipt
version stamp are integrated. Independent review of `c3e54bda` found that
run-exit-3 cleanup failures can be mislabeled as stale-base and the changed
reason text no longer matched a follow-up prefix check. Commit `7046479e`
corrected both paths; the rebased equivalent at
`71b1d1699313414013062b0d2bc9c44af13fadb4` passed independent review and exact-HEAD
full CI. That commit is integrated and pushed to CE `master`. ISSUE-042 tracks the
disposable-fixture acceptance. ISSUE-043 tracked the separate post-cleanup
Worktrunk inventory recovery gap revealed by the cleanup fixture and was closed
by TASK-439.

## Completion Criteria

- [x] A stale-base `run-finish --json` verdict and its receipt report the task branch base commit and current source-tip commit | verify: human — the tracked disposable fixture evidence matches both SHA values to Git's merge-base and current source ref
- [x] Regression tests preserve stale-base reason/rebase advice, distinguish incomplete-recovery output, and prove run-finish does not attempt `git rebase` | verify: `ce-agent-kit@71b1d169` tests stale Check failure and run-exit-3 cleanup failure separately; exact-HEAD full CI run `6a9e362bf6b20ff9176d889f37c68978` PASS (`build/logs/ci-20260925-095448-1132.log`)
- [x] Disposable fixtures capture both stale-base and incomplete-recovery `run-finish` verdicts without force-pushing or modifying the real runtime | verify: human — `tasks/done/evidence/TASK-438/` records both commands, receipts, pre/post execution identity, task path/ref state, and the observed partial cleanup

## Fixture evidence

- The stale fixture returned exit 1 and names base `a0e3fa51a3bcd57695195a611c9bdd61e94c7b1a` and source tip `b6f4d8bbc0470322daeaff74d4618bb8b8369a4f`. Its next action is `rebase onto the source tip in the worktree, then retry run-finish`; task identity and Git worktree/path/ref snapshots match before and after.
- The cleanup fixture returned exit 3 after local integration/push. Its reason remains `integration succeeded but task recovery cleanup failed`; cleanup flags are false, path exists before/after, and the post-failure `run-status` inventory mismatch is recorded under ISSUE-043.
- Both fixtures use local bare remotes and the reviewed CE implementation at `7046479e71d04b14d17a7e60f8612bb8f751d52b`; the equivalent rebased source `71b1d1699313414013062b0d2bc9c44af13fadb4` passed full CI. They do not touch the DVA runtime or origin and use no force push. Full commands and snapshots: [`TASK-438 evidence`](evidence/TASK-438/README.md).

The exact CI-tested commit `71b1d1699313414013062b0d2bc9c44af13fadb4` is now
`master` and `origin/master`. `ce task run-status` recorded `DONE`,
`sourcePushed=true`, and successful worktree/local-branch/remote-branch removal.

## Sources

- [ISSUE-008](../_archive/issue/008-run-finish-does-not-name-a-stale-base-as-the-reason-it-blocked.md)
- [TASK-425](425-name-stale-base-on-run-finish.md)
- [TASK-433](433-stamp-tool-version-on-gate-verdicts.md)
- [ISSUE-042](../_archive/issue/042-preserve-incomplete-recovery-verdicts.md)
- [ISSUE-043](../_archive/issue/043-run-finish-cleanup-failure-leaves-recovery-outside-worktrunk-inventory.md)
