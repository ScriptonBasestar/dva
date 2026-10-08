# sigdock-idp

## 대상

- 경로: `/Users/archmagece/mydevbox/sigdock-idp-devbox`
- 브랜치: `master`
- HEAD: `5c81ef25cc4becdace49d562094f6bf38add4059`
- DVA: `/Users/archmagece/go/bin/dva` 0.3.0
- 루트 `dva.yml` 있음. `.gz-git.yaml`의 `legacy:`는 없음.

## 판정과 모드

- 판정: **partial**
- 모드: **Preserve**
- `stack`/`plans`(`infra`, `hybrid`, `infra-mq`, `dev-full`, `observability`)는 동작하는 의도다. validator JSON에 `modes`/`applications` 등 deprecated 섹션은 없다. Rewrite 근거 없음.
- partial인 이유: `subprojects.sigdock-idp-sdks`가 자식 `dva.yml` 없이 선언되어 있다. validate warning도 남아 있다.

## 자식 인벤토리

| workspace | exists | 실행 표면 | dva.yml | 루트 subproject |
| --- | --- | --- | --- | --- |
| conformance-suite | 예 | 예 (`docker-compose.yml`, `pom.xml`) | 아니오 | 없음. 제외 |
| sigdock-idp-rs | 예 | 예 (Makefile `check`/`lint`, Cargo.toml) | 예 | `sigdock-idp-rs` |
| sigdock-idp-ts | 예 | 예 (package scripts `dev`/`build`/`lint`/`check`) | 예 | `sigdock-idp-ts` |
| sigdock-idp-sdk-py | 예 | 아니오 | 아니오 | 없음 |

`sigdock-idp-sdks`는 workspace가 아니다. 디렉터리는 있고 루트 명령 파일은 없다. 중첩 `sdk-go`/`sdk-python`/`sdk-ruby` 등은 매니페스트만 있다. 그런데 루트 subproject로 선언되어 있고 `dva.yml`은 없다.

`conformance-suite`는 `.gz-git.yaml`이 벤더 클론, `access: read-only`, `sync.strategy: skip`으로 제외한다. 자식 설정을 요구하지 않는다.

`sigdock-idp-sdk-py`는 `pyproject.toml` 라이브러리다. 루트에 Makefile, package scripts, compose, run/test/build 명령 파일이 없다. 결함 아님.

## 기준선

- `dva config validate --json`: exit 0, valid, error 0, warning 5. `config_drift` 1 (`compose.e2e.yaml`, `compose.e2e.delivery-tls.yaml`, `compose.multi-region-postgres.yaml`). `config_suggestion` 4 (`hooks-matrix`, `hooks-matrix-verify`, `observe-only-guard-verify`, `pilot-observe-only`). owner: DVA config.
- `dva doctor --json`: exit 0, fail 1 / 17. `Encrypted env source declared` — `.sops.yaml`, `.env.sops`가 있는데 `sops_source`가 없다. owner: DVA config.

## 발견

| owner | 증거 |
| --- | --- |
| DVA config | `dva.yml` `subprojects.sigdock-idp-sdks` → `sigdock-idp-sdks`. 그 경로에 `dva.yml`이 없다. |
| DVA config | validate `config_drift` 3개 compose. 파일 주석으로 의도적 미등록이 아니다. stack에 넣으면 plan 의미가 바뀐다. |
| DVA config | doctor: `env_file`에 `sops_source` 없음. 경로는 `.env.sops`. |
| DVA config | Makefile 타겟 4개가 interaction으로 제안됨. `command: make …` 래핑은 이관이 아니다. |

## 제안 표

| 기존 표면 | DVA 이름 | alias/보류 | 이유 |
| --- | --- | --- | --- |
| `subprojects.sigdock-idp-sdks` | 선언 제거 또는 표면이 생긴 뒤 자식 설정 | 보류 | 빈 `dva.yml`을 만들지 않는다. subproject 가감은 wave-1이 아니다. |
| 미등록 e2e/multi-region compose | 현행 plan 유지 | 보류 | 등록은 plan 의미 변경. 의도 주석이 없어 `drift_ignore`도 단정하지 않는다. |
| `.env.sops` | `env_file` `sops_source` | 보류 | doctor 항목이다. validate만으로 재확인되지 않고 비밀 파일 참조다. |
| `make hooks-matrix` 등 | 유지 | 보류 | 워크플로 소유를 바꾸지 않는다. |

적용하지 않음: 읽기 전용 감사다.

## wave-1

아니오. 위 항목은 plan 의미, 비밀 참조, 또는 subproject 배선이다.
