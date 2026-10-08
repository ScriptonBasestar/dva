# familybook

## 대상

- 경로: `/Users/archmagece/mydevbox/familybook-devbox`
- 브랜치: `develop`
- HEAD: `5b42c086e9ae44a6e37534f89b84b35b50ee6e54`
- DVA: `/Users/archmagece/go/bin/dva` 0.3.0
- 루트 `dva.yml` 있음. `.gz-git.yaml`의 `legacy:`는 없음.

## 판정과 모드

- 판정: **partial**
- 모드: **Preserve**
- `plans` `infra`/`backend`/`dev`/`monitoring`과 `default_plan: infra`는 동작하는 의도다. deprecated 섹션 없음.
- partial: 실행 표면과 자식 `dva.yml`이 있는 `familybook-engine-fiber`가 루트 `subprojects`에 없다. validate warning도 남는다.

## 자식 인벤토리

| workspace | exists | 실행 표면 | dva.yml | 루트 subproject |
| --- | --- | --- | --- | --- |
| familybook-engine-fiber | 예 | Makefile, go.mod | 예 (`stack`/`plans`) | 없음 |
| familybook-client-kmp | 아니오 | 불가 | 불가 | 없음. unavailable. 클론하지 않음 |

## 기준선

- validate exit 0. warning 39. error 0.
  - `config_drift` 1: `compose.apps.yaml`, `compose.devcontainer.yaml`, `compose.gateway.yaml`, `compose.local-dev.yaml`, `compose.sigdock-dev.yaml`, `compose.sigdock.yaml`, `compose.test.yaml`가 stack 목록에 없다. stack은 `compose.yaml`과 `compose.monitoring.yaml`만 쓴다. owner: DVA config.
  - `config_suggestion` 38: Makefile 타깃 미매핑. owner: DVA config.
- doctor exit 1. JSON fail 2 / 14.
  - `Encrypted env source declared`: `.env` 항목에 `sops_source` 없음. owner: DVA config.
  - `KMP checkout`: `familybook-client-kmp/gradlew` 없음. owner: Environment (미클론). 체크를 지워 숨기지 않음.

## 발견

| owner | 증거 |
| --- | --- |
| DVA config | `familybook-engine-fiber/dva.yml`이 있는데 루트에 `subprojects` 키가 없다. 루트 `backend`는 `./scripts/devbox.sh backend-run`을 그대로 둔다. import 없는 path 선언은 계획 의미를 바꾸지 않는다. |
| DVA config | Compose drift 7파일. 어느 plan에 넣을지는 계획 의미 변경이라 wave-1이 아니다. |
| DVA config | 사용자 체크 `Docker daemon`은 doctor 내장 Docker 체크와 겹친다. |
| Environment | KMP workspace 부재. `FAMILYBOOK_KMP_DIR` 미설정. |

## 제안 표

| 기존 표면 | DVA 이름 | alias/보류 | 이유 |
| --- | --- | --- | --- |
| `familybook-engine-fiber` `dva.yml` | `subprojects.familybook-engine-fiber.path` | 보류 | import 없이 연결만. 이번 실행은 제안만. |
| 미등록 compose 7파일 | 기존 또는 새 plan의 compose runner | 보류 | 서비스 집합을 정해야 한다. |
| Makefile `db-*`, `env-*`, `deploy-*`, `kmp`는 일부 이미 interaction | 나머지 38개는 보류 | 보류 | 한 기계적 변경이 아니고 make 포장이 이관은 아니다. |
| KMP `gradlew` | 유지 (`kmp-*`) | 보류 | 체크아웃이 없다. |
| sops | `sops_source: .env.sops` | 보류 | 비밀 로딩 변경. |

적용하지 않음: 읽기 전용. KMP 실패를 설정 삭제로 막지 않음.

## wave-1

아니오. `familybook-engine-fiber` subproject 추가는 새 연결이라 이번 wave-1이 아니다. 제안 표에 남긴다. KMP 경로는 선언하지 않는다.

## wave-2

해당 없음. `sops_source: .env.sops`는 감사 전 커밋 `3f6980f`부터 선언돼 있었다. 위 doctor 발견의 `Encrypted env source declared`는 같은 dva 0.3.0으로 primary 체크아웃에서 다시 재면 나오지 않는다. 감사 기록의 원인은 확인하지 못했다. 남은 fail은 `KMP checkout`다.
