---
id: TASK-633
title: "health: handle intended foreign listeners and noise in the port owner check"
type: feature
priority: P3
effort: S
exec-tier: standard
status: todo
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

- [ ] An entry can declare that its health port is served by another process and skip the owner mismatch, documented in USAGE.md. | verify: human — 설정 테스트와 USAGE 설명을 독립 리뷰가 확인
- [ ] Remote health hosts do not emit an owner-unverified warning on every `up`. | verify: human — internal/lifecycle 테스트를 독립 리뷰가 확인

## Notes

- Also from review: a fixture test for the Linux `/proc/net/tcp` parser and end-to-end tests of
  `waitEntryReady`/`ProcessPlugin.Status` on a mismatch.
