# flow-pipechain

## 대상

- 경로: `/Users/archmagece/mydevbox/flow-pipechain-devbox`
- 브랜치: `develop`
- HEAD: `c7733941254c413f4693b17acedee98421c8f952`
- DVA: 0.3.0 (`/Users/archmagece/go/bin/dva`, commit `13dc89398b921c6535ec80ab890f4b04b8f3020c`)
- 루트 설정: `dva.yml`. `legacy:` 없음. deprecated 섹션 없음.

## 판정 and 모드

- 판정: **applied** — validate exit 0이고 workspace 7곳이 모두 자식 `dva.yml`과 root `subprojects`에 있다. import는 생략되어 루트 `dva ls`에는 안 나오지만, 연결 자체는 성립한다.
- 모드: **Preserve**. warning 7은 error/deprecation이 아니다. `flow-pipechain-examples`와 `optional/`은 workspace가 아니라 연결 대상이 아니다.

## 자식 인벤토리

| workspace | 로컬 | 실행 표면 | 자식 dva | root subproject |
| --- | --- | --- | --- | --- |
| flow-pipechain-admin-ui | 예 | 예, Makefile+npm | 예 | 예 `admin-ui` |
| flow-pipechain-agent | 예 | 예, Makefile+go.mod | 예 | 예 `agent` |
| flow-pipechain-cli | 예 | 예, Makefile+go.mod | 예 | 예 `cli` |
| flow-pipechain-plugin-sdk | 예 | 예, Makefile+go.mod | 예 | 예 `plugin-sdk` |
| flow-pipechain-portal | 예 | 예, Makefile+npm | 예 | 예 `portal` |
| flow-pipechain-server | 예 | 예, Makefile+go.mod | 예 | 예 `server` |
| flow-pipechain-shared | 예 | 예, Makefile+go.mod | 예 | 예 `shared` |

## 기준선

- `dva config validate --json`: exit 0. warning 7, error 0. suppressed suggestion 107.
- warning: `config_drift` 1, `config_suggestion` 1, `ignore_stale` 5. 소유자 DVA config.
- `dva doctor --json`: exit 0. JSON 실패 1.
- 실패: `Encrypted env source declared` — DVA config. 후보는 `.sops.yaml`, `.env.sops`, 그리고 `.env.sops.bak.*` 두 개. 백업 파일을 `sops_source`로 선언하면 안 된다.

## 발견

- DVA config. `ignore_stale` 중 실제 타깃이 없는 3건. 증거: 루트 `dva.yml` `suggestion_ignore`의 `dev-keycloak-export-realm`, `dev-keycloak-only`, `log-search-bench perf-log-search`. 같은 파일의 `log-search-bench`와 `perf-log-search` 단독 항목은 stale가 아니다.
- DVA tool. `env-edit-*`, `env-show-*`는 `.make/env.mk`의 `env-edit-%`, `env-show-%` 패턴 룰을 가리킨다. validator는 `%` 룰을 타깃으로 보지 않아 stale로 보고한다. 이 항목은 지우지 않는다.
- DVA config. `config_drift`: `deploy/local/compose.app.yaml`, `compose.https.yaml`, `compose.infra.yaml`이 stack `files`에 없다. 루트 stack은 `compose.yaml`만 쓴다.
- DVA config. `config_suggestion`: Makefile `doc-links-selftest`.
- DVA config. 사용자 체크 `Docker daemon accessible`이 doctor 기본 체크와 중복이고 둘 다 통과했다.
- DVA config. doctor sops 실패. 비밀 로딩이라 wave-1이 아니다.
- 평가: 루트 native(`api`/`agent`/`portal-ui`/`admin-ui`)와 자식 interaction이 같이 있다. 자식 파일은 있으므로 미연결은 아니다.

## 제안 표

| 기존 표면 | DVA 이름 | alias/보류 | 이유 |
| --- | --- | --- | --- |
| 없는 타깃 ignore 3줄 | 해당 리스트 항목 삭제 | 예, 미적용 | plan/비밀/compose를 바꾸지 않는다. validate만으로 재확인 |
| `env-edit-*`, `env-show-*` | 유지 | 보류 | DVA tool. `%` 패턴 룰을 stale로 오인 |
| `deploy/local/compose.*.yaml` | stack files | 보류 | 실행 파일 집합이 바뀐다 |
| `doc-links-selftest` | interaction | 보류 | 편입 여부가 기계적이지 않다 |
| 중복 Docker 체크 | 기본 doctor | 보류 | 효과는 doctor이지 validate가 아니다 |
| sops | `sops_source: .env.sops`만 후보 | 보류 | 비밀. `.bak`는 제외 |

감사 직후에는 파일을 고치지 않았다. 적용과 통합은 아래 wave-1이다.

## wave-1

예. 워크트리 `dev/grok/mbp/chore/dva-wave1` 커밋 `e93b2b4`에서 `dev-keycloak-export-realm`, `dev-keycloak-only`, `log-search-bench perf-log-search`만 삭제했다. `env-edit-*`와 `env-show-*`는 남겼다. 적용 후 `dva config validate` exit 0, warning 4. 남은 `ignore_stale` 2건은 그 패턴 룰이다.

`branch-integrate`가 이 커밋을 `develop`에 fast-forward 했고 태스크 브랜치를 회수했다. primary `develop`은 `e93b2b4`다. `make check`와 `make lint`는 자식 체크아웃이 없는 워크트리와 깨끗한 기준 트리가 달라 기준선을 재지 못했다. 브랜치 진단은 0건이라 `--allow-skipped-checks`로 그 비교만 경고로 내렸다.

## wave-2

예. `env_file`의 `.env` 항목에 `sops_source: .env.sops`를 더한 커밋 `7f03ac4`가 `develop`에 있다. `make check`와 `make lint`는 기준선을 재지 못해 `--allow-skipped-checks`로 통합했다. `.env.sops.bak.*`는 선언하지 않았다. 이 리포트가 sops를 "비밀 로딩 변경"으로 보류한 근거는 틀렸다. `sops_source`는 로드 경로가 읽지 않는 선언 메타데이터다([follow-ups](follow-ups.md#sops_source-선언-wave-2)). primary 체크아웃 재측정에서 doctor fail은 1에서 0이다. `Encrypted env source declared` 행은 없다.
