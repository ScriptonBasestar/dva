# primeno1

## 대상

- 경로: `/Users/archmagece/mydevbox/primeno1-devbox`
- 브랜치: `master`
- HEAD: `82757e496c2612b126a3b5733f4fb4cb5dc71fa2`
- DVA: `0.3.0` (`/Users/archmagece/go/bin/dva`)
- 루트·`primeno1-engine-kt`·`primeno1-frontend`에 `dva.yml`.

## 판정

- 판정: `applied`
- 모드: `Preserve`
- validate exit 0. `modes`/`applications` 없음. 로컬 자식 둘이 `subprojects`로 연결됨. import는 없어 루트 `dva ls`에는 숨지만, 연결 자체는 되어 있다.

## 자식 인벤토리

`workspaces:`와 `legacy:` 키는 없다. 자식 선언은 `repositories:`.

| repositories | exists | executable surface | dva.yml | root subproject |
| --- | --- | --- | --- | --- |
| primeno1-engine-kt | 예 | 예 (Makefile, docker-compose.yml, gradlew) | 예 | `primeno1-engine-kt` |
| primeno1-frontend | 예 | 예 (Makefile, package.json scripts) | 예 | `primeno1-frontend` |

## 기준선

- validate exit 0, warning 6, error 0
  - `config_drift` — `env/docker-compose/docker-compose.task44.verify.yml`, `task45.verify.yml`. owner: DVA config
  - `ignore_stale` ×5 — `suggestion_ignore[0..4]`: `ci-build*`, `deploy*`, `backup*`, `prepare`, `ws-*`. 루트 Makefile에 해당 타깃 없음. owner: DVA config
- doctor exit 1. 재실행(2026-10-08) JSON fail 2. 초고의 Node 실패는 재실행에서 재현되지 않았다.
  - `Compose config resolves` — compose config가 non-zero. owner: Environment
  - `Required compose env inputs` — `./scripts/check-compose-env.sh` 실패. owner: Environment

## 발견

- DVA config: stale ignore 5줄. 증거: `dva.yml` `suggestion_ignore`, validate `ignore_stale`. verify compose 둘은 일회성 파일로 보이며 stack에 넣지 않음.
- Environment: 언실된 compose 비밀과 doctor PATH. 설정으로 침묵시키지 않음.
- 자식은 연결되어 있다. `setup`, `validate*`, `version` ignore는 stale가 아님.

## 제안

| old surface | DVA name | alias/보류 | reason |
| --- | --- | --- | --- |
| suggestion_ignore 5패턴 | 항목 삭제 | wave-1 후보, 미적용 | validate가 stale로 지목. 이번 실행은 읽기 전용 |
| task44/45 verify compose | stack files | 보류 | compose 파일 집합이 바뀐다 |
| 자식 interaction | 루트 import | 보류 | import 추가는 wave-1이 아님 |
| compose env 체크 | 유지 | 보류 | 실패 원인은 없는 비밀값 |

감사 직후에는 파일을 고치지 않았다. 적용과 통합은 아래 wave-1이다.

## wave-1

예. 워크트리 `dev/grok/mbp/chore/dva-wave1` 커밋 `13abac8`에서 `ci-build*`, `deploy*`, `backup*`, `prepare`, `ws-*`를 삭제했다. `setup`, `validate*`, `version`은 남겼다. 적용 후 `dva config validate` exit 0, warning 1. 남은 경고는 verify compose drift다.

`ce task run-finish`가 이 커밋을 `master`에 fast-forward 했고, 워크트리와 태스크 브랜치를 회수했다. primary `master`는 `13abac8`이다.
