# sigdock-pki

## 대상

- 경로: `/Users/archmagece/mydevbox/sigdock-pki-devbox`
- 브랜치: `master`
- HEAD: `9e47b4fa9adf8000be4259f47b8d6f3c3d6cd45e`
- DVA: `/Users/archmagece/go/bin/dva` 0.3.0
- 루트 `dva.yml`/`dva.yaml` 없음. `.gz-git.yaml`에 `workspaces:`와 `legacy:`가 없다.

## 판정과 모드

- 판정: **absent**
- 모드: **New**
- 독립 제품 루트다.

## 자식 인벤토리

workspace 없음. legacy 없음. 따라서 연결 의무가 있는 로컬 workspace도 없다.

루트 Makefile은 `.make/build.mk`와 `.make/test.mk`를 include 한다. 주석은 `sigdock-pki-rs/`를 별도 클론이라고 한다. 그 디렉터리는 없고 `.gitignore`에도 그 이름이 없다. 클론하지 않았다. unavailable.

## 기준선

루트 설정이 없어 validate와 doctor를 실행하지 않았다. exit는 null.

## 발견

| owner | 증거 |
| --- | --- |
| DVA config | 루트 `dva.yml` 없음. |
| Project | Makefile이 가리키는 `sigdock-pki-rs`가 없고 workspace 선언도 없다. 없는 자식을 subproject로 만들지 않는다. |

## 제안 표

| 기존 표면 | DVA 이름 | alias/보류 | 이유 |
| --- | --- | --- | --- |
| 루트 Makefile include | 루트 `stack`/`plans` (이름 미정) | 보류 | 모드 New. 대상 Rust 트리가 로컬에 없다. |

## wave-1

아니오. 기존 `dva.yml`이 없고, 없는 자식 링크는 wave-1이 아니다.
