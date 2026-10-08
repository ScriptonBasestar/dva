# scripton-nd-stack

## 대상

- 경로: `/Users/archmagece/mydevbox/scripton-nd-stack-devbox`
- 브랜치: `develop`
- HEAD: `e7ddadddabd27d96359dd4d1f671ea34515c3897`
- DVA: `/Users/archmagece/go/bin/dva` 0.3.0 (commit `13dc89398b921c6535ec80ab890f4b04b8f3020c`)

## 판정 and 모드

- 판정: **applied**
- 모드: **Preserve**
- `plans`/`stack`이 있고 deprecated 섹션은 없다. validate exit 0. 활성 워크스페이스 `ndea-rs`, `ndea-webui`는 자식 `dva.yml`과 subproject로 연결된다. `legacy/*`는 `.gz-git.yaml`의 `access: read-only`와 키 접두로 제외한다.

## 자식 인벤토리

별도 `legacy:` 키는 없다. 아래 `legacy/…`는 `workspaces` 항목이다.

| workspace | exists | 실행 표면 | dva.yml | 루트 subproject |
| --- | --- | --- | --- | --- |
| ndea-rs | 예 | 예 (Makefile, compose/, Cargo.toml) | 예 | `ndea-rs` |
| ndea-webui | 예 | 예 (Makefile, package.json scripts) | 예 | `ndea-webui` |
| legacy/proxynd-cli | 예 | 예 (Makefile, go.mod) | 없음 | 없음, 제외 |
| legacy/proxynd-ce | 예 | 예 (Makefile, docker-compose.yml, go.mod) | 없음 | 없음, 제외 |
| legacy/proxynd-ee | 예 | 예 (Makefile, go.mod) | 없음 | 없음, 제외 |
| legacy/proxynd-cloud | 아니오 | 없음 | 없음 | unavailable, 제외 |
| legacy/proxynd-sdk | 예 | 예 (Makefile, go.mod) | 없음 | 없음, 제외 |

## 기준선

- `dva config validate --json`: exit 0. warning 13 (`config_drift` 1, `config_suggestion` 7, `ignore_stale` 5). error 0.
- `dva doctor --json`: exit 1, stderr `1 user check(s) failed`. JSON fail 2.

| 항목 | owner |
| --- | --- |
| `config_drift`, `config_suggestion`, `ignore_stale` | DVA config |
| Encrypted env source declared | DVA config |
| Local environment file exists (`.env` 없음) | Environment |

## 발견

- DVA config: `deploy/local/compose.disposable-project.yaml`, `compose.persistent-artifacts.yaml`, `compose.schema-isolation.yaml`이 있고 스택 compose `files`는 `deploy/local/compose.yaml`만 가리킨다. 파일을 넣으면 스택 의미가 바뀌므로 wave-1이 아니다.
- DVA config: Makefile `test-cloud-settings`, `test-component-orphans`, `test-context-docs`, `test-guard-usage`, `test-kubeconform`, `test-task-index`, `test-task-links`에 같은 이름 interaction이 없다. 제안일 뿐 일괄 이관하지 않는다.
- DVA config: `suggestion_ignore`의 `build-*`, `tidy`, `test-sdk`, `test-cli`, `test-coverage*`는 루트 Makefile·`.make`에 없다.
- DVA config: sops 파일(`.sops.yaml`, `.env.sops`, `.env.docker.sops`)에 `sops_source` 없음. 비밀이라 wave-1 제외.
- Environment: 사용자 체크가 `.env` 부재를 보고한다.
- legacy 읽기 전용 모듈과 없는 `proxynd-cloud`는 결함이 아니다. 클론하지 않았다.

## 제안 표

| 옛 표면 | DVA 이름 | alias/보류 | 이유 |
| --- | --- | --- | --- |
| 세 compose 파일 | 기존 `stack.compose`에 추가 | 보류 | 포함 범위가 계획 의미. 읽기 전용 |
| Make `test-*` 7개 | 동명 interaction | 보류 | 워크플로인지 불명. `make` 포장 이관 금지 |
| stale ignore 5개 | 항목 삭제 | 보류 | wave-1 후보, 미적용 |
| legacy proxynd | 없음 | 보류 | read-only 아카이브로 제외 |

## wave-1

예. 워크트리 `dev/grok/mbp/chore/dva-wave1` 커밋 `887a332b`에서 그 5개를 삭제했다. 기준은 조사 당시 HEAD가 아니라 당시 `develop` 팁 `b6da3565`다. 적용 후 `dva config validate` exit 0, warning 8. 남은 경고는 drift 1건과 suggestion 7건이다.

`branch-integrate`가 이 커밋을 `develop`에 fast-forward 했고 태스크 브랜치를 회수했다. primary `develop`은 `887a332b`다. `make check`와 `make lint`는 통과했다. `origin/dev/claude/mst/chore/npm-tenant-isolation-gap-card`와 cross-merge 충돌 경고가 있었고, 그 브랜치는 그대로 두었다.
