# airouter

## 대상

- 경로: `/Users/archmagece/mydevbox/airouter-devbox`
- 브랜치: `master`
- HEAD: `ca1372e2b3015c04fcc8587eb1a6c29b7db41939`
- DVA: `/Users/archmagece/go/bin/dva` 0.3.0
- 루트 `dva.yml` 있음. `.gz-git.yaml`의 `legacy:`는 없음.

## 판정과 모드

- 판정: **partial**
- 모드: **Preserve**
- 루트 `stack`/`plans`(`local-dev`, `daemon-only`)는 동작하는 의도다. validator JSON에 `modes`/`applications` 등 deprecated 섹션은 없다. Rewrite 근거 없음.
- partial인 이유: 실행 표면이 있는 `airouter-macos-app`에 자식 `dva.yml`이 없다.

## 자식 인벤토리

| workspace | exists | 실행 표면 | dva.yml | 루트 subproject |
| --- | --- | --- | --- | --- |
| airouter-cli | 예 | Makefile, go.mod | 예 | `airouter-cli` → `airouter-cli` |
| airouter-webui | 예 | Makefile, package scripts | 예 | `airouter-webui` → `airouter-webui` |
| airouter-design | 예 | Makefile | 예 | `airouter-design` → `airouter-design` |
| airouter-macos-app | 예 | Makefile (`build`/`test`/`run`), Package.swift | 아니오 | 없음 |

다른 `dva.yml`: `airouter-cli`, `airouter-webui`, `airouter-design`.

## 기준선

- `dva config validate --json`: exit 0, valid, warning 0, error 0. owner 해당 없음.
- `dva doctor --json`: exit 0, fail 0 / 11. sops는 `env_file.files[].sops_source: .env.sops`로 선언되어 있다.

## 발견

| owner | 증거 |
| --- | --- |
| DVA config | `airouter-macos-app/Makefile`은 `build`/`test`/`run`이 있으나 `airouter-macos-app/dva.yml`이 없다. 루트 `dva.yml` subprojects에도 없다. 자식 파일 없이 subproject를 선언하면 깨진 선언이 된다. |

## 제안 표

| 기존 표면 | DVA 이름 | alias/보류 | 이유 |
| --- | --- | --- | --- |
| `airouter-macos-app` `make build/test/run` | 자식 `stack`/`interaction` (이름 미정) | 보류 | 자식 저장소 소유 설정이 먼저 필요하다. 빈 `dva.yml`을 만들지 않았다. |
| 루트 `plans.local-dev` / `daemon-only` | 유지 | 보류 | 계획 의미를 바꾸는 이관이 아니다. |

적용하지 않음: 읽기 전용 감사다. macOS 앱 설정은 루트 파일 한 줄로 끝낼 수 없다.

## wave-1

아니오. 자식 `dva.yml` 신설은 기존 루트 `dva.yml` 안의 기계적 한 변경이 아니다.
