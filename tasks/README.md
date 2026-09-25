# DVA Task Management Board

## 현재 릴리스 및 후속 큐 (2026-09-25)

v0.3.0 공개와 postflight는 완료됐다. PLAN-010 공개 릴리스 범위는 8/8이며,
TASK-411은 ignored 런타임 잔재를 보존하는 독립 리뷰를 마쳤다. 상류 후속 큐의 최신 의존 순서는
[[PLAN-011]]이다. PLAN-011 대상 TASK-420–436·438–439은 현재 todo 0, doing 0, review 0, blocked 6, done 13장이다.
TASK-407은 기준 정합화 뒤 `review/`에서 독립 재검증을 기다린다. TASK-437은 보드 현행화와 독립 완료 리뷰를 마쳤다.

- ISSUE-037: local `master`와 `origin/master` 동기화가 확인되어 해결됨.
- ISSUE-014: 영수증 디렉터리 `unknown` 분류는 TASK-403으로 닫혀 아카이브됐다.
- ISSUE-005/TASK-422: 영수증 종단 상태 기록과 조회 보존이 상류에 이미 구현되어 독립 리뷰 후 done 처리했다.
- ISSUE-024: finalize 차단은 우회하지 않는다. 남은 작업은 TASK-430이다.
- ISSUE-004: TASK-421은 historical controller 삭제 후 새 owner/CLI route가 없어 blocked다 (ISSUE-040).
- ISSUE-043: TASK-438 fixture에서 provider가 통합한 뒤 Worktrunk 회수에 실패하면 후속 `run-status`가 inventory 불일치로 막히는 recovery gap을 기록했다. TASK-439는 source containment 확인, 잔여 path/ref 보존, 제한된 receipt에 대한 `run-status`·`run-list`의 `run-recover` 안내, idempotence와 `--no-fetch` 지원을 구현하고 독립 리뷰를 통과해 이슈를 해결·아카이브했다.
- ISSUE-032/TASK-435: source-branch 승격 차단은 `ce-agent-kit` master `71b1d169`에 포함된 `05b0b4ae` 수정으로 해결됐다.
- ISSUE-013/019/020/022/028/031/032는 상류 수정 확인 후 `tasks/_archive/issue/`에 보관했다.
- TASK-431 source는 통합됐지만 live install의 보존 대상 자격 증명 설정 drift 대기로 blocked다. TASK-432, TASK-433, TASK-420, TASK-438, TASK-439는 상류 통합·독립 리뷰를 마치고 done이다. TASK-420은 legacy completion evidence 세 형상에 fresh `quality-review-evidence`를 기록한 뒤 canonical receipt가 validator까지 통과하는 것을 검증했다. TASK-438의 exact-CI commit `71b1d169`와 TASK-439의 exact-CI commit `7eaf596a`는 CE master/origin에 통합됐고 각 task worktree/branch 회수도 확인했다. ISSUE-008/042/043은 회귀 수정과 fixture·테스트 증거를 근거로 해결·아카이브됐다. TASK-437은 최종 보드 증거 리뷰 PASS 후 `done/`으로 옮겼다. ISSUE-001은 ce-workbook controller run이 남아 todo를 유지한다. TASK-424 상류 구현은 live-zone 기준만 충족해 ISSUE-041 parity 해결 전까지 blocked이며, DVA의 기존 guard를 유지한다. TASK-430은 canonical review receipt 보존 계약 대기로 blocked다.
- ISSUE-039: TASK-436은 stale·미푸시 local worktree의 ref 처분 결정 대기로 blocked다. 훅 재설계는 그 밖이다.
- TASK-420–439의 상태와 실행 의존성은 [[PLAN-011]]에서 유지한다. TASK-437은 보드 현행화 카드라 PLAN-011 child 수에는 포함하지 않는다.
- PLAN-006~009의 아카이브와 `tasks/` 아래 네 `.ce` 잔재 삭제도 기존 사람 결정 대기다.
  루트 `.ce/task-runtime.yaml`은 ACTIVE 런타임 선언이므로 보존한다.

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
완료 카드 13장과 단독 TASK-411, TASK-437이 `tasks/done/`에 기록되어 현재 done 카드는 15장이다.

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
