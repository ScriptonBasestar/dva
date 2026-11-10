# 비교안 동시 실행 설계

상태: **채택(2026-11-06), 부분 구현**. [ROADMAP.md](../ROADMAP.md)가 채택했고 실행 카드로 나눴다.
구현된 범위: `compare:` 선언 검증(TASK-589), 기록·snapshot 체크아웃·stale 검증(TASK-590),
snapshot 모드 `dva compare up|status|down`(TASK-591), worktree 모드(TASK-592, git provider만.
worktrunk·CE provider와 `dva target`은 아직), config 모드(TASK-593), 사용법은
[USAGE.md](../USAGE.md).

## 문제

화면 구조나 서버 동작을 비교할 때 A/B, A/B/C/D처럼 여러 비교안을 동시에 띄운다.
지금은 체크아웃을 만들고, 포트를 바꾸고, 각자 `dva up`을 호출하고, 끝나면 하나씩
내리는 일을 사람이나 에이전트가 매번 손으로 한다. 반복 비용이 크고 실행마다 조건이
달라진다. 같은 plan을 N개 체크아웃에서 격리해 실행하는 결정적 경로가 없다.

## 채택 기준 대조

[기능 채택 기준](../SOUL.md#기능을-채택하는-기준) 네 질문에 대한 답이다.

1. 같은 개발 환경을 여러 코드 상태로 동시에 실행하고 비교하는 문제를 해결한다.
2. 기존 plan을 그대로 쓰고, 비교안별 차이(체크아웃, 포트 변수)만 선언한다.
3. 실행 전 비교안별 경로, 커밋, 포트, URL을 보여주고 같은 선언으로 상태와 종료를 확인한다.
4. Git 작업 생성, 브랜치, 커밋, 통합은 소유하지 않는다. 스냅샷 모드에서 DVA가
   만드는 것은 자신이 회수하는 detached 체크아웃뿐이다.

## 모드

| 모드 | 체크아웃 출처 | 생성·회수 | 편집 |
|---|---|---|---|
| snapshot | 지정한 ref를 DVA가 detached worktree로 생성 | DVA | 하지 않음 |
| worktree | provider가 알려주는 기존 작업 worktree | provider(worktrunk, CE 등) | 그 worktree에서 함 |
| config | 호출한 체크아웃 하나, 비교안별 vars | 체크아웃 없음 | 하지 않음 |

- **snapshot**: `up` 시점에 ref를 커밋으로 고정하고
  `.sb/dva/variants/<compare>/<variant>/`에 detached worktree를 만든다. 브랜치를 만들지 않는다.
  `down`은 DVA가 만든 경로만 `git worktree remove`(force 없음)로 회수한다. dirty하면
  회수를 거부하고 경로를 보고한다.
- **worktree**: 후보 조회와 재검증은 [TASK-538](../tasks/todo/538-worktree-development-targets.md)의
  provider 계약을 따른다. DVA는 그 경로를 만들거나 지우지 않는다.

한 비교 실행 안에서 모드를 섞을 수 있다(예: `a`는 snapshot, `b`는 작업 worktree). config 모드는 결정 기록을 따른다.

## 선언 스케치

필드 이름은 가칭이다.

```yaml
compare:
  web-layout:
    plan: web-dev
    shared: [api, db]        # 호출한 체크아웃에서 한 번만 기동
    per_variant: [web]       # 비교안마다 각 체크아웃에서 기동
    ports:
      WEB_PORT: [3001, 3002, 3003, 3004]
    variants: { a: main, b: feat/layout-a }   # 선택: 인자 없이 호출할 때의 기본 묶음
```

```bash
dva compare up web-layout a=feat/layout-a b=feat/layout-b c=main
dva compare up web-layout a=feat/layout-a b=worktree:TASK-12
dva compare status web-layout
dva compare down web-layout
```

`shared`와 `per_variant`는 plan의 엔트리 이름이고, 둘을 합치면 plan의 엔트리 전체와
같아야 한다. 프론트 비교는 API와 DB를 공유하고, 서버 비교는 DB까지 `per_variant`에 둘 수 있다.

## 격리 규칙

- **포트**: 비교안 i는 각 포트 목록의 i번째 값을 plan vars로 받는다. 목록이 비교안 수보다
  짧으면 실행 전에 실패한다. 점유된 포트를 다른 값으로 옮겨 성공으로 보고하지 않는다.
- **compose 프로젝트**: `project_name:`이 없으면 체크아웃 디렉터리 이름으로 갈린다.
  `per_variant` 엔트리에 고정 `project_name:`이 있으면 충돌이므로 실행 전에 실패한다.
- **프로세스 신원**: `shared`는 호출한 설정 디렉터리, `per_variant`는 각 비교안 설정
  디렉터리의 `.sb/dva/pids/`에 남긴다. 새 레지스트리를 만들지 않는다.
- **config 모드의 인스턴스**(TASK-593): 비교안이 호출한 체크아웃을 공유하므로 DVA가 비교안
  이름을 인스턴스로 삼아 `per_variant` 엔트리의 pid·로그를 `.sb/dva/{pids,logs}/<비교안>/`로
  나누고, compose·podman-compose 프로젝트 이름은 `<디렉터리 이름>-<비교안>`을 compose가 요구하는
  형식(소문자·숫자·`-`·`_`)으로 정규화해 쓴다. `per_variant` compose 항목의 `project_name:`은 검증이
  거부하므로 접두사가 될 수 없다. 대소문자만 다른 비교안 이름은 한 디렉터리로 겹치므로 거부한다. 인스턴스 없는 일반 `dva up`의 경로와
  이름은 바뀌지 않는다. 비교안 이름은 plan var `DVA_VARIANT`로도 전달된다. OS 환경에 `DVA_VARIANT`가 이미 있으면 그 값이 모든
  비교안을 덮어쓰므로 포트 var와 같이 실행 전에 거부한다.
- **config 모드의 빌드 상태**: DVA가 알고 있는 도구(`next`, `nuxt`, `nuxi`, `astro`,
  `gatsby`)를 `per_variant` 엔트리가 실행하는데 그 명령·dir·env가 `DVA_VARIANT`를 쓰지 않으면
  실행 전에 실패한다. 임의 명령이 체크아웃에 쓰는 것은 DVA가 볼 수 없으므로 이 목록은
  증명이 아니라 알려진 경우의 방어선이다.
- **환경 입력**: 비교안은 자기 체크아웃의 `dva.yml`과 env 입력을 쓴다. 비교안 사이에서
  암호화 입력을 복사하지 않는다.

## 수명주기

- `up`: 모든 비교안의 출처를 해석하고 포트를 배정해 표시한 뒤, `shared`를 한 번,
  `per_variant`를 비교안마다 시작한다. 기존 `--dry-run`은 생성과 시작 없이 해석 결과만 낸다.
  이미 같은 출처와 커밋으로 떠 있는 비교안은 다시 만들지 않는다. 그래서 다시 호출하면
  빠지거나 실패한 비교안만 시작된다.
- `status`: 비교안별 출처(ref 또는 작업 ID), 커밋, 포트, 실제 관찰된 URL, readiness,
  프로세스 신원, 로그 경로를 사람과 기계 출력으로 낸다. snapshot의 ref가 그 뒤 움직였으면
  drift로 표시한다. 읽기 전용이며 아무것도 시작하거나 고치지 않는다.
- `down [<variant>...]`: 지정한 비교안을, 생략하면 전부 내린다. `per_variant`를 먼저 내리고
  마지막 비교안이 내려가면 `shared`를 내린다. snapshot만 회수하고 작업 worktree는
  건드리지 않는다. 종료 실패는 해당 비교안을 남긴 채 실패로 보고한다.

### 일부 실패

`up` 도중 일부 비교안이 실패하면 **이미 시작한 것은 그대로 두고 실패로 끝낸다**.
자동으로 되돌리지 않는다. 출력에는 비교안별 결과(시작됨, 실패와 원인, 시도 안 함)와,
되돌릴 때 쓸 정확한 `dva compare down` 명령을 함께 낸다. 되돌리기는 사용자나 에이전트가
그 명령을 **명시적으로 호출할 때만** 일어난다. 결과는 전부 성공, 일부 실패,
시작 전 실패를 구분해 보고한다.

## 사람과 에이전트의 사용

같은 명령을 사람은 손으로, 에이전트는 기계 출력으로 쓴다. 별도의 에이전트 전용 경로는 두지 않는다.

- 수동: `up`·`status`·`down` 세 동사와 `이름=출처` 인자만 알면 된다. 자주 쓰는 비교안
  묶음은 `compare.<name>.variants`로 선언해 인자 없이 호출할 수 있다.
- 에이전트: 같은 명령에 기존 기계 출력 플래그를 붙인다. 판단에 필요한 사실(비교안별 결과,
  URL, 다음에 실행할 명령)을 모두 출력에 담는다. 그래서 에이전트가 로그를 읽거나 포트를
  탐색할 필요가 없다. 재호출이 안전하므로(`up`은 빠진 것만 시작) 실패 후 재시도에 추측이 필요 없다.

## 미리보기 워크플로우와의 연동

`ui-iterate` 같은 미리보기 워크플로우는 서버 수명주기, URL, 포트, readiness를 호스트에
맡긴다. `dva compare`는 그 호스트 수명주기가 될 수 있다. 이 연동은 선택이며 DVA가
그 워크플로우에 의존하지 않는다.

- 전형적인 조합은 `a=main`(snapshot, 고정 기준선)과 `b=worktree:<현재 작업>`(편집 중,
  hot reload)이다. 워크플로우는 `b`에 바인딩하고, `a`는 나란히 보는 비교 대상으로 쓴다.
- 워크플로우의 start는 `up`을, status는 `status`의 비교안별 URL·readiness·프로세스 신원을,
  stop은 `down`을 쓴다. 다른 작업이 띄운 비교안은 그 작업의 `down`만 내린다.
- 선언된 포트는 고정 바인딩이다. 워크플로우는 이를 바꾸지 않고 `status`가 보고한 주소를 사용한다.

## TASK-538·587과의 관계

worktree 모드는 538의 후보 조회와 재검증을 그대로 쓴다. 이 설계는 538의 범위(Git 작업
생성 제외)를 바꾸지 않는다. snapshot은 Git 작업이 아니라 DVA가 소유한 실행 산출물로 본다.
587의 `dva target`은 단일 대상, `dva compare`는 N개 대상의 묶음이다.

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

ROADMAP 채택 후 실행 카드는 TASK-589~593이다.
