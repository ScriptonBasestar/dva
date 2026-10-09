# careerarchive

## 대상

- 경로: `/Users/archmagece/mydevbox/careerarchive-devbox`
- 브랜치: `master`
- HEAD: `81be52cda072ac0e3f937b09f4e83b826a89b91a`
- DVA: `/Users/archmagece/go/bin/dva` 0.3.0
- 루트 `dva.yml` 있음. `.gz-git.yaml`의 `legacy:`는 없음.

## 판정과 모드

- 판정: **partial**
- 모드: **Preserve**
- `plans` `verify-workspace` / `design` / `verify`와 Penpot·RustFS stack은 유지할 의도다. deprecated 섹션 보고 없음.
- partial: 실행 표면 workspace 4곳에 자식 `dva.yml`이 없고, 같은 저장소 `prototype/dva.yml`이 루트 subproject가 아니며, validate warning과 doctor 실패가 남는다.

## 자식 인벤토리

| workspace | exists | 실행 표면 | dva.yml | 루트 subproject |
| --- | --- | --- | --- | --- |
| careerarchive-server-go | 예 | Makefile (타깃 16), go.mod | 아니오 | 없음 |
| careerarchive-web-react | 예 | Makefile, package scripts | 아니오 | 없음 |
| careerarchive-e2e-playwright | 예 | Makefile, package scripts | 아니오 | 없음 |
| careerarchive-design | 예 | Makefile | 아니오 | 없음 |
| prototype (workspace 아님, 같은 저장소) | 예 | `dva.yml` interaction `test`/`lint`/`serve` | 예 | 없음 |

`packages/`, `ops/`, `design/`은 workspace가 아니고 루트에 Makefile/package scripts가 없다. `ops` Compose는 루트 stack이 이미 참조한다.

## 기준선

- validate exit 0. warning 1: `semantic` — plan 3개에 `default_plan` 없음. owner: DVA config.
- doctor exit 1. JSON fail 2 / 25.
  - `Encrypted env source declared` — `.sops.yaml`, `.env.sops`가 있는데 `env_file`이 없다. owner: DVA config.
  - `Verification-environment listeners stay on loopback` — 사용자 체크. owner: DVA config (아래).

## 발견

| owner | 증거 |
| --- | --- |
| DVA config | 위 네 workspace는 실행 표면이 있는데 자식 `dva.yml`이 없다. 루트 subproject로 선언할 수 없다. |
| DVA config | `prototype/dva.yml`은 checks/interaction을 가지지만 루트 `subprojects`가 없다. cwd를 바꿔야만 보인다. |
| DVA config | `default_plan` 부재. 값을 고르면 bare `dva up`의 계획 의미가 바뀐다. |
| DVA config | `dva.yml` loopback 체크는 `lsof` 줄 전체를 비고정 grep한다. 재실행 시 매칭된 행은 `ssh *:14020`뿐이었고 19000/19001/8080/5173 리스너가 아니었다. 장치 주소의 `8080` 부분문자열로 실패할 수 있다. 체크를 지우면 안 된다. |
| DVA config | 사용자 체크 `Penpot Compose file exists`, `RustFS Compose file exists`는 doctor 내장 Compose 파일 존재 검사와 겹친다. |

## 제안 표

| 기존 표면 | DVA 이름 | alias/보류 | 이유 |
| --- | --- | --- | --- |
| 자식 Makefile/npm | 각 자식 `dva.yml` | 보류 | 자식 저장소 소유. 빈 파일을 wave-1로 만들지 않음. |
| `prototype` interaction | 루트 `subprojects.prototype.path` (import 없음) | 보류 | 계획 의미는 그대로다. 이번 실행은 제안만. |
| `make check` 등 루트 interaction | 유지 | 보류 | 이미 DVA 이름이다. make를 구현으로 감싼 이관은 별도 과제. |
| `default_plan` | 보류 | 보류 | 어떤 plan이 bare lifecycle인지는 의미 변경이다. |
| sops 선언 | `env_file` + `sops_source` | 보류 | 비밀 로딩을 바꾼다. validate만으로 재확인되지 않는다. |

적용하지 않음: 읽기 전용. 환경 실패를 설정으로 숨기지 않았다. loopback은 환경 노출이 아니라 체크 식의 오탐이다.

## wave-1

아니오. `subprojects.prototype` 추가는 새 서브프로젝트 연결이라 이번 wave-1이 아니다. 제안 표에만 남긴다.

## 후속

`subprojects.prototype.path`(import 없음)를 `3b5d366`으로 `master`에 반영했다. 위 제안 표의 `prototype` 보류 항목이다. 통합 과정은 [follow-ups](follow-ups.md#다음으로-할-가치가-있는-것)에 있다.
