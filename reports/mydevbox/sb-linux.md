# sb-linux

## 대상

- 경로: `/Users/archmagece/mydevbox/sb-linux-devbox`
- 브랜치: `master`
- HEAD: `eae052c9142c2242d0e69b1d3633d37d7def3327`
- DVA: `/Users/archmagece/go/bin/dva` 0.3.0
- 루트 `dva.yml`/`dva.yaml` 없음. README는 SB Linux 워크스테이션 프로토타입 모노레포라고 한다.

## 판정 and 모드

- 판정: **absent**
- 모드: **New**
- 워크스페이스 자식이 없고, 검사·패키지 표면은 루트 저장소 안에 있다.

## 자식 인벤토리

`.gz-git.yaml`에 `workspaces:`와 `legacy:`가 없다. depth 1 자식 git 없음.

루트 실행 표면: Makefile `lint`/`fmt`/`check`/`test`/`smoke`/`smoke-install`과 `utm-share`/`utm-base`/`utm-provision`. `packages/arch`, `packages/debian`, `packages/benchmarks`는 이 저장소 안의 디렉터리다. depth 3 Compose 없음.

## 기준선

- validate exit: 없음. warning: 없음. doctor exit: 없음. doctor 실패: 없음.

## 발견

- DVA config. 루트 `make test`/`check`가 있는데 `dva.yml`이 없다. 없는 자식을 만들어 연결할 근거는 없다.
- Project. `utm-*`는 가상머신 수명이다. 로컬 Compose가 없어 제품 infra 스캐폴드를 만들지 않는다.

## 제안 표

| 기존 표면 | DVA 이름 | alias/보류 | 이유 |
| --- | --- | --- | --- |
| `make test`/`lint`/`check`/`smoke` | interaction | 보류 | New. make 포장은 이관이 아니다. 스캐폴드는 보류 |
| `make utm-*` | 없음 | 보류 | UTM 수명을 DVA plan으로 표현할 로컬 Compose 증거가 없다 |
| 루트 스캐폴드 | `dva.yml` | 보류 | 패키지 경계가 아직 루트 한 파일로 닫히지 않는다 |

적용하지 않음: 읽기 전용.

## wave-1

아니오.
