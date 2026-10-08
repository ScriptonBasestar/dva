# scripton-suphyul-router

## 대상

- 경로: `/Users/archmagece/mydevbox/scripton-suphyul-router-devbox`
- 브랜치: `master`
- HEAD: `f4e6eddd4d23870ce0d9891ca0033e019de11dfa`
- DVA: `/Users/archmagece/go/bin/dva` 0.3.0
- 루트 `dva.yml`/`dva.yaml` 없음. 게이트웨이 모노레포다. `legacy:` 없음.

## 판정 and 모드

- 판정: **absent**
- 모드: **New**
- 앱과 로컬 Postgres가 루트 저장소에 있으나 DVA 선언이 없다.

## 자식 인벤토리

`.gz-git.yaml`에 `workspaces:`와 `legacy:`가 없다. depth 1 자식 git 없음.

루트 실행 표면: Makefile `build`/`run`/`test-unit`과 `db-up`/`db-down`/`db-schema`/`db-seed`. `gateway/go.mod`와 `gateway/cmd`. `deploy/local/compose.yaml` 서비스는 `postgres`뿐이다. 최상위 `name:`은 없다. 호스트 포트는 `POSTGRES_PORT`(기본 12610)다. `db-up`은 `docker compose -f deploy/local/compose.yaml up -d`다.

## 기준선

- validate exit: 없음. warning: 없음. doctor exit: 없음. doctor 실패: 없음.

## 발견

- DVA config. 루트에 plan으로 올릴 Compose와 `make build`/`run`이 있는데 `dva.yml`이 없다. `db-up`을 `command: make`로 감싸는 것은 이관이 아니다.
- Project. Compose에 최상위 `name:`이 없다. 수정 위치는 Compose 파일이며, 없는 DVA 설정의 wave-1이 아니다.

## 제안 표

| 기존 표면 | DVA 이름 | alias/보류 | 이유 |
| --- | --- | --- | --- |
| `deploy/local/compose.yaml` postgres | stack + `local-infra` | 보류 | 스캐폴드는 보류. `name:`은 Project 수정이다 |
| `make build`/`run` (gateway) | `local-dev`의 native | 보류 | infra plan과 분리해야 한다. 루트 한 줄이 아니다 |
| `make test-unit`/`validate*` | interaction | 보류 | make 포장은 이관이 아니다 |
| 루트 스캐폴드 | `dva.yml` | 보류 | New |

적용하지 않음: 읽기 전용.

## wave-1

아니오. 루트 `dva.yml`이 없어 validate가 지목할 Preserve 수정이 없다.
