# hek

## 대상

- 경로: `/Users/archmagece/mydevbox/hek-devbox`
- 브랜치: `master`
- HEAD: `d8dc679661a660e8f44de14187b0c0f56e364d5b`
- DVA: `0.3.0` (`/Users/archmagece/go/bin/dva`)
- 루트 `dva.yml`만 있음. 자식 `dva.yml` 없음.

## 판정

- 판정: `partial`
- 모드: `Migrate`
- validator가 `modes`와 `stack.*.order`를 deprecated로 보고함. 실행 표면이 있는 workspace 둘이 루트에 연결되지 않음.

## 자식 인벤토리

`legacy:` 없음. path는 workspace 이름과 같다.

| workspace | exists | executable surface | dva.yml | root subproject |
| --- | --- | --- | --- | --- |
| hek-engine-fiber | 예 | 예 (Makefile, go.mod) | 아니오 | 없음 |
| hek-extension-chrome | 예 | 아니오 (확장 manifest.json만) | 아니오 | 없음 |
| hek-web-next | 예 | 예 (Makefile, package.json scripts) | 아니오 | 없음 |

`hek-extension-chrome`은 표면이 없으므로 빈 `dva.yml`을 제안하지 않음.

## 기준선

- validate exit 0, warning 7, error 0
  - `semantic` `modes` — plans/environments/sites로 이전. owner: DVA config
  - `semantic` `stack.*.order` — `plans.*.entries[].order`. owner: DVA config
  - `semantic` plans 없음. owner: DVA config
  - `config_drift` `compose/compose.test.yml`. owner: DVA config
  - `config_suggestion` package.json `build`, `lint`, `test`. owner: DVA config
- doctor exit 1 (`ERROR: 1 user check(s) failed`), JSON fail 3
  - `Encrypted env source declared` (`.sops.yaml`, `.env.sops`). owner: DVA config
  - `Compose config resolves` — `POSTGRES_PASSWORD` 미설정. owner: Environment
  - `Compose env file exists` — `compose/.env` 없음. owner: Environment

## 발견

- DVA config: `dva.yml`의 `modes`, `default_mode: full`, `stack.compose.order`. 실행 표면 자식 둘에 자식 `dva.yml`과 루트 `subprojects`가 없음 (`hek-engine-fiber`, `hek-web-next`).
- Environment: compose env 파일이 없어 interpolation이 실패. 설정으로 숨기지 않음.
- `hek-extension-chrome`은 평가만. 결함 아님.

## 제안

| old surface | DVA name | alias/보류 | reason |
| --- | --- | --- | --- |
| `modes.core` / `modes.full` | `local-infra` 등 named plan | 보류 | plan 의미를 바꾸는 Migrate. wave-1 조건(Preserve, 기계적 한 줄) 불충족 |
| `make dev` / `make dev-infra` (파일 주석) | `dva up <plan>` | 보류 | 위와 같음 |
| hek-engine-fiber, hek-web-next | 자식 `dva.yml` + 루트 subproject | 보류 | subproject/자식 링크 추가는 wave-1이 아님 |
| package.json build/lint/test | 동명 interaction | 보류 | 명령 표면 추가 |
| `compose/.env` | 유지 | 보류 | 없는 비밀 파일. Environment |

적용하지 않음: 읽기 전용 감사이고 모드가 Migrate다.

## wave-1

아니오.
