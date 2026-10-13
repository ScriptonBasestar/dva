---
id: ISSUE-510
title: "Docker agreement test changes daemon endpoint after its precondition"
type: bug
priority: P2
effort: S
exec-tier: standard
status: todo
severity: medium
ownership: local
discovered-in: "Docker agreement test changes daemon endpoint after its precondition"
discovered-at: 2026-10-14
created: 2026-10-14
---

## Summary

`internal/cli/doctor_docker_socket_test.go`의 `TestDockerSocketAndDaemonAgreeWhenDaemonReachable`는
실제 daemon 접근을 확인한 뒤 `DOCKER_HOST`를 빈 문자열로 바꾼다. 이때 Docker의
endpoint 해석이 달라지면 동일 daemon을 비교한다는 전제가 깨져 CLI 테스트가 실패한다.
소유자는 DVA의 테스트이며 제품 문맥 정리 범위에서는 등록만 하고 수정하지 않는다.

## 소유권 — 이 저장소다

이 저장소의 CLI 테스트가 전제 확인 뒤 입력 환경을 바꾸므로 DVA가 소유한다.
Docker나 Colima 설정을 변경하는 대신 테스트의 비교 전제를 후속 작업에서 수정해야 한다.

## Reproduction

2026-10-14 macOS에서 `DOCKER_HOST=unix:///Users/archmagece/.colima/default/docker.sock`,
`DOCKER_CONTEXT` 미설정 상태로 다음을 실행했다.

```bash
go test ./internal/cli -run '^TestDockerSocketAndDaemonAgreeWhenDaemonReachable$' -count=1
```

master `f83b4440`과 제품 문맥 변경 `b56cf013` 모두 exit 1이며
`precondition: checkDocker should pass when DockerDaemonReachable is true`를 보고했다.
두 소스 사이 이 테스트와 doctor 구현은 바뀌지 않았다.

```bash
DOCKER_CONTEXT=colima go test ./internal/cli -run '^TestDockerSocketAndDaemonAgreeWhenDaemonReachable$' -count=1
```

위 프로세스 한정 실행은 exit 0이다. 호스트의 Docker context나 환경 파일은 변경하지 않았다.
이는 현재 검증 환경의 명시적 선택이며 테스트 결함을 해결한 것은 아니다.

## Expected vs Actual

- Expected: 동일한 daemon endpoint에서 접근 결과를 비교하거나, 전제 조건을 변경 후 확인한다.
- Actual: 환경 변경 전에 통과한 probe를 변경 후에도 유효한 전제로 사용한다.
