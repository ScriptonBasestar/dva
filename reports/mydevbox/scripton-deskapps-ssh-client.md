# scripton-deskapps-ssh-client

## 대상

- 경로: `/Users/archmagece/mydevbox/scripton-deskapps-ssh-client-devbox`
- 브랜치: `master`
- HEAD: `04c738ed2427c7b1d8a74b40eba5fd8031ed5e81`
- DVA: `/Users/archmagece/go/bin/dva` 0.3.0
- 루트 `dva.yml`/`dva.yaml` 없음. `legacy:` 없음.

## 판정 and 모드

- 판정: **absent**
- 모드: **New**
- 루트 Makefile은 `scripton-ssh-terminal-manager`만 활성으로 위임한다. 그 자식에 `dva.yml`이 없다.

## 자식 인벤토리

`path` 키 없음.

| workspace | 로컬 | 실행 표면 | 자식 dva |
| --- | --- | --- | --- |
| scripton-ssh-desktop-client | 예, master `027f1ff`, 추적 파일 `.gitignore`뿐 | 아니오 | 아니오 |
| scripton-ssh-server | 예, master `20c285e`, 추적 파일 `.gitignore`뿐 | 아니오 | 아니오 |
| scripton-ssh-terminal-manager | 예, master `b90338a` | Makefile(`dev`/`build`/`run`/`test`), Cargo.toml | 아니오 |

없는 workspace 없음. 빈 두 곳에는 의식적 `dva.yml`을 만들지 않는다. 루트 `dev`/`run`/`build`/`test`/`lint`는 terminal-manager로 위임한다. Compose 없음.

## 기준선

- validate/doctor: 실행 안 함 (루트 dva 없음).

## 발견

- DVA config. 실행 표면이 있는 자식은 terminal-manager뿐이고 `dva.yml`이 없다. 루트 `subprojects`는 그 파일 다음에만 가능하다.
- Project. desktop-client와 server는 체크아웃이 비어 있다. 이름만으로 표면을 만들지 않는다.

## 제안 표

| 기존 표면 | DVA 이름 | alias/보류 | 이유 |
| --- | --- | --- | --- |
| terminal-manager `make dev`/`build`/`test` | 자식 interaction, native는 장기 프로세스일 때 | 보류 | 자식 파일이 먼저다. 스캐폴드는 보류 |
| 빈 desktop-client/server | 없음 | 보류 | 실행 표면 없음 |
| 루트 `make dev`/`test` | 자식 alias | 보류 | 지금은 `TM` 위임이다. make 포장은 이관이 아니다 |

적용하지 않음: 읽기 전용.

## wave-1

아니오.
