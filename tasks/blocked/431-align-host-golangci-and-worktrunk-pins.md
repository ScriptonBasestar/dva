---
id: TASK-431
title: "Align host golangci-lint and worktrunk pins"
type: chore
priority: P2
effort: S
exec-tier: standard
needs-human: true
status: blocked
created: 2026-09-24
---

## Summary

[ISSUE-026](../issue/026-host-tool-pins-drift-from-the-versions-the-toolchain-expects.md)의
현행 측정에서 `which -a` 첫 항목은 mise shim이지만, `mise which golangci-lint`는
Aqua 2.13.2를 선택하고 실행 버전도 2.13.2다. PATH 재정렬은 필요하지 않다.
devenv source pin은 정확한 worktrunk 0.74.0으로 고쳤고 master에 통합했다.
남은 일은 기존 live `config/docker/config.json`의 로컬 자격 증명 drift를 보존하면서
통합 소스를 설치하는 것이다. 설치기가 drift를 거부해 사람의 처분 결정을 기다린다.

## Completion Criteria

- [x] mise shim이 Aqua 2.13.2를 선택하고 실행된 golangci-lint가 2.13.2를 보고한다 | verify: human — `mise which golangci-lint`와 `golangci-lint --version` 결과를 이 카드에 기록한다
- [ ] 통합된 source의 worktrunk 핀 0.74.0이 live install에 안전하게 반영된다 | verify: human — drift를 보존한 설치가 끝난 뒤 설치본 config와 `mise which worktrunk`를 확인한다

## Out of scope

- 설치된 `~/devenv`를 직접 고치는 일.
- DVA의 golangci 설정을 2.12.2에 맞추는 일.

## Sources

- [ISSUE-026](../issue/026-host-tool-pins-drift-from-the-versions-the-toolchain-expects.md)

## Current evidence

- Integrated source commit: `ce-devenv@7273dd62d9eaced4428fe769a4f1cdff7f3e4c2a`; only `cargo:worktrunk = "latest"` changed to `"0.74.0"`.
- `mise which golangci-lint` resolves to `aqua-golangci-golangci-lint/2.13.2`; `golangci-lint --version` reports 2.13.2. `which -a` starts with the mise shim, so its first raw PATH entry is not the resolved binary.
- `ce task run-finish` integrated and pushed the source commit, removed the task branch/worktree, and reported baseline make-check/make-lint failures as non-worsening.
- `make install` refused because live `config/docker/config.json` contains local Docker Hub credential entries absent from source. No force was used; live worktrunk config remains uninstalled.
