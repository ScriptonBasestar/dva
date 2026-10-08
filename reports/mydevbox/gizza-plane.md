# gizza-plane

## 대상

- 경로: `/Users/archmagece/mydevbox/gizza-plane-devbox`
- 브랜치: `master`
- HEAD: `856d222de62629a0f3197a8a75e156dee0188ef8`
- DVA: `/Users/archmagece/go/bin/dva` 0.3.0
- 루트 `dva.yml` 있음. `.gz-git.yaml`의 `legacy:`는 없음. sops/env 파일 없음.

## 판정과 모드

- 판정: **applied**
- 모드: **Preserve**
- 루트는 stack/plan 없이 interaction·checks·subproject path만 둔다. 파일 주석이 그 범위를 명시한다. 자식 lifecycle을 루트로 올리는 Rewrite가 아니다. deprecated 섹션 없음.
- validate exit 0, 두 실행 표면 자식이 path로 연결된다. suggestion 3건은 error가 아니다.

## 자식 인벤토리

| workspace | exists | 실행 표면 | dva.yml | 루트 subproject |
| --- | --- | --- | --- | --- |
| gizza-plane-daemon | 예 | Makefile, go.mod | 예 | `daemon` → `gizza-plane-daemon` (import 없음) |
| gizza-plane-web | 예 | Makefile, package scripts | 예 | `web` → `gizza-plane-web` (import 없음) |

주석은 import를 일부러 생략하고 `dva run --project <name>`만 연다고 적는다. 연결 조건은 path다.

## 기준선

- validate exit 0. warning 3, 모두 `config_suggestion`: `board-consistency`, `board-consistency-test`, `tasks-validate`. owner: DVA config. error 0. suppressed 4 (`suggestion_ignore`의 clone/sync/status/ci/clean/web-build/validate).
- doctor exit 0. JSON fail 0 / 9.

## 발견

| owner | 증거 |
| --- | --- |
| DVA config | 세 Makefile 타깃은 interaction도 ignore도 아니다. `dva.yml` 38–49행은 clone/sync/status와 `validate`만 의도적으로 제외한다. |
| 평가 | 자식 `dva.yml`을 루트 command namespace로 import하지 않은 것은 선언된 범위다. 결함으로 보지 않는다. |

## 제안 표

| 기존 표면 | DVA 이름 | alias/보류 | 이유 |
| --- | --- | --- | --- |
| `make board-consistency` 등 3타깃 | 동명 interaction 또는 ignore | 보류 | 워크플로 포함 여부가 의미 변경이다. 한 줄 alias는 make 포장이라 이관이 아니다. |
| `make -C gizza-plane-*` CI 단계 | 유지 | 보류 | 루트 fan-out으로 이미 적혀 있다. |
| 자식 stack/plan | 자식 소유 | 보류 | 루트에 plan을 새로 만들지 않는다. |

적용하지 않음: 읽기 전용. 동작 중인 범위를 새 plan 모델로 다시 쓰지 않음.

## wave-1

아니오. suggestion 3건을 interaction으로 올리거나 ignore로 지우는 일은 계획/명령 의미를 바꾸고, 어느 쪽인지도 확정할 수 없다.
