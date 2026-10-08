# dripter

## 대상

- 경로: `/Users/archmagece/mydevbox/dripter-devbox`
- 브랜치: `develop`
- HEAD: `7559d7ba5c8cf009d67b11d6fe43f530a5449f96`
- DVA: `/Users/archmagece/go/bin/dva` 0.3.0
- 루트 `dva.yml` 있음.

## 판정과 모드

- 판정: **applied**
- 모드: **Preserve**
- `plans`(`local-infra`, `local-dev`, `full-stack` 등)와 두 자식 import는 유지할 의도다. deprecated 섹션 없음.
- validate error는 없다. warning·doctor 실패는 발견으로만 남긴다.

## 자식 인벤토리

`.gz-git.yaml`에 `workspaces:`와 `legacy:`는 없다. 인벤토리는 `repositories:`다. 주석 처리된 `dripter-chrome-ext`는 항목이 아니다.

| repositories 이름 | exists | 실행 표면 | dva.yml | 루트 subproject |
| --- | --- | --- | --- | --- |
| dripter-engine-ktor | 예 | Makefile, compose.yaml, build.gradle.kts | 예 | `backend` + import `plans.native` 등 |
| dripter-frontend-astro | 예 | Makefile, package scripts | 예 | `frontend` + import `plans.native` 등 |

## 기준선

- validate exit 0. warning 8, 모두 `ignore_stale`. error 0. suppressed 61. owner: DVA config.
- doctor exit 0. JSON fail 1 / 12: `Encrypted env source declared`. `env_file.files`의 `.env`에 `sops_source`가 없다. `.sops.yaml`, `.env.sops`는 있다. owner: DVA config.

## 발견

| owner | 증거 |
| --- | --- |
| DVA config | stale `suggestion_ignore`: `_infra`. 대응 Make 타깃이 없다. |
| DVA tool | `env-edit-%`, `env-reseal-%`, `env-seal-%`, `env-show-%`, `env-unseal-%`, `k8s-secret-apply-%`, `k8s-secret-edit-%`는 `.make/env.mk` 패턴 룰이다. validator가 `%`를 타깃으로 보지 않는다. 유지한다. |
| DVA config | sops 소스 미선언. wave-1 제외 (비밀, doctor 재확인). |
| 평가 | `repositories` 키 이름은 구 스키마다. 두 경로가 존재하고 연결되므로 결함으로 보지 않는다. |

## 제안 표

| 기존 표면 | DVA 이름 | alias/보류 | 이유 |
| --- | --- | --- | --- |
| `_infra` | 항목 삭제 | 예, 미적용 | 계획 의미 불변. 재확인은 validate |
| `%` ignore 7개 | 유지 | 보류 | DVA tool. 패턴 룰을 stale로 오인 |
| 자식 `make`/`pnpm` 별칭 | 기존 import interaction | 보류 | 이미 `backend`/`frontend` import로 연결되어 있다. |
| sops | `sops_source: .env.sops` | 보류 | 비밀 로딩 변경. |

감사 직후에는 파일을 고치지 않았다. 적용과 통합은 아래 wave-1이다.

## wave-1

예. 워크트리 `dev/grok/mbp/chore/dva-wave1` 커밋 `c2938ec`에서 `_infra`만 삭제했다. `%` 패턴 7개는 남겼다. 적용 후 `dva config validate` exit 0, warning 7. 자식 체크아웃은 부모 git에 없어서, 검증 때만 기본 체크아웃으로 임시 링크를 두고 커밋 전에 지웠다.

통합 시점의 `develop`보다 27커밋 뒤였다. `origin/develop` 위로 rebase 했고 충돌은 없었다. rebase 커밋 `40c5ef0`은 `origin/develop`에 있다. integration gate가 없어 `--allow-skipped-checks`로 통합했다. 워크트리와 태스크 브랜치는 회수했다.
