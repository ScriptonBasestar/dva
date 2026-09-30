---
id: TASK-459
title: "Implement cloudflared tunnel prerequisite for remote entries"
type: feature
priority: P2
effort: M
exec-tier: standard
allowed-paths: [internal/config, internal/lifecycle, internal/cli, docs, examples, tasks, USAGE.md, ARCHITECTURE.md, CHANGELOG.md]
status: todo
created: 2026-09-30
---

## Summary

[docs/68](../../docs/68-remote-access-tunnel.md)의 `tunnel:` 접근 전제조건을
kubectl/helm 엔트리에 구현한다. interactive와 service-token 인증을 모두 지원한다.

## Completion Criteria

- [ ] `tunnel:` 스키마가 파싱·검증되고, provider가 cloudflared가 아니거나 service-token 모드에 `service_token_env`가 없거나, `local`이 루프백이 아니거나, kubectl/helm이 아닌 엔트리에 쓰이면 거부된다 | verify: `go test ./internal/config/ -v 2>&1 | /usr/bin/grep -q '^--- PASS: TestTunnelConfigValidation ('`
- [ ] interactive 모드에서 토큰이 없고 TTY도 없으면 로그인 명령을 안내하며 실패하고, JWT는 출력하거나 로그에 남기지 않는다 | verify: `go test ./internal/lifecycle/ -v 2>&1 | /usr/bin/grep -q '^--- PASS: TestTunnelAuthInteractiveNoTTY ('`
- [ ] 준비 판정은 인증과 TCP 연결을 모두 요구하고, 이미 사용 중인 포트는 충돌로 보고한다 | verify: `go test ./internal/lifecycle/ -v 2>&1 | /usr/bin/grep -q '^--- PASS: TestTunnelReadyRequiresAuthAndTCP ('`
- [ ] service-token 모드는 선언된 환경변수 값을 cloudflared에 넘기고, 비어 있으면 터널 시작 전에 실패한다 | verify: `go test ./internal/lifecycle/ -v 2>&1 | /usr/bin/grep -q '^--- PASS: TestTunnelAuthServiceToken ('`
- [ ] 같은 터널 선언은 한 번만 열고, DVA가 시작한 cloudflared 프로세스만 종료한다 | verify: `go test ./internal/lifecycle/ -v 2>&1 | /usr/bin/grep -q '^--- PASS: TestTunnelLifecycleOwnership ('`
- [ ] 만료된 토큰에서 `cloudflared access token --app`의 종료 코드를 실측해 docs/68 §7에 기록한다 | verify: human — docs/68 §7
- [ ] `dva doctor`가 cloudflared 설치 여부, interactive 인증 상태, service-token 환경변수 존재 여부를 값 없이 보고한다 | verify: human — run dva doctor on a tunnel config
- [ ] 문서 게이트가 통과한다 | verify: `make doc-check` (regression-guard)
- [ ] 전체 테스트와 lint가 통과한다 | verify: human — `make test && make lint` output is linked in Evidence (exceeds the 30s binding budget)

## Dependency

TASK-457 (design, docs/68) must be integrated first.
