---
id: TASK-628
title: "validate: warn when a process entry health check probes another entry's port"
type: feature
priority: P3
effort: S
exec-tier: standard
status: done
delivered-by: 401225f1
quality-review: pass
quality-review-evidence: "ce review-evidence 922c35a55e08d2ba118be64201e82f81720215e679824780a1cd715d4e480695"
created: 2026-11-22
allowed-paths: [internal/config/, internal/cli/, internal/lifecycle/port_guard.go, USAGE.md, CHANGELOG.md, tasks/]
---

## Summary

Since 3e2d3927 (TASK-538), `ProcessPlugin.Up` refuses to start when an http/tcp health check
address already accepts connections. A process/native entry whose health check targets a
dependency (for example a DB on 5432) therefore fails `up` whenever that dependency runs. The
behaviour is documented in USAGE.md but only surfaces at `up` time. `dva validate` should warn
before then.

## Completion Criteria

- [x] `dva validate` emits a warning naming the entry and address when a process/native entry's http/tcp health check resolves to a port that another stack entry declares or probes. | verify: `go test ./internal/config -run HealthPortShared -count=1`
- [x] Entries whose health check targets their own port produce no warning. | verify: `go test ./internal/config -run HealthPortNoWarning -count=1`

## Decision (2026-11-22, user)

- A1: use only information already in dva.yml — other entries' http/tcp health check addresses
  and docker `ports:` host ports (legacy `docker:` block or `runners.docker`). No new port field.
- Compose file `ports:` are out of scope: validation does no file I/O (TASK-248) and dva has no
  static compose parser. Asked and confirmed by the user.

## Outcome

- `HealthCheckConfig.Endpoint` (internal/config/health_endpoint.go) now owns the host:port
  resolution; `lifecycle.healthCheckAddress` delegates to it so the `up` guard and `validate`
  agree on the port. That is why `internal/lifecycle/port_guard.go` joined `allowed-paths`.
- Ports are compared by number only, so `localhost:8080` and `127.0.0.1:8080` collide.
- No warning fires on the repository's own dva.yml or the 11 example/fixture yml files.

## Review Attempts

### 1. 2026-11-22 — reviewer: 독립 opus 서브에이전트(읽기 전용), executor: 구현 실행(401225f1)

- 기준 전부 pass. 근거: `ce review-evidence 922c35a5…`.
- 포트 가드 리팩터는 동작 보존. 비차단: 포트 번호만 비교, 플랜 간 경고 소음, native·legacy docker 테스트 공백.
- 다음 전환: done.
