# reviewrary

## 대상

- 경로: `/Users/archmagece/mydevbox/reviewrary-devbox`
- 브랜치: `develop`
- HEAD: `99cad10ff830c8a3fbd68178ec0f889ccf042089`
- DVA: `0.3.0` (`/Users/archmagece/go/bin/dva`)
- 루트 `dva.yml`, `cmd/rollback-webhook/dva.yml`.

## 판정

- 판정: `partial`
- 모드: `Migrate`
- validator가 `stack.*.order`에 대해 `dva config migrate`를 가리키고 plans가 없다. 설정은 아직 유효하나(validate exit 0) deprecation이 남아 partial이다.

## 자식 인벤토리

`workspaces:` / `legacy:` 없음. 선언은 `repositories:`.

| 이름 | exists | executable surface | dva.yml | root subproject |
| --- | --- | --- | --- | --- |
| reviewrary-engine-spring | 아니오, unavailable | 미확인 | 없음 | 없음 |
| reviewrary-client-sveltekit | 아니오, unavailable | 미확인 | 없음 | 없음 |
| cmd/rollback-webhook (같은 저장소) | 예 | 예 (Makefile, go.mod) | 예 | `rollback-webhook` |

없는 저장소는 clone하지 않음.

## 기준선

- validate exit 0, warning 65, error 0
  - `semantic` ×1 — `stack.compose.order`는 plan order로 옮겨야 함. owner: DVA config
  - `semantic` ×1 — plans 없음. owner: DVA config
  - `config_suggestion` ×63 — Makefile 타깃(예: `dev`, `local-infra`, `test`, `skaffold-dev`, `ws-status`). owner: DVA config
- doctor exit 0, JSON fail 3 (built-in)
  - `Encrypted env source declared` — `.sops.yaml`, `.env.sops`. owner: DVA config
  - `Compose config resolves` — `MINIO_ROOT_PASSWORD` 없음. owner: Environment
  - `.sb/dva/ is ignored in .gitignore` — ignore 아님. owner: Project

## 발견

- DVA config: `dva.yml` stack `order: 10`, plans 부재, `provision.dev`가 `docker compose`를 직접 실행. 증거: 루트 `dva.yml`.
- Environment: 언실된 compose 변수. 값을 채우거나 체크를 지우라고 제안하지 않음.
- Project: `.gitignore`에 `.sb/dva/` 예외가 없음.
- 로컬에 있는 실행 자식 `rollback-webhook`은 연결됨. 나머지 둘은 unavailable.

## 제안

| old surface | DVA name | alias/보류 | reason |
| --- | --- | --- | --- |
| `stack.order` + provision compose up | named plan `local-infra` | 보류 | Migrate이며 plan 의미가 바뀐다 |
| Makefile dev/test/skaffold 63개 | interaction | 보류 | 일괄 이관은 기계적 한 줄이 아님 |
| 없는 engine/client | subproject | 보류 | 체크아웃이 없음. clone 금지 |
| `.gitignore` `.sb/dva/` | gitignore 한 줄 | 보류 | Project 파일이고 `dva.yml` 수정이 아님 |
| `.env.sops` | `sops_source` | 보류 | 비밀 |

적용하지 않음: 읽기 전용, 모드가 Migrate.

## wave-1

아니오.
