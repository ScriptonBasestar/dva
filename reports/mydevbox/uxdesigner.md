# uxdesigner

## 대상

- 경로: `/Users/archmagece/mydevbox/uxdesigner-devbox`
- 브랜치: `master`
- HEAD: `d0a30f9f0fa5464a2f7fe3e86fa6578a25dc354e`
- DVA: `/Users/archmagece/go/bin/dva` 0.3.0
- 루트 `dva.yml`/`dva.yaml` 없음. `.gz-git.yaml`에 `workspaces:`와 `legacy:`가 없다.

## 판정과 모드

- 판정: **absent**
- 모드: **New**
- 독립 제품 루트다. `uxdesigner-rs`와 `uxdesigner-ts`는 이 저장소 안의 디렉터리다.

## 자식 인벤토리

workspace 없음. legacy 없음.

| 디렉터리 | 실행 표면 | dva.yml |
| --- | --- | --- |
| (루트) | 예. Makefile이 `.make/build.mk`, `dev.mk`, `test.mk`를 include | 아니오 |
| uxdesigner-rs | 아니오. Cargo workspace 매니페스트만. 별도 git 아님 | 아니오 |
| uxdesigner-ts | 예. package scripts `build`/`dev`/`lint`/`test`. 별도 git 아님 | 아니오 |

같은 저장소 안의 표면이지 gz-git workspace가 아니다. 자식 저장소 `dva.yml`을 요구하지 않는다.

## 기준선

루트 설정이 없어 validate와 doctor를 실행하지 않았다. exit는 null.

## 발견

| owner | 증거 |
| --- | --- |
| DVA config | 루트 `dva.yml` 없음. 실행 진입점은 루트 Makefile과 `uxdesigner-ts` scripts다. |

## 제안 표

| 기존 표면 | DVA 이름 | alias/보류 | 이유 |
| --- | --- | --- | --- |
| 루트 `make` build/dev/test, `uxdesigner-ts` scripts | 루트 `stack`/`plans`와 interaction (이름 미정) | 보류 | 모드 New. 읽기 전용. 빈 파일을 만들지 않는다. |

## wave-1

아니오. 기존 루트 `dva.yml`이 없다.
