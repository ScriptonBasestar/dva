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

- [x] `tunnel:` 스키마가 파싱·검증되고, provider가 cloudflared가 아니거나 service-token 모드에 `service_token_env`가 없거나, `local`이 루프백이 아니거나, kubectl/helm이 아닌 엔트리에 쓰이면 거부된다 | verify: `go test ./internal/config/ -v 2>&1 | /usr/bin/grep -q '^--- PASS: TestTunnelConfigValidation ('`
- [x] interactive 모드에서 토큰이 없고 TTY도 없으면 로그인 명령을 안내하며 실패하고, JWT는 출력하거나 로그에 남기지 않는다 | verify: `go test ./internal/lifecycle/ -v 2>&1 | /usr/bin/grep -q '^--- PASS: TestTunnelAuthInteractiveNoTTY ('`
- [x] 준비 판정은 인증과 TCP 연결을 모두 요구하고, 이미 사용 중인 포트는 충돌로 보고한다 | verify: `go test ./internal/lifecycle/ -v 2>&1 | /usr/bin/grep -q '^--- PASS: TestTunnelReadyRequiresAuthAndTCP ('`
- [x] service-token 모드는 선언된 환경변수 값을 cloudflared에 넘기고, 비어 있으면 터널 시작 전에 실패한다 | verify: `go test ./internal/lifecycle/ -v 2>&1 | /usr/bin/grep -q '^--- PASS: TestTunnelAuthServiceToken ('`
- [x] 같은 터널 선언은 한 번만 열고, DVA가 시작한 cloudflared 프로세스만 종료한다 | verify: `go test ./internal/lifecycle/ -v 2>&1 | /usr/bin/grep -q '^--- PASS: TestTunnelLifecycleOwnership ('`
- [ ] 만료된 토큰에서 `cloudflared access token --app`의 종료 코드를 실측해 docs/68 §7에 기록한다 | verify: human — docs/68 §7
- [ ] `dva doctor`가 cloudflared 설치 여부, interactive 인증 상태, service-token 환경변수 존재 여부를 값 없이 보고한다 | verify: human — run dva doctor on a tunnel config
- [x] 문서 게이트가 통과한다 | verify: `make doc-check` (regression-guard)
- [x] 전체 테스트와 lint가 통과한다 | verify: human — `make test && make lint` output is linked in Evidence (exceeds the 30s binding budget)

## Evidence

- 2026-10-02, 워크트리 `claude__mbp__feat__task-459`에서 실측:
  - 카드 바인딩 5개 전부 PASS (`go test ./internal/config/ -v | grep '^--- PASS: TestTunnelConfigValidation ('` 및 lifecycle 4개 동일 패턴).
  - `make build` exit 0, `make doc-check` exit 0 (6개 도구 OK), `make lint` exit 0 (gofmt 537 files, ci-lint 0 issues), `make test` exit 0 (전체 패키지 PASS).
  - 단위 테스트 상세: `TestTunnelConfigValidation` 15 서브테스트(config 파싱·거부 테이블), lifecycle 4개 테스트 10 서브테스트(no-TTY 실패 + JWT 비누출, 준비=인증+TCP, service-token 전달/사전 실패, 소유·중복·정리·포트충돌), doctor 2개 테스트(설치 행 단독 실패, env 값 비누출).
- 인간 확인 대기: 만료 토큰 종료 코드 실측(docs/68 §7), 실제 터널 설정에서 `dva doctor` 실행.

## Dependency

TASK-457 (design, docs/68) must be integrated first.
