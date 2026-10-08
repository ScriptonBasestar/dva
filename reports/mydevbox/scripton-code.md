# scripton-code

## 대상

- 경로: `/Users/archmagece/mydevbox/scripton-code-devbox`
- 브랜치: `master`
- HEAD: `88ff74fbf3fd89251e924e715f156cd28e0cb1d2`
- DVA: `/Users/archmagece/go/bin/dva` 0.3.0
- 루트 `dva.yml`/`dva.yaml` 없음. README는 Scripton Code 개발 환경이라고 한다.

## 판정 and 모드

- 판정: **absent**
- 모드: **New**
- `workspaces:` 키는 있으나 항목이 없고, 루트 make 타깃은 TODO echo다.

## 자식 인벤토리

`legacy:` 없음. `workspaces:` 아래 이름이 없다. 없는 경로로 보고할 자식도 없다. depth 1 자식 git 없음.

루트 `Makefile`에 `dev`/`install`/`build`/`test`/`lint`가 있으나 본문은 echo와 `# TODO: add … when language/framework is chosen`이다. `src/`, `scripts/`, `.make/`는 없다. 있는 트리는 `docs/use-cases`와 `e2e/persona`, `e2e/scenario`다. Compose 없음.

## 기준선

- validate/doctor: 실행 안 함 (루트 dva 없음).

## 발견

- DVA config. 루트 `dva.yml`이 없다. 다만 현재 make는 실행 구현이 아니다.
- Project. 빈 `workspaces:`와 없는 `src/`가 맞다. 언어가 정해지기 전의 껍질이다. 의식적 `dva.yml`을 만들지 않는다.

## 제안 표

| 기존 표면 | DVA 이름 | alias/보류 | 이유 |
| --- | --- | --- | --- |
| `make dev`/`build`/`test` | 없음 | 보류 | TODO echo다. DVA로 옮겨도 명령이 생기지 않는다 |
| 빈 `workspaces:` | 없음 | 보류 | 로컬 자식이 없다. 클론하지 않는다 |
| 루트 스캐폴드 | `dva.yml` | 보류 | New이며 실행 표면이 아직 없다 |

적용하지 않음: 읽기 전용.

## wave-1

아니오.
