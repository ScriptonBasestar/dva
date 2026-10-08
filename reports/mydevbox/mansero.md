# mansero

## 대상

- 경로: `/Users/archmagece/mydevbox/mansero-devbox`
- 브랜치: `master`
- HEAD: `768603491dfbfbcf0a2137c97f99bfd47ac73ba6`
- DVA: `/Users/archmagece/go/bin/dva` 0.3.0
- 루트 `dva.yml`/`dva.yaml` 없음. `legacy:` 키는 없다.

## 판정 and 모드

- 판정: **absent**
- 모드: **New**
- 등록된 자식 중 실행 표면이 있는 곳이 있으나 `dva.yml`은 어디에도 없다.

## 자식 인벤토리

| workspace | 로컬 | 실행 표면 | 자식 dva |
| --- | --- | --- | --- |
| mansero-specs | 예, master, 커밋 없음(`.git`만) | 아니오 | 아니오 |
| mansero-kmp | 예, master | Makefile(`check`/`test`), `docker-compose.yml`, `build.gradle.kts`+`gradlew`, package `lint:openapi` | 아니오 |
| mansero-design | 예, master | Makefile(`check`/`lint`) | 아니오 |

`mansero-client`(Flutter, `pubspec.yaml`)와 `mansero-server`(Makefile, go.mod, compose)는 로컬 git이다. `.gz-git.yaml` 주석이 deprecated·접근 금지로 등록에서 뺐다. 제외하며 빈 `dva.yml`을 만들지 않는다. `path` 키는 없고 디렉터리 이름이 경로다.

## 기준선

- validate/doctor: 실행 안 함 (루트 dva 없음).

## 발견

- DVA config. `mansero-kmp`와 `mansero-design`은 실행 표면이 있고 자식 `dva.yml`이 없다. specs는 표면이 없어 의식적 설정을 만들지 않는다.
- Project. client/server는 파일이 아니라 `.gz-git.yaml`의 제외 지시가 근거다. 다시 연결하지 않는다.

## 제안 표

| 기존 표면 | DVA 이름 | alias/보류 | 이유 |
| --- | --- | --- | --- |
| `mansero-kmp` compose/gradle/make | 자식 stack·interaction | 보류 | 자식 파일이 먼저다. 스캐폴드는 보류 |
| `mansero-design` `make check` | 자식 interaction | 보류 | 동일 |
| client/server | 없음 | 보류 | `.gz-git.yaml`이 등록 제외를 말한다 |
| 루트 스캐폴드 | `dva.yml` | 보류 | New |

적용하지 않음: 읽기 전용.

## wave-1

아니오.
