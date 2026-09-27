# DVA Task Management Board

## 현재 릴리스 및 후속 큐 (2026-09-27)

v0.3.0 공개와 postflight는 완료됐다. PLAN-010 공개 릴리스 범위는 8/8이다. 상류 후속 큐의 최신 의존 순서는
[[PLAN-011]]이다. PLAN-011의 19장 중 완료 16, 이관 2, 차단 1장이다.
TASK-421/423은 `61ead57b`에서 task-manager-devbox W13/W14로
이관되어 superseded 보관됐다. TASK-431은 source 0.74.0 핀의 live 설치와
호스트 lint 2.13.2 검증을 마쳤다. TASK-443은 독립 done-review PASS를 받아 보드 현행화를 완료했다.
TASK-444의 fresh PASS와 first durable receipt로 ISSUE-001을 fixed 처리해 archive했다.
TASK-444도 독립 done-review PASS를 받아 완료했다.
TASK-445는 TASK-431 측정 범위와 TASK-436 기존 회수 승인을 현행화했다.

- ISSUE-037: local `master`와 `origin/master` 동기화가 확인되어 해결됨.
- ISSUE-014: 영수증 디렉터리 `unknown` 분류는 TASK-403으로 닫혀 아카이브됐다.
- ISSUE-005/TASK-422: 영수증 종단 상태 기록과 조회 보존이 상류에 이미 구현되어 독립 리뷰 후 done 처리했다.
- ISSUE-007/041: TASK-424가 DVA의 strict status dialect 선택, paired evidence, identity regressions, independent done-review PASS를 마쳐 fixed 처리 후 `tasks/_archive/issue/`에 보관했다.
- ISSUE-024: finalize 차단은 우회하지 않는다. TASK-430의 `ed1f4574`는 receipt 존재·tracking·digest 검증, missing/untracked/malformed 음성 회귀, preview/apply 일치, ISSUE-051 supersession을 구현했다. exact implementation CI와 upstream independent review PASS, `1e408857` master 통합 완료. TASK-391/393/394 및 TASK-410 DVA previews는 모두 `WOULD REMOVE`, clean worktree로 기록됐으며 DVA 독립 done-review PASS 후 fixed로 resolve했다.
- ISSUE-004/006: TASK-421/423은 task-manager-devbox W13/W14로 이관됐고 DVA 이슈는 제품 구현·채택 확인까지 열린다.
- ISSUE-043: TASK-438 fixture에서 provider가 통합한 뒤 Worktrunk 회수에 실패하면 후속 `run-status`가 inventory 불일치로 막히는 recovery gap을 기록했다. TASK-439는 source containment 확인, 잔여 path/ref 보존, 제한된 receipt에 대한 `run-status`·`run-list`의 `run-recover` 안내, idempotence와 `--no-fetch` 지원을 구현하고 독립 리뷰를 통과해 이슈를 해결·아카이브했다.
- ISSUE-032/TASK-435: source-branch 승격 차단은 `ce-agent-kit` master `71b1d169`에 포함된 `05b0b4ae` 수정으로 해결됐다.
- ISSUE-013/019/020/022/028/031/032는 상류 수정 확인 후 `tasks/_archive/issue/`에 보관했다.
- TASK-431의 source worktrunk pin `7273dd62`를 지정 설치기로 live에 반영했다. `/tmp` 호스트 scope에서 explicit wt 0.74.0과 Aqua golangci-lint 2.13.2가 선택·실행된다. DVA 프로젝트와 CI의 2.12.2는 제품 핀이다. 선언된 로컬 입력 10개는 추적 evidence와 제한된 해시 감사로 보존을 확인했고 generated shared drift는 사용자 선택대로 소스 정본을 유지한다. 독립 done-review 뒤 ISSUE-026을 fixed로 닫았다.
- TASK-432, TASK-433, TASK-420, TASK-438, TASK-439는 상류 통합·독립 리뷰를 마쳤고 현재 tasks/_archive/2026-09/에 있다. TASK-420은 legacy completion evidence 세 형상에서 fresh non-empty quality-review-evidence를 먼저 기록하고 canonical receipt를 validator까지 왕복 검증했다. TASK-438의 exact-CI commit 71b1d169와 TASK-439의 exact-CI commit 7eaf596a는 CE master/origin에 통합됐고 task branch/worktree 회수도 확인했다. ISSUE-008/042/043은 회귀 수정과 fixture·테스트 증거로 해결·아카이브됐다. ISSUE-001의 마지막 DVA-host TASK-312 migration도 TASK-444에서 독립 PASS와 첫 canonical receipt를 기록해 fixed/archive 처리했다. TASK-312는 archive validation에서 history로 skip되므로 저장 receipt는 별도 canonical-digest round trip으로 확인했다.
- TASK-424는 CE `96bb3674`의 strict CLI producer round trips와 `dba2348b` 통합을 바탕으로 DVA가 `card-dialect.strict-status`를 선택했다. paired DVA/CE zone·archive fixtures, 최종 전체 board strict validation 492/0, duplicate-ID 회귀 테스트, `make doc-check`가 통과했고 `checkCardStatus` 중복은 제거됐다. CE resolver가 만든 `issue/ + status: done + resolution` 예외는 old guard와 다르며 evidence에 기록했다. 독립 done-review PASS로 `done/`에 이동했다.
- TASK-430의 `ed1f4574`는 receipt ownership 경계를 보완하고 exact implementation CI 및 upstream independent review를 통과했다. review evidence commit `1e408857`는 master에 통합·push됐고 task branch/worktree 회수 완료. 네 DVA cleanup previews가 `WOULD REMOVE`로 통과해 tracked evidence에 저장됐고, fresh DVA independent done-review PASS 및 ISSUE-024 fixed resolve까지 끝났다.
- ISSUE-039: TASK-436의 stale worktree는 clean, branch HEAD는 4ce30f26, upstream은 없고 lifecycle은 ACTIVE다. 회수 승인은 이미 기록됐다. CE의 지원되는 discard 수명주기 동작이 없어 blocked다.
- ISSUE-044/045: 상류 통합 후 2026-09-27 1662bb68에서 해결 이슈 아카이브로 이동했다. 현재 active issue 목록에 남기지 않는다.
- TASK-420–439의 상태와 실행 의존성은 [[PLAN-011]]에서 유지한다. TASK-437·TASK-443·TASK-445는 PLAN-011 child가 아닌 보드 현행화 카드이고 TASK-444는 ISSUE-001 마감 카드라 PLAN-011 child 수에는 포함하지 않는다.
- ISSUE-004/006의 ce-agent-kit#1은 역사적 보고처다. `61ead57b`가 두 카드의 후속 owner를 taskchain-task-manager로 정해 task-manager-devbox W13/W14로 이관했다.
- ISSUE-030은 PLAN-011 밖의 ce-agent-kit 작업이었다. TASK-331 구현 `16af8443`은 세 번째 독립 리뷰 PASS와 exact CI PASS를 받았고, review-evidence 카드 커밋 `f3cfa169`까지 `master/origin`에 통합·push됐다. `run-finish`가 source push와 task worktree/branch 회수를 완료했다. 최종 tree CI run `ci-20260927-175252-77713` exit 0; source-built CE `v0.8.4-372-gf3cfa169`의 validate 출력은 16 valid / 0 invalid, `ce task gate --json`은 READY와 revision `f3cfa169`를, review-receipt JSON은 canonical digest와 같은 version/revision을 기록했다. legacy/mismatch/unknown advisory와 digest hard error는 exact CI regression tests 및 독립 리뷰에서 확인해 ISSUE-030을 fixed/archive했다.
- PLAN-006~009는 `tasks/_archive/plan/`에 보관했고 `make doc-check`가 링크·계획 진행률을 확인했다.
  `tasks/` 아래 네 `.ce` 텔레메트리 디렉터리의 파일 12개(30,336바이트)는
  `/Users/archmagece/backups/dva/task-ce-telemetry-20260927`로 이동해 SHA-256을 대조했다.
  [백업 목록과 해시](done/evidence/TASK-446/telemetry-backup-manifest.json)를 추적한다.
  루트 `.ce/task-runtime.yaml`은 ACTIVE 런타임 선언으로 남아 있다.

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
2026-09-27까지 이 완료 배치들은 _archive/2026-09/로 이동했다. 현재 tasks/done/에는
TASK-431, TASK-442, TASK-443, TASK-444, TASK-445, TASK-446 여섯 장이 있고 durable review evidence는 tasks/done/evidence/에 보관한다.

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
