---
id: TASK-433
title: "Stamp the tool version on gate verdicts"
type: bug
priority: P3
effort: S
exec-tier: standard
status: done
created: 2026-09-24
quality-review: pass
quality-reviewed-at: 2026-09-25
quality-review-evidence: "Independent done-review PASS: gate JSON and new run-finish receipts share the BuildToolVersion stamp; fallback tests and commit CI pass; TASK-433 integrated at 33deb918."
---

## Summary

[ISSUE-030](../issue/030-gate-verdicts-carry-no-tool-version-stamp.md)의 gate와 신규 run-finish 영수증 범위를 닫는다. 상류 `ce-agent-kit` master `33deb918`가 동일한 build identity를 두 판정에 기록한다. validate와 리뷰 영수증의 출처 표시는 ISSUE-030에 남아 있다.

방향: gate JSON과 신규 run-finish 영수증이 같은 도구 버전·revision을 기록한다. validate 결과와 리뷰 영수증의 남은 범위는 이 카드에서 다루지 않는다.

## Completion Criteria

- [x] `ce task gate --json`과 신규 run-finish 영수증에 도구 버전 필드가 있다 | verify: human — 상류 반영 후 이 저장소 출력에 그 필드가 있다

## Evidence

- `ce task gate --json` reports `tool_version` and `tool_revision`; the isolated current DVA board gate reports `v0.8.4-345-g5d70c9d8` / `5d70c9d84f34574488a6dd80ddd04d1ae2c9442f`.
- New finish receipts share the gate's build identity; regression tests cover describe, revision-only, and `unknown` fallback.
- Integrated source: `ce-agent-kit@33deb9182f5255bb4fb5dbe0df2f41e7786e4b97` on `master`.
- DVA commit CI run `3b7006a2dfcdb6d2817e4bbc6c17a548` passed in 6m25.047s with unchanged input attestation.
- Independent done-review: PASS. `ce task run-finish` integrated and pushed to `master`, then removed the task worktree and local/remote branch.

## Out of scope

- DVA 게이트 로그에 버전을 따로 받아 적는 래퍼.

## Sources

- [ISSUE-030](../issue/030-gate-verdicts-carry-no-tool-version-stamp.md)
