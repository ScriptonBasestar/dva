---
id: TASK-431
title: "Align host golangci-lint and worktrunk pins"
type: chore
priority: P2
effort: S
exec-tier: standard
needs-human: true
status: todo
created: 2026-09-24
---

## Summary

[ISSUE-026](../issue/026-host-tool-pins-drift-from-the-versions-the-toolchain-expects.md)은
2026-09-24에도 재현된다. PATH의 첫 golangci-lint는 mise shim을 거쳐 2.12.2이고,
aqua 2.13.2는 그 뒤에 있다. devenv mise config의 `"cargo:worktrunk"`는
`latest`다. 고치는 곳은 devenv 소스 체크아웃의 mise config와 PATH다.
설치된 `~/devenv` 스냅샷은 수정하지 않는다. 보고는 ce-agent-kit#6이지만
착지는 devenv다.

방향: aqua 2.13.2가 PATH에서 먼저 잡히게 하고, worktrunk 핀을 정확한 버전으로
고정한다. 이 저장소의 lint 설정을 그 버전에 맞춰 낮추지 않는다.

## Completion Criteria

- [ ] PATH의 첫 golangci-lint가 aqua 2.13.2다 | verify: human — devenv 소스에서 배포한 뒤 `which -a golangci-lint`의 첫 줄이 aqua 경로인지 확인한다
- [ ] mise의 worktrunk 핀이 `latest`가 아니다 | verify: human — devenv 소스의 mise config에서 `cargo:worktrunk` 값이 정확한 버전이다

## Out of scope

- 설치된 `~/devenv`를 직접 고치는 일.
- DVA의 golangci 설정을 2.12.2에 맞추는 일.

## Sources

- [ISSUE-026](../issue/026-host-tool-pins-drift-from-the-versions-the-toolchain-expects.md)
