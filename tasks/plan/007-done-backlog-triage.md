---
id: PLAN-007
title: "Retire the unreviewed done backlog and move quality review to card close"
type: plan
scope: "tasks/done/ 58장의 일회성 트리아지, done-disposition 루프에서 발견된 소유자 없는 결함 3건, 닫기 시점 독립 리뷰 전환의 외부 의존, 그리고 그 트리아지 이후 다시 채워진 대기열(2026-09-09 추가)"
progress: 91
total-tasks: 12
completed-tasks: 11
children: [TASK-325, TASK-326, TASK-327, TASK-330, TASK-332, TASK-333, TASK-334, TASK-335, TASK-337, TASK-342, TASK-367, TASK-368]
target-date: "2026-09-30"
created: 2026-09-05
---

## Goal

`tasks/done/`가 리뷰 대기열로 쌓이지 않게 한다. 지금은 카드를 닫는 세션이 `done-review`를 돌리지 않아
done 카드 58장 전부에 `quality-review`가 없고, 아카이브 직전의 done-disposition 루프 하나가 리뷰를 전담한다.
리뷰가 변경 시점에서 분리돼 직렬 병목이 됐고, 유입(여러 세션의 병렬 닫기)이 배출(한 프롬프트에 한 장)보다 빠르다.

완료 상태:

1. 58장이 아래 티어대로 처리되어 `tasks/done/`에 Tier C(PLAN-006 소유) 9장만 남는다.
   문자 그대로 비우는 것은 이 계획의 목표가 아니다 — Tier C는 명시적으로 범위 밖이고,
   그 카드들은 PLAN-006이 닫는 시점에 리뷰된다.
2. 리뷰 없이 아카이브한 카드는 `quality-review` 값으로 그 사실이 정직하게 남아 있다.
3. 루프가 관찰로만 남겼던 결함 3건이 자식 카드로 처리됐다.
4. 이후 done 진입 조건에 독립 리뷰가 요구된다 (외부 의존, §External).

## Starting state (2026-09-05)

- done 58장, `quality-review` 보유 0장. `_archive/done` 245장.
- done-disposition 4사이클(TASK-244·245·246·284, 2026-09-05) 수확: 실질 결함 0건, 문서·메타 정리 4건.
  TASK-284의 major finding은 루프가 아니라 독립 리뷰가 찾은 것을 루프가 기록했을 뿐이다.
  기록: `tmp/task-management/direct/done-disposition/` (gitignored).
- 엔진(`ce-workbook/task_management`)은 이미 done-review → `quality-review` → done-disposition → archive를
  분리해 두고 있고, `ce task archive`는 `quality-review: pass|conditional` 없는 카드를 거부한다.
  빠진 것은 설계가 아니라 **호출 시점**이다.
- PLAN-006이 다른 세션에서 활성이며 TASK-303~317을 소유한다. 그 카드들은 이 계획이 건드리지 않는다.

## Triage

### Tier A — 실질 done-review (13장)

보안·비밀·데이터 손실·롤백 표면이거나 P0 결정. 지금 방식(binding 재실행 + diff 대조)으로 한 장씩.
리뷰어는 구현 세션과 달라야 한다.

| Task | 이유 |
|---|---|
| TASK-248 | P0. required env 정책이 진단을 깨지 않게 강제 |
| TASK-252 | P0 결정. top-level env promotion 안전성 |
| TASK-259 | P0 결정. qualified project addressing |
| TASK-260 | P0 결정. cross-project 조합 계약 |
| TASK-261 | P0 결정. vNext 어휘·마이그레이션 약속 |
| TASK-263 | P0 결정. addressing 노출·충돌 |
| TASK-265 | P0 결정. interaction env_file 호환 |
| TASK-277 | env_file 보간 순서 비결정성 — TASK-246 리뷰가 실재 확인 |
| TASK-281 | env bridge seal/show 계약 동결 |
| TASK-282 | env bridge seal/show 구현 — 비밀 취급 표면 |
| TASK-286 | agent-runtime deny rule 투영 — 파괴적 명령 차단 |
| TASK-291 | composition 런타임·LIFO 롤백 |
| TASK-296 | readiness 실패 시 자식 방치·롤백 누락 |

### Tier B — 기계적 아카이브, `waived` 표기 (36장)

docs/chore/P2·P3, 라우트 결정·cobra 도움말·문서 수정·릴리스 절차·PLAN-005 조합 구현의 나머지.
`ce task validate` 통과 + 부모 링크 재지정 + `quality-review: waived`(엔진이 값을 받으면) 또는
`conditional` + evidence 한 줄 `waived: PLAN-007 backlog triage, no independent review`. 한 커밋으로.

250, 254, 255, 256, 257, 258, 266, 267, 268, 269, 270, 271, 272, 273, 274, 275, 276, 278, 279, 280,
283, 285, 287, 288, 289, 290, 292, 293, 294, 295, 297, 298, 299, 300, 301, 302

### Tier C — 보류, 활성 소유자 있음 (9장)

PLAN-006 자식. 그 계획을 진행하는 세션이 닫는 시점에 리뷰하도록 맡긴다. 이 계획은 건드리지 않는다.

303, 304, 305, 306, 308, 310, 312, 313, 317

## 2026-09-09 중간 점검 — 재고는 비웠고 유량은 그대로다

`progress`를 90에서 **75**로 내렸다. 자식이 둘 늘어서다(TASK-367·368). 숫자가 뒤로 간
것이 아니라 분모가 정직해진 것이다 — 아래가 근거다.

**측정**: `tasks/done/` **19장**, `quality-review` 보유 **0장**. 2026-09-05 시작 상태가
"58장, 0장"이었고 §Order 3·4b가 그 58장을 다 비웠다. 이틀 만에 19장이다.

§Goal 1은 "처리되어 `tasks/done/`에 Tier C 9장만 남는다"였다. 지금 19장 중 Tier C는
다섯(308·310·312·313·317)뿐이다. **§Goal 자신이 진단한 것은 재고가 아니라 유량이었는데**
("유입이 배출보다 빠르다") 이 계획이 실행한 것은 재고 비우기뿐이었다. 유량을 건드리는
항목은 §External 1번 하나였고 그것을 카드 없이 남겼다.

그래서 이 계획은 **부분적으로만 닫힌다**:

| §Goal | 상태 |
|---|---|
| 1. 58장 처리, Tier C만 잔류 | **일회성으로는 완료.** 그 58장은 전부 처분됐다. 지속 상태로는 거짓 — 19장이 다시 있다 → TASK-367 |
| 2. 리뷰 없이 아카이브한 카드에 그 사실이 정직히 남음 | **완료**, 단 아래 `waived` 항 참조 |
| 3. 루프가 관찰만 남긴 결함 3건 → 자식 카드 | **완료** (§Order 4c, 여섯 장) |
| 4. 이후 done 진입에 독립 리뷰 요구 | 엔진 강제는 외부 소유로 남고, DVA가 할 수 있는 문서화는 TASK-368로 완료 |

**두 항목을 카드로 만든 이유**는 이 저장소가 같은 실수를 이미 한 번 고쳤기 때문이다.
PLAN-006 §Order 10행이 여덟 장의 필요를 서술하고 소유자를 적지 않았고, 그 여덟 장은
2026-09-09에 PLAN-008이 생길 때까지 소유자 없이 놓여 있었다. 필요를 문장으로 적는 것과
카드로 만드는 것은 다르다.

### `waived`가 상류에 추가됐다 — §External 2번은 해결됐다

§External 2번은 `host-schema.yaml`에 `waived` 값을 추가하는 것이었고 "카드 없음"으로
남겼다. **2026-09-09 재확인: 값이 존재한다.** `ce task archive --help`가
`quality-review: pass|conditional|waived`를 받는다고 명시하고, `waived`는
`quality-review-evidence`를 함께 요구한다. 이 계획이 요청한 형태 그대로다.

대가가 하나 남았다. §Order 3이 Tier B 36장을 값이 없던 시절 `conditional` + evidence
문구로 우회했고, 지금 `tasks/_archive/done/`의 `quality-review: conditional`은 **48장**
이다. 그중 상당수는 리뷰된 카드가 아니라 우회된 waiver다. **소급 수정하지 않는다** —
evidence 문구(`waived: PLAN-007 backlog triage, no independent review`)가 각 카드에
남아 있어 정보는 손실되지 않았고, 아카이브를 다시 쓰는 비용이 이득을 넘는다. 기록만
남긴다: **그 48장을 세어 "48장이 리뷰됐다"고 읽으면 틀린다.**

## Order

| # | 작업 | 조건 |
|---|---|---|
| 1 | TASK-325, 326, 327 처리 | 완료 2026-09-10. 세 카드는 독립으로 처리됐고, TASK-325는 현재 flowcheck 출력으로 예시를 갱신했다 |
| 1a | TASK-326 완료 후 세션 리뷰가 발견한 P2 결함을 TASK-330으로 분리 | TASK-326의 후속. 완료 및 통합됨 |
| 2 | §External의 엔진 정책 변경 | **카드 없음, 추적 안 함.** ce-workbook 소유이고 3의 우회로 실무 필요는 이미 충족됐다 (§External) |
| 3 | Tier B 일괄 아카이브 | **완료** 2026-09-07, 커밋 `8d3031b`. `waived` 값이 없어 `conditional` + evidence 문구로 대체 |
| 4a | Tier A 13장 done-review | **완료** 2026-09-07, 커밋 `a575ccb`(7장) + `242cd5b`(6장) |
| 4b | Tier A 아카이브 | **완료** 2026-09-08. 12장은 4a 리뷰 직후 아카이브됐고, `fail`로 남아 있던 TASK-282는 결함 카드 둘(TASK-334 결정, TASK-335 수정)이 닫힌 뒤 별도 세션이 재리뷰해 `conditional`을 냈다 — 코드 결함 없음, 막고 있던 것은 카드 본문의 과장 주장 셋이었고 `1e73a99`가 정정했다. Tier A 13장 전부 아카이브 완료 |
| 4c | 4a가 낳은 결함 카드 6장 | TASK-332·333·342·334·335·337. 이 계획의 자식으로 편입됨. 2026-09-08 기준 332·333·334·335·337·342 여섯 장 전부 닫힘 — 4c 완료 |
| 5 | Tier C | 이 계획 범위 밖. PLAN-006 종료 시 잔여분만 재평가 |

## External

`ce-workbook/task_management` 엔진 소스(생성물 `generated/` 아님)에 두 가지:

- `run-finish` 또는 `move --to done` 전제조건에 `quality-review` 존재 요구. 대기열 재발을 막는 유일한 지점.
- `host-schema.yaml` `quality-review_values`에 `waived` 추가. 리뷰 생략을 `conditional`로 위장하지 않기 위해.

**상태 (2026-09-08): 둘 다 카드가 없고, 만들지 않는다.** 엔진 소스는 ce-workbook 소유라 이 저장소의
태스크 보드가 진척을 강제할 수 없고, 여기에 카드를 두면 영원히 열린 채 남는다. 두 번째 항목의 실무
필요는 Tier B 아카이브(`8d3031b`)가 `conditional` + `waived: PLAN-007 backlog triage, no independent
review` evidence 문구로 이미 충족했다 — 값이 정직하게 남아 있고 리뷰됨으로 위장되지도 않았다.
첫 번째 항목(닫기 시점 강제)은 정책 변경으로 남으며, 그 저장소에서 다룬다.

DVA 쪽 문서는 이 계획이 끝난 뒤 `AGENTS.md`에 "카드를 닫으려면 별도 세션의 done-review가 필요하다"
한 줄과 정책 링크만 둔다. 규칙 본문을 복제하지 않는다.

## Rules for this plan

- 동시 개발 중이다. 기존 done 카드를 수정하는 커밋은 통합 직전 origin/master로 rebase하고,
  카드 파일 외 경로를 건드리지 않는다.
- done-disposition 루프 중 소유자 없는 결함을 보면 같은 사이클에서 todo 카드를 만든다. 기록만 남기는
  관찰은 손실이다.
- Tier B 아카이브가 "검토됨"으로 읽히면 안 된다. `waived`/대체 표기 없이는 아카이브하지 않는다.

## Children

- ~~TASK-325~~ — docs/51 flowcheck 예시 출력 갱신 (done, 2026-09-10)
- TASK-326 — doccheck 앵커 슬러그가 `_`를 버려 GitHub와 어긋남 — 완료(아카이브)
- TASK-327 — PLAN-002에 TASK-284 temp-name supersession 기록 — 완료
- TASK-330 — doccheck 언더스코어 강조가 문자 단위가 아닌 run 단위로 페어링하도록 수정 (TASK-326 세션 리뷰가 발견한 후속 결함) — 완료(아카이브)
Tier A done-review(4a)가 발견한 결함 카드 — 리뷰가 관찰만 남기지 않고 카드를 만든다는 §Rules 규칙의 결과다.

- TASK-332 — 죽은 interaction `env_file` deprecation 경고 제거 (TASK-259 배치 1) — 완료
- TASK-333 — import 항목의 owner·canonical/alias 정체 노출 (TASK-259 배치 1) — 완료, 정체를 두 map 키가 공유하는 clone 하나에 한 번 찍어 출력이 키에서 되유도하지 않게 함. `plans:` import는 아직 비어 있고 TASK-366가 소유
- TASK-342 — TASK-263의 subproject 예약어·부모 라우트 거부 규칙 강제 (TASK-263 배치 1) — 완료
- TASK-334 — `config env show`의 real-sops 커버리지 한계 결정 (TASK-282 배치 2) — 완료, fake 주도 한계 수용
- TASK-335 — seal create-only TOCTOU 창 차단 (TASK-282 배치 2) — 완료, `link(2)` 원자 배치
- TASK-337 — `GatedCommands`를 살아 있는 커맨드 트리에 결속 (TASK-286 배치 2) — 완료, 게이트 표면을 선언이 아니라 `internal/cli` AST에서 도출

2026-09-09 중간 점검이 §Goal 1·4의 미완 절반에 소유자를 붙인 카드 — 위 §2026-09-09 중간 점검 참조.

- TASK-367 — 트리아지 이후 다시 채워진 done 대기열 처분 (19장, `quality-review` 0장). 이 계획 자신의 자식 일곱 장이 소유자 없이 그 안에 있다
- TASK-368 — `AGENTS.md`에 독립 `done-review` 요구와 canonical 정책 링크를 기록 — 완료
