---
status: done
---

# TASK-438 disposable runtime evidence

Generated from CE source commit `7046479e71d04b14d17a7e60f8612bb8f751d52b`.
The CE binary was built with `go build -o tmp/task-438-runtime-evidence/bin/ce ./cmd/ce`.
After rebase, the equivalent implementation at `71b1d1699313414013062b0d2bc9c44af13fadb4`
passed exact-HEAD full CI (`6a9e362bf6b20ff9176d889f37c68978`,
`build/logs/ci-20260925-095448-1132.log`) and was integrated to `master` and
`origin/master`.

Both scenarios used fresh local bare repositories and independent control/task
worktrees. `GIT_CONFIG_GLOBAL` and `GIT_CONFIG_SYSTEM` were `/dev/null`; no
DVA runtime, DVA origin, or force push was used.

## Stale-base scenario

The fixture ran `ce task run-start stale --type test --json`, committed and
pushed the task branch, pushed a separate commit to local `origin/master`, and
then ran `ce task run-finish stale --json` from the control checkout.

Expected: exit `1`, `status: BLOCKED`, both base/source SHA values in the
reason, and `nextAction` equal to `rebase onto the source tip in the worktree,
then retry run-finish`.

Observed: all assertions passed. The JSON field is camelCase `nextAction`.
`receipt.baseHead`, `receipt.sourceHeadBefore`, and `receipt.taskHead` match the
captured identity files; `gz-git integrate check` exited `1`; and the task
worktree path remained present before and after.

Evidence: `stale-start.json`, `stale-status-before.json`,
`stale-status-after.json`, `stale-finish.json`, `stale-finish.exit`,
`stale-before-finish.txt`, `stale-after-finish.txt`,
`stale-path-before.txt`, `stale-path-after.txt`, and `stale-identities.txt`.

## Incomplete-recovery scenario

The fixture ran `ce task run-start cleanup --type test --json`, committed and
pushed the task branch, then made the task worktree directory read-only before
running `ce task run-finish cleanup --json`. This produced a genuine provider
cleanup failure while leaving the task directory on disk. Worktrunk removed its
inventory entry, so the subsequent `run-status` is blocked on an inventory
mismatch; that recovery gap is tracked separately by ISSUE-043. A nested
repository was also attempted, but did not trigger this provider failure by itself.

Expected: exit `3`, receipt reason `integration succeeded but task recovery
cleanup failed`, no stale-base rebase advice, and `gz-git integrate run` exit
`3` in diagnostics.

Observed: all assertions passed. The receipt reports `sourcePushed: true`,
while `worktreeRemoved`, `localBranchRemoved`, and `remoteBranchRemoved` are
all `false`. `test -e` and `test -d` remained true for the recorded task
worktree path before and after. The snapshots record control checkout, Git
worktree registration, local/remote task refs, and task HEAD. The provider
diagnostic contains the exact `Permission denied` cleanup error.

Evidence: `cleanup-start.json`, `cleanup-status-before.json`,
`cleanup-status-after.json`, `cleanup-finish.json`, `cleanup-finish.exit`,
`cleanup-before-finish.txt`, `cleanup-after-finish.txt`,
`cleanup-path-before.txt`, `cleanup-path-after.txt`, and
`cleanup-identities.txt`.

## Verification and cleanup

The fixture runner performed Python assertions over both JSON receipts,
including status, reason, `nextAction`, exit codes, SHA identity, provider
diagnostics, recovery flags, and path preservation. A second independent
assertion pass over the saved evidence also passed.

The exact temporary root was listed and validated after capture, then removed:
`tmp/task-438-runtime-evidence/fixture-root/`. The temporary CE binary and
runner were removed as well. Only this tracked evidence directory remains.
