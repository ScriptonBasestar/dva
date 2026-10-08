# sadawiki

## 대상

- 경로: `/Users/archmagece/mydevbox/sadawiki-devbox`
- 브랜치: `master`
- HEAD: `29ead617cc747aca6e7ad380148dc1870d7ce558`
- DVA: `0.3.0` (`/Users/archmagece/go/bin/dva`)
- `dva.yml`은 루트만.

## 판정

- 판정: `applied`
- 모드: `Preserve`
- 루트에 plans(`infra`, `infra-tools`, `test-infra`)가 있고 validate exit 0. 연결할 로컬 workspace 자식이 없다.

## 자식 인벤토리

`.gz-git.yaml`의 `workspaces:`는 빈 키다. `legacy:` 없음. 로컬 자식 디렉터리를 클론하거나 빈 `dva.yml`을 만들지 않음. 실행 표면은 루트 Makefile, `package.json`, `compose.yaml`이다.

| workspace | exists | executable surface | dva.yml | root subproject |
| --- | --- | --- | --- | --- |
| (없음) | — | — | — | subprojects 없음 |

## 기준선

- validate exit 0, warning 0, error 0. `suggestion_ignore`는 매칭되어 suppressed.
- doctor exit 1 (`3 user check(s) failed`), JSON fail 4
  - `.env file exists` — `.env` 없음. owner: Environment
  - `Node.js available` — 감사 PATH에 없음. mise shim `v24.19.0` 존재. owner: Environment
  - `pnpm available` — 같은 이유. shim `12.8.2` 존재. owner: Environment
  - `.sb/dva/ is ignored in .gitignore`. owner: Project

## 발견

- Environment: 로컬 `.env`와 doctor PATH. `dva.yml`의 `env_file`은 `.env`를 `required: false`로 이미 선언. 체크 실패를 설정으로 지우지 않음.
- Project: `.gitignore`가 `.sb/dva/`를 무시하지 않음.
- DVA config 결함으로 볼 deprecation·미연결 자식은 없다.

## 제안

| old surface | DVA name | alias/보류 | reason |
| --- | --- | --- | --- |
| 루트 Make/npm (이미 ignore) | 유지 | 보류 | validate가 누락으로 보지 않음. 래핑 이관은 plan 밖 작업 |
| `.env` | 유지 | 보류 | Environment |
| `.gitignore` `.sb/dva/` | gitignore | 보류 | `dva.yml` 밖 Project 수정 |

적용하지 않음: 읽기 전용이고 wave-1에 해당하는 `dva.yml` 한 줄이 없다.

## wave-1

아니오.
