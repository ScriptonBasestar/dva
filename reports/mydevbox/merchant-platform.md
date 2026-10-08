# merchant-platform

## 대상

- 경로: `/Users/archmagece/mydevbox/merchant-platform`
- 브랜치: `master`
- HEAD: `fdc4f3c2dcf7f5d1ed174a9846004965f5f30041`
- DVA: `/Users/archmagece/go/bin/dva` 0.3.0
- 루트 `dva.yml`/`dva.yaml` 없음. 제품 문서는 `PRODUCT.md`(사장업). `.gz-git.yaml`에 `workspaces:`와 `legacy:`가 없다.

## 판정 and 모드

- 판정: **absent**
- 모드: **New**
- 루트는 통합 게이트만 있고, 앱 체크아웃은 workspace로 선언되어 있지 않다.

## 자식 인벤토리

등록된 workspace 없음. 없는 선언 자식도 없다.

로컬에만 있는 git: `sajangup-server` (master `6224c79`, origin `sajangup/sajangup-server`). 부모 `.gitignore`가 `/sajangup-server/`를 무시한다. 실행 표면은 Makefile(`build`/`test`/`image-build`), `build.gradle.kts`+`gradlew`, `deploy/compose.*.yml`. 자식 `dva.yml` 없음.

루트 Makefile은 `gate-scripts`와 `branch-integrate`뿐이다. 앱 기동이 아니다.

## 기준선

- validate exit: 없음. warning: 없음. doctor exit: 없음. doctor 실패: 없음.

## 발견

- DVA config. 앱 표면에 자식 `dva.yml`이 없고 루트 선언도 없다. 자식 파일 없이 `subprojects`를 만들지 않는다.
- Project. `.gz-git.yaml`은 아직 `workspaces:`가 없는데 `sajangup-server` 체크아웃은 있다. 클론하지 않는다. 루트 `make branch-integrate`는 git 통합이라 DVA lifecycle이 아니다.

## 제안 표

| 기존 표면 | DVA 이름 | alias/보류 | 이유 |
| --- | --- | --- | --- |
| `sajangup-server` gradle/make/compose | 자식 `dva.yml`, 이후 root subproject | 보류 | 자식 소유 파일이 먼저다. 스캐폴드는 보류 |
| 루트 `make gate-scripts` | 없음 | 보류 | 태스크 게이트. DVA 이관 증거가 없다 |
| 루트 스캐폴드 | `dva.yml` | 보류 | New. workspace 선언과도 별개다 |

적용하지 않음: 읽기 전용.

## wave-1

아니오.
