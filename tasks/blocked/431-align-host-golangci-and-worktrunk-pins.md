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
devenv source master/origin commit `7273dd62`에 `worktrunk` 핀 0.74.0이 통합·push됐고,
`ce task run-finish pin-worktrunk-0740`이 DONE으로 종료되어 task worktree와 branch가
회수됐다. 최종 readiness 비교에서 `make check` 실패 수는 22→22, `make lint`는 23→23이며
변경 경로 진단은 없었다. 이전 준비 실행의 22→21 측정 대신 이 최종 run-finish 결과를
정본으로 기록한다. 최초 전체 `make check`는 24 failed, 1627 passed, 6 skipped로
끝났고 실패는 설치기 fixture의 canonical marketplace identity, 낡은 CE binary,
upstream exclusion cache, CLAUDE.md prose-size 검사에 집중됐다.

live 설치에는 managed `config/docker/config.json` drift와 generated Claude/Codex 설정
drift가 남아 있다. non-force 설치가 이를 중단하며 force는 기존 자격 증명과 공유 설정을
덮어쓸 수 있으므로 실행하지 않았다.

## Completion Criteria

- [x] mise shim이 Aqua 2.13.2를 선택하고 실행된 golangci-lint가 2.13.2를 보고한다 | verify: human — `mise which golangci-lint`와 `golangci-lint --version` 결과를 이 카드에 기록한다
- [x] Source master pins worktrunk to 0.74.0 and task-relevant checks do not worsen the unchanged-source baseline | verify: human — integrated commit `7273dd62`; final `run-finish` readiness reports `make check` 22→22 and `make lint` 23→23 with no changed-path diagnostics
- [ ] 통합된 source의 worktrunk 핀 0.74.0이 기존 live 설정을 보존하며 설치된다 | verify: human — 지정 설치 결과와 설치본 config, `mise which wt`, `wt --version` 확인

## Out of scope

- 설치된 `~/devenv`를 직접 고치는 일.
- DVA의 golangci 설정을 2.12.2에 맞추는 일.

## Sources

- [ISSUE-026](../issue/026-host-tool-pins-drift-from-the-versions-the-toolchain-expects.md)

## Current evidence

- Integrated source master commit `7273dd62` pins `config/mise/config.toml` from `"latest"` to `"0.74.0"`; the isolated task diff contained only that config line.
- Full `make check`: 24 failed, 1627 passed, 6 skipped in 564.94s. Failing groups report stale CE binary revision (`5d70c9d`, 9 commits behind source), missing canonical ce-agent-kit marketplace identity in installer fixtures, old upstream file-size exclusions, and the CLAUDE.md prose-size threshold. None references worktrunk. This run did not include an unchanged-source baseline comparison; do not report it as passing.
- Read-only follow-up confirmed the branch diff is exactly this one config line and static search found no `config/mise/config.toml` or `cargo:worktrunk` reference in Makefile, scripts, `.github`, or `config/tests`. The stale installed-ce failure is separately tied to the installed binary being 9 commits behind declared source. These findings show no static dependency between the pin and reported failure groups, but the 24 exact failing nodeids were not retained, so they do not substitute for the declared unchanged-source baseline comparison. Use the repository readiness runner; do not create a comparison worktree manually.
- `mise which golangci-lint` resolves to `aqua-golangci-golangci-lint/2.13.2`; `golangci-lint --version` reports 2.13.2. `which -a` starts with the mise shim, so its first raw PATH entry is not the resolved binary.
- `ce task run-finish pin-worktrunk-0740 --json` completed DONE. It verified readiness against `origin/master`, integrated and pushed `7273dd62d9eaced4428fe769a4f1cdff7f3e4c2a`, then removed the task worktree and local/remote branches. Final baseline counts were `make check` 22→22 and `make lint` 23→23, with no changed-path diagnostics; these supersede the earlier 22→21 preliminary count.
- The executable provided by the `cargo:worktrunk` package is `wt`, not `worktrunk`: `mise which wt` resolves to the installed `cargo-worktrunk/latest/bin/wt`, and `wt --version` reports 0.74.0. A normal `mise install cargo:worktrunk@0.74.0` reports that version already installed. The host executable is already at the pinned version, but the live `~/devenv` config and a drift-preserving install remain unverified.
- Installer drift audit found managed `config/docker/config.json` differences and generated `config/claude/settings.json` plus `config/codex/config.toml` differences (four shared preference values). No force install was run. Preserve these live files until a safe supported install path exists.
