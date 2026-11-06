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
- `status`: 비교안별 출처(ref 또는 작업 ID), 커밋, 포트, URL, 상태를 사람과 기계 출력으로 낸다.
  snapshot의 ref가 그 뒤 움직였으면 drift로 표시한다.
- `down`: `per_variant`를 모두 내린 뒤 `shared`를 내리고, snapshot만 회수한다.
  종료 실패는 해당 비교안을 남긴 채 실패로 보고한다.

## TASK-538·587과의 관계

worktree 모드는 538의 후보 조회와 재검증을 그대로 쓴다. 이 설계는 538의 범위(Git 작업
생성 제외)를 바꾸지 않는다. snapshot은 Git 작업이 아니라 DVA가 소유한 실행 산출물로 본다.
587의 `dva target`은 단일 대상, `dva compare`는 N개 대상의 묶음이다.

## 미결정 질문

1. 일부 비교안만 시작에 실패하면 이미 시작한 것을 되돌릴지, 남기고 실패로 보고할지.
2. snapshot마다 필요한 의존 설치(`pnpm install` 등)를 plan의 `provision`에 맡길지.
3. 비교안이 코드가 아니라 설정만 다를 때 체크아웃 없이 vars만 바꾸는 세 번째 모드를 둘지.
4. worktree 모드의 키를 작업 ID에서 provider가 정의하는 ID(worktrunk는 branch)로 일반화할지.
   이는 587 계약의 변경이다.
5. 비교 실행 묶음 자체(어떤 비교안이 떠 있는지)를 status가 무엇으로 재구성할지.
   새 레지스트리 없이 snapshot 경로와 provider 재조회로 충분한지 확인이 필요하다.
