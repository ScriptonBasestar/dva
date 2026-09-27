---
id: TASK-431
title: "Align host golangci-lint and worktrunk pins"
type: chore
priority: P2
effort: S
exec-tier: standard
needs-human: false
status: done
created: 2026-09-24
quality-review: pass
quality-reviewed-at: 2026-09-27
quality-review-evidence: "Independent review session /root/review_task443_board_refresh verified all three criteria against source 7273dd62, live release/receipt, explicit wt 0.74.0 and Aqua lint 2.13.2, restricted before/after audit of 10 local inputs, generated shared reset chosen by user, and secret-hash redaction. Final board consistency review followed README/PLAN-011/ISSUE-026 correction."
---

## Summary

Source `7273dd62`의 `worktrunk` 핀 0.74.0을 지정 설치기로 live host에 설치했다.
프로젝트 밖 `/tmp`에서 `mise which wt`는 명시적 `cargo-worktrunk/0.74.0`을,
`wt --version`은 0.74.0을 보고한다. 전역 Aqua lint는 2.13.2이고 DVA 프로젝트와
CI의 2.12.2는 의도된 제품 핀이다. 이전 릴리스의 로컬 입력 10개와 새 릴리스
receipt를 열거·대조한 결과를 추적 증거에 기록했다.

## Completion Criteria

- [x] 프로젝트 밖 호스트 scope에서 mise가 Aqua 2.13.2를 선택하고 실행 버전도 2.13.2를 보고한다 | verify: human — 2026-09-27 `/tmp`에서 `mise which golangci-lint`가 `aqua-golangci-golangci-lint/2.13.2`를 가리키고 `golangci-lint --version`은 2.13.2; DVA의 `.mise.toml`과 CI는 2.12.2를 선언한다
- [x] Source master pins worktrunk to 0.74.0 and task-relevant checks do not worsen the unchanged-source baseline | verify: human — integrated commit `7273dd62`; final `run-finish` readiness reports `make check` 22→22 and `make lint` 23→23 with no changed-path diagnostics
- [x] 통합된 source의 worktrunk 핀 0.74.0이 선언된 로컬 입력을 보존하고 generated shared drift를 정본으로 복귀시키며 설치된다 | verify: human — 지정 설치 결과·설치본 config·`mise which wt` 경로·`wt --version`, 설치 전후 실제 로컬 입력 목록·바이트 비교·receipt, 제한된 로컬 해시 감사와 shared drift 백업을 확인한다

## Out of scope

- 설치된 `~/devenv`를 직접 고치는 일.
- DVA의 golangci 설정을 2.12.2에 맞추는 일.

## Sources

- [ISSUE-026](../_archive/issue/026-host-tool-pins-drift-from-the-versions-the-toolchain-expects.md)

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

## 2026-09-27 설치 검증

[추적 증거](evidence/TASK-431/host-install-20260927.json)는 설치 전후
release ID와 receipt SHA-256, 실제 로컬 입력 10개의 경로·크기·바이트 일치 판정,
설치기 `forced_drift`, 새 receipt sidecar 일치, host 선택 경로·실행 버전을 기록한다.
비밀 파일의 원문 SHA-256 감사 기록은 제한된 외부 백업에만 둔다.
일반 `make install`은 `config/docker/config.json` drift를 감지해 exit 2로
거부했다. 기존 Docker 설정을
`/Users/archmagece/backups/devenv/task431-20260927/docker-config-before.json`에
보존한 후 지정 `FORCE=1 make install`로 source `7273dd62`를 설치했다.
새 릴리스 `7273dd62d9ea-20260927T143640807928Z`는 기존 10개 입력을 모두
열거했다. 8개는 바이트까지 같고, 두 Codex local TOML은 설치기의
`absorb-local` 처리로 바뀌었다. TOML 키 비교에서 기존 키 삭제는 0개이며
추가·변경된 17개 leaf의 값이 모두 이전 live generated 값과 같다.
Docker 설정도 기존 키 10개가 값까지 동일하고 source의 키 3개가 추가됐다.
생성된 Claude·Codex 설정의 shared drift는 `--force`가 소스 값으로 되돌렸다.
기존 model/effort·알림·플러그인 관련 키의 변경·삭제 목록은 증거에 있고,
직전 릴리스 파일 세 개를 제한된 외부 백업에 보존했다. 따라서 선언된 로컬
입력은 보존됐지만 이전 생성 설정의 모든 값이 live에 유지됐다고 주장하지 않는다.
새 receipt SHA-256은 sidecar와 일치한다. `/tmp`에서 `mise which wt`는
`cargo-worktrunk/0.74.0/bin/wt`, `wt --version`은 0.74.0,
`mise which golangci-lint`와 실행 버전은 Aqua 2.13.2다.
