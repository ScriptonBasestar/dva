# scripton-dashboard

## 대상

- 경로: `/Users/archmagece/mydevbox/scripton-dashboard-devbox`
- 브랜치: `develop`
- HEAD: `3b4b104e09eb2f713dba85b4c1e59a1ae848b769`
- DVA: `/Users/archmagece/go/bin/dva` 0.3.0
- 루트 `dva.yml`/`dva.yaml` 없음. `legacy:` 없음. `integrationBranch`는 `develop`.

## 판정 and 모드

- 판정: **absent**
- 모드: **New**
- 워크스페이스 둘과 같은 저장소의 `dashboard-webui`가 실행 표면을 가지지만 `dva.yml`은 없다.

## 자식 인벤토리

`path` 대신 `targetPath`다. 두 디렉터리 모두 그 값과 같다.

| workspace | 로컬 | 실행 표면 | 자식 dva |
| --- | --- | --- | --- |
| scripton-mfe-protocol | 예, develop `e16b7d0` | package scripts `build`/`check` | 아니오 |
| scripton-ui-components | 예, develop `ce99d1d` | package scripts `dev`/`build`/`check` | 아니오 |

`dashboard-webui`는 workspace가 아니다. 루트 git이 92개 경로를 추적한다. Makefile(`dev`/`build`)과 package scripts(`dev`/`build`/`test`/`test:e2e`)가 있다. 루트 Makefile은 `dev-dashboard`/`dev-components`/`build`/`check`다. depth 3 Compose 없음.

## 기준선

- validate exit: 없음. warning: 없음. doctor exit: 없음. doctor 실패: 없음.

## 발견

- DVA config. 자식 둘과 중첩 앱 모두 `dva.yml`이 없다. 자식 파일 없이 root `subprojects`를 선언하지 않는다. 중첩 앱도 설정 파일이 생기기 전에는 subproject가 아니다.
- 루트 `make dev`를 `command: make`로 감싸는 것은 이관이 아니다.

## 제안 표

| 기존 표면 | DVA 이름 | alias/보류 | 이유 |
| --- | --- | --- | --- |
| mfe/ui `pnpm` scripts | 각 자식 interaction | 보류 | 자식 저장소 파일이 먼저다 |
| `dashboard-webui` `make dev`/`pnpm test` | 루트 또는 중첩 `dva.yml` | 보류 | 같은 저장소 앱이다. 스캐폴드는 보류 |
| 루트 스캐폴드 | `dva.yml` | 보류 | New |

적용하지 않음: 읽기 전용.

## wave-1

아니오.
