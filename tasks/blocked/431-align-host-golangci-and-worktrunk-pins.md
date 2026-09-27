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
아직 끝나지 않았다. DVA 작업 디렉터리의 `golangci-lint` 2.12.2는
프로젝트 `.mise.toml`과 CI의 의도된 핀이다. 프로젝트 밖에서는 전역 Aqua
2.13.2가 선택·실행돼 호스트 기준을 만족한다. `mise which wt`는 여전히
`cargo-worktrunk/latest`를 선택한다. 과거 설치 기록은
`FORCE=1` 실행과 11개 입력 보존을 주장하지만 설치 receipt가 없고 보고된 source commit
`03c1466e`도 canonical checkout에서 확인되지 않았다. 남은 차단 사유는
worktrunk 핀의 live 설치와 실제 보존 입력의 추적 가능한 증거다.

## Completion Criteria

- [x] 프로젝트 밖 호스트 scope에서 mise가 Aqua 2.13.2를 선택하고 실행 버전도 2.13.2를 보고한다 | verify: human — 2026-09-27 `/tmp`에서 `mise which golangci-lint`가 `aqua-golangci-golangci-lint/2.13.2`를 가리키고 `golangci-lint --version`은 2.13.2; DVA의 `.mise.toml`과 CI는 2.12.2를 선언한다
- [x] Source master pins worktrunk to 0.74.0 and task-relevant checks do not worsen the unchanged-source baseline | verify: human — integrated commit `7273dd62`; final `run-finish` readiness reports `make check` 22→22 and `make lint` 23→23 with no changed-path diagnostics
- [ ] 통합된 source의 worktrunk 핀 0.74.0이 기존 live 설정을 보존하며 설치된다 | verify: human — 지정 설치 결과·설치본 config·`mise which wt` 경로·`wt --version` 및 설치 전후 실제 로컬 입력 목록·해시·receipt를 추적 가능한 evidence에 기록한다

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

## 2026-09-27 재판정

앞 절의 `golangci-lint` 실패 판정은 DVA 프로젝트 scope와 호스트 scope를 혼동했다.
`tasks` 이외의 `/tmp`에서 전역 Aqua 2.13.2가 실제 선택·실행됐다. DVA 저장소의
tracked `.mise.toml:16`과 CI `golangci-lint-action`은 함께 2.12.2를 선언한다.
둘은 각각의 범위에서 정합하다. 호스트 설치 증거는 전역 scope에서 재측정하고,
DVA 제품 핀은 이 카드의 대상으로 바꾸지 않는다. devenv source master의
`cargo:worktrunk = "0.74.0"`은 설치 snapshot에는 아직 반영되지 않았다.
기존 release receipt는 `forced: false`, 보존 입력 10개를 기록한다. 새 설치는
설치 직전 실제 입력을 열거·해시하고 지정 installer의 drift 판정과 로컬 overlay
기능으로 보존한 뒤 새 receipt와 함께 검증한다. 과거 11개 주장으로 수를 고정하지 않는다.
