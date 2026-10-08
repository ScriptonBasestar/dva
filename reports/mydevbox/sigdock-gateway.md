# sigdock-gateway

## 대상

- 경로: `/Users/archmagece/mydevbox/sigdock-gateway-devbox`
- 브랜치: `master`
- HEAD: `f56101c7dc37cf18afc4dcbe8e6693a19b3da699`
- DVA: `/Users/archmagece/go/bin/dva` 0.3.0
- 루트 `dva.yml`/`dva.yaml` 없음. `.gz-git.yaml`에 `workspaces:`와 `legacy:`가 없다.

## 판정과 모드

- 판정: **absent**
- 모드: **New**
- 독립 제품 루트다. 다른 devbox의 자식이 아니다.

## 자식 인벤토리

workspace 없음. legacy 없음.

| 디렉터리 | 실행 표면 | dva.yml |
| --- | --- | --- |
| (루트) | 예. Makefile `build`/`test`/`run-http`/`run-grpc` | 아니오 |
| sigdock-gateway-rs | 아니오. 같은 저장소의 Cargo workspace. 루트 명령 파일 없음 | 아니오 |

`sigdock-gateway-rs`는 별도 git 저장소가 아니다.

## 기준선

루트 설정이 없어 validate와 doctor를 실행하지 않았다. exit는 null.

## 발견

| owner | 증거 |
| --- | --- |
| DVA config | 루트 `dva.yml` 없음. 실행 진입점은 루트 Makefile이다. |

## 제안 표

| 기존 표면 | DVA 이름 | alias/보류 | 이유 |
| --- | --- | --- | --- |
| `make build` / `test` / `run-http` / `run-grpc` | 루트 native `stack`+`plans` (이름 미정) | 보류 | 모드 New. 읽기 전용. `make` 래핑은 이관이 아니다. |

## wave-1

아니오. 기존 루트 `dva.yml`이 없다.
