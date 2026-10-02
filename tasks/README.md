# DVA Task Management Board

## 현재 상태 (2026-10-03)

v0.3.0 공개와 postflight는 완료됐고 PLAN-010(8/8)과 PLAN-011(19/19)은
`tasks/_archive/plan/`에 보관됐다. 열린 계획은 없다. 과거 판정은 보관된 카드와 Git 이력이 소유한다.
2026-10-03 [[TASK-482]]가 done 카드 30장(TASK-449~481)을 재검증해 전부 `tasks/_archive/2026-10/`로 보관했다.
`tasks/done/`에는 자기 자신을 닫는 TASK-482와 durable review evidence만 남는다 —
TASK-482는 다음 정리 주기에 독립 재검증을 받는다.

열린 이슈 (전부 DVA 측에서 처리할 수 있는 작업이 없다):

- ISSUE-004/006 (upstream-waiting): taskchain-task-manager W13/W14 제품 구현은 완료됐고 DVA 실행 소비자 채택과 사람 전용 terminal 실측을 기다린다.
- ISSUE-453 (blocked, 사람 승인): W07c2a 배포 artifact 승인과 pin 활성화, 양성·음성 host 실측을 기다린다.
- ISSUE-454 (upstream-waiting): DVA 측 작성 시점 검사([[TASK-460]])는 체크됐고, `ce` 런타임 재귀 가드만 ce-agent-kit 보고를 기다린다.
- ISSUE-461 (upstream-waiting): bare `ce task run-finish` 자동 선택은 ce-agent-kit 소유다.

열린 todo: TASK-459는 체크된 기준 7개(기계 6 + `make test`·`make lint` 1)가 통과했고, 만료 토큰 종료 코드 실측과 라이브 `dva doctor` 두 사람 확인만 남았다.

보드 현행화는 `tasks/`만이 아니라 `git worktree list`와 `ce task run-list`도 함께 확인한다.
완료 카드는 현행화 태스크 끝에 `tasks/_archive/YYYY-MM/`로 보관한다(`done-finalize`는 카드를 삭제해 plan children을 깨뜨린다).

이 문서는 DVA 프로젝트의 태스크 및 보드 구조, 아카이브 상태를 안내하는 정본 인덱스입니다.

## 보드 구조

- `tasks/todo/`: 현재 대기 중인 실행 가능한 작업 카드
- `tasks/done/`: 구현 및 마감 리뷰가 완료되어 아카이빙을 대기 중인 카드
- `tasks/plan/`: 다중 태스크 묶음을 관리하는 상위 계획 카드
- `tasks/issue/`: 발견된 버그, 구조 결함, 상류 의존성 이슈 카드
- `tasks/review/`: 리뷰 대기 중인 카드
- `tasks/backlog/`: 아직 착수 순번이 오지 않은 카드
- `tasks/_archive/YYYY-MM/`: 모든 완료 기준 및 실측 검증을 통과하여 보관된 아카이브 카드.
  옛 철자 `tasks/archive/`는 TASK-411에서 접혔고 카드가 남아 있지 않습니다 (`doccheck`·
  `planprogress`는 TASK-410 이후 두 철자를 같은 zone으로 읽습니다)

## 아카이빙 진행 상태

2026-09-22 전수 재검증 시점에 `tasks/done/`은 비어 있었습니다 (`evidence/`만 유지).
누적 97건 전수 재검증 — 96건 아카이브 이관, 2건 복귀(TASK-370, TASK-407).
그 뒤 다시 done에 모인 12장을 2026-09-23에 재검증했고, **그 시점에는**
`tasks/done/`에 `evidence/`만 남았다. 2026-09-25 후속 작업에서 PLAN-011의
완료 카드 15장, 단독 TASK-411·TASK-437, TASK-407이 tasks/done/에 기록되어 당시 done 카드는 18장이었다.
2026-09-27까지 이 완료 배치들은 _archive/2026-09/로 이동했다. 2026-09-28에는
남은 TASK-436과 TASK-447도 독립 검토 기록을 확인한 뒤 _archive/done/에 보관했다.
2026-10-02까지 TASK-449~481 30장이 다시 done에 모였고, 2026-10-03 [[TASK-482]]가
전부 재검증해 `_archive/2026-10/`로 보관했다(아래 Batch 9).

| 배치 | 범위 | 대상 수 | 상태 |
|:---:|:---|:---:|:---:|
| Batch 1 | ISSUE-002, 003, 029, TASK-307 ~ 320 | 15 | 완료 (아카이브 이관 및 검증 완료) |
| Batch 2 | TASK-321 ~ 338 | 15 | 완료 (15 아카이브 이관 및 검증 완료) |
| Batch 3 | TASK-339 ~ 353 | 15 | 완료 (15 아카이브 이관 및 검증 완료) |
| Batch 4 | TASK-354 ~ 368 | 15 | 완료 (15 아카이브 이관 및 검증 완료) |
| Batch 5 | TASK-369 ~ 385 | 17 | 완료 (16 아카이브 + TASK-370은 미완료 기준 보유로 `todo/` 복귀) |
| Batch 6 | TASK-386 ~ 404 | 17 | 완료 (17 아카이브 이관 및 검증 완료) |
| Batch 7 | TASK-405 ~ 409 | 5 | 완료 (4 아카이브 + TASK-407은 기준 1 미충족으로 `todo/` 복귀) |
| Batch 8 | TASK-370, TASK-407, TASK-410 ~ 419 | 12 | 완료 (10 아카이브 + TASK-407·411은 `blocked/` 복귀) |
| Batch 9 | TASK-449 ~ 481 | 30 | 완료 (30 아카이브, todo 환류 0) |

### Batch 7 상세 (2026-09-22)

| 카드 | 판정 | 근거 |
|:---|:---|:---|
| TASK-405 | 아카이브 | `--near-limit` 23건 나열, `go test ./tools/doccheck/` ok, 분리 규약 섹션 확인 |
| TASK-406 | 아카이브 | 위조/정식 경로 대조 §2·§3, TASK-376 evidence 정정 확인. `docs/406` §4의 옛 경로 표기 수정 |
| TASK-407 | `todo/` 복귀 | AGENTS.md 규칙이 `max+1`이라 격리된 워크트리 둘이 같은 번호를 고른다 — 기준 1 미충족 (ISSUE-033) |
| TASK-408 | 아카이브 | 위키링크 나열 인식 테스트 통과, 역방향·중복 제거는 TASK-394·391에서 이미 닫힘 재측정 |
| TASK-409 | 아카이브 | `prose.go` 파일 코멘트와 `TestIssue021PairingResidue`가 양 어순을 고정 |

### Batch 8 상세 (2026-09-23)

| 카드 | 판정 | 근거 |
|:---|:---|:---|
| TASK-370 | 아카이브 | v0.2.0 노트 SHA-256 유지, CHANGELOG 예시 3, v0.3.0 노트는 `depends_on`을 step에 둠 |
| TASK-407 | `blocked/` 복귀 | 기준 1 미충족. 격리된 두 워크트리가 여전히 같은 `max+1`을 고른다 |
| TASK-410 | 아카이브 | 두 철자 zone 테스트와 문서 게이트 통과. receipt는 수정하지 않음 |
| TASK-411 | `blocked/` 복귀 | 기준 2 미충족. 이 체크아웃 게이트가 `legacy-storage-dir`를 냄 |
| TASK-412 | 아카이브 | commit/full dry-run이 yamlcheck와 changelogcheck를 실행 |
| TASK-413 | 아카이브 | 프로파일 위임 확인. yamlcheck 제거 시 `make doc-check`가 실패하고 원복함 |
| TASK-414 | 아카이브 | review/backlog zone, 교차 중복, 미선언 디렉터리 테스트 통과 |
| TASK-415 | 아카이브 | doing/blocked zone이 상태·중복 검사에 포함 |
| TASK-416 | 아카이브 | PLAN-006 29장 각 1회, 문서·보드 게이트 통과 |
| TASK-417 | 아카이브 | 0.3.0 문서 정합, `make check-generate`, `make release-check` 통과 |
| TASK-418 | 아카이브 | v0.3.0 postflight가 원격 identity와 7개 자산을 다시 확인 |
| TASK-419 | 아카이브 | 빈·불일치·과거 후보는 changelogcheck가 거부 |

통합 시점(2026-09-23) 추가 발견: 이 배치를 master에 올리는 중 TASK-412 카드가
`review/`와 `todo/`에 동시에 존재했는데 `doccheck`가 `duplicate: 0`을 보고했다.
`cardZones`가 `review/`·`backlog/`를 선언하지 않아 네 장이 세 검사 모두에서 빠진다 —
TASK-407이 방어선으로 지목한 DUP-ID 검사가 침묵하는 경로다. [[TASK-414]]로 분리했다.

### Batch 9 상세 (2026-10-03)

30장 전부 아카이브. 기계 바인딩 94개 재실행 PASS. `make lint`·`make test` exit 0.
사람 기준은 카드와 다른 에이전트가 독립 재검증했다(split 14장은 커밋 diff의 순수 이동 대조).

| 카드 | 판정 | 근거 |
|:---|:---|:---|
| TASK-449~452 | 아카이브 | 읽기 전용 queue·verdict·start 브리지 테스트 통과. 452 host 증거의 외부 커밋 `2ee38b2`(ce-agent-kit)·`d0635bf`(task-manager-devbox)·`9e8fac7`(taskchain) 실존 확인 |
| TASK-454~456 | 아카이브 | pin/start 테스트 통과. ISSUE-453 본문이 production pin 비활성을 정확히 기술 |
| TASK-457·458 | 아카이브 | docs/68 §2~6, kubernetes Secret target 테스트와 docs/62·69·USAGE·ARCHITECTURE 정합 |
| TASK-460 | 아카이브 | doccheck 재귀 검사 테스트 통과 → ISSUE-454 DVA 기준 체크 |
| TASK-462~475 | 아카이브 | 14 커밋 모두 원본 0 insertion, 제거·추가 라인 multiset 일치(package/import 제외). 473·474 빌드 바인딩이 루트에 바이너리를 남겨 `-o /dev/null`로 수정 |
| TASK-476 | 아카이브 | `make lint` 출력 링크 누락 → 재실행 결과(0 issues)를 Evidence에 기록 |
| TASK-477~480 | 아카이브 | 존·status·decision 마커 바인딩 통과 |
| TASK-481 | 아카이브 | doctor 터널 테스트 통과. 기존 docs/69와 번호가 겹친 초안을 `docs/70`으로 이동 |
