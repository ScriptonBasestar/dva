# flow-taskchain

## 대상

- 경로: `/Users/archmagece/mydevbox/flow-taskchain-devbox`
- 브랜치: `develop`
- HEAD: `7211e54d797aa34f5520f6c03a01d0662361f5af`
- DVA: 0.3.0 (`/Users/archmagece/go/bin/dva`, commit `13dc89398b921c6535ec80ab890f4b04b8f3020c`)
- 루트 설정: `dva.yml`. `legacy:` 없음. deprecated 섹션 없음.

## 판정 and 모드

- 판정: **partial** — validate exit 0이지만 `flow-taskchain-cli`가 실행 표면만 있고 자식 `dva.yml`/subproject가 없다.
- 모드: **Preserve**. `local-infra`/`local-dev` 등과 자식 plan import가 동작 의도로 유효하다.

## 자식 인벤토리

| workspace | 로컬 | 실행 표면 | 자식 dva | root subproject |
| --- | --- | --- | --- | --- |
| flow-taskchain-admin | 예 | 예, Makefile+npm | 예, stack `admin` plan `dev` | 예 `admin` import plans |
| flow-taskchain-cli | 예 | 예, Makefile include+go.mod (`build`/`dev`) | 아니오 | 아니오 |
| flow-taskchain-engine | 예 | 예, Makefile+compose+go.mod | 예, stack `engine` | 예 `engine` |
| flow-taskchain-mcp | 예 | 예, Makefile+compose+go.mod | 예, stack `mcp` | 예 `mcp` |
| flow-taskchain-portal | 예 | 예, Makefile+npm+compose | 예, stack `portal` | 예 `portal` |
| tasuku-repo | 예 | 예, go.mod+cmd | 아니오 | 아니오 |

`tasuku-repo`는 `.gz-git.yaml`이 제3자 upstream 참조, access read-only, 원격 쓰기 금지로 적는다. 제품 모듈에서 제외하고 빈 `dva.yml`을 제안하지 않는다. `tests/fixtures/dva-invalid/dva.yml`은 고의로 깨진 픽스처이며 workspace가 아니다.

## 기준선

- `dva config validate --json`: exit 0. warning 9, error 0. suppressed suggestion 102.
- warning: `config_drift` 1, `config_suggestion` 5, `ignore_stale` 3. 소유자 DVA config.
- `dva doctor --json`: exit 0. JSON 실패 1.
- 실패: `Encrypted env source declared` — DVA config. `.sops.yaml`, `.env.sops`가 있고 `env_file`에 `sops_source`가 없다.
- 사용자 체크 `compose.yaml exists`(`deploy/local/compose.yaml`)는 기본 체크 `Compose file exists: deploy/local/compose.yaml`과 중복이고 둘 다 통과. 소유자 DVA config.

## 발견

- DVA config. `flow-taskchain-cli` 미연결. 증거: `.gz-git.yaml` workspace, `flow-taskchain-cli/Makefile`, 루트 `subprojects`에 cli 없음. 루트 interaction `cli`는 `go run` 래핑이지 자식 설정이 아니다.
- DVA config. `ignore_stale`: `local-compose-dev-*`. 대응 패턴 룰이 없다.
- DVA tool. `env-edit-*`, `env-show-*`는 `.make/env.mk`의 `env-edit-%`, `env-show-%`다. validator가 `%` 룰을 타깃으로 보지 않는다. 지우지 않는다.
- DVA config. `config_drift`: `deploy/local/compose.e2e.yaml`이 stack `files`(`deploy/local/compose.yaml`)에 없다.
- DVA config. `config_suggestion`: `check-ci-child-deps`, `gap-303-status`, `validate-dva-config`, `validate-task-graph`, `verify-cross-repo-evidence`.
- DVA config. sops 미선언. 비밀이라 wave-1 아님.
- 루트 interaction 상당수가 아직 `make`를 호출한다. 일괄 이전은 이번 범위가 아니다.

## 제안 표

| 기존 표면 | DVA 이름 | alias/보류 | 이유 |
| --- | --- | --- | --- |
| `local-compose-dev-*` | 그 항목만 삭제 | 예, 미적용 | plan/비밀/compose 불변. 재확인은 validate |
| `env-edit-*`, `env-show-*` | 유지 | 보류 | DVA tool. `%` 패턴 룰 오인 |
| `flow-taskchain-cli` make | 자식 `dva.yml` 후 subproject | 보류 | 자식 파일이 먼저다 |
| tasuku `cmd` | 없음 | 보류 | upstream 참조, read-only |
| `compose.e2e.yaml` | stack files | 보류 | e2e를 local plan에 넣으면 의미가 바뀐다 |
| Makefile 제안 5개 | interaction | 보류 | CI/증거 타깃을 개발 명령으로 올릴지 불명 |
| sops / 중복 compose 체크 | 보류 | 보류 | 비밀, 또는 validate로 효과가 안 보임 |

적용하지 않음: 읽기 전용.

## wave-1

처음에는 적용하지 않았다. `ce task run-doctor`가 BLOCKED였고, 체크아웃의 `wt`는 0.74.0이며 런타임은 0.80 이상을 요구했다.

툴체인 핀 `20975563`이 `develop`에 반영되어 `cargo:worktrunk`가 0.80.0이 된 뒤 doctor는 ACTIVE다. `local-compose-dev-*`만 지운 커밋 `f18c6d35`를 `ce task run-finish`로 `develop`에 반영하고 태스크 브랜치와 워크트리를 회수했다. `env-edit-*`, `env-show-*`, `local-native-dev-*`는 남겼다. 적용 후 `dva config validate`는 exit 0이고, 남은 `ignore_stale`은 `env-edit-*`와 `env-show-*`다.

## wave-2

예. `env_file`의 `.env` 항목에 `sops_source: .env.sops`를 더한 커밋 `ae4f7bb7`가 `develop`에 있다. `ce task run-finish`로 통합했다. 이 리포트가 sops를 "비밀 로딩 변경"으로 보류한 근거는 틀렸다. `sops_source`는 로드 경로가 읽지 않는 선언 메타데이터다([follow-ups](follow-ups.md#sops_source-선언-wave-2)). primary 체크아웃 재측정에서 doctor fail은 1에서 0이다. `Encrypted env source declared` 행은 없다.
