# flow-knowchain

## 대상

- 경로: `/Users/archmagece/mydevbox/flow-knowchain-devbox`
- 브랜치: `develop`
- HEAD: `646751d2dd4c7e1790fcd338a263a6f9793ad2ce`
- DVA: 0.3.0 (`/Users/archmagece/go/bin/dva`, commit `13dc89398b921c6535ec80ab890f4b04b8f3020c`)
- 루트 설정: `dva.yml`. `legacy:` 키 없음. deprecated 섹션 없음.

## 판정 and 모드

- 판정: **partial**
- 모드: **Preserve** — `local-infra`/`local-dev`/`docker-full`/`observability`와 native stack이 동작 의도로 유효하다. validate error와 deprecated 섹션이 없어 Migrate/Rewrite가 아니다.
- 자식 5곳 모두 실행 표면이 있고 자식 `dva.yml`이 없다. root는 그중 4곳을 `subprojects`로 선언한다(자식 파일 없는 선언).

## 자식 인벤토리

| workspace | 로컬 | 실행 표면 | 자식 dva | root subproject |
| --- | --- | --- | --- | --- |
| flow-knowchain-ai | 예 | 예, Makefile+pyproject | 아니오 | 예 `ai` |
| flow-knowchain-backend | 예 | 예, Makefile+go.mod | 아니오 | 예 `backend` |
| flow-knowchain-client | 예 | 예, Makefile+package scripts | 아니오 | 아니오 |
| flow-knowchain-frontend | 예 | 예, Makefile+package scripts | 아니오 | 예 `frontend` |
| flow-knowchain-admin | 예 | 예, Makefile+package scripts | 아니오 | 예 `admin` |

없는 workspace는 없다. `apps/compose.yaml`은 workspace가 아니다.

## 기준선

- `dva config validate --json`: exit 0. warning 2, error 0. suppressed suggestion 67.
- warning: `config_drift` 1, `config_suggestion` 1. 소유자 모두 DVA config.
- `dva doctor --json`: exit 0. JSON 실패 2 (exit 0은 건강이 아님).
- 실패: `Encrypted env source declared` — DVA config (`.sops.yaml`, `.env.sops`, `env_file`에 `sops_source` 없음).
- 실패: `Compose config resolves` — Environment. `MEILI_MASTER_KEY` 값이 없다. compose 파일을 고쳐 침묵시키지 않는다.

## 발견

- DVA config. 네 subproject path에 자식 `dva.yml`이 없다. 증거: 루트 `dva.yml` `subprojects` (`flow-knowchain-backend`, `flow-knowchain-ai`, `flow-knowchain-frontend`, `flow-knowchain-admin`). 설치된 validator는 exit 0이라 이 공백을 error로 보지 않는다. 정책상 깨진 선언이다.
- DVA config. `flow-knowchain-client`는 실행 표면이 있고 자식 설정과 root subproject가 없다.
- DVA config. native `run`이 `make dev-native-*` / `pnpm dev`이고 interaction 다수가 `make`/`pnpm` 래핑이다. 이전이 아니다.
- DVA config. `config_drift`: 루트 옆 `compose.debug.override.yaml`, `compose.production.override.yaml`, `compose.production.yaml`, `compose.test.yaml`이 stack `files`에 없다.
- DVA config. `config_suggestion`: Makefile `card-check`에 같은 이름 interaction이 없다.
- Environment. Compose interpolation의 `MEILI_MASTER_KEY`.

## 제안 표

| 기존 표면 | DVA 이름 | alias/보류 | 이유 |
| --- | --- | --- | --- |
| 자식 5개 Makefile/npm | 자식 `dva.yml`, 그 다음 subproject | 보류 | 자식 파일 없이 subproject를 추가·유지하면 안 된다. 루트 한 줄이 아니다 |
| `make dev-native-*` / `pnpm dev` | 기존 stack native | 보류 | 구현이 아직 Make/pnpm이다. 계획 의미를 유지한 채 한 줄로 바꾸지 못한다 |
| override/production/test compose | stack files 추가 | 보류 | 넣으면 실행 파일 집합이 바뀐다 |
| `card-check` | interaction | 보류 | 워크플로 편입 여부가 기계적이지 않다 |
| sops / 빠진 master key | 없음 | 보류 | 비밀·Environment |

적용하지 않음: 읽기 전용. 기존 subproject 4줄을 지워도 자식 공백은 남고, 선언 의미를 바꾼다.

## wave-1

아니오.
