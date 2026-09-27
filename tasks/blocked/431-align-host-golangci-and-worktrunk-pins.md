---
id: TASK-431
title: "Align host golangci-lint and worktrunk pins"
type: chore
priority: P2
effort: S
exec-tier: standard
needs-human: false
status: blocked
created: 2026-09-24
---

## Summary

source 쪽 `worktrunk` pin과 readiness 비교는 통과했지만 live host acceptance는
독립 재측정에서 실패했다. 현재 `mise which golangci-lint`와 실행 버전은 2.12.2이고,
`mise which wt`는 `cargo-worktrunk/latest`를 선택한다. 과거 설치 기록은
`FORCE=1` 실행과 11개 입력 보존을 주장하지만 설치 receipt가 없고 보고된 source commit
`03c1466e`도 canonical checkout에서 확인되지 않았다. source 통합과 두 live 기준은
별도로 판정해야 하므로 이 카드를 blocked로 되돌린다.

## Completion Criteria

- [ ] mise가 Aqua 2.13.2를 선택하고 실행된 golangci-lint가 2.13.2를 보고한다 | verify: human — `mise which golangci-lint`와 `golangci-lint --version` 결과를 같은 evidence에 기록한다
- [x] Source master pins worktrunk to 0.74.0 and task-relevant checks do not worsen the unchanged-source baseline | verify: human — integrated commit `7273dd62`; final `run-finish` readiness reports `make check` 22→22 and `make lint` 23→23 with no changed-path diagnostics
- [ ] 통합된 source의 worktrunk 핀 0.74.0이 기존 live 설정을 보존하며 설치된다 | verify: human — 지정 설치 결과·설치본 config·`mise which wt` 경로·`wt --version` 및 force 전후 11개 입력 보존을 추적 가능한 evidence에 기록한다

## Out of scope

- 설치된 `~/devenv`를 직접 고치는 일.
- DVA의 golangci 설정을 2.12.2에 맞추는 일.

## Sources

- [ISSUE-026](../issue/026-host-tool-pins-drift-from-the-versions-the-toolchain-expects.md)

## Current evidence

- Integrated source master commit `7273dd62` pins `config/mise/config.toml` from `"latest"` to `"0.74.0"`; the isolated task diff contained only that config line.
- Full `make check`: 24 failed, 1627 passed, 6 skipped in 564.94s. Failing groups report stale CE binary revision (`5d70c9d`, 9 commits behind source), missing canonical ce-agent-kit marketplace identity in installer fixtures, old upstream file-size exclusions, and the CLAUDE.md prose-size threshold. None references worktrunk. This run did not include an unchanged-source baseline comparison; do not report it as passing.
- Read-only follow-up confirmed the branch diff is exactly this one config line and static search found no `config/mise/config.toml` or `cargo:worktrunk` reference in Makefile, scripts, `.github`, or `config/tests`. The stale installed-ce failure is separately tied to the installed binary being 9 commits behind declared source. These findings show no static dependency between the pin and reported failure groups, but the 24 exact failing nodeids were not retained, so they do not substitute for the declared unchanged-source baseline comparison. Use the repository readiness runner; do not create a comparison worktree manually.
- 2026-09-25 measurement: `mise which golangci-lint` resolved to `aqua-golangci-golangci-lint/2.13.2` and the executable reported 2.13.2. The 2026-09-27 independent review supersedes this observation: its current selection and executable both report 2.12.2.
- `ce task run-finish pin-worktrunk-0740 --json` completed DONE. It verified readiness against `origin/master`, integrated and pushed `7273dd62d9eaced4428fe769a4f1cdff7f3e4c2a`, then removed the task worktree and local/remote branches. Final baseline counts were `make check` 22→22 and `make lint` 23→23, with no changed-path diagnostics; these supersede the earlier 22→21 preliminary count.
- The executable provided by the `cargo:worktrunk` package is `wt`, not `worktrunk`: `mise which wt` resolves to the installed `cargo-worktrunk/latest/bin/wt`, and `wt --version` reports 0.74.0. A normal `mise install cargo:worktrunk@0.74.0` reports that version already installed. The host executable is already at the pinned version, but the live `~/devenv` config and a drift-preserving install remain unverified.
- Installer drift audit found managed `config/docker/config.json` differences and generated `config/claude/settings.json` plus `config/codex/config.toml` differences (four shared preference values). At that point no force install was run. A later 2026-09-27 install report claims FORCE=1 was used, but the independent reviewer could not verify its source commit or the 11-input before/after preservation, so it is not acceptance evidence.
- 2026-09-27 install report (not independently reproducible): the prior session reported adopting live jq = 1.8.1 at 03c1466e, running make install FORCE=1, preserving 11 declared inputs, and selecting cargo-worktrunk/0.74.0/bin/wt. The independent reviewer could not resolve 03c1466e in canonical ce-devenv; the current selection is cargo-worktrunk/latest/bin/wt. Treat the install and preservation statements as unverified historical claims, not acceptance evidence.

## Independent review (2026-09-27) — FIX NEEDED

The reviewer measured `mise which golangci-lint` under the installed 2.12.2 directory and
`golangci-lint --version` as 2.12.2, so criterion 1 fails. `mise which wt` selected
`cargo-worktrunk/latest/bin/wt` even though `wt --version` reported 0.74.0; this does not
prove the source pin reached the live selection. The force-install record has no tracked
before/after evidence for the 11 claimed local inputs, and its cited source commit was not
found in the canonical checkout. Criterion 2 passes: canonical `ce-devenv` commit
`7273dd62` contains the one-line pin and the lifecycle receipt records `make check` 22→22,
`make lint` 23→23, no changed-path diagnostics, source push, and task worktree/branch cleanup.

To unblock, capture one tracked install evidence record with current `mise which` and version
outputs for both tools; a selected explicit `cargo-worktrunk/0.74.0` path; non-force drift
output; exact source commit and lifecycle receipt; and, if force is required, an enumerated
before/after comparison of the 11 protected inputs.
