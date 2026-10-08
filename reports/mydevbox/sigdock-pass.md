# sigdock-pass

## 대상

- 경로: `/Users/archmagece/mydevbox/sigdock-pass-devbox`
- 브랜치: `master`
- HEAD: `8126742d9afab7fdef9a6e49ff82c11a721022a3`
- DVA: `/Users/archmagece/go/bin/dva` 0.3.0
- 루트 `dva.yml` 있음. `.gz-git.yaml`의 `legacy:`는 없음. `sigdock-pass-server-go`는 주석상 아카이브이며 로컬에 없다.

## 판정과 모드

- 판정: **partial**
- 모드: **Preserve**
- `stack`/`plans`(`primary`, `ha-postgres`, `ha-redis`, `federation`)는 동작하는 의도다. deprecated 섹션 없음. Rewrite 근거 없음.
- partial인 이유: 실행 표면이 있는 workspace 5곳에 자식 `dva.yml`이 없고 루트 `subprojects`도 없다.

## 자식 인벤토리

| workspace | path | exists | 실행 표면 | dva.yml | 루트 subproject |
| --- | --- | --- | --- | --- | --- |
| sigdock-pass-server | sigdock-server-rs | 예 | 예 (`src/main.rs`, `tests/`, Dockerfile) | 아니오 | 없음 |
| sigdock-pass-ts | sigdock-ts | 예 | 예 (scripts `dev`/`build`/`test`) | 아니오 | 없음 |
| sigdock-pass-vscode | sigdock-vscode | 예 | 예 (scripts `compile`/`test`) | 아니오 | 없음 |
| sigdock-pass-client | sigdock-client | 예 | 예 (`[[bin]]` `sdctl`) | 아니오 | 없음 |
| sigdock-pass-extension | sigdock-extension | 예 | 예 (script `package`) | 아니오 | 없음 |
| sigdock-pass-shared | sigdock-shared | 예 | 아니오 | 아니오 | 없음 |

`sigdock-shared`는 openapi/proto와 evidence 스크립트만 있다. 루트 매니페스트·Makefile·compose가 없다. 결함 아님.

## 기준선

- `dva config validate --json`: exit 0, valid, error 0, warning 1. `config_drift` 1: `compose-airgap.yaml`, `compose-chaos.yaml`, `compose-cpd.yaml`, `compose-saas.yaml`, `compose-saas-aws.yaml`, `compose-saas-azure.yaml`, `compose-saas-gcp.yaml`. owner: DVA config. 파일 헤더가 이 overlay를 의도적으로 미등록이라고 적는다.
- `dva doctor --json`: exit 0, fail 3 / 11.
  - `Encrypted env source declared` — `.sops.yaml`, `.env.sops`, `sops_source` 없음. owner: DVA config.
  - `Compose config resolves` — exit 1. `.env` 없음. 필수 변수 미설정 (`POSTGRES_PASSWORD`, `RUSTFS_*`, `SD_RUSTFS_APP_USER`, `SD_RUSTFS_APP_SECRET`, `SD_RUSTFS_DATA_STATE`, `SD_AUDIT_MANIFEST_SIGNING_KEY_HEX`, `SD_WEBHOOK_MASTER_KEY`, `SD_WORKLOAD_MASTER_KEY`). 값은 기록하지 않음. owner: Environment.
  - `.sb/dva/ is ignored in .gitignore` — `.gitignore`에 `.sb` 없음. owner: Project.

## 발견

| owner | 증거 |
| --- | --- |
| DVA config | 위 5개 workspace에 자식 `dva.yml`이 없다. 없는 자식 파일로 `subprojects`를 선언하면 깨진 선언이다. |
| DVA config | `config_drift`. 의도는 `dva.yml` 헤더와 일치한다. |
| DVA config | `sops_source` 누락. |
| Environment | compose interpolation. 설정을 고쳐 경고를 끄지 않는다. |
| Project | `.gitignore`가 `.sb/dva/`를 무시하지 않는다. |

## 제안 표

| 기존 표면 | DVA 이름 | alias/보류 | 이유 |
| --- | --- | --- | --- |
| 자식 `make`/`cargo`/`pnpm`/`vsce` | 각 저장소 interaction (이름 미정) | 보류 | 자식 `dva.yml`이 먼저다. 루트 subproject 추가는 wave-1이 아니다. |
| 미등록 deploy/CI compose | 현행 plan 유지 + `drift_ignore` | 아래 wave-1 | 헤더가 미등록을 이미 말한다. stack에 넣지 않는다. |
| `.env.sops` / 빈 `.env` | 기존 `provision.init-sops` | 보류 | 비밀 복호화는 Environment. `sops_source`는 doctor 전용이다. |

감사 직후에는 파일을 고치지 않았다. 적용과 통합은 아래 wave-1이다.

## wave-1

예. 워크트리 `dev/grok/mbp/chore/dva-wave1` 커밋 `043ac033`에서 헤더가 이미 뺀 overlay만 `drift_ignore`에 적었다. stack과 plans는 그대로다. 적용 후 `dva config validate` exit 0, warning 0. 이 커밋은 `origin/dev/grok/mbp/chore/dva-wave1`에 있다.

통합 시점의 `master`보다 13커밋 뒤였다. `origin/master` 위로 rebase 했고 충돌은 없었다. rebase 커밋은 `35337616`다. rebase 뒤 `make lint`는 통과했다. `branch-integrate`는 원격 태스크 브랜치와 로컬이 달라 `push — upstream differs`에서 멈췄다. 워크트리 `/Users/archmagece/worktrees/sigdock-pass/sigdock-pass-devbox/grok__mbp__chore__dva-wave1`를 남겨 두었다.
