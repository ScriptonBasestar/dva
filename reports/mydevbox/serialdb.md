# serialdb

## 대상

- 경로: `/Users/archmagece/mydevbox/serialdb-devbox`
- 브랜치: `master`
- HEAD: `323190fb6bd1c28cb15a7c2b7142207af8d67d4d`
- DVA: `/Users/archmagece/go/bin/dva` 0.3.0
- 루트 `dva.yml`/`dva.yaml` 없음. `.gz-git.yaml`에 `workspaces:`와 `legacy:`가 없다.

## 판정과 모드

- 판정: **absent**
- 모드: **New**
- 독립 제품 루트다.

## 자식 인벤토리

workspace 없음. legacy 없음.

| 디렉터리 | 실행 표면 | dva.yml |
| --- | --- | --- |
| (루트) | 예. Makefile `build`/`test`/`scenario-test` | 아니오 |
| serialdb-go | 예. `go.mod`와 `cmd/serialdb`. 같은 저장소. 루트 Makefile이 이 트리를 빌드한다 | 아니오 |
| serialdb-rs | 아니오. Cargo.toml만. 별도 git 아님 | 아니오 |

`serialdb-go`는 gz-git workspace가 아니다. 별도 자식 `dva.yml`을 요구하지 않는다.

## 기준선

루트 설정이 없어 validate와 doctor를 실행하지 않았다. exit는 null.

## 발견

| owner | 증거 |
| --- | --- |
| DVA config | 루트 `dva.yml` 없음. 실행 진입점은 루트 Makefile → `serialdb-go`다. |

## 제안 표

| 기존 표면 | DVA 이름 | alias/보류 | 이유 |
| --- | --- | --- | --- |
| `make build` / `test` / `scenario-test` | 루트 native `stack`+`plans` (이름 미정) | 보류 | 모드 New. 읽기 전용. `command: make build` 래핑은 이관이 아니다. |

## wave-1

아니오. 기존 루트 `dva.yml`이 없다.
