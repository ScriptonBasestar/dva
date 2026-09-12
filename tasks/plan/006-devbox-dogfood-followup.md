---
id: PLAN-006
title: "Work the devbox dogfood follow-up queue in dependency order"
type: plan
scope: "TASK-311..323 from the 2026-09-05 mydevbox migration, plus the needs-human cards that gate the rest"
progress: 84
total-tasks: 26
completed-tasks: 22
children: [TASK-339, TASK-340, TASK-328, TASK-329, TASK-312, TASK-313, TASK-317, TASK-311, TASK-324, TASK-314, TASK-316, TASK-320, TASK-322, TASK-315, TASK-318, TASK-323, TASK-249, TASK-307, TASK-309, TASK-319, TASK-321, TASK-345, TASK-346, TASK-347, TASK-348, TASK-351]
target-date: "2026-10-31"
created: 2026-09-05
---

## Goal

2026-09-05 mydevbox dogfood(23개 devbox 저장소를 dva v0.1.48 plans 스키마로 이전)에서 나온 결함·
설계 카드를 하나씩 처리한다. `reports/`·`tmp/`는 ignore 대상이라 이 카드가 상태의 정본이다.
카드는 `tasks/todo/`, 완료는 `tasks/done/`, 이 문서는 순서·의존·상태만 소유한다.

## Starting state (2026-09-05)

- 완료·통합: TASK-303, 304, 305, 306, 308, 310 (dva master d7636a3, e3c562d, 9b74de9, b2c2d13).
- devbox 이전 커밋 `chore(dva): migrate dva.yml to plans and clear validate warnings`가 19개
  저장소 trunk에 착지. 미착지 4개는 §Blocked 참조.
- 설계 대기 문서: docs/55(alias/extends), docs/56(suppression). 결정 항목은 각 문서 §5.

## Order (decision-free first)

| # | Task | Why here |
|---|---|---|
| 1 | ~~TASK-312~~ dry-run up이 native health를 기다림 | 완료 2026-09-05. P1 S. 다른 카드의 dry-run 검증을 막았음 |
| 2 | ~~TASK-313~~ local 러너 workdir 무시 | 완료 2026-09-05. P1 S. 독립 |
| 3 | ~~TASK-317~~ migrate 힌트 오류·legacy 누락 | 완료 2026-09-05. P1 M. 306 스캐폴드와 별개 |
| 4 | ~~TASK-311~~ down <plan> volume/network 잔존 | 완료 2026-09-05. P1 M. `--purge`가 프로젝트 전체 down |
| 4a | ~~TASK-324~~ composition plan 중복 경고 오탐 | 완료 2026-09-05. P2 S. `Composes` 비교 추가 |
| 5 | ~~TASK-314~~ logs/build plan 범위 | 완료 2026-09-05. P2 S |
| 6 | ~~TASK-316~~ drift 감지 결함 | 완료 2026-09-07(quality-review conditional). P2 M. docs/56 `drift_ignore`(309)보다 먼저 — 감지 폭이 억제 설계의 입력 |
| 7 | ~~TASK-320~~ suggestion 파서·manifest | 완료·통합 2026-09-08. P3 S. 항목 1·2 수정, 항목 4는 `dva clean`이 실제로 라우팅되므로 변경 없음(테스트로 고정), 항목 3은 TASK-351로 분리 |
| 8 | ~~TASK-322~~ init 탐지 결함 | 완료 2026-09-07. P2 M. 249 재설계와 겹치지 않는 탐지 버그만. 잔여 항목은 339·340으로 분리 |
| 9 | ~~TASK-315~~ compose profiles | 완료·통합 2026-09-08(master 5f2d85d). P2 M. 후속 345·346·347·348 파생 |
| 10 | ~~TASK-318~~ 섹션 순서 자동 정렬 | 완료 2026-09-08. P3 S. 재리뷰가 결함 8장(TASK-358~365)을 파생시켰고, 그 묶음은 PLAN-008이 소유한다(2026-09-09 신설) |
| 10a | 실기동 검증 회차 (TASK-328) | 311 완료로 착수 가능. primeno1 native 6종(gate 체인+`exec`), familybook/flow-taskchain composition plan을 `dva up`/`status`/`down --purge` 실제 실행으로 확인하고 각 리포트에 출력 첨부. 결함이 나오면 카드로 승격 |
| 11 | ~~TASK-323~~ 문서 의미 공백 | 완료 2026-09-08. P3 S. `--env` 문구는 307 결정 전이라 보류했고, USAGE.md 분할·예제 검증은 356·357로 분리 |

## Needs-human (결정 후 착수)

- ~~TASK-307~~ plan alias/extends (완료 2026-09-11). TASK-309 → docs/56 §5 (5항목). 309의 선행 316·320은 둘 다 완료.
- TASK-319 native entry ergonomics, TASK-321 destructive interaction agent-deny: 설계 결정 기록 필요.

## Devbox integration state (2026-09-05 저녁 갱신)

- 통합 완료: db-orchestrator 6d0e3e07(master), primeno1 768cbb4(native 6 엔트리), familybook 9d559fb(자식 import,
  engine-fiber 3d30df87·client-flutter b792a5b9), flow-taskchain 2134c8c + 자식 4개, dns-bridge 원격 브랜치 삭제 완료.
- 미통합 1건: scripton-dashboard `dev/claude/mst/chore/dva-yml-migrate`(25dd928). 해법 작업 중 브랜치
  `dev/claude/mst/chore/check-prepares-subprojects`(50cf918, 체크포인트): `check`가 order-only 전제로 `prepare`·
  `pnpm install`을 자동 수행. 남은 결함 — dashboard-webui의 `file:` 링크가 install 시점에 복사되므로
  protocol/components `dist`를 dashboard install보다 먼저 빌드해야 함(`make install` 순서와 동일). 워크트리
  `~/worktrees/scripton-dashboard/scripton-dashboard-devbox/claude__mst__chore__check-prepares-subprojects` 유지.
  완료 후 이 브랜치 → develop 통합 → dva-yml-migrate 브랜치 rebase 후 통합 순. 래퍼가 target을 임시 트리에
  체크아웃해 `make check`를 돌리므로 gitignore된 `scripton-mfe-protocol`이 없어 baseline 측정 불가.
  `--allow-skipped-checks`로도 우회 안 됨. 해법은 devbox Makefile `check`가 `prepare`를 선행하거나 사람이 통합.
- ~~사람 통합 대기: familybook `dev/claude/mst/chore/readiness-dva-yml`(3538cf2)~~ — **2026-09-08 확인: 유실.**
  `~/mydevbox/familybook-devbox`에 커밋 3538cf2가 없고(`git cat-file -t` 실패), readiness 브랜치도 워크트리도
  없으며 HEAD는 `be6d0fd [develop]`, 파일은 여전히 `dva.yaml`. 통합 대기가 아니라 **재작성 대상**이다.
  상세와 정정된 기준은 TASK-329 참조.
- 세션 종료 2026-09-05 저녁(dva 2cb184e): 이 세션은 TASK-324 완료, PLAN-006/TASK-323 현행화, familybook 자식 통합
  (`.gz-git.yaml` workspace 항목 + `integration: {}`)까지. 다른 세션이 311·313·314·317을 병행 처리했으므로
  다음 착수 전 `git log origin/master`와 `tasks/todo/` 재확인.
- 실기동 검증 미실시: primeno1 native 엔트리(gate 체인 + `exec`)와 familybook/flow-taskchain composition plan은
  dry-run까지만 확인. TASK-311 완료 후 실기동 회차 필요.

## Session handoff (2026-09-05 밤)

- 세션 1회차 종료 시점: 1~5·4a 통합 완료(master 2cb184e). 316은 분석만 카드에 기록, worktree 없음.
- 다음 착수(2026-09-09 갱신): 318·323까지 닫혀 §Order의 기계적 항목은 전부 소진됐다. **남은 것은 10a 실기동(TASK-328) 하나뿐**이고, 그것도 사람이 실제로 돌려야 닫힐 카드다.
  사람 결정 대기는 4장(307, 309, 319, 321)이다. 315 후속 345~348은 §Order에 편입하지 않고 compose 묶음으로 함께 처리.
- 잔여 사람 결정: 위 §Needs-human, §Devbox integration state의 scripton-dashboard·familybook 2건.

## Children

착수 순서는 §Order와 §Needs-human이 정본이다. 아래는 26장 전부를 **현재 위치와 함께**
나열한 것으로, 목적은 순서를 다시 적는 것이 아니라 **어느 자식이 어느 절에도 놓이지
않았는지 드러내는 것**이다. frontmatter `children:`는 planprogress가 세고, 이 절은
validate가 요구한다 — 둘은 다른 질문에 답한다.

### §Order에 놓인 13장

- TASK-312 — `--dry-run up`이 native health를 기다림 (done)
- TASK-313 — local 러너가 `interaction.workdir`를 무시 (done)
- TASK-317 — config migrate 힌트 오류·legacy 필드 누락 (done)
- TASK-311 — `down <plan>`이 named volume/network를 남김 (archive/done)
- TASK-324 — composition plan 중복 경고 오탐 (done)
- TASK-314 — `logs`/`build <plan>` 범위 좁히기 (done)
- TASK-316 — compose drift 감지 결함 (archive/done)
- TASK-320 — Makefile suggestion 파서·manifest usage_example (done)
- TASK-322 — init 탐지 결함 6종 (done)
- TASK-315 — compose profiles: stack 러너 옵션과 PlanEntry profiles (done)
- TASK-318 — 섹션 순서 자동 정렬 (done). 재리뷰가 파생시킨 8장은 **PLAN-008이 소유**한다
- TASK-323 — 문서 의미 공백 (done)
- TASK-328 — 실기동 검증 회차 (todo). **§Order에 남은 유일한 미완 항목이다**

### §Needs-human에 놓인 4장 — 결정 전에는 착수 불가

- TASK-307 — plans `alias`/`extends` (done)
- TASK-309 — suppression ergonomics (todo)
- TASK-319 — native entry ergonomics (todo)
- TASK-321 — destructive interaction agent-deny (done)

### 남은 순서 미지정 1장 — TASK-329

완료된 TASK-249·339·340은 더 이상 실행 순서를 필요로 하지 않고, compose 번들은 아래
§Compose follow-up sequence가 소유한다. 이 plan의 미완 자식 중 §Order·§Needs-human 어느
쪽에도 아직 놓이지 않은 것은 TASK-329 하나다.

- TASK-329 — familybook devbox `dva.yaml` → `dva.yml` rename (todo). 외부 readiness 브랜치가
  실제로 착지했는지 확인한 뒤에만 다시 순서를 부여한다. 그 전에는 다른 plan 자식이나 내부
  코드 작업의 선행 조건으로 취급하지 않는다.

### Compose follow-up sequence (currentized 2026-09-10)

1. TASK-345 is done: a single compose entry now consumes its matching explicit name.
2. TASK-347 is done independently: generator guidance selects profiles on plan entries.
3. TASK-346 is done: validation checks the compose profile source of truth without creating a
   second dva.yml declaration.
4. TASK-348 remains the final human-owned real-image-build observation. It follows 345 and
   346 because it verifies their user-facing compose path, and agents must not perform the
   lifecycle build it requires.

## Rules

- 카드 하나 = worktree 하나 = 통합 하나. 완료 시 카드를 `tasks/done/`으로 옮기고 이 문서의
  `completed-tasks`·`progress`를 같은 커밋에서 갱신한다.
- dry-run 검증이 필요한 카드(311, 315)는 1번 완료 전에는 착수하지 않는다.
