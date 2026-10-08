# flow-agent-mesh

## 대상

- 경로: `/Users/archmagece/mydevbox/flow-agent-mesh-devbox`
- 브랜치: `master`
- HEAD: `93c334b472335bd31cb1ce185f16df077091c716`
- DVA: 0.3.0 (`/Users/archmagece/go/bin/dva`, commit `13dc89398b921c6535ec80ab890f4b04b8f3020c`)
- 루트 설정: `dva.yml` (stack/plans/interaction). `modes`/`applications` 없음.

## 판정 and 모드

- 판정: **partial**
- 모드: **Preserve** — 루트 plan `infra`/`infra-test`/`full-stack`가 동작 의도다. validator JSON에 deprecated 섹션이 없어 Migrate가 아니고, 설정이 사용 불가가 아니므로 Rewrite가 아니다.
- 활성 workspace 3곳이 실행 표면이 있는데 자식 `dva.yml`과 root `subprojects`가 없다.

## 자식 인벤토리

`.gz-git.yaml`의 `legacy:` 키는 없다. `workspaces:`만 있다.

| workspace | 로컬 | 실행 표면 | 자식 dva | root subproject |
| --- | --- | --- | --- | --- |
| agent-mesh | 예 | 예, Makefile+go.mod | 아니오 | 아니오 |
| agent-mesh-flows | 예 | 예, Makefile+ops compose | 아니오 | 아니오 |
| agent-mesh-runner | 예 | 예, go.mod+cmd | 아니오 | 아니오 |
| agent-mesh-exts | 예 | 아니오, 작업 트리 없음(`.git`만), access read-only | 아니오 | 아니오 |
| agent-mesh-cli 등 7개 (`legacy/<name>`) | 예 | 예, Makefile(portal은 npm 포함) | 아니오 | 아니오 |

`legacy/*` 7개는 path가 `legacy/`이고 access가 read-only다. prompts는 강등 기록이 있다. 아카이브 경로로 보고 빈 `dva.yml`을 제안하지 않는다. exts는 표면 없음(결함 아님).

## 기준선

- `dva config validate --json`: exit 0. warning 0, error 0. suppressed suggestion 27.
- `dva doctor --json`: exit 1. stderr `ERROR: 1 user check(s) failed`. JSON 실패 2.
- 실패: `Encrypted env source declared` — DVA config (`.sops.yaml`, `.env.sops` 존재, `env_file`에 `sops_source` 없음).
- 실패: `.env file exists` — Environment (루트 `.env` 없음). `env_file`은 `.env`를 required false로 둔다. exit 1의 사용자 체크다.
- 통과 중복: 사용자 체크 `Docker daemon accessible`, `compose.yaml exists`가 doctor 기본 체크와 겹친다. 소유자 DVA config. 실패 원인은 아니다.

## 발견

- DVA config. `agent-mesh`, `agent-mesh-flows`, `agent-mesh-runner`는 실행 표면이 있고 `dva.yml`이 없다. 루트 `dva.yml`에 `subprojects`가 없다. 루트 native `am-server`는 이 세 저장소를 대신하지 않는다.
- DVA config. doctor `Encrypted env source declared`. 증거: 루트 `dva.yml` `env_file`, 파일명 `.sops.yaml`, `.env.sops`.
- DVA config. 사용자 체크가 Docker daemon과 `compose.yaml` 존재를 기본 체크와 중복한다. 증거: 루트 `dva.yml` `checks`.
- Environment. 루트 `.env` 부재. 설정으로 침묵시키지 않는다.
- exts와 `legacy/*`는 위 표의 평가이며 결함으로 세지 않는다.

## 제안 표

| 기존 표면 | DVA 이름 | alias/보류 | 이유 |
| --- | --- | --- | --- |
| `agent-mesh` 등 3개 Makefile/go | 자식 `dva.yml` + root `subprojects` | 보류 | 자식 파일이 먼저 있어야 선언할 수 있다. 루트 한 줄 수정이 아니다 |
| `legacy/*` make/npm | 없음 | 보류 | `legacy/` + read-only. 빈 설정을 만들지 않음 |
| sops 파일 | `env_file.sops_source` | 보류 | 비밀 로딩이 바뀐다 |
| 중복 Docker/compose 체크 | 기본 doctor에 위임 | 보류 | validate만으로 효과가 확인되지 않는다 |
| 없는 `.env` | 없음 | 보류 | Environment |

적용하지 않음: 이 실행은 읽기 전용이고, 자식 연결은 루트 `dva.yml` 한 건의 기계적 수정이 아니다.

## wave-1

아니오. 활성 자식 연결은 새 자식 파일이 필요하고, sops 선언은 비밀을 건드린다.
