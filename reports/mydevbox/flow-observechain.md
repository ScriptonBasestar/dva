# flow-observechain

## 대상

- 경로: `/Users/archmagece/mydevbox/flow-observechain-devbox`
- 브랜치: `develop`
- HEAD: `16a21744d04a75a533130dbfab0a1a29de9a8fa7`
- DVA: 0.3.0 (`/Users/archmagece/go/bin/dva`, commit `13dc89398b921c6535ec80ab890f4b04b8f3020c`)
- 루트 설정: `dva.yml`. `legacy:` 없음. deprecated 섹션 없음.

## 판정 and 모드

- 판정: **applied** — 루트 설정, validate exit 0, 실행 표면이 있는 workspace 4곳이 모두 자식 `dva.yml`과 root `subprojects`로 연결됨.
- 모드: **Preserve**. doctor의 sops 실패는 자식 누락·deprecation·validate error가 아니라서 partial로 내리지 않는다. 발견에는 남긴다.

## 자식 인벤토리

| workspace | 로컬 | 실행 표면 | 자식 dva | root subproject |
| --- | --- | --- | --- | --- |
| flow-observechain-ai | 예 | 예, Makefile+pyproject | 예 | 예 `ai` (interactions import) |
| flow-observechain-admin | 예 | 예, Makefile+package scripts | 예 | 예 `admin` |
| flow-observechain-core | 예 | 예, Makefile+go.mod | 예 | 예 `core` |
| flow-observechain-portal | 예 | 예, package scripts | 예 | 예 `portal` |

없는 workspace 없음. 자식 설정은 interaction/provision이고 stack/plans는 없다. import를 생략하지 않았다.

## 기준선

- `dva config validate --json`: exit 0. warning 0, error 0. suppressed suggestion 51.
- `dva doctor --json`: exit 0. JSON 실패 1.
- 실패: `Encrypted env source declared` — DVA config. 루트 `.sops.yaml`, `.env.sops`가 있고 `env_file`은 `.env.example`과 optional `.env`만 가리킨다. `sops_source` 없음.
- Compose config resolves는 통과. Docker daemon 통과.

## 발견

- DVA config. doctor `Encrypted env source declared`. 증거: 루트 `dva.yml` `env_file`. core 쪽 `.env.sops`는 자식 파일이며 이번 기준선은 루트만 실행했다.
- DVA config, 보류 평가. 루트 interaction `make-validate`/`test`/`codegen`/`e2e-*`가 여전히 `make`를 호출한다. 자식 연결과는 별개이고 plan 의미를 바꾸는 이전이 아니다.
- 자식 누락 결함 없음.

## 제안 표

| 기존 표면 | DVA 이름 | alias/보류 | 이유 |
| --- | --- | --- | --- |
| 자식 Makefile/npm `dev`/`test` | 이미 자식 interaction, root import | 유지 | 연결은 되어 있다 |
| 루트 `make test` 등 | 기존 interaction 이름 | 보류 | Make 래핑을 구현으로 바꾸는 일은 한 줄 수정이 아니다 |
| `.sops.yaml` / `.env.sops` | `env_file.sops_source` | 보류 | 비밀 입력 경로가 바뀐다 |

적용하지 않음: 읽기 전용. sops 선언은 wave-1 조건(비밀 불변)을 깨뜨린다.

## wave-1

아니오.

## wave-2

예. `env_file`의 `.env` 항목에 `sops_source: .env.sops`를 더한 커밋 `ea712fd`가 `develop`에 있다. 선언된 gate가 없어 `--allow-skipped-checks`로 통합했다. 이 리포트가 sops를 "비밀 로딩 변경"으로 보류한 근거는 틀렸다. `sops_source`는 로드 경로가 읽지 않는 선언 메타데이터다([follow-ups](follow-ups.md#sops_source-선언-wave-2)). primary 체크아웃 재측정에서 doctor fail은 1에서 0이다. `Encrypted env source declared` 행은 없다.
