# matdosa

## 대상

- 경로: `/Users/archmagece/mydevbox/matdosa-devbox`
- 브랜치: `master`
- HEAD: `32bba7ec2dbc73b78e51c78de3a3150d4957de7b`
- DVA: `0.3.0` (`/Users/archmagece/go/bin/dva`)
- 루트 `dva.yml`, `matdosa-engine-fiber/dva.yml`, `matdosa-web-svelte/dva.yml`.

## 판정

- 판정: `applied`
- 모드: `Preserve`
- validate exit 0, deprecation 없음. 로컬 workspace 둘이 루트 `subprojects.engine` / `web`과 plan import(`engine-dev`, `web-dev`)로 연결됨.

## 자식 인벤토리

`legacy:` 없음.

| workspace | exists | executable surface | dva.yml | root subproject |
| --- | --- | --- | --- | --- |
| matdosa-engine-fiber | 예 | 예 (Makefile, docker-compose.yaml, go.mod) | 예 | `engine` import plan `dev` as `engine-dev` |
| matdosa-web-svelte | 예 | 예 (package.json scripts) | 예 | `web` import plan `dev` as `web-dev` |

## 기준선

- validate exit 0, warning 0, error 0. `suggestion_ignore`는 실제 타깃과 맞음.
- doctor exit 1 (`2 user check(s) failed`), JSON fail 3
  - `Encrypted env source declared` — 루트 `.sops.yaml`, `.env.sops`. owner: DVA config
  - `pnpm available` — 감사 PATH에 없음. mise shim `pnpm 12.8.2`는 존재. owner: Environment
  - `golangci-lint available` — 같은 이유. shim `2.12.2` 존재. owner: Environment

## 발견

- DVA config: `provision.default`와 `interaction.build` 등이 `dva run` / `dva build` / `dva stop`을 다시 호출함. 증거: `dva.yml` provision·interaction. 비밀 선언(`sops_source`)도 없음.
- Environment: doctor가 mise shim 없이 돌아 도구 체크가 실패. 설정을 고쳐서 지우지 않음.
- 자식 연결 결함 없음.

## 제안

| old surface | DVA name | alias/보류 | reason |
| --- | --- | --- | --- |
| provision의 `dva run install-*` | 자식 leaf 명령 | 보류 | 실행 의미를 바꾸는 재구성. 한 줄 기계 수정이 아님 |
| `.env.sops` | `sops_source` | 보류 | 비밀 입력 |
| pnpm / golangci-lint 체크 | 유지 | 보류 | Environment. shim은 설치되어 있음 |

적용하지 않음: 읽기 전용. 동작 중인 plan 구성을 유지하는 편이 맞다.

## wave-1

아니오.

## wave-2

예. 축약형 `files: [.env]`를 객체형으로 바꾸고 `sops_source: .env.sops`를 더한 커밋 `6785c54`가 `master`에 있다. 축약형 항목도 `required: false`로 읽히므로 로드는 같다. `branch-integrate` readiness는 플래그 없이 통과했다. primary 체크아웃 재측정에서 doctor fail은 1에서 0이다.
