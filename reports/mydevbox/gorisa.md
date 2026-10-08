# gorisa

## 대상

- 경로: `/Users/archmagece/mydevbox/gorisa-devbox`
- 브랜치: `master`
- HEAD: `84060797616dbb21e770495315deb006bbc8e7c9`
- DVA: `0.3.0` (`/Users/archmagece/go/bin/dva`, commit `13dc89398b921c6535ec80ab890f4b04b8f3020c`)
- 루트 `dva.yml` 있음. `.omo/evidence/**/malformed*/dva.yml` 2개는 깨진 fixture라 자식으로 세지 않음.

## 판정

- 판정: `applied`
- 모드: `Preserve`
- validate exit 0, `modes`/`applications` deprecation 없음. workspace 두 곳과 로컬 `grabber`가 루트 `subprojects`에 연결됨.

## 자식 인벤토리

`.gz-git.yaml` `legacy:` 없음.

| workspace | exists | executable surface | dva.yml | root subproject |
| --- | --- | --- | --- | --- |
| gorisa-rails | 예 | 예 (Makefile, package.json scripts, compose.yaml) | 예 | `gorisa-rails` |
| gorisa-prc-history-archive | 예 | 예 (Makefile, pyproject.toml) | 예 | `gorisa-prc-history-archive` |
| grabber (workspaces 밖) | 예 | 아니오 (pyproject.toml만, Makefile/compose 없음) | 예 | `grabber` |

`grabber`는 표면이 없어 빈 `dva.yml`을 만들 대상이 아니다. 이미 자식 설정과 루트 subproject가 있다.

## 기준선

- validate exit 0, warning 3, error 0
  - `config_drift` ×2 — `compose/compose.rustfs.yaml`, `gorisa-rails/compose/compose.dev.yaml`가 stack `runners.compose.files`에 없음. owner: DVA config
  - `config_suggestion` ×1 — Makefile `runtime-ownership-contract`. owner: DVA config
- doctor exit 0, JSON fail 1 (built-in이라 exit는 0)
  - `Encrypted env source declared` — 루트 `.sops.yaml`, `.env.sops`가 있는데 `sops_source` 없음. owner: DVA config

## 발견

- DVA config: 위 drift 2건, suggestion 1건, sops 미선언. 증거: 루트 `dva.yml`, `compose/compose.rustfs.yaml`, `gorisa-rails/compose/compose.dev.yaml`, `.sops.yaml`.
- workspace 자식은 연결됨. 미연결 실행 표면 없음.

## 제안

| old surface | DVA name | alias/보류 | reason |
| --- | --- | --- | --- |
| `compose/compose.rustfs.yaml` | 기존 stack files에 추가 | 보류 | compose 집합을 바꾸면 plan 의미가 바뀐다. wave-1 아님 |
| `gorisa-rails/compose/compose.dev.yaml` | 자식 compose runner | 보류 | 자식 파일 변경이며 루트 한 줄 수정이 아님 |
| `make runtime-ownership-contract` | interaction 동명 | 보류 | interaction 추가는 wave-1이 아님 |
| `.env.sops` | `env_file` `sops_source` | 보류 | 비밀 입력 선언. 이번 감사는 읽기 전용 |

적용하지 않음: 감사 전용이고, 위 항목은 plan·비밀·compose 파일을 건드린다.

## wave-1

아니오.
