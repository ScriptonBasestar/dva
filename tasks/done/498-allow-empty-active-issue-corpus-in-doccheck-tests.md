---
id: TASK-498
title: "Allow an empty active issue corpus while preserving inventory coverage"
type: bug
priority: P1
effort: S
exec-tier: strong
execution-mode: implementation
needs-human: false
allowed-paths: [tools/doccheck/upstreamref_test.go]
status: done
quality-review: pass
quality-review-evidence: tasks/done/evidence/TASK-498/independent-review.json
created: 2026-10-07
---

## Summary

TASK-496 final gate failed the TASK-494 binding after all three issues were legitimately
resolved and archived. TestUpstreamRefsSweepsTheRealCorpus requires Seen/Read > 0,
although a valid board may have zero active issues. Preserve the real inventory sweep
and all ownership/ref failure fixtures; remove the workload quota only.

## Steps

1. tools/doccheck/upstreamref_test.go:249–281: retain repoRoot and LoadInventory. Require
   the tracked tasks/README.md sentinel in that inventory so an absent task inventory
   cannot silently pass. Count eligible regular Markdown tasks/issue entries from the
   loaded inventory and require Seen and Read to equal that count, including zero.
   Require no read errors and retain Unclassified, Mismatched, Unrefed assertions;
   also require Unreasoned zero. Update the comment and honest sweep log.
2. tools/doccheck/upstreamref_test.go:60: add TestUpstreamRefEmptyIssueCorpusIsValid.
   Use disposable absent/empty issue-directory fixtures with a valid nonissue Markdown
   inventory; include archived issue coverage if useful. Assert real Check succeeds
   and ownership/ref defect counters remain zero. No live files or issue mutation.
3. Keep all existing reported/unreported/split/local-only fixtures unchanged in meaning.
   Run targeted race tests, then separate grok-4.7 quality/done review. Coordinator
   runs declared commit CI and board gates, commits/pushes and CE integrates.
4. TASK-496 coordinator rebases on this source fix and reruns its empty-board gate.

## Stop conditions

- Do not change production checkUpstreamRefs, LoadInventory, ownership policy or severity.
- Do not skip the real inventory test or weaken negative failure assertions.
- Do not retain a fake issue to make the board nonempty.
- Do not edit README, TASK-494 historical card or TASK-493 seals.
- No self review, synthetic CE receipt, secrets, network service or Git lifecycle edits.

## Completion Criteria

- [x] New empty-corpus regression exists and passes; baseline lacks this named function | verify: `/usr/bin/grep -q '^func TestUpstreamRefEmptyIssueCorpusIsValid(' tools/doccheck/upstreamref_test.go && go test -count=1 -race -run '^TestUpstreamRefEmptyIssueCorpusIsValid$' ./tools/doccheck/`
- [x] Real sweep compares exact eligible inventory rather than requiring live issues | verify: `/usr/bin/grep -q 'tasks/README.md' tools/doccheck/upstreamref_test.go && go test -count=1 -race -run '^TestUpstreamRef' ./tools/doccheck/`
- [x] Separate final review passes | verify: `python3 -c 'import json,pathlib; d=json.loads(pathlib.Path("tasks/done/evidence/TASK-498/independent-review.json").read_text()); assert d["verdict"]=="PASS" and d["sessionId"]'`

## Attempts

- Discovery: TASK-496 final doc-check and validate passed; ce task gate failed the
  TASK-494 binding with seen/read 0/0 after resolving ISSUE-453/490/497. Treat as
  a task-related regression, not a baseline failure exemption. TASK-496 integration
  waits for this test-only correction.

## Verification

Independent Grok 4.7 01a111ce-bad3-7852-8b9a-fd37ca154144 PASS; implementation
01a111c9-eb7d-7430-b28b-b23f4dd43b6c is a different session. Targeted race tests
exit 0. CI 86694121893b0dd207493b28ebedb1fa commit succeeded (2m14.420692167s),
with process-local KUBECONFIG removed to avoid known host configuration influence.
Only the allowed test file changed; existing negative fixtures and production are intact.
Actual sweep in this task worktree has 2 active issues. TASK-496 performs final
zero-active-issue live sweep after rebase, before its integration.
