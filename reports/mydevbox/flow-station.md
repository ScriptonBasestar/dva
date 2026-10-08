# flow-station

## 대상

- 경로: `/Users/archmagece/mydevbox/flow-station-devbox`
- 브랜치: `master`
- HEAD: `4095dea186df1f0e94c2d00d46ab35748a5af791`
- DVA: 0.3.0 (`/Users/archmagece/go/bin/dva`, commit `13dc89398b921c6535ec80ab890f4b04b8f3020c`)
- 루트 `dva.yml`/`dva.yaml` 없음. 제품 루트다(`PRODUCT.md`, `.gz-git.yaml`, 독립 workspace). worktree dump나 다른 devbox의 자식이 아니다.

## 판정 and 모드

- 판정: **absent**
- 모드: **New**
- 루트 설정이 없어 validate/doctor를 실행하지 않았다.

## 자식 인벤토리

`legacy:` 키 없음.

| workspace | 로컬 | 실행 표면 | 자식 dva | root subproject |
| --- | --- | --- | --- | --- |
| flow-station | 예 | 예, Makefile(`build`/`test`/`dist`/`check`)+go.mod | 아니오 | 해당 없음 (루트 dva 없음) |
| flow-station-dogfood | 예 | 예, Makefile(`check`/`lint`) | 아니오 | 해당 없음 |

없는 workspace 없음. 루트 Makefile은 `gz-git` sync/validate와 `scripts/check-docs.sh`뿐이고, 주석대로 문서 게이트다.

## 기준선

- validate exit: 없음 (루트 dva 없음)
- warning: 없음
- doctor exit: 없음
- doctor 실패: 없음

## 발견

- DVA config. 실행 표면이 있는 자식 둘이 자식 `dva.yml` 없이 있고, 루트도 선언이 없다. 증거: `.gz-git.yaml` `workspaces`, `flow-station/Makefile`, `flow-station-dogfood/Makefile`.
- 자식 없는 루트에 빈 설정만 두는 것은 wave-1이 아니다. 자식 파일을 만들기 전에 root `subprojects`를 선언하면 안 된다.

## 제안 표

| 기존 표면 | DVA 이름 | alias/보류 | 이유 |
| --- | --- | --- | --- |
| 루트 `make check` | 없음 (문서 스크립트) | 보류 | DVA가 문서 규칙 전체를 대체한다는 증거가 없다 |
| 루트 `make prepare`/`validate` | 없음 | 보류 | gz-git/SSH라 네트워크 상태다. Environment에 가깝고 설정으로 감추지 않는다 |
| `flow-station` `make build`/`test`/`check` | 자식 interaction, 이후 root subproject | 보류 | 자식 저장소 파일이 먼저다 |
| `flow-station-dogfood` `make check` | 자식 interaction | 보류 | 동일 |

적용하지 않음: 모드가 New이고 이 실행은 읽기 전용이다. 루트 스캐폴드는 자식 연결 설계가 같이 가야 해서 한 파일의 기계적 수정이 아니다.

## wave-1

아니오.
