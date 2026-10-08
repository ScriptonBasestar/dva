# gzh-cli

## 대상

- 경로: `/Users/archmagece/mydevbox/gzh-cli-devbox`
- 브랜치: `master`
- HEAD: `715a986a9ce21da72c588a259d188ee328cc1951`
- DVA: `/Users/archmagece/go/bin/dva` 0.3.0
- 루트 `dva.yml`/`dva.yaml` 없음. `.gz-git.yaml`의 `legacy:`는 없다.

## 판정 and 모드

- 판정: **absent**
- 모드: **New**
- 워크스페이스 12곳이 로컬에 있고 실행 표면이 있으나, 루트와 자식 모두 `dva.yml`이 없다.

## 자식 인벤토리

`path`는 모두 `./<name>`. 없는 자식 없음. 자식 `.gz-git.yaml`의 `workspaces:`는 비어 있다. 루트 subproject는 해당 없음.

| workspace | 로컬 | 실행 표면 |
| --- | --- | --- |
| gzh-cli, gzh-cli-dev-env, gzh-cli-gitforge, gzh-cli-mcp-plugin, gzh-cli-net-env, gzh-cli-os-env, gzh-cli-package-manager, gzh-cli-project, gzh-cli-quality, gzh-cli-shellforge, gzh-cli-template | 예, master | Makefile, go.mod, `cmd/` |
| gzh-cli-core | 예, master | Makefile(`build`/`test`/`check`), go.mod |

12곳 모두 자식 `dva.yml` 없음. 루트 Makefile에 `build`/`test`/`install`/`check`가 있다. Compose 없음.

## 기준선

- validate exit: 없음. doctor exit: 없음. 루트 dva가 없어 실행하지 않았다.

## 발견

- DVA config. 실행 표면이 있는 자식 12곳에 자식 `dva.yml`이 없다. 자식 파일 없이 root `subprojects`를 선언하면 안 된다.
- 루트 `make build`를 `command: make`로 감싸는 것은 이관이 아니다.

## 제안 표

| 기존 표면 | DVA 이름 | alias/보류 | 이유 |
| --- | --- | --- | --- |
| 자식 `make build`/`test` | 자식 interaction, 이후 root subproject | 보류 | 자식 저장소 파일이 먼저다 |
| 루트 `make build`/`test`/`check` | 없음 | 보류 | 자식 연결 전 루트 스캐폴드는 포장이다 |
| 루트 스캐폴드 | `dva.yml` | 보류 | New. wave-1 조건 밖 |

적용하지 않음: 읽기 전용.

## wave-1

아니오.
