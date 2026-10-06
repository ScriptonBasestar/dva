---
id: TASK-495
title: "Treat empty cloudflared access token stdout as unauthenticated"
type: bug
priority: P2
effort: S
exec-tier: standard
execution-mode: implementation
needs-human: false
allowed-paths: [internal/lifecycle/tunnel.go, internal/lifecycle/tunnel_test.go, internal/cli/doctor_tunnel.go, internal/cli/doctor_tunnel_test.go, docs/68-remote-access-tunnel.md]
status: todo
created: 2026-10-05
---

## Summary

[ISSUE-488](../_archive/issue/488-measure-expired-tunnel-token-and-live-doctor.md) 실측에서
cloudflared 2026.9.3의 `access token --app`은 만료 토큰에 **exit 0**과 빈 stdout을 돌려주고
토큰 파일을 지웠다. [docs/68](../../docs/68-remote-access-tunnel.md) §3.1 1단계와 두 구현은
종료 코드만 본다. 그래서 만료 토큰을 인증됨으로 판정한다.

- `internal/lifecycle/tunnel.go` `runCloudflaredDiscard` — interactive 준비 판정이 로그인을 건너뛴다.
- `internal/cli/doctor_tunnel.go` `checkTunnelAuthState` — doctor가 만료 토큰을 `[pass]`로 보고한다.

인증됨은 종료 코드 0 **그리고** stdout 바이트 수 > 0이다. stdout은 바이트 수만 세고 내용은
버리며 로그·오류 메시지에 넣지 않는다.

## Steps

1. `internal/lifecycle/tunnel.go:160` `runCloudflaredDiscard` — exit 0이고 stdout 바이트 수가 0이면 미인증이다. TTY가 없으면 로그인 명령을 안내하고 실패한다.
2. `internal/lifecycle/tunnel_test.go:179` `TestTunnelAuthInteractiveNoTTY`는 유지한다. 빈 stdout 회귀 `TestTunnelAuthExpiredTokenEmptyStdout`를 이 파일에 추가한다.
3. `internal/cli/doctor_tunnel.go:68` `checkTunnelAuthState` — 같은 조건에서 Access token 행은 `[FAIL]`이다. stdout 내용은 출력하지 않는다.
4. `internal/cli/doctor_tunnel_test.go:150` `TestDoctorTunnelInteractiveAuthFailure`는 유지한다. `TestDoctorTunnelAuthEmptyStdout`를 이 파일에 추가한다.
5. `docs/68-remote-access-tunnel.md:61` §3.1 1단계가 종료 코드와 비어 있지 않은 stdout을 함께 요구하게 한다.

## Stop conditions

- 토큰 바이트를 로그, 오류, 테스트 출력에 남기지 않는다.
- exit 0만으로 인증됨을 판정하지 않는다. `.lock`이 남은 exit 1 경로는 그대로 둔다.
- 이 카드의 Completion Criteria 문구를 구현 없이 체크하지 않는다. quality-review는 리뷰 전에 두지 않는다.
- 터널 인증 밖의 파일은 고치지 않는다.

## Completion Criteria

- [ ] 가짜 `cloudflared`가 exit 0과 빈 stdout을 내면 interactive 준비 판정이 미인증으로 처리하고 TTY 없이는 로그인 명령을 안내하며 실패한다 | verify: `go test -count=1 -v -run '^TestTunnelAuth' ./internal/lifecycle/ 2>&1 | /usr/bin/grep -q '^--- PASS: TestTunnelAuthExpiredTokenEmptyStdout '`
- [ ] 같은 조건에서 `dva doctor`의 Access token 행이 `[FAIL]`이고, 출력에 가짜 토큰 바이트가 없다 | verify: `go test -count=1 -v -run '^TestDoctorTunnel' ./internal/cli/ 2>&1 | /usr/bin/grep -q '^--- PASS: TestDoctorTunnelAuthEmptyStdout '`
- [ ] docs/68 §3.1 1단계가 종료 코드와 비어 있지 않은 stdout을 함께 요구한다 | verify: `awk '/^### 3.1/{p=1} /^### 3.2/{p=0} p' docs/68-remote-access-tunnel.md | /usr/bin/grep -q 'stdout이 비어 있지 않'`
- [ ] 문서 게이트가 통과한다 | verify: `make doc-check` (regression-guard)

## Notes

- 기존 `TestTunnelAuthInteractiveNoTTY`, `TestDoctorTunnelInteractiveAuthFailure`는 비영 종료만 다룬다. 둘 다 유지한다.
- `.lock`이 남은 만료 토큰은 exit 1이라 이미 미인증으로 처리된다.
