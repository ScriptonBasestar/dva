---
id: ISSUE-001
title: "Task runtime cannot review legacy done cards without verification evidence"
type: bug
status: todo
priority: P1
effort: S
exec-tier: strong
severity: medium
discovered-in: "Task runtime cannot review legacy done cards without verification evidence"
discovered-at: 2026-09-10
created: 2026-09-10
---

## Summary

The task-management controller cannot apply an evidence-backed `done-review`
to legacy DVA done cards that predate controller-issued completion receipts.
`TASK-312` has a reproducible implementation and a fresh independent full-test
review, but its frontmatter lacks `verification-evidence`; the controller
rejects the review while trying to create its review receipt. The same legacy
shape can prevent the remaining soft done backlog from reaching disposition.

## Reproduction

1. In a clean worktree at `c90bb07`, select `TASK-312` with
   `task_management.engine.operate.entry_interpreter.select_task`; it selects
   `done-review`.
2. Supply a valid `pass` proposal with `quality-review`,
   `quality-reviewed-at`, and `quality-review-evidence` after independently
   rerunning `dva test`.
3. Apply that proposal through `entry_interpreter.py`.

## Expected vs Actual

- Expected: the controller records the independent review evidence and adds a
  canonical `quality-review-receipt`, or it supplies a documented migration
  path for the legacy completion record.
- Actual: `_materialize_task()` reads the absent `verification-evidence` as a
  required string and rejects the transition with `verification-evidence must
  be text` before the review receipt can be written.

## Evidence

- Independent review: implementation commit `4f267fc` contains the DryRun
  health-wait guard and `TestUpDryRunSkipsEntryHealthWait`; `dva test` passed
  on 2026-09-10.
- Direct-controller attempts and timing records are in the ignored
  `tmp/task-management/direct/queue-run/` directory of the review worktree.

## Resolution Criteria

- [ ] `TASK-312` can receive a `done-review` verdict without fabricating a
  historical completion receipt | verify: human — a task-management source
  fix or documented migration procedure is linked here and a fresh controller
  run records the receipt
- [ ] The compatibility rule is exercised against a legacy done card and a
  current controller-created done card | verify: human — upstream test or
  executable receipt is linked here
