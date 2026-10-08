# lottomaster

## 대상

- 경로: `/Users/archmagece/mydevbox/lottomaster-devbox`
- 브랜치: `master`
- HEAD: `642f9a331cfd5fa6eb88efaebf67c1ccdb54e1af`
- DVA: `/Users/archmagece/go/bin/dva` 0.3.0
- 루트 `dva.yml`/`dva.yaml` 없음. `.gz-git.yaml` metadata 이름은 `lottokit-devbox`. `legacy:` 없음.

## 판정 and 모드

- 판정: **absent**
- 모드: **New**
- 앱 표면은 자식 `lottokit-kmp`에 있고, 그 자식과 루트 모두 `dva.yml`이 없다.

## 자식 인벤토리

`path` 키는 없다. 디렉터리 이름과 같은 `lottokit-kmp`가 로컬에 있다.

| workspace | 로컬 | 실행 표면 | 자식 dva |
| --- | --- | --- | --- |
| lottokit-kmp | 예, master `6916356` | Makefile(e2e 에뮬레이터), `docker-compose.yml`(`db`, `server`), `build.gradle.kts`+`gradlew` | 아니오 |

없는 workspace 없음. 루트 Makefile은 문서·플래닝 저장소 주석이고 `.make`의 base/help/setup/utils/validate를 include한다. 루트 subproject 해당 없음.

## 기준선

- validate exit: 없음. warning: 없음. doctor exit: 없음. doctor 실패: 없음.

## 발견

- DVA config. 실행 표면이 있는 자식에 `dva.yml`이 없고 루트 선언도 없다. 자식 파일보다 root `subprojects`를 먼저 두면 안 된다.
- Project. 자식 Compose는 `db`와 `server`를 한 파일에 둔다. 앱과 인프라 분리는 자식 소유 파일이라 루트 한 줄 수정이 아니다.

## 제안 표

| 기존 표면 | DVA 이름 | alias/보류 | 이유 |
| --- | --- | --- | --- |
| 자식 `docker-compose.yml` | 자식 stack, 이후 `local-infra`/`local-dev` | 보류 | 자식 `dva.yml`이 먼저다. 스캐폴드는 보류 |
| 자식 `gradlew`와 e2e Makefile | 자식 interaction | 보류 | make/gradle 포장은 이관이 아니다 |
| 루트 스캐폴드 | `dva.yml` | 보류 | New |

적용하지 않음: 읽기 전용.

## wave-1

아니오. 루트 `dva.yml`이 없다.
