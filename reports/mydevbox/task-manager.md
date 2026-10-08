# task-manager

## 대상

- 경로: `/Users/archmagece/mydevbox/task-manager-devbox`
- 브랜치: `master`
- HEAD: `6933b4264e8cc1350fe8c960008d45c7bea50775`
- DVA: `0.3.0` (`/Users/archmagece/go/bin/dva`)
- 루트 `dva.yml`만 있음.

## 판정

- 판정: `partial`
- 모드: `Preserve`
- 루트 interaction은 validate exit 0이고 deprecation이 없다. 그러나 로컬 workspace 자식에 실행 표면이 있는데 `dva.yml`이 없다. 루트 주석 "No services or child executable surface exist yet"는 현재 Makefile과 맞지 않는다.

## 자식 인벤토리

`legacy:` 없음. `selfSync.enabled: false`.

| workspace | exists | executable surface | dva.yml | root subproject |
| --- | --- | --- | --- | --- |
| taskchain-task-manager (`path: ./taskchain-task-manager`) | 예 | 예 (Makefile `build` `test` `lint` `check`, go.mod) | 아니오 | 없음 |

## 기준선

- validate exit 0, warning 0, error 0.
- doctor exit 0, JSON fail 0.
- sops 파일 없음.

## 발견

- DVA config: 실행 표면이 있는 자식에 자식 `dva.yml`이 없고 루트 `subprojects`도 없다. 증거: `.gz-git.yaml` `workspaces.taskchain-task-manager`, `taskchain-task-manager/Makefile`.
- 루트 interaction `repo-status` / `repo-remote` / `workspace-check`는 조회용이라 그 자체는 결함이 아니다.
- 빈 자식 `dva.yml`을 wave-1로 만들지 않음. 표면이 있으므로 나중에 실제 명령이 들어가야 한다.

## 제안

| old surface | DVA name | alias/보류 | reason |
| --- | --- | --- | --- |
| `make -C taskchain-task-manager build/test/lint/check` | 자식 interaction + 루트 subproject | 보류 | 자식 파일과 subproject 추가. wave-1 아님 |
| 루트 주석 "no executable surface" | 주석 정정 | 보류 | 주석만 고치면 연결이 생기지 않음 |

적용하지 않음: 읽기 전용. 자식 저장소에 설정을 새로 쓰는 작업이다.

## wave-1

아니오.
