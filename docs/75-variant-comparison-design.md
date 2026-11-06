# 비교안 동시 실행 설계 초안

상태: **설계 초안, 미구현**. 이 문서는 명령이나 설정 필드를 지원한다고 선언하지 않는다.
[ROADMAP.md](../ROADMAP.md)가 채택하고 실행 카드로 나누기 전에는 후보다.

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

## 두 모드

| 모드 | 체크아웃 출처 | 생성·회수 | 편집 |
|---|---|---|---|
| snapshot | 지정한 ref를 DVA가 detached worktree로 생성 | DVA | 하지 않음 |
| worktree | provider가 알려주는 기존 작업 worktree | provider(worktrunk, CE 등) | 그 worktree에서 함 |

- **snapshot**: `up` 시점에 ref를 커밋으로 고정하고
  `.sb/dva/variants/<compare>/<variant>/`에 detached worktree를 만든다. 브랜치를 만들지 않는다.
  `down`은 DVA가 만든 경로만 `git worktree remove`(force 없음)로 회수한다. dirty하면
  회수를 거부하고 경로를 보고한다.
- **worktree**: 후보 조회와 재검증은 [TASK-538](../tasks/todo/538-worktree-development-targets.md)의
  provider 계약을 따른다. DVA는 그 경로를 만들거나 지우지 않는다.

한 비교 실행 안에서 두 모드를 섞을 수 있다(예: `a`는 snapshot, `b`는 작업 worktree).

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
그 명령을 **명시적으로 호출할 때만** 일어난다. 종료 코드는 전부 성공, 일부 실패,
시작 전 실패를 구분한다.

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

## 미결정 질문

1. snapshot마다 필요한 의존 설치(`pnpm install` 등)를 plan의 `provision`에 맡길지.
2. 비교안이 코드가 아니라 설정만 다를 때 체크아웃 없이 vars만 바꾸는 세 번째 모드를 둘지.
3. worktree 모드의 키를 작업 ID에서 provider가 정의하는 ID(worktrunk는 branch)로 일반화할지.
   이는 587 계약의 변경이다.
4. 비교 실행 묶음 자체(어떤 비교안이 떠 있는지)를 status가 무엇으로 재구성할지.
   새 레지스트리 없이 snapshot 경로와 provider 재조회로 충분한지 확인이 필요하다.
   `up` 재호출의 안전성과 `down`의 대상 결정도 같은 재구성에 기댄다.
5. 기계 출력의 필드 이름과 종료 코드 값. 기존 출력 규약을 확인한 뒤 정한다.
