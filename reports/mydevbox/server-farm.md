# server-farm

## 대상

- 경로: `/Users/archmagece/mydevbox/server-farm-devbox`
- 브랜치: `master`
- HEAD: `b55bcc782080e155b8c6b3e74a3427ce9bd893ef`
- DVA: `0.3.0` (`/Users/archmagece/go/bin/dva`)
- 루트 `dva.yml`과 `server-farm-backend-go/dva.yml`. `.gz-git.yaml` 없음.

## 판정

- 판정: `partial`
- 모드: `Preserve`
- 루트 plan `local-infra`는 유효하다(validate exit 0, deprecation 없음). 같은 트리의 실행 자식이 루트 `subprojects`에 없다.

## 자식 인벤토리

workspace/legacy 선언 없음. 아래는 루트 안의 별도 git 체크아웃이다. 자식 HEAD `7206328`, 브랜치 `master`.

| 경로 | exists | executable surface | dva.yml | root subproject |
| --- | --- | --- | --- | --- |
| server-farm-backend-go | 예 | 예 (Makefile, go.mod) | 예 | 없음 |

## 기준선

- validate exit 0, warning 0, error 0. 루트 `suggestion_ignore`는 매칭됨.
- doctor exit 0, JSON fail 1
  - `Encrypted env source declared` — 루트 `.sops.yaml`, `.env.sops`. owner: DVA config

## 발견

- DVA config: `server-farm-backend-go/dva.yml`이 있는데 루트 `dva.yml`에 `subprojects`가 없다. 자식 interaction은 `make build/test/lint` 래핑이다. 증거: 두 `dva.yml`.
- 루트 stack/plan(postgres, rustfs, `compose.yaml`)은 유지할 동작이 있다.
- sops 미선언은 비밀 표면이라 이번 범위에서 고치지 않음.

## 제안

| old surface | DVA name | alias/보류 | reason |
| --- | --- | --- | --- |
| server-farm-backend-go | `subprojects` path | 보류 | subproject 추가는 wave-1이 아님 |
| 자식 `make test` 등 | leaf 명령으로 교체 | 보류 | 래핑 제거는 명령 구현을 바꾼다 |
| `.env.sops` | `sops_source` | 보류 | 비밀 |

적용하지 않음: 읽기 전용. 루트 plan 의미는 유지한다.

## wave-1

아니오.

## wave-2

예. `env_file`의 `.env` 항목에 `sops_source: .env.sops`를 더한 커밋 `630c697`가 `master`에 있다. 워크트리에 gitignore된 `server-farm-backend-go/`가 없어 `make check`와 `make lint` 기준선을 재지 못해 `--allow-skipped-checks`로 통합했다. 이 리포트가 sops를 "비밀 로딩 변경"으로 보류한 근거는 틀렸다. `sops_source`는 로드 경로가 읽지 않는 선언 메타데이터다([follow-ups](follow-ups.md#sops_source-선언-wave-2)). primary 체크아웃 재측정에서 doctor fail은 1에서 0이다. `Encrypted env source declared` 행은 없다.
