---
id: ISSUE-008
title: "run-finish does not name a stale base as the reason it blocked"
type: bug
status: todo
priority: P2
effort: S
exec-tier: standard
severity: medium
discovered-in: "2026-09-13 TASK-378 integration"
discovered-at: 2026-09-13
created: 2026-09-13
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
refuse.

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

## Resolution Criteria

- [ ] `run-finish` distinguishes a stale base from an incomplete recovery in its
      verdict line | verify: human — 두 조건을 각각 재현해 서로 다른 메시지가 나오는지 확인한다
- [ ] the stale-base message names the branch's base commit, the source tip, and
      the rebase remedy | verify: human — 메시지에 두 커밋과 조치가 모두 있는지 확인한다
- [ ] `run-finish` still refuses rather than rebasing on the operator's behalf |
      verify: human — 충돌하는 리베이스를 만들어 두고 run-finish가 스스로 처리하지 않는지 확인한다
- [ ] the diagnosis is reproduced with a stale base and no force-push before the
      fix lands | verify: human — 재현 절차와 결과가 이 카드에 기록되었는지 확인한다

## Notes

This is a `ce` task-runtime defect and cannot be fixed in this repository; it
belongs with [[ISSUE-005]], [[ISSUE-006]] and [[ISSUE-007]] as upstream work.
It is filed here because this board is where the evidence was produced.
