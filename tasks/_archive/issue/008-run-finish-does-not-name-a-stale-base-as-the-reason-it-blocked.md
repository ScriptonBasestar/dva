---
id: ISSUE-008
title: "run-finish does not name a stale base as the reason it blocked"
type: bug
status: done
priority: P2
effort: S
exec-tier: standard
severity: medium
discovered-in: "2026-09-13 TASK-378 integration"
discovered-at: 2026-09-13
ownership: upstream
created: 2026-09-13
upstream-ref: "ce-agent-kit#2"
resolution: fixed
resolved-at: 2026-09-25T01:10:07Z
resolution-summary: "Resolved as fixed by TASK-438."
---

## Summary

`ce task run-finish` refused to integrate a task branch whose base had gone
stale, and the message it printed named neither the stale base nor the rebase
that fixes it. It reported a recovery failure and pointed at "gz-git
diagnostics" — a place where the answer was not.

## Reproduction

1. `ce task run-start <task> --type <type>` and commit work in the worktree.
2. From another session, integrate anything into the source branch so its tip
   moves past the task branch's base.
3. `ce task run-finish <task>` from the primary checkout.

Observed on 2026-09-13 with TASK-378 in this repository.

## Expected vs Actual

**Expected** — the verdict names the condition that caused the refusal: the
branch is based on a commit that is no longer the source tip, and the remedy
is a rebase in the worktree.

**Actual** — the verdict reports a recovery failure and directs the operator
to `gz-git` diagnostics, which say nothing about the base. The stale base is
never mentioned, though `run-finish` compared the two commits in order to
refuse. The incomplete-recovery message should remain for the case it actually
describes rather than be reused for a stale base.

## Evidence

TASK-378's branch was cut from `17949ec`. While the work was in progress
another session integrated eight commits, moving `master` to `b18f783`.
`run-finish` then said:

```text
BLOCKED familybook-readiness-landing-check: integration did not complete;
post-operation evidence failed: task recovery is incomplete
next: resolve gz-git diagnostics and retry
Error: integration did not complete; post-operation evidence failed: task
recovery is incomplete
```

Nothing here is false. "Task recovery is incomplete" is accurate — the
worktree and both branches were still present, because integration had not
happened. But it describes the *consequence*, and the `next:` line sends the
reader to diagnostics that do not mention a base at all. The one fact that
would have ended the investigation — this branch is based on a commit that is
no longer the tip — is absent.

Recovering it took a separate round of `git log`, `git merge-base
--is-ancestor` against three commits, and a `git worktree list`, none of which
`run-finish` needed the operator to run: it had already compared the two
commits itself in order to refuse.

After `git rebase origin/master` in the worktree and a force-with-lease push,
the identical `run-finish` invocation returned `DONE` and reclaimed the
worktree and both branches.

### What this evidence does not establish

The rebase was not the only change between the two invocations — the branch
was also force-pushed, so its remote ref moved. The stale base is strongly
implied as the cause (it is the condition that existed before and not after,
and the force-push was a consequence of the rebase, not an independent fix)
but it was not isolated. Whoever fixes this should reproduce with a stale base
and no force-push before settling on the diagnosis.

## Impact

The failure is not rare: it happens whenever two sessions integrate into the
same source branch, which is the normal condition for parallel task work in
this repository — the concurrent session that moved `master` here was doing
exactly the work this board asks for.

The cost is not the extra rebase, which is required and correct. It is that
the message steers away from it. An operator who trusts `next:` goes looking
at `gz-git` state, finds nothing wrong, and is left with a lifecycle command
that refuses without a reason. That is the shape of failure that gets worked
around with a manual merge, which is exactly what the lifecycle commands exist
to prevent.

`CLAUDE.md` already prescribes the fix ("Before resuming an existing
unattended task branch, rebase it in its worktree onto the current tip of the
configured source branch"). The rule holds; the tool that detects the
violation does not cite it.

## Recommended Resolution

Have `run-finish` report the stale base as its own condition, distinct from
recovery failure, and name both commits and the remedy. Something with the
shape of:

```text
BLOCKED <task>: base is stale — branch is based on 17949ec, master is at b18f783
next: rebase in the worktree onto origin/master, push, and retry run-finish
```

Two things to preserve. The command should keep refusing rather than rebasing
on its own: a rebase can conflict, and CLAUDE.md requires stopping on
conflicts rather than silently choosing a side. And the existing recovery
message should stay for the case it actually describes — a genuinely
incomplete reclaim — instead of being repurposed as the message for both.

## 소유권 — 상류다 (2026-09-15 명시)

`run-finish`의 verdict 메시지는 `ce` 런타임 소유다 — Notes가 "cannot be fixed in this
repository"로 적는다. 기준 전부가 메시지 변경이므로 이 저장소가 고칠 부분은 없다.
보고는 [[TASK-399]]가 `ce-agent-kit#2`로 수행했다.

## Resolution Criteria

- [x] `run-finish` distinguishes a stale base from an incomplete recovery in its
      verdict line | verify: human — TASK-438 stale and exit-3 cleanup fixtures retain their distinct verdicts
- [x] the stale-base message names the branch's base commit, the source tip, and
      the rebase remedy | verify: human — TASK-438 evidence records both SHAs and the exact rebase nextAction
- [x] `run-finish` still refuses rather than rebasing on the operator's behalf |
      verify: human — TASK-438 regression coverage proves run-finish never invokes rebase
- [x] the diagnosis is reproduced with a stale base and no force-push | verify: human — the isolated fixture records source/task commits and ancestry without force-pushing either branch

## Notes

This is a `ce` task-runtime defect and cannot be fixed in this repository; it
belongs with [[ISSUE-005]], [[ISSUE-006]] and [[ISSUE-007]] as upstream work.
It is filed here because this board is where the evidence was produced.

## Follow-up (2026-09-25)

TASK-425 is done for the narrower stale-base reason, rebase advice, and real
no-force-push reproduction. Its evidence was captured after the fix landed, so
the earlier timing phrase “before the fix lands” cannot be re-established and
has been removed from the criterion. The remaining commit-identity and
no-automatic-rebase regression work belongs to
[TASK-438](../../done/438-name-both-commits-in-stale-base-finish-verdict.md).
The independent review of its first SHA-bearing implementation (`c3e54bda`)
found that run-exit-3 cleanup failure can be reclassified as stale-base and
the changed reason no longer matches a suppression-prefix check. Commit
`7046479e` fixes both; rebased commit `71b1d169` passed exact-HEAD full CI and
is integrated to CE master/origin. The disposable-fixture evidence is recorded
under TASK-438 and its independent DVA done-review passed. ISSUE-042 is resolved.
The separate post-cleanup inventory recovery gap remains ISSUE-043.
