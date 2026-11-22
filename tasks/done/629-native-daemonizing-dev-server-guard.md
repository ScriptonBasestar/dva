---
id: TASK-629
title: "process: detect native dev servers that daemonize and escape tracking"
type: feature
priority: P3
effort: S
exec-tier: standard
status: done
delivered-by: 3a3c64a6
quality-review: pass
quality-review-evidence: "ce review-evidence 8ade547761a5ca2d6c6bc47a9c526a24f9d66f5e2693ddfa5afdb7a97364e408"
created: 2026-11-22
allowed-paths: [internal/lifecycle/, USAGE.md, CHANGELOG.md, tasks/]
---

## Summary

In the TASK-538 pilot (airadartoday-web), Astro's dev server backgrounded itself by default.
The tracked pid exited, and the real server kept running untracked until it was killed by PID.
`ASTRO_DEV_BACKGROUND=0` worked around it. dva should notice when the tracked process exits
while its health check endpoint still answers, and report it instead of claiming the entry
stopped or failed.

## Completion Criteria

- [x] When the tracked process exits but the entry's health endpoint keeps answering, `up`/`status` report an "escaped from tracking" error naming the address. | verify: `go test ./internal/lifecycle -run Escaped -count=1`
- [x] USAGE.md documents the case and the foreground setting for known dev servers (Astro `ASTRO_DEV_BACKGROUND=0`). | verify: human — USAGE.md의 해당 설명을 독립 리뷰가 확인

## Decision (2026-11-22, user)

- B1: an escaped server is reported as an error and never killed. The only identity left is
  the port, and a port is not proof of ownership; no name- or pattern-based kills.

## Outcome

- `up` (composition readiness wait, internal/lifecycle/composition_wait_ready.go): when the
  tracked pid is dead and a http/tcp health address answers — either on the liveness tick or
  right after the probes pass — the wait fails with `escapedError` naming pid and address.
- `status` (ProcessPlugin.Status): a pidfile whose pid is dead while a health address answers
  reports state `error` with `extra.escaped`; the text table prints a `!` line.
- The readiness wait functions moved out of composition_orchestrator.go (over the 500-line
  limit) into composition_wait_ready.go unchanged apart from the new check.
- Limit: a server that daemonizes only after the probes pass while its parent is still alive
  is not caught by `up`; the next `status` reports it.

## Review Attempts

### 1. 2026-11-22 — reviewer: 독립 opus 서브에이전트(읽기 전용), executor: 구현 실행(3a3c64a6)

- 기준 전부 pass. 근거: `ce review-evidence 8ade5477…`.
- 대기 함수 이동은 동작 보존, error 상태는 소비자를 깨지 않는다. 비차단: dva stop 뒤 무관한 프로세스가 포트를 잡으면 status가 escaped로 오보 — TASK-632로 이관.
- 다음 전환: done.
