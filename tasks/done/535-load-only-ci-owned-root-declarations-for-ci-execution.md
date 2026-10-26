---
id: TASK-535
title: "Load only CI-owned root declarations for CI execution"
type: bug
priority: P1
effort: S
exec-tier: strong
status: done
quality-review: pass
quality-reviewed-at: 2026-10-26
quality-review-evidence: tasks/done/evidence/TASK-535/independent-review.json
created: 2026-10-26
---

## Summary

TASK-533 skipped child imports but still finalized unrelated root lifecycle
declarations. A real Singlepageweb dry-run then failed because `local-dev`
composed `textcove`, whose imported plan was intentionally not loaded.
Introduce an explicit CI-owned load mode: preserve root discovery, decoding,
merges, version/defaults, environment input declarations and CI validation;
leave lifecycle/import/remote/plan/endpoint finalization to normal full loading.
Keep default Load fully strict and preserve selected-child file/profile errors.

## Completion Criteria

- [x] CI loads root profiles with imported composition aliases absent, while ordinary loading stays strict. | verify: `go test ./internal/config -run 'TestLoadForCI|TestLoadConfigWithImportedMissingSubproject' -count=1`
- [x] Root CI and selected child CI preserve their separate ownership. | verify: `go test ./internal/cli -run 'TestCIRootProfileSkipsUnrelatedImportedChildren|TestCIProjectLoadsOnlySelectedChild' -count=1`

## Attempts

- 2026-10-26 main: `nice -n 19 env -u GOFLAGS GOMAXPROCS=2 make recovery-check` passed completely: build, full race/coverage tests, vet/format/lint (0 issues), and all documentation gates. Raw log: tmp/task-535-recovery-check.log. No implementation changes followed this run; only this verification record was appended.

- 2026-10-26: `/Users/archmagece/.local/bin/dva ci predeploy --subproject singlepageweb-calculator --dry-run` in Singlepageweb failed on the missing imported `textcove` plan after TASK-533 integration (aa63ffb2). The prior focused and full tests lacked a root composition alias. Preserve this failure rather than treating TASK-533 as actual product success.
- 2026-10-26: static investigation and independent design review confirmed root CI consumes root vars, environment, env_file and CI profiles, not lifecycle plans/stack/remote/endpoints. Implementation and regression verification in progress.
- 2026-10-26: explicit ForCI mode implemented; focused config/CLI regressions passed (attempt 2, after correcting the expected existing CI timeout error text). Independent source review PASS; reviewer did not rerun tests. Selected child retains its existing file/profile checks, not unrelated full lifecycle finalization. Built binary dry-run in the actual Singlepageweb checkout passed, producing the expected site-bindings plan without deployment. Full recovery verification follows serially after the host's current test run. Logs: tmp/ci-root-composition-scoped-attempt2.log and tmp/ci-root-composition-build.log; actual product dry-run: /tmp/dva-task-535-product-dry-run.log.
