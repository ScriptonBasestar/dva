---
id: TASK-605
title: "compose ps에 -a를 붙여 죽은 프로젝트를 not ready로 보고"
type: feature
priority: P2
status: todo
created: 2026-11-11
---

## Summary

compose 상태 조회가 `docker compose ps --format json`을 `-a` 없이 실행해, 컨테이너가 모두
종료·크래시한 프로젝트는 서비스가 0개로 보였고 `dva compare status`의 readiness가 `not ready`가
아니라 `-`로 나왔다. 두 plugin이 공용 `composePsArgs`로 ps 인자를 만들고, docker compose는
`ps -a --format json`으로 멈춘 컨테이너까지 조회한다. podman-compose는 `ps`가 이미 내부에서
`podman ps -a`를 쓰고 자체 파서에 `-a`가 없어(1.6.0 `compose_ps_parse`) 플래그를 붙이지 않는다.
실행 중인 서비스 옆에서 exit 0으로 끝난 one-shot(init·migration)은 이전처럼 목록에서 빠져 정상
프로젝트를 `not ready`로 만들지 않고, 아무것도 실행 중이 아니면 모든 컨테이너를, 비0 종료는
항상 보고한다.

## Completion Criteria

- [ ] 종료된 컨테이너만 남은 compose 프로젝트의 상태 조회가 서비스 목록을 반환해 readiness가 not ready가 된다 | verify: `go test ./internal/lifecycle/ -run 'TestComposePsIncludesStopped' -count=1`
- [ ] docker compose와 podman-compose의 ps 인자가 같은 규칙을 쓰고 기존 lifecycle 테스트가 통과한다 | verify: `go test ./internal/lifecycle/ -count=1`
