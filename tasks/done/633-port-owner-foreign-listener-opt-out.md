---
id: TASK-633
title: "health: handle intended foreign listeners and noise in the port owner check"
type: feature
priority: P3
effort: S
exec-tier: standard
status: done
delivered-by: 4f2b443f
quality-review: pass
quality-review-evidence: "ce review-evidence a8b24d697964b25b2e8ee526a381d90e593900488552391251ae3de5776c34ce"
created: 2026-11-22
allowed-paths: [internal/lifecycle/, internal/config/, USAGE.md, CHANGELOG.md, tasks/]
---

## Summary

TASK-631 (bc76a5be) fails `up` when a local health port's listener is outside the tracked
process group. A process entry that runs `docker run -p …` or `docker compose up` in the
foreground is served by Docker Desktop's backend on macOS, outside the group, so `up` fails
with no opt-out. Remote health hosts also log an "owner unverified" warning on every `up`.
Found by the TASK-631 independent review.

## Completion Criteria

- [x] A health check can declare `owner: external` (the only accepted value; others fail `dva validate`), which skips the owner comparison and is shown in `status` as declared, documented in USAGE.md. | verify: `go test ./internal/config -run HealthCheckOwner -count=1`
- [x] With `owner: external`, a foreign listener no longer fails `up` or marks `status` error, end to end. | verify: `go test ./internal/lifecycle -run PortOwner -count=1`
- [x] Remote health hosts do not emit an owner-unverified warning on every `up` (debug log) but stay listed as unverified in `status`. | verify: `go test ./internal/lifecycle -run PortOwnerRemote -count=1`
- [x] The Linux `/proc/net/tcp` parser has a fixture test. | verify: human — linux 빌드 태그 테스트, macOS에서는 `GOOS=linux go vet`로 컴파일만 확인

## Outcome

- `HealthCheckConfig.Owner` + schema enum `["external"]` (top-level and stack entries), merged like other fields.
- `portOwnerReport` gains `External` and `Remote`; `extra()` builds status keys (`port_owner_external` new).
- Mutation check: removing the external skip, the remote split, the external-first ordering, or remote in status extra each fails a test.

## Notes

- Also from review: a fixture test for the Linux `/proc/net/tcp` parser and end-to-end tests of
  `waitEntryReady`/`ProcessPlugin.Status` on a mismatch.

## Review Attempts

### 1. 2026-11-23 — reviewer: 독립 opus 서브에이전트(읽기 전용), executor: 구현 실행(4f2b443f, 6d830e75)

- 기준 전부 pass. 근거: `ce review-evidence a8b24d69…`.
- 전체 리뷰(ac49403d)는 판정 로직·스키마 두 위치·병합·e2e·Linux fixture·문서를 확인. 비차단 지적 중 stack 엔트리 거부 테스트 공백은 6d830e75로 보강했고, 그 커밋과 rebase(range-diff 동일)는 별도 델타 리뷰가 확인. manifest에 `owner` 미표시는 승인 범위 밖이라 남김.
- 통합 전 master의 ISSUE-627 `ownership:` 누락으로 doc-check가 실패해 749ff02c로 먼저 바로잡았다.
- 다음 전환: done.
