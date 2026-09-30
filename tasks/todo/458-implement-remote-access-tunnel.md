---
id: TASK-458
title: "Implement cloudflared tunnel prerequisite for remote entries"
type: feature
priority: P2
effort: M
exec-tier: standard
allowed-paths: [internal/config, internal/lifecycle, internal/cli, docs, examples, tasks/todo]
status: todo
created: 2026-09-30
---

## Summary

[docs/68](../../docs/68-remote-access-tunnel.md)의 `tunnel:` 접근 전제조건을
kubectl/helm 엔트리에 구현한다. interactive와 service-token 인증을 모두 지원한다.

## Completion Criteria

- [ ] `tunnel:` 스키마가 파싱·검증되고, provider가 cloudflared가 아니거나 service-token 모드에 `service_token_env`가 없으면 거부된다 | verify: `go test ./internal/config/ -v 2>&1 | /usr/bin/grep -q -- '--- PASS: TestTunnel'`
- [ ] interactive 모드에서 토큰이 없고 TTY도 없으면 로그인 명령을 안내하며 실패하고, JWT는 출력하거나 로그에 남기지 않는다 | verify: `go test ./internal/lifecycle/ -v 2>&1 | /usr/bin/grep -q -- '--- PASS: TestTunnelAuth'`
- [ ] 준비 판정은 인증과 TCP 연결을 모두 요구하고, 이미 사용 중인 포트는 충돌로 보고한다 | verify: `go test ./internal/lifecycle/ -v 2>&1 | /usr/bin/grep -q -- '--- PASS: TestTunnelReady'`
- [ ] 만료된 토큰에서 `cloudflared access token --app`의 종료 코드를 실측해 docs/68 §7에 기록한다 | verify: human — docs/68 §7
- [ ] `dva doctor`가 cloudflared 설치 여부와 인증 상태를 값 없이 보고한다 | verify: human — run dva doctor on a tunnel config
- [ ] 전체 게이트가 통과한다 | verify: `make doc-check` (regression-guard)

## Dependency

TASK-457 (design, docs/68) must be integrated first.
