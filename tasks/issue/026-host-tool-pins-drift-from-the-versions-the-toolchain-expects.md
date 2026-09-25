---
id: ISSUE-026
title: "Host tool pins drift from the versions the toolchain expects"
type: bug
status: todo
priority: P2
severity: medium
ownership: upstream
created: 2026-09-15
discovered-at: 2026-09-15
discovered-in: "mst 호스트 온보딩 후속 (2026-09-15)"
upstream-ref: "ce-agent-kit#6"
---

## Summary

워크스테이션 도구 두 개의 원래 drift를 추적한다. golangci-lint는 재측정 결과
mise가 의도한 Aqua 2.13.2를 정상 선택해 별도 수정이 필요 없다. worktrunk source pin은
devenv master commit `7273dd62`에 0.74.0으로 통합됐다. live 설정에 pin을 반영하는
안전한 설치는 기존 로컬 drift를 보존할 경로가 없어 미완료다.

**1. (해결됨) golangci-lint 2.12.2가 mise가 고른 2.13.2보다 PATH에서 앞선다.** mise 전역
설정은 `"aqua:golangci/golangci-lint" = "2.13.2"`를 고르고, 2.13.2는 go1.27.0으로
빌드돼 프로젝트 go1.26.6을 파싱한다. 그런데 물리 경로
`~/.local/share/mise/installs/golangci-lint/2.12.2/...`(구 백엔드 설치 잔재)가
PATH에 직접 박혀 shim보다 앞선다. ce-agent-kit의 `make lint`는 툴체인 검증에서
즉시 실패한다:

```
❌ golangci-lint is built with go1.26.2 but this project compiles with go1.26.6.
```

같은 실패가 `branch-integrate`의 baseline 측정을 무력화했다 — target tip(내 변경
이전)마저 진단 없이 실패해 "baseline unmeasurable"로 2026-09-15 통합 하나가
`--allow-skipped-checks` downgrade로 갈 수밖에 없었다.

**2. (source 수정 진행 중) mise의 `"cargo:worktrunk" = "latest"`가 ce의 정확 핀과 충돌한다.** ce는
`worktrunkVersion = "0.74.0"`을 substring 정확 매치로 검사한다. mise의 `latest`는
지금 우연히 0.74.0이지만, worktrunk가 다음 버전을 내면 모든 저장소의
`run-doctor`가 같은 차단(BLOCKED)으로 선다. 2026-09-14 mst 온보딩이 하루 종일
걸린 것이 정확히 이 조합이다. 수정은 devenv 소스의 mise config에서
`"cargo:worktrunk" = "0.74.0"`으로 핀하는 것 — `~/.config/mise/config.toml`은
devenv 릴리스로 향하는 symlink라 설치본을 손대지 않는다(rust=1.96.0이 이미
symlink를 통해 쓰여 있어 릴리스-소스 간 드리프트도 함께 생겼다).

## Evidence

2026-09-15 실측:

```
$ which -a golangci-lint | head -2
/Users/.../mise/installs/golangci-lint/2.12.2/...   # 구 백엔드 — PATH가 여기를 쓴다
/Users/.../mise/installs/aqua-golangci-golangci-lint/2.13.2/...

$ mise current | /usr/bin/grep golangci
aqua:golangci/golangci-lint 2.13.2                   # mise의 선택과 어긋난다
```

## Reproduction

1. `which -a golangci-lint | head -2`를 돌린다 — 물리 경로 2.12.2가 첫 줄이다.
2. ce-agent-kit 체크아웃에서 `make lint`를 돌린다 — 툴체인 검증에서 즉시 실패한다.
3. `mise current`에서 worktrunk를 본다 — `latest`로 나온다.

셋 다 워크스테이션 상태라 매번 같은 결과로 재현된다.

## Expected vs Actual

| | |
|---|---|
| 기대 | mise가 고른 버전이 실제로 실행되고, worktrunk는 ce가 요구하는 버전에 핀된다 |
| 실제 | PATH의 잔재 2.12.2가 lint를 죽이고, worktrunk는 `latest`로서 다음 배포에서 doctor를 선다 |

## Impact

중간 — 다음 누군가 kit 저장소를 통합할 때 같은 baseline 무력화가 재현되고,
worktrunk 다음 버전 배포일에 모든 저장소의 작업이 하루 막힌다. 둘 다 이미 한 번
일어난 일이다.

## 소유권 — 상류다

고치는 곳은 devenv 소스 체크아웃의 mise config와 PATH 구성, 그리고 mise의 레거시
설치 정리다. 이 저장소는 증거만 제공한다. 1의 임시 회피는
`PATH=<aqua-2.13.2-경로>:$PATH make lint`로 그 실행만 바로잡는 것이고, 실제로 그렇게
해서 브랜치가 lint-clean(`0 issues.`)임은 증명했다.

## Resolution Criteria

- [x] 실행된 golangci-lint가 mise가 선택한 Aqua 2.13.2다 | verify: human — `mise which golangci-lint`가 Aqua 2.13.2를 가리키고 `golangci-lint --version`이 2.13.2인지 확인한다
- [ ] worktrunk 0.74.0 핀이 live `~/devenv` 설치본에 반영된다 | verify: human — 설치본 mise config와 `mise which wt`, `wt --version`이 0.74.0을 확인한다

## 후속 (2026-09-24)

2026-09-25 재측정에서 `which -a golangci-lint`의 첫 항목은 mise shim이다.
`mise which golangci-lint`는 Aqua 2.13.2 바이너리를 가리키고 실행도 2.13.2를
보고한다. 따라서 raw PATH 항목 순서는 실제 선택을 나타내지 않는다.

devenv source task worktree는 `cargo:worktrunk = "0.74.0"`으로 바뀌었다. `make check`는
24 failed, 1627 passed, 6 skipped로 종료됐다. 실패는 installed CE의 9-commit 지연,
canonical marketplace identity 누락, 오래된 upstream exclusion cache, CLAUDE.md prose-size
기준에 집중된다. diff는 `config/mise/config.toml` 한 줄이며 실패 출력에는 worktrunk가
없다. 다만 변경 전 동일 source baseline과 비교하지 않았으므로 task CI 통합 전 이 차이를
입증해야 한다.

설치기 drift audit은 live `config/docker/config.json`의 managed 변경과 generated
`config/claude/settings.json`, `config/codex/config.toml`의 네 preference 차이를 찾았다.
non-force 설치는 이를 거부하며 force가 live 자격 증명과 설정을 덮을 수 있다. 자격 증명
및 설정을 보존하는 지정 설치 경로를 확인할 때까지 install은 보류한다.
작업 기록은 [TASK-431](../blocked/431-align-host-golangci-and-worktrunk-pins.md)이다.

2026-09-25 read-only audit confirmed the source branch changes only the worktrunk pin and
found no static reference to that config key from the failing check targets or tests. The
installed-ce failure is independently explained by installed build `5d70c9d` lagging its
declared source by 9 commits. The original full-check output did not retain all exact failing
nodeids, so this is evidence of no direct dependency, not a measured unchanged-source
baseline. Let the declared readiness runner perform the isolated comparison; do not integrate
or force-install until its result and a drift-preserving installation path are available.

The package executable is `wt`, not `worktrunk`. `mise which wt` resolves to the currently
installed `cargo-worktrunk/latest/bin/wt` and `wt --version` reports 0.74.0; a normal
`mise install cargo:worktrunk@0.74.0` reports already installed. This verifies the current
host binary only: although the source pin is now integrated, the live `~/devenv` config and
drift-preserving install contract remain unverified.

## 2026-09-25 source integration and baseline comparison

The worktrunk pin is present in devenv master/origin commit `7273dd62`. The declared readiness
runner compared the task branch against its unchanged-source baseline: `make check` failed
counts remained 22 to 22 and `make lint` remained 23 to 23, with no diagnostics on changed
paths. The source criterion is satisfied; the remaining criterion is applying the pin through
a supported installer while preserving existing live config drift. The exact final
`ce task run-finish pin-worktrunk-0740` result was DONE: it integrated/pushed
`7273dd62d9eaced4428fe769a4f1cdff7f3e4c2a` and reclaimed the worktree and local/remote branches.
This final comparison supersedes the earlier preliminary 22-to-21 count.
