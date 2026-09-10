---
id: ISSUE-001
title: "Review pipeline cannot migrate legacy done cards into CE-compatible durable receipts"
type: bug
status: todo
priority: P0
effort: M
exec-tier: strong
severity: medium
discovered-in: "TASK-312 done-review and TASK-354 gate currentization"
discovered-at: 2026-09-10
created: 2026-09-10
---

## Summary

The current review pipeline has three separate compatibility breaks between
the workbook controller and the installed CE validator. It cannot turn legacy
DVA done cards into review receipts that are both truthful and accepted by
`ce task validate`:

1. Without `ce-tasks.yaml`, the controller uses its legacy dialect and requires
   `verification-evidence` to be a path to an existing file. `TASK-312` has no
   field and fails with `verification-evidence must be text`; `TASK-344` and
   `TASK-371` do have the field, but its value is prose rather than a file and
   therefore fails as `review-evidence-invalid`.
2. The controller hashes a PyYAML-normalized frontmatter document. CE 0.8.4
   validates `reviewed-card-sha256` against its own canonical JSON
   serialization. A controller receipt can therefore be well-formed JSON and
   still fail CE's digest check.
3. The controller writes the receipt under `tmp/task/...`. DVA ignores `tmp/`,
   so that artifact is not a durable repository receipt even if it validates
   in the worktree that created it.

As of 2026-09-10 this is an active repository gate blocker, not only a backlog
disposition defect. At DVA HEAD `af7f6e6`, installed CE 0.8.4 (`8034cc4`),
`ce task gate --json` exits 1 at `validate`: 65 of 67 cards are valid and the
two failures are `TASK-344` and `TASK-371`, both done blockers with no canonical
`quality-review-receipt`. `TASK-371` directly blocks `TASK-354`.

## Reproduction

1. At DVA `af7f6e6`, run `ce task gate --json`; it returns
   `status: not-ready`, `summary: task_validate_failed`, and
   `failed_step: validate`.
2. Run `ce task validate --all`; it reports exactly two invalid cards:
   `TASK-344` blocks `TASK-343` and `TASK-371` blocks `TASK-354`, but neither
   declares a `quality-review-receipt`.
3. The underlying legacy-controller reproduction remains: in a clean worktree at `c90bb07`, select `TASK-312` with
   `task_management.engine.operate.entry_interpreter.select_task`; it selects
   `done-review`.
4. Supply a valid `pass` proposal with `quality-review`,
   `quality-reviewed-at`, and `quality-review-evidence` after independently
   rerunning `dva test`.
5. Apply that proposal through `entry_interpreter.py`.

## Expected vs Actual

- Expected: the controller records the independent review evidence and adds a
  CE-compatible `quality-review-receipt` at a tracked durable path, or the two
  runtimes supply a documented migration path for the legacy completion
  record.
- Actual for `TASK-312`: `_materialize_task()` reads the absent
  `verification-evidence` as a required string and rejects the transition with
  `verification-evidence must be text` before the receipt can be written.
- Actual for `TASK-344` and `TASK-371`: their existing prose
  `verification-evidence` is interpreted as a file path in the legacy dialect
  and is rejected as `review-evidence-invalid`. Enabling the other dialect
  would still leave the controller/CE digest mismatch and ignored `tmp/` path.

## Evidence

- Current board verdict: `ce task gate --json` exits 1 with
  `task_validate_failed`; `ce task validate --all` reports 65 valid and 2
  invalid cards out of 67, naming only `TASK-344` and `TASK-371`.
- Both failed cards carry prose `verification-evidence` but no controller-issued
  receipt. Adding a hand-written receipt would fabricate the review provenance
  the validator is designed to require, so this issue records the blocker
  instead.
- Source comparison confirms that the workbook controller's
  `review_subject_sha256()` hashes sorted PyYAML plus the body, while CE 0.8.4's
  `canonicalCardDigest()` hashes sorted canonical JSON, a separator, and the
  body. CE's source explicitly states that workbook-produced digests are not
  compatible.
- The controller fixes its output path at `tmp/task/<TASK-ID>/...`; the durable
  contract documented by the workbook is
  `tasks/receipts/<TASK-ID>/done-review-<sha>.json`, tracked by Git.
- Independent review: implementation commit `4f267fc` contains the DryRun
  health-wait guard and `TestUpDryRunSkipsEntryHealthWait`; `dva test` passed
  on 2026-09-10.
- Direct-controller attempts and timing records are in the ignored
  `tmp/task-management/direct/queue-run/` directory of the review worktree.

## P0 Blocker

- `p0_reason`: the shared board gate is red, so TASK-354 cannot truthfully claim
  readiness or attach that verdict to an integration runner. Removing the
  `blocks` edges or inventing receipts would only hide the missing reviews.
- `owner`: joint external ownership. `ce-agent-kit` owns the validator's
  canonical digest and a compatible migration/issuance interface;
  `ce-workbook/task_management` owns the review issuer, legacy evidence
  handling, and durable output path. A separate DVA reviewer owns each actual
  review verdict; TASK-354 owns none of those verdicts.
- `next_action`: the two runtime owners define and test one receipt contract:
  the issuer obtains or computes CE's canonical digest, accepts an explicit
  fresh-review evidence mode for legacy cards whose completion evidence is
  absent or prose, and writes the machine receipt to a tracked path such as
  `tasks/receipts/<TASK-ID>/done-review-<sha>.json`. After that support lands,
  independent reviewers perform fresh reviews of TASK-344 and TASK-371 and
  commit the generated receipts and human-readable review records. No receipt
  is reconstructed from an assumed historical review.
- `next_check`: `ce task validate --all` reports 67 valid and 0 invalid, then
  `ce task gate --json` exits 0 with `status: ready`. Each blocking done card
  names a readable Git-tracked receipt under `tasks/receipts/` whose
  `reviewed-card-sha256` matches CE's canonical card digest. Upstream fixtures
  cover all three shapes: missing evidence (`TASK-312`), prose evidence
  (`TASK-344`/`TASK-371`), and a current controller-created card.

## Resolution Criteria

- [ ] `TASK-312` can receive a fresh `done-review` verdict despite absent
  completion evidence, without claiming that a historical completion receipt
  existed | verify: human — a joint runtime fix or documented migration
  procedure is linked here and a fresh controller run records the review
- [ ] Missing, prose, and current file-backed completion-evidence shapes all
  produce CE-compatible canonical digests | verify: human — upstream tests for
  both the ce-agent-kit validator contract and ce-workbook issuer are linked
  here
- [ ] New review receipts are written to a durable tracked location rather
  than remaining under ignored `tmp/` | verify: human — upstream issuance test
  and migration responsibility name `tasks/receipts/<TASK-ID>/`
- [ ] `TASK-344` and `TASK-371` carry genuine controller-produced review
  receipts under `tasks/receipts/` and the DVA board is ready | verify: `ce task gate --json`
