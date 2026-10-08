# gizzahub

## 대상

- 경로: `/Users/archmagece/mydevbox/gizzahub-devbox`
- 브랜치: `develop`
- HEAD: `65bcbc7a4f9ceddcf70501f1c75e86302cb5d980`
- DVA: `/Users/archmagece/go/bin/dva` 0.3.0
- 루트 `dva.yml` 있음. `.gz-git.yaml`의 `legacy:`는 없음.

## 판정과 모드

- 판정: **partial**
- 모드: **Preserve**
- 루트 `plans`(`local-infra`, `local-dev`, `full-stack` 등)와 세 자식 import는 유지할 의도다. deprecated 섹션 없음.
- partial: `grabber-social-web-py`와 `grabber-social-web-go`는 로컬에 있고 자식 `dva.yml`이 있는데 루트 subproject가 아니다.

## 자식 인벤토리

| workspace | exists | 실행 표면 | dva.yml | 루트 subproject |
| --- | --- | --- | --- | --- |
| gizzahub-engine-fiber | 예 | Makefile, go.mod | 예 | 예, import `plans.backend` |
| gizzahub-infocenter | 예 | Makefile, package scripts | 예 | 예, import `plans.web` |
| gizzahub-web-svelte | 예 | Makefile, package scripts, compose.yaml | 예 | 예, import `plans.frontend` |
| grabber-social-web-py | 예 | Makefile, compose.yaml, pyproject.toml | 예 | 없음 |
| grabber-social-web-go | 예 | Makefile, go.mod | 예 | 없음 |

`.omo/evidence/**/dva.yml` 두 파일은 fixture다. workspace가 아니므로 인벤토리에서 제외했다.

## 기준선

- validate exit 0. warning 14, 모두 `config_suggestion` (task-1035/1036 계약 타깃, otel/idempotency 테스트, `validate-product-plan`). owner: DVA config. error 0. suppressed 142.
- doctor exit 0. JSON fail 1 / 13: `Encrypted env source declared`. `env_file.files`의 `.env`에 `sops_source` 없음. `.sops.yaml`, `.env.sops`는 있다. owner: DVA config.

## 발견

| owner | 증거 |
| --- | --- |
| DVA config | `.gz-git.yaml` `workspaces.grabber-social-web-py` / `grabber-social-web-go`와 각 `dva.yml`. 루트 `subprojects`는 engine, infocenter, web-svelte만 있다. 이름 때문에 건너뛰지 않는다. |
| DVA config | suggestion 14건은 계약·픽스처 Make 타깃이다. interaction으로 올리면 명령 표면이 바뀐다. |
| DVA config | sops 미선언. 비밀 로딩이라 wave-1 제외. |

## 제안 표

| 기존 표면 | DVA 이름 | alias/보류 | 이유 |
| --- | --- | --- | --- |
| grabber 두 자식 `dva.yml` | `subprojects.<name>.path`만 | 보류 | import 없으면 기존 plan 목록이 유지된다. |
| grabber plan/interaction | 루트 import | 보류 | 루트 `dva ls`에 올릴 이름은 별도 선택이다. |
| task-1035/1036 Make | 보류 | 보류 | 단발 계약 타깃. make 포장은 이관이 아니다. |
| sops | `sops_source: .env.sops` | 보류 | 비밀 입력 변경. doctor로만 재확인된다. |

적용하지 않음: 읽기 전용.

## wave-1

아니오. grabber 두 경로의 subproject 추가는 새 연결이라 이번 wave-1이 아니다. 제안 표에 남긴다.

## wave-2

예. `env_file`의 `.env` 항목에 `sops_source: .env.sops`를 더한 커밋 `425f9134`가 `develop`에 있다. `branch-integrate` readiness 13개를 통과했다. `origin/dev/grok/mbp/issue-holds`와의 cross-merge 경고는 그 브랜치를 두고 넘겼다. 이 리포트가 sops를 "비밀 로딩 변경"으로 보류한 근거는 틀렸다. `sops_source`는 로드 경로가 읽지 않는 선언 메타데이터다([follow-ups](follow-ups.md#sops_source-선언-wave-2)). primary 체크아웃 재측정에서 doctor fail은 1에서 0이다. `Encrypted env source declared` 행은 없다.
