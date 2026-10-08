# ci-toolchain

## 대상

- 경로: `/Users/archmagece/mydevbox/ci-toolchain`
- 브랜치: `master`
- HEAD: `062ea7b51567701215965c9b6651872d6bef02b5`
- DVA: `/Users/archmagece/go/bin/dva` 0.3.0
- 루트 `dva.yml`/`dva.yaml` 없음. 단일 git 저장소이며 Harbor용 툴체인 이미지다(`Dockerfile`, `Makefile`, `VERSION`).

## 판정 and 모드

- 판정: **absent**
- 모드: **New**
- 루트 설정이 없어 validate/doctor를 실행하지 않았다. 장시간 서비스가 아니라 이미지 빌드다.

## 자식 인벤토리

`.gz-git.yaml`에 `workspaces:`와 `legacy:`가 없다. 주석은 하위 저장소가 생기면 `workspaces:`를 붙이라고만 한다. depth 1에 자식 git은 없다.

루트 실행 표면: `Makefile`의 `image-builder`, `image-build`, `image-push`, `image-digest`. Compose 없음.

## 기준선

- validate exit: 없음
- warning: 없음
- doctor exit: 없음
- doctor 실패: 없음

## 발견

- DVA config. 빌드 표면이 있는데 루트 `dva.yml`이 없다. 증거: `Makefile`, `Dockerfile`. depth 3에서 `dva.yml`/`dva.yaml` 0건.
- `make image-*`를 `command: make`로 감싸는 것은 이관이 아니다. 레지스트리 자격은 Environment이며 설정으로 만들지 않는다.

## 제안 표

| 기존 표면 | DVA 이름 | alias/보류 | 이유 |
| --- | --- | --- | --- |
| `make image-build`/`image-push`/`image-digest` | interaction | 보류 | New. 구현을 옮기는 설계가 먼저다. make 포장은 이관이 아니다 |
| 루트 스캐폴드 | `dva.yml` | 보류 | 루트 파일이 없고 기계적 Preserve 수정이 아니다 |

적용하지 않음: 읽기 전용.

## wave-1

아니오. 루트 `dva.yml`이 없어 validate가 지목할 수정이 없다.
