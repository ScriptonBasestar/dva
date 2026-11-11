# 비교안 동시 실행 설계 — 세부 규칙과 결정 기록

[75-variant-comparison-design.md](75-variant-comparison-design.md)에서 분리한 세부 문서다. 모드·선언·수명주기는 그 문서가 소유한다.

## config 모드 격리 규칙

- **config 모드의 인스턴스**(TASK-593): 비교안이 호출한 체크아웃을 공유하므로 DVA가 비교안
  이름을 인스턴스로 삼아 `per_variant` 엔트리의 pid·로그를 `.sb/dva/{pids,logs}/<비교안>/`로
  나누고, compose·podman-compose 프로젝트 이름은 일반 실행에서 compose가 정하는 프로젝트
  (첫 compose 파일의 최상위 `name:`, 없으면 그 파일 디렉터리 이름)에 `-<비교안>`을 붙여 compose가 요구하는
  형식(소문자·숫자·`-`·`_`)으로 정규화해 쓴다(같은 디렉터리의 파일들은 일반 실행처럼 한 프로젝트로 남는다). `per_variant` compose 항목의 `project_name:`은 검증이
  거부하므로 접두사가 될 수 없다. 대소문자만 다른 비교안 이름은 한 디렉터리로 겹치므로 한 명령 안에서도, 이미 기록된 비교안과도(snapshot·worktree 포함) 거부한다. 인스턴스 없는 일반 `dva up`의 경로와
  이름은 바뀌지 않는다. 비교안 이름은 plan var `DVA_VARIANT`로도 전달된다. OS 환경에 `DVA_VARIANT`가 이미 있으면 그 값이 모든
  비교안을 덮어쓰므로 포트 var와 같이 실행 전에 거부한다.
- **config 모드의 빌드 상태**: DVA가 알고 있는 도구(`next`, `nuxt`, `nuxi`, `astro`,
  `gatsby`)를 `per_variant` 엔트리가 실행하는데 그 명령·dir·env가 `DVA_VARIANT`를 쓰지 않으면
  실행 전에 실패한다. 임의 명령이 체크아웃에 쓰는 것은 DVA가 볼 수 없으므로 이 목록은
  증명이 아니라 알려진 경우의 방어선이다.

## 결정 기록

- 2026-11-06 사용자 결정: snapshot과 worktree 두 모드를 모두 지원한다. 의존 공유는
  `shared`/`per_variant` 선언으로 정한다. 일부 실패는 남기고 실패로 보고하며, 되돌리기는
  명시적 `down` 호출로만 한다. 수동 사용과 에이전트 사용이 같은 명령으로 간단해야 하며,
  `ui-iterate` 같은 미리보기 워크플로우와 함께 쓸 수 있어야 한다.
- 2026-11-06 사용자 결정: snapshot의 의존 설치는 plan의 `provision`에 맡긴다. worktree
  모드의 키는 provider가 정의하는 ID(worktrunk는 branch)로 넓힌다. 이는 TASK-587 계약의
  변경이며 TASK-587에 반영했다.
- 2026-11-06 사용자 결정(권장안 수용):
  - **config 모드**를 세 번째 모드로 둔다. 한 체크아웃에서 vars만 바꿔 N개를 띄우며,
    DVA가 비교안 이름으로 pid 경로와 compose 프로젝트 이름을 나눈다. 체크아웃 안에
    빌드 상태를 쓰는 도구(예: `.next`)는 그 경로를 var로 나누지 않으면 실행 전에 실패한다.
    snapshot 모드 다음에 구현한다. 구현(TASK-593): 출처는 `config`(브랜치 이름이 `config`면
    `refs/heads/config`), 기록의 모드는 `config`와 `instance`이며, `down`은 프로세스만
    멈추고 체크아웃은 건드리지 않는다.
  - **실행 기록**: 호출한 체크아웃의 `.sb/dva/compare/<name>.json`에 비교안 이름·모드·출처·
    경로·포트 배정만 남긴다. pid 파일과 같은 "시작한 것"의 기록이며 대상 선택 레지스트리가
    아니다. 매 호출에서 실제 pid와 provider로 재검증하고 어긋나면 stale로 보고한다.
  - **기계 출력**: 종료 코드는 기존 규약대로 성공 0, 실패 1이다. 일부 실패와 시작 전 실패는
    기존 실패 envelope의 `error.code`로 구분한다. 비교안별 결과는 기존 plan 출력
    (`status`, `endpoints[].url`)을 감싸고 출처·커밋·결과를 더한다. 최상위에 다음에 실행할
    명령을 둔다. 정확한 필드 이름은 구현 카드가 기존 출력 코드에 맞춰 정한다.
