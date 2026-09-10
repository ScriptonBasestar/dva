# 63. init native 선언 탐지 결정

> 상태: **결정됨** (TASK-339, 2026-09-10). 상위 계약은
> [capability-driven init 설계](58-capability-driven-init-design.md)가 소유한다.

## 결정

**Makefile recipe를 읽는 것은 명령 추측이 아니다.** `dev-dashboard`가
`cd dashboard-webui && pnpm dev`를 선언했다면 저장소가 실행 디렉터리와 명령을 직접 소유한
것이다. Go `dva init`은 그 문자열을 그대로 옮길 수 있다. `package.json`이나 `mise.toml`만 보고
`pnpm dev`, `go run .` 같은 관례를 만드는 것은 TASK-249의 금지 대상이며 계속 만들지 않는다.
runtime pin과 package manifest는 프로젝트 분류 근거일 뿐 실행 명령 근거가 아니다.

탐지와 충돌 해소 규칙은 다음으로 확정한다.

1. 포트 manifest는 `PORT_MAPPINGS.yaml`, `PORT_MAPPINGS.yml`, `port_mappings.yaml`,
   `port_mappings.yml` 순으로 첫 파일 하나만 읽는다. 여러 철자를 병합하지 않는다.
2. `.gz-git.yaml` workspace 경로는 `targetPath`, `path`, workspace key 순으로 고른다. 절대 경로와
   상위 디렉터리 이탈 경로는 생성하지 않는다.
3. `dev-<name>` recipe가 native entry 이름과 `run`을 소유한다. 단순 Make 변수는 실제 문자열로
   확장하며, `cd <dir> && <command>`는 `dir`과 `run`으로 분리한다.
4. `build-<name>`이 있으면 같은 entry의 `build`를 소유한다. 없으면 공용 `build` recipe에서 같은
   `dir`을 명시한 명령만 사용한다. `test` target은 native runner 필드가 아니므로 `make test`
   interaction으로 노출한다. hybrid scaffold에서 같은 interaction을 언어 template이 추측했다면
   Makefile의 직접 증거가 그 항목만 대체한다. Compose stack 선언은 template이 계속 소유하며,
   `dev-compose` 이름이 충돌하면 plan 참조도 compose runner로 맞춰 해석 불가능한 plan을 만들지 않는다.
5. entry와 plan 순서는 이름순이며 plan entry에는 10 단위 `order`만 기록한다. 서로 다른 recipe의
   의존 관계는 Makefile이 직접 선언하지 않는 한 증명할 수 없으므로 `depends_on`을 만들지 않는다.
   hybrid의 `dev` plan에는 Compose entry를 먼저 넣어 bare `dva up`도 두 runner type을 모두 실행한다.
6. `services.application` 포트는 endpoint를 만들고 같은 이름의 native entry에는 HTTP health check를
   붙인다. manifest의 `env` 이름도 읽지만 애플리케이션이 실제로 소비하는지는 확인할 수 없으므로
   env를 주입하지 않고 생성 파일에 확인 주석을 남긴다.

이 범위는 저장소가 선언한 capability만 옮기므로 TASK-249 계약을 확장하지만 뒤집지 않는다. 포트의
프로토콜과 entry 간 더 복잡한 의존성처럼 manifest가 답하지 않는 사항은 후속 사람이 확인한다.

Make 변수는 flavor를 보존한다. `:=`는 그 줄에서 알려진 값으로 즉시 확장하고, `=`와 `?=`는 recipe를
읽을 때 최종 값으로 지연 확장한다. 관련 `dev-*`, `build*`, `test` recipe에 동적 함수나 해석할 수 없는
참조가 남으면 잘못된 경로를 생성하지 않고 해당 변수와 target을 명시한 오류를 반환한다.
Make가 shell용 단일 `$`로 바꾸는 `$$`는 생성 명령에서도 `$` 하나로 보존한다. `$@`, `$<`, `$^`,
`$?`, `$*`, `$%`, `$|`를 포함한 escape되지 않은 자동·단일 문자 참조는 Make 밖에서 값이 달라지므로
관련 발견 target을 오류로 거부한다.
