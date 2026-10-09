# scripton-db-orchestrator

## 대상

- 경로: `/Users/archmagece/mydevbox/scripton-db-orchestrator-devbox`
- 브랜치: `master`
- HEAD: `4bdbe5b9604b1f1b278d718bb8c31ca9e4ba1c8d`
- DVA: `/Users/archmagece/go/bin/dva` 0.3.0 (commit `13dc89398b921c6535ec80ab890f4b04b8f3020c`)

## 판정 and 모드

- 판정: **partial**
- 모드: **Preserve**
- `plans`/`stack` 의도가 있고 `modes`/`applications`는 없다. 감사 당시 validate는 `interaction.ci` 예약어 때문에 exit 1이었다. 반영은 아래 후속이다.

## 자식 인벤토리

별도 `legacy:` 키는 없다. `legacy/…` 워크스페이스는 주석상 archived lineage이고 `access: read-only`라 제외한다.

| workspace / 경로 | exists | 실행 표면 | dva.yml | 루트 subproject |
| --- | --- | --- | --- | --- |
| db-orchestrator-ui | 예 | 예 (package.json scripts) | 예 | `db-orchestrator-ui` |
| db-orchestrator-rs | 예 | 아니오 | 없음 | `db-orchestrator-rs` (선언만) |
| legacy/db-orchestrator-api, cli, core, sdk, worker | 예 | 아니오 (go.mod만) | 없음 | 없음, 제외 |
| legacy/db-orchestrator-operator | 예 | 예 (Makefile, go.mod) | 없음 | 없음, 제외 |
| `deploy/test` (gz 워크스페이스 아님, 같은 저장소) | 예 | 예 (docker-compose.yaml) | 예 | `deploy-test` |

`db-orchestrator-rs` 루트에는 Cargo.toml·Makefile·package.json·Compose가 없다. 중첩 디렉터리는 `.ce`와 `src` 수준이라 표면 없음(평가). 빈 `dva.yml`을 제안하지 않는다. 제외된 legacy는 결함이 아니다.

## 기준선

- `dva config validate --json`: exit 1. warning 2 (`ignore_stale`). error 1. 감사 당시 값이다.
- `dva doctor --json`: exit 0. JSON fail 1. stderr는 `interaction.ci` 예약어 경고를 반복한다.

| 항목 | owner |
| --- | --- |
| `interaction.ci` 예약 명령 | DVA config |
| `ignore_stale` `k8s-secret-apply-*`, `k8s-secret-edit-*` | DVA tool |
| Encrypted env source declared | DVA config |

error 원인: `'ci'`는 `dva ci` 빌트인이라 이 interaction은 `dva run ci`로만 도달한다.

## 발견

- DVA config: `interaction.ci` 이름 충돌. 이름을 바꾸면 호출 표면이 바뀌고 Make와 문서가 같은 이름을 쓸 수 있어, 계획 불변인 한 줄 수정으로 보지 않는다.
- DVA tool: ignore는 `k8s-secret-apply-*` 형태인데 실제 타깃은 `.make/env.mk`의 `k8s-secret-apply-%` 패턴 룰이다. validator가 `%` 룰을 못 본 것으로 보는 stale이다. ignore를 삭제하지 않는다.
- DVA config: `.sops.yaml`·`.env.sops`에 `sops_source` 없음. 비밀이라 wave-1 아님.
- `db-orchestrator-rs` subproject 선언은 자식 `dva.yml`이 없지만 실행 표면이 없어 결함으로 세지 않는다.

## 제안 표

| 옛 표면 | DVA 이름 | alias/보류 | 이유 |
| --- | --- | --- | --- |
| `interaction.ci` / `dva run ci` | `ci-check` | 적용 `132a43f8` | 호출부가 없어 이름만 바꿨다. `command: make ci`는 그대로다 |
| `k8s-secret-*-%` | 기존 ignore 유지 | 보류 | DVA tool |
| `db-orchestrator-rs` | 없음 | 보류 | 루트 실행 표면 없음 |
| legacy operator Makefile | 없음 | 보류 | read-only 아카이브 |

## wave-1

아니오. 적용하지 않았다.

## 후속

예. `interaction.ci`를 `ci-check`로 바꾼 커밋 `132a43f8`이 `master`에 있다. `command: make ci`와 tags는 그대로다. 저장소 안에 `dva ci` 호출은 없다. 설치본 `dva` 0.3.0(`a0deef70`)으로 primary에서 `dva config validate`는 exit 0이다. `k8s-secret-apply-*`와 `k8s-secret-edit-*` ignore는 그대로 두었고, 그 설치본은 `%` 패턴 룰 수정(`56cc792c`)보다 앞이라 이 둘을 아직 stale로 본다. `sops_source`는 선언하지 않았다.
