# policy-gate

## 대상

- 경로: `/Users/archmagece/mydevbox/policy-gate-devbox`
- 브랜치: `master`
- HEAD: `e36226e9f6ee676127540eeefd9e7e1b8101ee30`
- DVA: `/Users/archmagece/go/bin/dva` 0.3.0
- 루트 `dva.yml`/`dva.yaml` 없음. `legacy:` 없음.

## 판정 and 모드

- 판정: **absent**
- 모드: **New**
- 루트 `make build`/`test`는 `policy-gate-go`로 위임한다. 그 자식에 `dva.yml`이 없다.

## 자식 인벤토리

`path` 키 없음. 디렉터리 이름이 경로다.

| workspace | 로컬 | 실행 표면 | 자식 dva |
| --- | --- | --- | --- |
| policy-gate-go | 예, master `0f36461` | Makefile(`build`/`test`/`check`/`lint`/`smoke`), go.mod, `cmd/` | 아니오 |

없는 workspace 없음. depth 3 Compose 없음. 루트 Makefile 주석이 테스트 타깃을 이 자식으로만 위임한다고 적는다. 루트 subproject 해당 없음.

## 기준선

- validate exit: 없음. warning: 없음. doctor exit: 없음. doctor 실패: 없음.

## 발견

- DVA config. 실행 표면이 있는 유일한 자식에 `dva.yml`이 없다. 루트가 `$(MAKE) -C policy-gate-go`를 호출하므로, 루트에 `command: make`만 두면 이관이 아니다.
- 자식 파일 없이 root `subprojects`를 선언하지 않는다.

## 제안 표

| 기존 표면 | DVA 이름 | alias/보류 | 이유 |
| --- | --- | --- | --- |
| `policy-gate-go` `make build`/`test`/`check` | 자식 interaction | 보류 | 구현은 자식 소유다. 스캐폴드는 보류 |
| 루트 `make build`/`test` | 자식으로의 alias | 보류 | 지금은 `$(MAKE) -C`다. DVA를 만들기 전에 방향을 바꾸지 않는다 |
| 루트 스캐폴드 | `dva.yml` | 보류 | New |

적용하지 않음: 읽기 전용.

## wave-1

아니오.
