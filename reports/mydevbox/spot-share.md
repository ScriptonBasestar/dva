# spot-share

## 대상

- 경로: `/Users/archmagece/mydevbox/spot-share-devbox`
- 브랜치: `master`
- HEAD: `9d7eb7f2a69bd3a6663c72c343f845b7d32fe74a`
- DVA: `/Users/archmagece/go/bin/dva` 0.3.0
- 루트 `dva.yml`/`dva.yaml` 없음. `.gz-git.yaml`의 `legacy:`는 없음.

## 판정과 모드

- 판정: **absent**
- 모드: **New**
- 독립 제품 루트다. 루트 자체에는 Makefile, package scripts, compose가 없다.

## 자식 인벤토리

| workspace | exists | 실행 표면 | dva.yml | 루트 subproject |
| --- | --- | --- | --- | --- |
| spot-share-rails | 예 (path=이름) | 예. `compose.yml`, Rakefile, `bin/rails`, `bin/dev` | 아니오 (루트) | 루트 설정 없음 |

같은 저장소 중첩 설정: `spot-share-rails/ops/rustfs/dva.yml` (`plans.local-storage-rehearsal`). 자식 루트 설정이 아니다. cwd를 바꿔야만 보인다.

## 기준선

제품 루트에 dva 설정이 없어 validate와 doctor를 실행하지 않았다. 중첩 파일에도 돌리지 않았다. exit는 null.

## 발견

| owner | 증거 |
| --- | --- |
| DVA config | 루트 `dva.yml`이 없다. `spot-share-rails`는 실행 표면이 있는 workspace인데 자식 루트 `dva.yml`이 없다. |
| DVA config | `ops/rustfs/dva.yml`은 루트 subproject로 연결될 수 있는 중첩 설정이지만, 루트가 없어 연결하지 못했다. |

없는 자식 파일을 선언하지 않는다. rustfs 파일을 제품 루트 설정으로 승격하지 않는다.

## 제안 표

| 기존 표면 | DVA 이름 | alias/보류 | 이유 |
| --- | --- | --- | --- |
| `bin/dev`, `bin/rails`, `compose.yml` | 자식 루트 `stack`/`plans` (이름 미정) 후 루트 `subprojects` | 보류 | 모드 New. 자식 설정과 루트 스캐폴드가 둘 다 필요하다. |
| `ops/rustfs` `local-storage-rehearsal` | 유지 후 루트가 import | 보류 | 중첩 파일을 루트로 옮기면 소유가 바뀐다. |

적용하지 않음: 읽기 전용. 클론은 필요 없었다.

## wave-1

아니오. 루트 `dva.yml`이 없고, subproject·자식 링크 추가는 wave-1이 아니다.
