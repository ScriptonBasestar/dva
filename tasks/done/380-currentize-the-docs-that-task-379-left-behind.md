---
id: TASK-380
title: "Currentize the documents TASK-379 left behind"
type: docs
priority: P2
effort: S
exec-tier: standard
status: done
created: 2026-09-13
source: "TASK-379 완료 직후의 보드/문서 상태 재점검. 세 건 모두 하나의 원인 — 하네스와 카드는 움직였는데 그것을 읽는 문서가 따라오지 않았다"
depends-on: []
completion-summary: "TASK-379 뒤에 낡은 문서 세 곳을 현행화했다 — primeno1 리포트의 검증 한계(해소된 TASK-311/312 → 실제 blocker 둘), PLAN-009의 scope/Goal(여섯 장 → 일곱 장), TASK-328의 잔여 blocker. 상한 초과로 2026-09-05 이력을 primeno1-migration-log.md로 분리했고 그 systemic 조건은 ISSUE-009로 분리 기록했다"
verification-status: verified
verification-evidence: "완료 기준 8개의 verify 바인딩 전부 exit 0. make doc-check 0, make lint 0, bash -n + shellcheck 0, bash tools/dogfoodrun/dogfood-run.sh --plan primeno1 rc 0. ce task validate --all은 ISSUE-001 기존 5건(TASK-344/371/376/377/378)만 — 이 카드가 새 부채를 더하지 않는다"
quality-review: pass
quality-reviewed-at: 2026-09-13
quality-review-evidence: "독립 리뷰 세션 review-380(Claude Opus 5, 저자 아님) 3라운드. 1라운드 conditional로 F1-F5, 이어서 F6까지 여섯 건 지적 — 전부 1차 출처에서 재도출해 수정. 2라운드 pass + 잔여 지적 R1/R2(non-blocking). 3라운드는 d3bae59가 2라운드 도중에 올라와 판정이 가리키는 커밋이 모호해진 것을 없애려고 리뷰어가 자발적으로 돌린 확인 회차이며 pass·verdict-final. 상세는 receipt의 timing-note에 있다"
quality-review-receipt: tmp/task-management/direct/queue-run/task-380-review-receipt.json
---

## Summary

[[TASK-379]]가 하네스를 plan `dev`로 재조준하고 닫히면서, 그 사실을 필요로 하는
문서 세 곳이 낡았다. 셋 다 순수 문서이고 파일이 겹치지 않으며 원인이 같다.

**한 장으로 묶는 이유**: 세 장으로 쪼개면 카드 오버헤드가 내용보다 크고, `blocks:`
간선이 늘어 [[ISSUE-001]]의 receipt 부채를 세 배로 만든다. 이 카드는 `blocks:`를
선언하지 않아 완료해도 게이트 실패를 늘리지 않는다.

## D-1. `docs/dogfood/primeno1.md:92`의 검증 한계가 해소된 blocker를 가리킨다

현재 문장: "검증 한계: `dva --dry-run up dev`는 TASK-312로 health 대기에 걸려
멈춤(kill 필요). 실기동 검증은 TASK-311/312 이후."

TASK-312는 `tasks/done/`, TASK-311은 `tasks/_archive/done/`에 있다. dry-run health
대기는 `Orchestrator.Up`이 `opts.DryRun`을 보게 고쳐져 해소됐다. 실제로 남은 blocker는
[[TASK-328]]의 사람 회차와 order 10 sigdock 게이트 선행조건이다. 이 문서에는 TASK-379의
재조준도 sigdock 선행조건도 없다.

## D-2. PLAN-009의 산문이 여섯 장, frontmatter가 일곱 장이다

`scope:`와 `## Goal`은 "여섯 장"이라 쓰고 TASK-371/344/350/343/354/338을 나열하는데,
`children:`은 TASK-377을 포함한 일곱 장이고 `total-tasks: 7`이다. `tools/planprogress`는
`total-tasks`/`completed-tasks`/`## Children`만 세므로 산문 드리프트를 보는 게이트가
없다. 이 검사기 자체는 [[TASK-381]]이 소유한다 — 이 카드는 009의 산문만 고친다.

## D-3. TASK-328이 sigdock 선행조건을 모른다

실기동 회차를 잡을 사람이 읽는 카드는 328이다. 그런데 "이 워크스테이션에서 order 10
게이트 조건 둘이 이미 위반돼 있다"는 사실은 `tools/dogfoodrun/dogfood-run.sh`의
`target_notes()`와 닫힌 카드 379에만 있다. 328 본문은 아직 blocker를 "하네스 재조준과
사람 회차"로 적어 두었는데 379가 닫혔으므로 지금은 "사람 회차와 선행조건 정리"다.

## D-4. ISSUE-001 §Summary 2번이 부분적으로 낡았다

"controller receipt는 CE의 정본 digest와 맞지 않아 DVA 안에서 만들 수 없다"고
적혀 있으나, TASK-379를 닫으며 **validator가 에러 메시지에 출력하는 정본 digest를
그대로 `reviewed-card-sha256`에 넣으면 DVA 안에서 통과하는 receipt를 만들 수 있다는
것이 실증됐다** — 카드 379가 지금 그 형태로 validate를 통과한다. 남은 진짜 장애는
3번(`tmp/`가 durable하지 않다)이다. 이 관측을 이슈에 기록한다. **이슈를 닫지 않는다** —
외부 소유와 P0 판정은 그대로다.

## D-5. 상한에 걸려 분리했다 (작업 중 발견)

`docs/dogfood/primeno1.md`는 작업 시작 시점에 10028바이트였다. `tools/doccheck`의
상한은 10240바이트이므로 여유가 **212바이트**였고, D-1의 갱신을 어떤 형태로 넣어도
넘는다. `tools/doccheck/policy.go`는 per-file 예외를 명시적으로 거부한다 — "a document
that cannot meet the limits is split, not exempted".

그래서 2026-09-05 회차별 적용 이력을 `docs/dogfood/primeno1-migration-log.md`로 분리하고
본 문서에는 현황·현재 상태·포인터만 남겼다. 옮긴 내용은 편집하지 않았다.

이것은 이 카드만의 문제가 아니다 — 회차 기록이 누적되는 dogfood 리포트 여러 장이 상한에
근접해 있고, [[TASK-328]]은 실기동 출력을 primeno1·familybook·flow-taskchain 셋에
붙여야 한다. 그 구조적 조건은 [[ISSUE-009]]가 소유한다.

## 리뷰 대응 (2026-09-13, review-380 — conditional)

독립 리뷰어가 게이트 셋과 verify 바인딩 일곱 개를 직접 재실행하고(전부 exit 0,
`validate --all`은 baseline 5건 그대로), 여섯 개의 사실 주장을 재유도했다. 넷 확인,
하나 부분 확인, 하나 반증. 다섯 건의 지적을 전부 재측정해 수용했다.

- **F1 (medium) — ISSUE-009의 측정이 틀렸다.** 초판은 `workflows/*.md`라는 **비재귀
  glob**으로 세어 `workflows/dva-dogfood/` 아래를 통째로 놓쳤다. 놓친 파일이 하필
  전체에서 가장 빠듯한 `ref-artifacts.md`(10234바이트, 여유 **6바이트**)다. "20장"도
  틀렸다 — HEAD 기준 21장. 재귀 `find`로 다시 세고 측정 명령을 이슈에 박았다.
  `tools/doccheck/policy.go:31`이 `workflows/`를 대상에 넣는데 내 측정이 그러지
  않았다 — 정책을 인용하면서 정책과 다른 범위를 잰 것이다.
- **F2 (low) — "편집 없이 옮겼다"가 거짓이었다.** 옮긴 67줄 중 정확히 한 줄, 하필 이
  카드가 다루는 `검증 한계` 줄을 다시 날짜 매기고 `TASK-312로` 귀속을 지웠다.
  원문을 복원하고 괄호 주석만 덧붙이는 형태로 바꿨다. 이력 문서의 규약도 명시했다 —
  원문은 고치지 않고 주석만 단다.
- **F3 (low) — 대명사가 가리킬 것을 잃었다.** "그 검증 한계" / "위 문장이 지목한"이
  분리 후 선행사 없는 문장이 됐다. 절 제목과 첫 문장에 원문을 인용해 복원했다.
- **F4 (low) — 선행조건 목록이 불완전했다. 그리고 세 곳에 복제돼 있었다.**
  게이트가 실제로 요구하는 것 중 셋이 카드에도 `target_notes()`에도 없었다:
  `require_loopback_provider_host`(`sigdock.localhost`가 loopback으로만 해석), contract
  스크립트의 실행 비트, `dva`/`docker`/`curl` 바이너리(`lsof`만 적혀 있었다). 또
  "인접 체크아웃 존재"는 불완전하다 — 게이트는 `$SIGDOCK_DEVBOX_DIR/dva.yml`도 파일로
  본다(이 문장 자체가 반대로 과했고, 아래 R2에서 다시 고쳤다). 셋 다 이 워크스테이션에서는
  충족돼 있어 **위반 건수는 여전히 둘**이지만, 목록이 완전하다고 읽히면 안 된다. 카드와
  하네스 양쪽에 추가하고 게이트의 실제 검사 순서도 적었다.
- **F6 (informational) — 기준 하나가 절반만 기계 검증됐다.** 기준 2가 "재조준 **과**
  선행조건"을 요구하는데 바인딩은 `SIGDOCK_CLIENTS_FILE` 하나만 grep했다 — 재조준 쪽이
  사라져도 통과한다. 리뷰어가 손으로 확인해 실제로는 충족돼 있었지만, 통과가
  기계 검증으로 읽히면 안 된다. 기준을 둘로 쪼개 각각 바인딩을 줬다.
- **F5 (low) — 5→6 관측이 재현되지 않는다.** 맞다. 리뷰어를 붙이기 전에 내가 receipt를
  워크트리로 복사해 원인을 제거했기 때문이다. 관측을 지우는 대신 순서를 명시한
  재현 절차를 ISSUE-001에 적었다(새 워크트리 → 6 → 복사 → 5). 원인을 제거하면 사라지는
  관측이라는 사실 자체가 이 이슈의 논점이다.

### 2차 리뷰의 잔여 지적 둘 (pass, non-blocking — 같은 커밋에서 접었다)

- **R1 (low) — ISSUE-009의 수사 한 문장이 자기 숫자와 맞지 않았다.** "앞 네 장에는 한 자도
  못 넣는다"는 틀렸다. `tools/doccheck/check.go:117`은 `nbytes > maxDocBytes`이므로 정확히
  10240바이트는 통과하고, 여유 6/7/15/25바이트는 한글 2/2/5/8자를 받는다. 실측치는 전부
  옳았고 그 위에 얹은 문장만 과했다 — 측정 정밀도를 주제로 하는 이슈에서 특히 나쁘다.
  숫자를 그대로 적는 형태로 고쳤다.
- **R2 (low) — F4의 교정이 반대로 과했다.** 게이트는 디렉터리와 `dva.yml`을 **둘 다** 본다:
  `sigdock-local-up.sh:258`이 `SIGDOCK_DEVBOX_DIR`을 cd 가능한 디렉터리로 요구하고,
  `:400`이 그 안의 `dva.yml`을 파일로 요구한다. 원래 문장은 틀린 것이 아니라 불완전했다.
  결과가 둘 더 있다 — (1) `:258`이 바이너리 검사보다 훨씬 앞이라 검사 순서의 진짜 머리가
  빠져 있었고, 위반 둘은 6번·8/9번이 아니라 **7번·9/10번**이며 앞 **여섯** 관문을 통과한
  뒤에 죽는다. (2) 두 검사가 **같은 실패 메시지**(`adjacent SigDock devbox not found`)를
  쓰고 뒤쪽만 경로를 덧붙인다 — 로그 판독에서 갈라야 하는 자리라 목록에 적었다.
  카드 328과 `target_notes()` 양쪽에 반영했다.

세 곳 복제(카드 · `target_notes()` · 리포트)는 F4에서 비용이 실증됐다 — 빠진 항목이
세 번 빠진다. 리포트 쪽은 이번에 포인터로 바꿔 두 곳으로 줄였고, 남은 두 곳의 단일화는
이 카드 범위 밖이다.

## Completion Criteria

- [x] `docs/dogfood/primeno1.md`의 검증 한계가 해소된 TASK-311/312 대신 실제 남은 blocker를 가리킨다 | verify: `/usr/bin/grep -q 'TASK-328' docs/dogfood/primeno1.md`
- [x] 같은 문서가 order 10 sigdock 선행조건을 기록한다 | verify: `/usr/bin/grep -q 'SIGDOCK_CLIENTS_FILE' docs/dogfood/primeno1.md`
- [x] 같은 문서가 TASK-379의 plan `dev` 재조준을 기록한다 | verify: `/usr/bin/grep -q '하네스 재조준(TASK-379)' docs/dogfood/primeno1.md`
- [x] PLAN-009의 `scope:`와 `## Goal`이 `children:` 일곱 장과 일치한다 | verify: `/usr/bin/grep -rq --include='009-work-the-doccheck-defect-bundle-in-dependency-order.md' 'TASK-377' tasks`
- [x] TASK-328 본문의 잔여 blocker가 사람 회차와 sigdock 선행조건 둘로 갱신된다 | verify: `/usr/bin/grep -rq --include='328-run-a-live-dogfood-verification-round-for-native-entries-and-composition-plans.md' 'SIGDOCK_CLIENTS_FILE' tasks`
- [x] ISSUE-001에 정본 digest 실증 관측이 기록되고 이슈는 열린 채 P0로 남는다 | verify: `/usr/bin/grep -rq --include='001-task-runtime-cannot-review-legacy-done-cards-without-verification-evidence.md' 'TASK-379' tasks`
- [x] `docs/dogfood/primeno1.md`가 10 KiB 상한 아래로 내려간다 | verify: `test 10240 -ge "$(/usr/bin/wc -c < docs/dogfood/primeno1.md)"`
- [x] 문서 게이트가 통과한다 | verify: `make doc-check` (regression-guard)
