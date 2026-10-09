# funbricks-notifire

## 대상

- 경로: `/Users/archmagece/mydevbox/funbricks-notifire-devbox`
- 브랜치: `develop`
- HEAD: `5c1cdc18e2fb7c925ea32f8cac337714aee6a430`
- DVA: `/Users/archmagece/go/bin/dva` 0.3.0 (commit `13dc89398b921c6535ec80ab890f4b04b8f3020c`)

## 판정 and 모드

- 판정: **applied**
- 모드: **Preserve**
- 루트 `dva.yml`에 `stack`/`plans`가 있고 deprecated 섹션은 없다. validate exit 0, warning 0. 실행 표면이 있는 워크스페이스 자식 둘 다 자식 `dva.yml`과 루트 `subprojects`로 연결된다.

## 자식 인벤토리

`workspaces`에 `path` 키가 없어 디렉터리명은 키와 같다. `legacy:` 없음.

| workspace | exists | 실행 표면 | dva.yml | 루트 subproject |
| --- | --- | --- | --- | --- |
| notifire-backend-phoenix | 예 | 예 (Makefile, mix.exs) | 예 | `backend` (`import.plans`: dev → `backend-dev`) |
| notifire-frontend-react | 예 | 예 (package.json scripts, compose.yml) | 예 | `frontend` (`import.plans`: dev → `frontend-dev`) |

추가 `dva.yml`: `.omo/evidence/supervised-task-execution/task-03-sec011-preflight/dva.yml`은 감독 과제 증빙 사본이다. 워크스페이스 자식이 아니므로 연결 대상이 아니다.

## 기준선

- `dva config validate --json`: exit 0. warning 0, error 0.
- `dva doctor --json`: exit 0. JSON fail 2. exit 0은 건강이 아니다.

| 항목 | 수 | owner |
| --- | --- | --- |
| Encrypted env source declared | 1 | DVA config |
| Compose config resolves | 1 | Environment |

## 발견

- DVA config: `.sops.yaml`과 `.env.sops`가 있는데 루트 `dva.yml`에 `env_file`이 없고 `sops_source`도 없다. 비밀 로딩 변경이라 wave-1이 아니다.
- Environment: `compose.yml` interpolation이 `POSTGRES_ADMIN_PASSWORD`를 요구해 `docker compose config`가 실패한다. 값을 채우는 일은 설정 침묵이 아니다.

## 제안 표

| 옛 표면 | DVA 이름 | alias/보류 | 이유 |
| --- | --- | --- | --- |
| sops 파일 | `env_file` `sops_source` | 보류 | 비밀 입력 선언. 계획 의미와 무관해도 비밀을 건드리므로 wave-1 제외 |
| Compose 필수 변수 | 기존 `local-infra` 등 plans | 보류 | Environment. 변수 부재를 러너 삭제로 숨기지 않음 |

루트 plans(`local-infra`, `review-backend`, `local-full`, `monitoring`, `test-resources`)는 유지 대상이다. 이번 감사는 읽기 전용이라 적용하지 않았다.

## wave-1

아니오. validate를 통과한 설정에, 루트 `dva.yml` 한 번의 기계적 수정으로 끝낼 계획·비밀·Compose 불변 항목이 없다.

## 후속

2026-10-09 설치본(`1aa8cd62`)으로 다시 재면 validate는 exit 0, warning 38이다. 기준선의 0에서 늘어난 이유는 `dva.yml`이 바뀌어서가 아니다. 저장소의 Makefile과 `dva.yml`에는 2026-10-06 이후 커밋이 없다. 늘어난 경고는 두 종류다.
- Make 타깃 수집이 고쳐지면서(`56cc792c`, `dc35fa93`) Makefile 제안이 34개 새로 보인다.
- 루트 interaction 4개가 자식 subproject의 같은 이름 키를 가린다는 경고다. `backend:test`, `backend:test-migrate`, `frontend:test`, `frontend:test-e2e-mock`이고, `dva <name>`은 subproject가 아니라 루트 키를 실행한다.

`env_file`·`sops_source` 결정은 `tasks/todo/` TASK-132로 올렸다(`90ad7cd`, `develop`). 저장소에는 이미 gitignore된 `.env`가 있어서, 선언하면 곧바로 로드되기 시작한다([follow-ups](follow-ups.md#제품-보드에-올린-카드)).
