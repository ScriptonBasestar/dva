---
id: TASK-533
title: "Scope CI config loading to the selected owner"
type: bug
priority: P2
effort: M
exec-tier: strong
allowed-paths:
- internal/config/load.go
- internal/config/subproject_test.go
- internal/cli/ci.go
- internal/cli/ci_test.go
- internal/cli/root.go
- docs/53-ci-profiles.md
- CHANGELOG.md
- tasks/issue/534-ci-subproject-run-resolves-unrelated-imports.md
status: done
quality-review: pass
quality-reviewed-at: 2026-10-26
quality-review-evidence: tasks/done/evidence/TASK-533/independent-review.json
created: 2026-10-26
---

## Summary

Root-owned CI profiles consume only the root CI config and, for `--subproject`, a declared child name passed to the steps. They should not require unrelated child config files to resolve imported plans or interactions. Keep ordinary config loading strict, preserve root parsing errors, and let `--project` load only its selected child config.

## Completion Criteria

- [x] Root CI loads its own profile without opening unrelated imported child configs, while --project loads only the selected child and reports a missing selected child. | verify: `go test ./internal/cli -run "TestCIRootProfileSkipsUnrelatedImportedChildren|TestCIProjectLoadsOnlySelectedChild" -count=1`
- [x] Default config loading continues validating all imported subprojects, and CI-specific loading still rejects malformed root YAML. | verify: `go test ./internal/config -run "TestLoadConfigWithImportedMissingSubproject|TestLoadSkipSubprojectImportResolution" -count=1`

## Attempts

- 2026-10-26 codex main: independent source review PASS. Recovery check attempt 1 failed because exporting GOFLAGS caused Make's trimpath option to reach path-sensitive tests. Attempt 2 with GOFLAGS unset passed build, full race tests and lint (0 issues); doc-check failed because the new docs/53 text exceeded 10240 bytes. Condensed only that documentation to 9994 bytes, obtained independent document review PASS, and reran make doc-check successfully. Logs: tmp/task-533-recovery-check.log, tmp/task-533-recovery-check-attempt2.log, tmp/task-533-final-doc-check.log. No source code changed after the passing race tests/lint.

- 2026-10-26 independent review: rejected the initial root-validation test oracle because it expected unknown YAML fields to be rejected, which is not a `Load` contract. Replaced it with malformed YAML, which the existing parser rejects; production behavior was not changed.
- 2026-10-26 codex: scoped Go regression run exit 1. `internal/config` passed; the CLI root-only dry-run test failed because its second JSON decode reused the prior scoped preview value, so an omitted `subproject` field left the old value in memory. Reset the fixture result before decoding and rerun the same scoped command.
- 2026-10-26 codex: reran the same scoped command after resetting the preview; both `internal/config` and `internal/cli` passed. Output: `tmp/task-533-scoped-tests-attempt2.log` (exit 0; shell PID 60532, exec session 48855).
- 2026-10-26 codex: a delayed successful `ce task new` invocation followed by a retry created duplicate draft TASK-534 beside TASK-533. The main session removed TASK-534 through its CE workflow; TASK-533 is the retained card and the reservation ledger was left unchanged.
