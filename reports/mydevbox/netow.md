# netow

## 대상

- 경로: `/Users/archmagece/mydevbox/netow-devbox`
- 브랜치: `master`
- HEAD: `3449b55fb0aef70029fca53e0279bb06285200e2`
- DVA: `/Users/archmagece/go/bin/dva` 0.3.0
- 루트 `dva.yml`/`dva.yaml` 없음. `legacy:` 없음. 루트에 `.sops.yaml`과 `.env.sops`가 있다.

## 판정 and 모드

- 판정: **absent**
- 모드: **New**
- 앱은 자식 `newtow-osx`이고, 루트 Compose는 Penpot/Excalidraw 디자인 도구다.

## 자식 인벤토리

`path` 키 없음. 디렉터리 `newtow-osx`가 있다.

| workspace | 로컬 | 실행 표면 | 자식 dva |
| --- | --- | --- | --- |
| newtow-osx | 예, master `d93ddc0` | Makefile(`build`/`run`/`dev`/`test`) | 아니오 |

없는 workspace 없음. 루트 `compose.design.yaml`의 `name`은 `netow-design`이고 프로필은 excalidraw/penpot이다. 루트 make는 태스크 목록과 `.make/design.mk`의 `design-up`/`penpot-up`이다. 제품 DB Compose는 depth 3에서 보이지 않는다.

## 기준선

- validate/doctor: 실행 안 함 (루트 dva 없음).

## 발견

- DVA config. `newtow-osx`에 실행 표면이 있고 자식 `dva.yml`이 없다. 루트 선언도 없다.
- Project. `compose.design.yaml`은 디자인 도구다. 제품 infra로 바꾸지 않는다.
- DVA config. sops 파일은 있으나 `dva.yml`의 `env_file`이 없다. 비밀 로딩이라 이번 배치에서 선언하지 않는다.

## 제안 표

| 기존 표면 | DVA 이름 | alias/보류 | 이유 |
| --- | --- | --- | --- |
| `newtow-osx` `make build`/`run`/`test` | 자식 native·interaction | 보류 | 자식 파일이 먼저다 |
| `make design-up` / Penpot Compose | `tools`/`design` | 보류 | 제품 `local-infra`가 아니다. 스캐폴드는 보류 |
| `.env.sops` | `sops_source` | 보류 | 비밀 입력 |
| 루트 스캐폴드 | `dva.yml` | 보류 | New |

적용하지 않음: 읽기 전용.

## wave-1

아니오.
