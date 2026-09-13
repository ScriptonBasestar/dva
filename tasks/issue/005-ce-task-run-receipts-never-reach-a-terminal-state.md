---
id: ISSUE-005
title: "ce task run-* receipts never reach a terminal state"
type: bug
status: todo
priority: P1
effort: M
exec-tier: standard
severity: medium
discovered-in: "2026-09-13 board currentization review"
discovered-at: 2026-09-13
created: 2026-09-13
---

## Summary

`ce task run-finish` retires a worktree and reports integration success, but the
execution receipt it was retiring stays in the run registry unchanged. The
registry therefore accumulates entries that describe work already integrated,
and `ce task run-list` refuses to report anything at all because it reads that
accumulation as blocked executions. The lifecycle commands are the declared
mutation path for every CE-adopting repository, so an operator has no
first-party way to see which executions are live.

The receipt schema is the root cause, not the accumulation: an entry carries
`task`, `type`, `source`, `owner`, `branch`, `worktree`, `startedAt`, and
`updatedAt` — and no terminal-state field. There is nowhere for `run-finish` to
record that it finished. `updatedAt` cannot stand in for one; it equals
`startedAt` on every entry, so it is a creation timestamp under another name.

Owner is external. `run-*` and its registry belong to ce-agent-kit and
ce-workbook/task_management; DVA owns only the cards it files. This is filed
here on the ISSUE-004 precedent — a DVA-observed defect in the shared runtime
gets a DVA card so the observation is not lost, and the fix lands upstream.

## Reproduction

1. In this repository, run `ce task run-list --json`.
2. Read `executions[]`: 20 entries on 2026-09-13.
3. Test each `worktree` path for existence: 19 of the 20 no longer exist. The
   one that does is the execution making the observation.
4. Compare `updatedAt` to `startedAt` on each: equal on all 20.
5. Run `ce task run-list` without `--json`: it prints
   `BLOCKED: one or more task executions are blocked` and exits non-zero,
   printing no execution at all.

Two of the 19 are `suppression-review-followups` and `record-309-verdict`,
whose `run-finish` invocations reported integration and recovery as completed
in this same session. Their commits are on `master` (`1e15275`, `99680ac`) and
their branches and worktrees are gone. The work is done; only the receipt
disagrees.

## Expected vs Actual

- Expected: a successful `run-finish` marks its receipt terminal, and
  `run-list` distinguishes live executions from retired ones.
- Actual: no field can express the distinction. Every receipt ever created is
  still live by the registry's own reading, and `run-list` degrades to a
  single BLOCKED line that names nothing.

## Impact

Two failures, and the second is the worse one:

- The registry cannot answer "what is running now", which is the question it
  exists to answer.
- BLOCKED is indistinguishable between a stale receipt and a genuinely
  abandoned worktree that needs recovery. A real abandonment arrives as one
  more line in a backlog of 19 false ones, so the signal that should stop an
  operator is the signal they have learned to skip.

## Recommended Resolution

Add an explicit terminal state to the receipt and have `run-finish` and
`run-abort` write it. A derived liveness check — "the worktree path is gone,
therefore it finished" — is not equivalent: it cannot tell integration from a
worktree someone deleted by hand, and it reads the answer off the filesystem
the receipt exists to be independent of.

Retiring the 19 existing entries is migration, not fix; a fix that leaves them
is still a fix, and this repository will accept whichever ordering upstream
prefers.

## Resolution Criteria

- [ ] The run receipt schema carries an explicit terminal-state field, and `run-finish` and `run-abort` set it | verify: human — upstream schema and tests are linked here
- [ ] `ce task run-list` reports live executions in this repository and exits 0 with no live execution outstanding | verify: human — fresh output from this repository is linked here
- [ ] BLOCKED names the executions it is blocked on | verify: human — upstream output distinguishes a stale receipt from an abandoned worktree
