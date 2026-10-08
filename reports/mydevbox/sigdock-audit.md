# sigdock-audit

## 대상

- 경로: `/Users/archmagece/mydevbox/sigdock-audit-devbox`
- 브랜치: `master`
- HEAD: `31f4508bdcf8b0941f81429dd0e03ca44bf8b003`
- DVA: `/Users/archmagece/go/bin/dva` 0.3.0
- 루트 `dva.yml`/`dva.yaml` 없음. `.gz-git.yaml`에 `workspaces:`와 `legacy:`가 없다.

## 판정과 모드

- 판정: **absent**
- 모드: **New**
- 독립 제품 루트다. worktree dump나 다른 devbox의 자식 체크아웃이 아니다.

## 자식 인벤토리

workspace 없음. legacy 없음.

저장소 내부 디렉터리(별도 git 아님, subproject 대상 아님):

| 디렉터리 | 실행 표면 | dva.yml |
| --- | --- | --- |
| (루트) | 예. Makefile `dev`/`dev-sqlite`/`dev-postgres`, `compose.yaml` | 아니오 |
| sigdock-audit-go | 아니오. `go.mod`만 | 아니오 |
| sigdock-audit-py | 아니오. `pyproject.toml`과 pytest 의존성, 루트 명령 파일 없음 | 아니오 |
| sigdock-audit-rs | 아니오. Cargo workspace 매니페스트만 | 아니오 |
| sigdock-audit-ts | 아니오. `package.json`에 scripts 없음 | 아니오 |

## 기준선

루트 설정이 없어 `dva config validate`와 `dva doctor`를 실행하지 않았다. exit는 null.

## 발견

| owner | 증거 |
| --- | --- |
| DVA config | 루트에 `dva.yml`이 없다. 실행 표면은 루트 Makefile과 `compose.yaml`이다. |

내부 언어 트리는 workspace가 아니다. 빈 자식 `dva.yml`을 만들지 않는다.

## 제안 표

| 기존 표면 | DVA 이름 | alias/보류 | 이유 |
| --- | --- | --- | --- |
| `make dev` / `compose.yaml` | 루트 `stack`+`plans` (이름 미정) | 보류 | 모드 New. 스캐폴드는 이번 읽기 전용 감사에서 쓰지 않는다. |

## wave-1

아니오. 루트 `dva.yml`이 없어 기존 파일 한 줄 수정이 아니다.
