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

워크스테이션 도구 두 개의 drift를 추적한다. devenv source `7273dd62`의
worktrunk 0.74.0 핀은 2026-09-27 live host에 설치돼 `/tmp`에서 명시적
`cargo-worktrunk/0.74.0`을 선택한다. 전역 Aqua lint도 2.13.2로 선택·실행된다.
DVA 프로젝트와 CI의 lint 2.12.2는 의도된 제품 핀이다. 상세 이전 관측은 아래
시간순 기록으로 보존하고 최종 판정은 TASK-431의 설치 증거를 따른다.

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

- [x] 프로젝트 밖 호스트 scope에서 golangci-lint가 Aqua 2.13.2로 선택·실행된다 | verify: human — 2026-09-27 `/tmp`에서 `mise which golangci-lint`와 `golangci-lint --version`이 모두 Aqua 2.13.2; DVA `.mise.toml`과 CI는 제품 2.12.2 핀으로 일치한다
- [x] worktrunk 0.74.0 핀이 live `~/devenv` 설치본에 반영된다 | verify: human — 설치본 mise config와 `mise which wt`, `wt --version`이 0.74.0을 확인한다

## 2026-09-27 범위 교정

2026-09-27 `/tmp`에서 `mise which golangci-lint`는 전역 Aqua 2.13.2를 선택하고
실행 버전도 2.13.2였다. DVA checkout의 tracked `.mise.toml`은 2.12.2를
선택하며 CI `golangci-lint-action`도 v2.12.2를 선언한다. 위 첫 기준의 호스트
scope는 충족됐고 DVA 제품 scope도 자체 선언과 일치한다. 이전의 "호스트 2.12.2
실패" 문장은 DVA cwd에서 호스트 기준을 측정한 결과다. source에 통합된
worktrunk 0.74.0 핀이 live release에서 `latest`로 남은 두 번째 기준만 미완료다.
설치 시점의 실제 로컬 입력을 열거·해시해 설치기 receipt와 대조한다.

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
작업 기록은 [[TASK-431]]이다.

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

## 2026-09-27 재발 증거

설치된 `ce`(0.8.4, a1a07dae 빌드)가 원격 master가 이미 쓰는 `ce-tasks.yaml`
`card-dialect.strict-status` 필드를 몰라 `ce task new`·`validate`·gate가 모두 실패했다.
kit을 dba2348b로 fast-forward하고 `make install_cli`로 재설치해 복구했다. 재빌드는
호스트 Go 환경의 `GOSUMDB=off` 때문에 go.mod가 요구하는 go 1.26.6 툴체인을 받지 못해
실패했고, 일회성 `GOSUMDB=sum.golang.org`로만 통과했다. 두 원인 모두 워크스테이션
상태이므로 수정 위치는 이 저장소가 아니라 devenv 정본(`GOSUMDB`)과 ce의 설정 스키마
대비 설치 버전 점검(`run-doctor`)이다.

## 2026-09-27 독립 재측정

TASK-431 reviewer는 `mise which golangci-lint`와 실행 버전 모두 2.12.2,
`mise which wt` 경로 `cargo-worktrunk/latest/bin/wt`, `wt --version` 0.74.0을 기록했다.
실행 버전만 맞는 것은 pinned installation 경로의 증거가 아니다. 과거 FORCE=1과
11개 입력 보존 주장은 durable before/after evidence가 없고 report의 03c1466e source
commit도 canonical checkout에서 확인되지 않았다. source pin 통합 및 readiness 비교는
통과 상태로 유지한다. 이 문단의 golangci 판정은 DVA cwd에서 호스트 기준을 측정한
역사적 오류이며 위 범위 교정이 현행 판정이다. wt live criterion이 검증될 때까지
이 이슈를 열어 둔다.

## 2026-09-27 live 설치 완료

[[TASK-431]]의 [추적 설치 증거](../done/evidence/TASK-431/host-install-20260927.json)에
source `7273dd62`, non-force drift 거부, 보존한 Docker 설정, 강제 설치 receipt,
이전·새 로컬 입력 10개의 목록·크기·바이트 일치 판정과 의미 비교를 기록했다.
원문 SHA-256은 제한된 외부 감사 파일에만 남겼다. 새 live release는
`7273dd62d9ea-20260927T143640807928Z`이고 `forced_drift` 세 경로를 receipt에
남겼다. `/tmp`에서 `mise which wt`는 explicit `cargo-worktrunk/0.74.0/bin/wt`,
`wt --version`은 0.74.0, golangci-lint의 선택·실행 버전은 Aqua 2.13.2다.
두 resolution criteria를 충족했으므로 독립 리뷰 뒤 fixed로 해결한다.
