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
upstream-ref: ""
---

## Summary

워크스테이션 도구 두 개가 **정본 핀과 어긋난 채로** 동작한다. 둘 다 이 저장소가
고칠 수 없는 위치에 있다.

**1. golangci-lint 2.12.2가 mise가 고른 2.13.2보다 PATH에서 앞선다.** mise 전역
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

**2. mise의 `"cargo:worktrunk" = "latest"`가 ce의 정확 핀과 충돌한다.** ce는
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

- [ ] PATH의 golangci-lint가 mise가 고른 aqua 2.13.2와 일치한다 | verify: human — devenv 수정 후 `which -a golangci-lint`의 첫 줄이 aqua 경로인지 확인한다
- [ ] mise의 worktrunk 핀이 exact 버전을 가리킨다 | verify: human — devenv 소스의 mise config에서 `"cargo:worktrunk"` 값이 `latest`가 아닌지 확인한다
