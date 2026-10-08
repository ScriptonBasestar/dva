# DVA Task Management Board

## 현재 상태 (2026-10-07)

v0.3.0 공개와 postflight는 완료됐고 PLAN-010(8/8)과 PLAN-011(19/19)은
`tasks/_archive/plan/`에 보관됐다. 열린 계획은 없다. 과거 판정은 보관된 카드와 Git 이력이 소유한다.
2026-10-03 [[TASK-482]]가 done 카드 30장(TASK-449~481)을 재검증해 전부 `tasks/_archive/2026-10/`로 보관했다.
2026-10-05 [[TASK-485]]가 TASK-482·483·484를 재검증해 `tasks/_archive/2026-10/`로 보관했다(아래 Batch 10).
`tasks/done/`에는 TASK-485, 운영 계약을 완료한 TASK-486, 승인 ceiling을 반영한 TASK-489, 수락된 범위 적용 TASK-492, 제품 source 보정 TASK-493, 2026-10-05 사람 실측으로 닫힌 TASK-459, durable review evidence가 남는다 —
TASK-485는 다음 정리 주기에 독립 재검증을 받는다.
DECISION-002는 2026-10-03에 Accepted다. 정본 upstream만 택했고 interim B는 쓰지 않는다.
인계는 [생성물 크기 규칙 보고](../docs/70-generated-artifact-upstream-report.md)다.
정본 구현은 ce-agent-kit `1270e1dc47bc7f3a2421de2074b92f619e4298a7`이다. 설치본은 자동으로 바뀌지 않는다.
2026-10-05 `ce version`은 0.8.4, commit `62db34ea9291c0cab9cc03222136e8a32fcaed16`, dirty false다.
이 커밋은 `1270e1dc47bc7f3a2421de2074b92f619e4298a7`와 `3656558b2bbd604224edf9c7de69454e96c379a8`의 자손이다.
플러그인 캐시 core 0.6.38은 이미 있다. 재설치하지 않았다. TASK-484가 저장한 소스 검사는 다시 실행하지 않았다.
이 작업의 시작 기준선(START)은 `23047d07`이다. 통합 뒤의 master를 이 해시로 단정하지 않는다. [[TASK-485]]가 TASK-482·483·484를 보관한 커밋이다.

미해결 이슈는 없다. 사용자가 승인한 로컬 설치 TASK-499는 설치 검증과 독립 Grok 4.7 완료 리뷰 PASS를 마치고 일회성 완료 기록으로 보관했다. 실행 대기 카드는 없다. TASK-498은 빈 active issue 코퍼스를 허용하는 테스트 보정을 독립 리뷰·전체 CI 후 master 7003a26e에 통합·push·reclaim했다. TASK-496의 실제 이슈 0개 보드에서도 회귀 테스트와 전체 문서·태스크 게이트가 통과했다. TASK-495는 빈 토큰 출력 판정을 수정하고 독립 Grok 리뷰·전체 CI 후 master 71b00be9에 통합·push·reclaim, 실제 CE DONE이다. TASK-496은 공개 TaskChain v0.1.0 darwin/arm64 자산·custom verified-release attestation·기본 PATH 해시·compiled pin 및 실제 CE start 1회, 해시 불일치 음성 queue/CE 0회, TASK-495 finish DONE을 검증했다. README의 이전 설명 때문에 최종 리뷰 attempt 1은 FAIL이었고, 사용자가 해당 문단만 승인해 보정한 뒤 attempt 2는 PASS다. 독립 세션이 TASK-496을 done으로 옮겼고 ISSUE-453/490/497은 fixed로 `_archive/issue/`에 보관했다. 증거는 `tasks/done/evidence/TASK-496/`다. TASK-496 호스트 증명 당시 전역 DVA는 교체하지 않았다. 이후 TASK-499에서 사용자가 승인한 master 2d08fcde 로컬 빌드를 기본 PATH 두 설치 경로에 적용했다. 과거 설치 범위 기록은 그대로다.

2026-10-05 TASK-491 옵션 1의 읽기 전용 범위로 ISSUE-004·006을 해결해 `_archive/issue/`에 보관했다. clean dd3ec0a 바이너리를 명시 선택한 실제 큐는 사람 TASK-459 한 장, agent 후보 0, verdict `human_required`다. 독립 Grok 4.7 PASS는 `tasks/done/evidence/TASK-492/live-acceptance-review.json`이다. 공개 pin 승인은 ISSUE-453과 별개다. 해당 독립 검증 당시 TASK-493은 source 통합 중인 doing이었다. 현재는 통합 완료됐으며 아래 최종 검증이 준비 집합을 다시 확인한다. 사람 terminal과 run-all은 이 범위가 아니다.

정본 소스로 닫아 `_archive/issue/`에 둔 이슈. 설치본은 자동으로 바뀌지 않는다 — 2026-10-05 설치본 `62db34ea`는 두 수정을 포함한다.

- ISSUE-454: 정본 ce-agent-kit `9b0b0305a553aec3faceeefd12bd6db6fd5a312d`(TASK-379)가 timeout 뒤 프로세스 그룹의 자손을 끝낸다. DVA 작성 시점 검사([[TASK-460]])와 설치본의 재귀 가드 관찰은 그대로다. 기록은 [[TASK-484]].
- ISSUE-461: 정본 `1f3f9a74be0cbe9cbb9aa8de943331eb05bdac2e`(TASK-378)가 같은 소유자의 활성 실행이 하나일 때 bare `ce task run-finish`를 고른다. 없거나 여럿이면 거부한다. 기록은 [[TASK-484]].

2026-10-05 이 저장소 안에서 닫은 이슈:

- [ISSUE-488](_archive/issue/488-measure-expired-tunnel-token-and-live-doctor.md): 사람이 고른 대상(`scripton-tonk-01-k8s.scripton.net`)으로 개발자 Mac에서 실측했다. 만료 토큰에서 `cloudflared access token --app`은 exit 0, 빈 stdout이고 JWT `-token` 파일을 지운다. 기록은 docs/68 §7이다. 종료 코드 단독 판정 결함은 [TASK-495](_archive/done/495-treat-empty-access-token-stdout-as-unauthenticated.md)다.

[TASK-495](_archive/done/495-treat-empty-access-token-stdout-as-unauthenticated.md)는 종료 코드 0과 stdout 바이트 수 > 0을 함께 요구하며 토큰 내용은 버린다. 독립 Grok 4.7 PASS와 CI 증거는 `tasks/done/evidence/TASK-495/`다. [TASK-496](_archive/done/496-activate-dd3ec0a-darwin-arm64-pin.md)은 공개 서명·설치·pin·실제 CE start/finish 검증을 완료했고 보호 문서 보정과 최종 독립 리뷰 PASS를 완료했다. ISSUE-497은 fixed로 보관했다. [TASK-459](_archive/done/459-implement-remote-access-tunnel.md)는 2026-10-05 ISSUE-488 실측으로 사람 기준 두 개를 채워 done이 됐다. 제품 native decision 호환 보정 [TASK-493](_archive/done/493-support-native-decision-queue-kind.md)은 독립 리뷰·전체 검사 후 제품 master에 `dd3ec0a`로 통합·push했고 task worktree와 local/remote 브랜치를 회수했다. [TASK-492](_archive/done/492-apply-approved-readonly-queue-acceptance.md)는 수락된 범위·문서 적용을 독립 Grok 4.7 재리뷰 PASS 후 완료했다. 004·006은 명시 읽기 전용 검증과 독립 PASS로 해결했다. TASK-493 source 통합도 완료됐다. [TASK-489](_archive/done/489-apply-approved-changelog-ceiling.md)는 실제 ceiling 트리거 반영과 후속 분석을 독립 Grok 4.7 재리뷰 PASS 후 완료했다. [TASK-486](_archive/done/486-record-session-followup-contract.md)은 독립 Grok 4.7 리뷰 PASS 후 완료됐고, 기준 2·5·6 바인딩은 TASK-489가 사실 정정한다. 독립 재검증 PASS 영수증은 `tasks/done/evidence/TASK-489/independent-review-attempt-2.json`이다. [TASK-487](decision/487-changelog-half-ceiling-trigger.md)은 2026-10-05 사용자가 옵션 2(실제 ceiling)를 골라 Accepted다. 측정은 87296 bytes와 물리 줄 1000 아래라 v0.2.x 아카이브는 하지 않는다.

[TASK-491](decision/491-queue-acceptance-scope-and-historical-evidence.md)은 2026-10-05 사용자가 옵션 1을 수락해 Accepted다. 004·006의 현재 범위는 읽기 전용 분류와 종료 판정이다. terminal/rollback과 공개 pin 승인은 이 수락에 포함되지 않는다.

## 자율 실행 계약

이 절이 운영 계약이다. 제품 루프 코드는 만들지 않는다.

- policy:model-grok-4.7 — 기본, strong, 독립 리뷰는 Grok 4.7이다. 호출은 grok(기본) 또는 grok --model grok-4.7 이다.
- policy:build-fast-cheap-standard — build-fast는 cheap와 standard만 실행한다. 호출은 grok --model grok-4.7-build-fast 이다.
- policy:runnable-all-default — 기본은 실행 가능한 카드를 모두 처리한다.
- policy:max-n — 상한이 있으면 max N이다.
- policy:dependency-order — 의존 순서대로 진행한다.
- policy:parallel-ownership-isolated — 독립 병렬은 소유 파일이 겹치지 않을 때만 한다.
- policy:one-repo-explicit-files — 카드 하나는 저장소 하나, 명시 파일, 결정적 Steps다.
- policy:human-excluded — needs-human, execution-mode external, tasks/decision의 미결은 자동 실행 집합에서 뺀다. 사람이 대상과 인가 맥락을 준 뒤 그 카드 Steps의 읽기 전용 확인은 실행할 수 있다.
- policy:human-grade-encoding — 사람 작업의 표기는 human-grade: human 이다. CE exec-tier 스키마는 cheap, standard, strong만 받으므로 exec-tier는 비운다. exec-tier: human 은 쓰지 않는다.
- policy:no-self-review — 작성 세션이 자기 카드를 통과시키지 않는다.
- policy:review-separate-session — 독립 리뷰는 같은 Grok 4.7의 다른 세션이다. 호출은 grok(기본) 또는 grok --model grok-4.7 이다.
- policy:fail-max-3 — FAIL이면 Attempts에 다른 접근을 적는다. 최대 3회.
- policy:blocked-push-only-continue — 3회 FAIL이면 이슈 제목 needs stronger review, blocked, 그 카드의 브랜치 push만 하고 다음 카드로 간다.
- policy:commit-immediate-push — 작업 커밋 직후 구현 세션이 그 브랜치를 Git으로 push한다.
- policy:source-branch-master — DVA의 설정된 소스 브랜치는 master다. develop이라고 가정하지 않는다.
- policy:run-finish-configured-branch — devbox/source의 PASS 통합은 설정된 브랜치로 ce task run-finish 다. ce가 처리한다.
- policy:branch-integrate-configured-source — subrepo는 저장소가 선언한 branch-integrate로 설정된 소스에 통합한다.
- policy:stop-own-card-conflict — 그 카드의 통합 충돌이나 관련 실패면 그 카드만 blocked로 두고 다음으로 간다.
- policy:preexisting-nonworsening — 기존 실패는 실패 자체와 악화되지 않은 증거를 함께 남긴다. 그 기록만으로 멈추지 않는다.
- policy:no-intercard-questions — 카드 사이에 질문하지 않는다.
- policy:integrate-after-review-pass — 자동 통합은 독립 리뷰 PASS 다음에만 한다. 작성자 PASS는 통합이 아니다. PASS 자동 통합은 설정된 소스 push와 worktree·브랜치 cleanup을 포함한다. ce가 처리한다.
- policy:final-report — 끝에 통합 해시, blocked 이유, 이슈 ID, 사람 항목, 의존 항목을 보고한다.
- policy:no-second-board-grader — 두 번째 보드 채점기를 만들지 않는다. 판정은 선언된 ce task gate 다.

보드 현행화는 `tasks/`만이 아니라 `git worktree list`와 `ce task run-list`도 함께 확인한다.
완료 카드는 현행화 태스크 끝에 `tasks/_archive/YYYY-MM/`로 보관한다(`done-finalize`는 카드를 삭제해 plan children을 깨뜨린다).

이 문서는 DVA 프로젝트의 태스크 및 보드 구조, 아카이브 상태를 안내하는 정본 인덱스입니다.

## 보드 구조

- `tasks/todo/`: 현재 대기 중인 실행 가능한 작업 카드
- `tasks/done/`: 구현 및 마감 리뷰가 완료되어 아카이빙을 대기 중인 카드
- `tasks/plan/`: 다중 태스크 묶음을 관리하는 상위 계획 카드
- `tasks/issue/`: 발견된 버그, 구조 결함, 상류 의존성 이슈 카드
- `tasks/decision/`: 미결 결정 카드와 인덱스. 수락된 정본은 `decisions/`다. CE의 내장 decision kind 디렉터리이며 별도 zones 선언 없이 검증한다.
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
전부 재검증해 `_archive/2026-10/`로 보관했다(아래 Batch 9). 2026-10-05 [[TASK-485]]가 남은
TASK-482~484를 재검증해 같은 곳으로 보관했다(아래 Batch 10).

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
| Batch 10 | TASK-482 ~ 484 | 3 | 완료 (3 아카이브, todo 환류 0) |
| Batch 11 | TASK-459, 485, 486, 489, 492–496, 498 | 10 | 완료 (10 아카이브, todo 환류 0) |

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

### Batch 10 상세 (2026-10-05)

3장 전부 아카이브. 체크된 기계 바인딩 18개(482: 5, 483: 7, 484: 6) 재실행 exit 0.
재검증은 세 카드의 작성·리뷰 세션과 다른 세션([[TASK-485]])이 했다.

| 카드 | 판정 | 근거 |
|:---|:---|:---|
| TASK-482 | 아카이브 | 기준 1–5 exit 0. 영수증 `evidence/TASK-482`·`TASK-484`의 independent-review.json 모두 PASS |
| TASK-483 | 아카이브 | 기준 1–7 exit 0. DECISION-002 `status: Accepted`. 영수증 outcome PASS, 작성·리뷰 세션 분리 |
| TASK-484 | 아카이브 | 기준 1–6 exit 0. 핵심 주장인 ISSUE-454 상류 기준(정본 소스 `go test` 두 패키지)을 재실행해 ok. 설치본 `62db34ea`가 `9b0b0305`·`1f3f9a74`·`1270e1dc`를 조상으로 포함 |

### Batch 11 상세 (2026-10-08)

10장 전부 아카이브. 체크된 기계 바인딩을 기준 문장 그대로 재실행해 exit 0.
TASK-459의 사람 기준은 docs/68 §7과 ISSUE-488에 남은 실측으로 충족했다.
`make test` exit 0. 당시 `make lint`는 미사용 `errors.As` 네 곳을 gopls가 거부했고,
그 호출을 `errors.AsType`으로 바꾼 뒤 lint가 0 issues다. 환류 0.

| 카드 | 판정 | 근거 |
|:---|:---|:---|
| TASK-459 | 아카이브 | 터널 테스트 5개·doc-check exit 0. 사람 실측은 docs/68 §7과 ISSUE-488에 남아 있다 |
| TASK-485 | 아카이브 | 482–484 보관 상태·링크·Batch 10 행 exit 0 |
| TASK-486 | 아카이브 | Accepted 결정, ceiling 마커, strict-status 바인딩 9개 exit 0 |
| TASK-489 | 아카이브 | 옵션 2 Accepted와 CHANGELOG 상한 바인딩 9개 exit 0 |
| TASK-492 | 아카이브 | 읽기 전용 004·006 기준과 TASK-491 Accepted 바인딩 5개 exit 0 |
| TASK-493 | 아카이브 | 제품 native decision 회귀 파일과 payload-checks 통합 기록 exit 0 |
| TASK-494 | 아카이브 | local-only 이슈 코퍼스 테스트와 리뷰 영수증 PASS |
| TASK-495 | 아카이브 | 빈 stdout 인증·doctor 테스트와 docs/68 §3.1 exit 0 |
| TASK-496 | 아카이브 | 공개 digest·PATH 바이너리·attestation·task-495 DONE exit 0 |
| TASK-498 | 아카이브 | 빈 active issue 코퍼스 테스트와 리뷰 영수증 PASS |

## 2026-10-05 읽기 전용 선택과 당시 증거

2026-10-05 당시 기본 설치본은 ISSUE-490의 legacy 바이너리였다. 설치·pin 전환을 승인한 것은
아니다. 내부 검증 바이너리는 확인된 제품 checkout에서 명시적으로 선택한다.
완료 기준은 변수가 없으면 아래 검증된 사용자 workspace 경로를 기본값으로 쓴다. 다른 장비에서는 확인한 checkout 경로로 변수만 바꾸고 존재 가드를 지킨다.

읽기 전용 명시 선택은 기본 PATH 교체가 아니다. dd3ec0a 내부 후보의 darwin/arm64 파일은
`build/native-decision-dd3ec0a/internal-candidate/taskchain-task-manager-darwin-arm64`이고
SHA-256은 `c01ce7aad3638ddc87acda2c9bd52e6d72a52f2a40d932d7e238f246f91e939e`다.
파일명에 플랫폼 접미사가 있으므로 그 디렉터리를 PATH에 넣는 것만으로는
`taskchain-task-manager`가 선택되지 않는다. `build/taskchain-task-manager`의 존재도
이 해시의 증거가 아니다. 아래 블록은 이름이 맞는 실행 파일이 있을 때의 선택이며,
해시가 위 값과 같음을 확인하기 전에는 이 후보로 부르지 않는다.

```bash
export TASKCHAIN_PRODUCT_REPO="$HOME/mydevbox/task-manager-devbox/taskchain-task-manager"
test -d "$TASKCHAIN_PRODUCT_REPO" && test -x "$TASKCHAIN_PRODUCT_REPO/build/taskchain-task-manager"
PATH="$TASKCHAIN_PRODUCT_REPO/build:$PATH" dva task-queue
PATH="$TASKCHAIN_PRODUCT_REPO/build:$PATH" dva task-queue-verdict
```

위 제품 경로는 사용자가 명시한 devbox와 그 workspace 선언으로 확인한 경로다.
폴더 이름이나 catalog alias로 생산자를 추정한 경로가 아니다. 제품 payload 검사는
그 저장소의 make check/make lint가 소유하고, 이 보드 판정은 ce task gate가 소유한다.
새 내부 산출물의 출처·3개 플랫폼 hash·clean checkout 재빌드 증거는
`tasks/done/evidence/TASK-493/internal-candidate-provenance.json`이다.
이 산출물은 미공개·미서명·미승인이고 기존 고정 후보를 덮어쓰지 않았다.
공개 승인 결정은 기존 devbox blocked 카드 007이 소유하므로 중복 결정 카드를
만들지 않는다. TASK-491은 이미 Accepted다.

source 통합·회수 후 최종 조회도 exit 0이다. 준비 큐는 TASK-459 한 장, agent 후보 0,
verdict는 human_required다. 카드 Markdown bytes는 두 읽기 전용 호출 전후 같았다.
증거는 `tasks/done/evidence/TASK-492/live-readonly-final.json`이다.

CI가 발견한 후속 [TASK-494](_archive/done/494-allow-local-only-issue-corpus.md)는 실제 보드에
상류 소유 이슈가 반드시 있어야 한다는 테스트 가정을 보정했다. 합성 fixture로
양성·음성 분류와 정상 local-only 보드를 검증했고 독립 Grok 4.7 PASS를 받았다.

2026-10-06 초기 공개 단계의 역사 기록: 제품 signer helper 8b42ba3 통합·push·reclaim 및 DD v0.1.0
tag push 후 draft 생성이 HTTP 403으로 거부됐다. 외부 권한 조치는 private devbox
ISSUE-057이 추적한다. TASK-496은 blocked이며 ISSUE-453/490은 미완료다. 공개 서명
검증 전에는 설치·pin·host start를 진행하지 않는다. TASK-495는 실제 positive 검증의
자동 후보로 남아 있다.

## 2026-10-06 공개 및 호스트 검증

v0.1.0 signer run 37472525561은 성공했고 공개 릴리스는 https://github.com/Gizzahub/taskchain-task-manager/releases/tag/v0.1.0 이다. public artifact source는 dd3ec0a이고 signer workflow source는 8dade71이다. 이것은 custom verified-release attestation이며 CI 빌드 주장이 아니다. 초기 HTTP 403과 draft 조회 HTTP 404는 해결됐고 private devbox TASK-056/059도 통합·회수됐다. 기존 내부 후보의 historical 승인 플래그와 TASK-493 봉인은 그대로다.

## 2026-10-07 최종 완료 검증

TASK-496 최종 독립 Grok 4.7 리뷰 01a111a0-d66d-7041-a9c4-1bf1d1b764fd attempt 2 PASS. CI 706974f34106187b71a04e5b38a82499 commit succeeded (1m12.992619041s); 검증 프로세스에서만 KUBECONFIG를 제외했다. 이슈 해결 기준 여덟 개 exit 0. historical TASK-493 봉인과 내부 후보 승인 플래그는 그대로다. 기본 PATH TaskChain만 승인 산출물로 교체했으며 Linux pin, CE writer 전면 전환, 전역 DVA 설치는 이 완료 범위에 포함하지 않는다.

빈 active issue 코퍼스의 최종 검증은 TASK-496 `final-empty-board-gates.json`이다.
TASK-498은 생산 판정을 변경하지 않았으며, 실제 inventory와 읽은 이슈 수를 정확히
대조하고 기존 소유권·상류 보고 누락 실패 fixture를 유지한다.

## 로컬 DVA 적용 (TASK-499)

기본 DVA는 버전 0.3.0, commit 2d08fcde 로컬 빌드다. 공식 dva install의
원자적 교체와 기본 PATH 버전·해시, 읽기 전용 queue/verdict를 검증했다.
이것은 새 공개 릴리스가 아니며 TASK-496의 역사적 host 실행 증거를 다시 만들지 않았다.
백업과 설치 증거는 tasks/done/evidence/TASK-499에 있다.
