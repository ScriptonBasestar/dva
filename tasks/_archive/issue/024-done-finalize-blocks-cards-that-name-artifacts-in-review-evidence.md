---
id: ISSUE-024
title: "done-finalize blocks cards that name artifacts in review evidence"
type: bug
status: done
priority: P2
severity: medium
ownership: upstream
created: 2026-09-15
discovered-at: 2026-09-14
discovered-in: "TASK-392 done-finalize (2026-09-14) — 통과한 카드와 막힌 카드의 차이에서"
upstream-ref: "ce-agent-kit#4"
resolution: fixed
resolved-at: 2026-09-25T04:32:54Z
resolution-summary: "Resolved as fixed by TASK-430."
---

## Summary

`ce task done-finalize`가 **리뷰 증거를 성실하게 적은 카드일수록 제거를 거부한다.**
두 가지 차단이 겹친다.

**1. 증거 서술의 경로가 아티팩트 소유로 해석된다.** `quality-review-evidence`는
리뷰어가 무엇을 봤는지 적는 자유 텍스트다. 그 안에 경로 모양 문자열
(`./tools/planprogress/`, `internal/cli`)이 보이면 finalize는 "explicit artifact
binding requires ownership reconciliation"을 내며 카드를 BLOCKED로 만든다. 리뷰
증거가 코드 영역을 언급하지 않는 것이 오히려 이상하므로, 이 검사는 **정상 작성을
벌하는** 방향으로만 작동한다. TASK-392가 통과한 이유는 증거 필드가 그 형태를
피했기 때문이지, 게이트가 그것을 지지해서가 아니다.

**2. 다른 카드의 역참조가 제거를 봉쇄한다.** done 카드를 위키링크로 인용하는
카드가 하나라도 있으면 "referenced by"로 BLOCKED다. 인용이 살아 있는 한 done
정리는 불가능하고, 인용은 정당하므로 지울 수도 없다 — 교착이다.

## Evidence

2026-09-15 현재 상태에서 세 카드의 dry-run 실측 (전부 BLOCKED, 전부 이 두 사유):

```
$ ce task done-finalize tasks/done/391-*.md --dry-run
  BLOCKED ...391: explicit artifact binding requires ownership reconciliation:
               ./tools/planprogress/ (from frontmatter:quality-review-evidence)
  BLOCKED ...391: referenced by tasks/done/394-*.md

$ ce task done-finalize tasks/done/393-*.md --dry-run
  BLOCKED ...393: explicit artifact binding requires ownership reconciliation:
               internal/cli (from frontmatter:quality-review-evidence)
  BLOCKED ...393: referenced by tasks/issue/022-*.md

$ ce task done-finalize tasks/done/394-*.md --dry-run
  BLOCKED ...394: explicit artifact binding requires ownership reconciliation:
               ./tools/planprogress/ (from frontmatter:quality-review-evidence)
  BLOCKED ...394: referenced by tasks/issue/017-*.md
```

역참조 사슬이 만든 교착의 모양: 017(이슈) → 394(done) → 391(done). 017은
살아 있는 이슈 카드고 참조는 정당하다. 사슬의 아무 고리도 스스로 풀 수 없다.

## Reproduction

1. done-review까지 끝난 카드 하나의 `quality-review-evidence`에 리뷰어가 본 코드
   영역을 경로로 적는다(정상적인 리뷰 증거 작성이다).
2. 그 카드를 위키링크로 인용하는 카드를 하나 둔다(정상적인 인용이다).
3. 그 카드로 `ce task done-finalize <카드> --dry-run`을 돌린다.

두 사유로 BLOCKED가 뜬다. 위 Evidence의 세 카드가 같은 조건에서 재현된 실례다 —
조건을 피한 카드만 제거된다.

## Expected vs Actual

| | |
|---|---|
| 기대 | 리뷰 증거의 서술은 소유 정합성 검사의 대상이 아니고(전용 바인딩 필드가 그 역할을 한다), 역참조는 인용 사실을 알리되 처분을 막지 않는다 |
| 실제 | 증거에 경로가 보이면 BLOCKED, 인용이 하나라도 있으면 BLOCKED — 두 조건을 피한 카드만 제거된다 |

## Impact

중간 — 지금은 세 장(TASK-391/393/394)이 이 차단에 걸려 있다. 세 장 모두
done-review까지 끝난 봉인 카드라 [[ISSUE-023]]의 done 다섯 장 대량 처분과 겹치면,
이 게이트가 풀리지 않는 한 보드의 done 존은 **자랄 수만 있다.** 기록된 원칙(증거는
Git 이력과 receipts가 보존)과 달리, 실제로는 카드 본문이 영구히 남는다.

## 소유권 — 상류다

두 검사 모두 `ce-agent-kit`의 cleanup 게이트(`internal/adapter/cli/commands/
task_cleanup*.go`의 정합성 검사)에 있다. 이 저장소가 할 수 있는 것은 증거 필드의
경로 표기를 피하는 것뿐인데, 그것은 1의 문제(정상 작성이 벌받는 구조)를
카드마다 우회로 복제하는 것이지 고치는 것이 아니다.

## TASK-391/393/394의 처분

세 장은 이 게이트가 고쳐질 때까지 finalize 불가다. 그때까지 done 존에 유지하며,
별도의 임시 조치(인용 제거·증거 재작성)는 하지 않는다 — 인용은 정당하고 증거
재작성은 봉인 카드 편집([[ISSUE-010]])이므로, 우회가 두 원칙을 모두 깨기 때문이다.

## Resolution Criteria

- [x] 상류에서 evidence 서술 경로가 소유 정합성 검사를 촉발하지 않게 된다 | verify: human — ce-agent-kit의 cleanup 검사가 전용 바인딩 필드만 검사하는지 읽어 확인한다
- [x] 세 카드의 dry-run이 더는 그 사유로 BLOCKED가 아니다 | verify: `ce task done-finalize $(/usr/bin/find tasks -name '391-stop-planprogress-from-pairing-any-korean-counter-with-an-enumeration.md') --dry-run 2>&1 | /usr/bin/grep -c "ownership reconciliation"` 출력이 0
- [x] 보드 게이트 통과 | verify: `ce task gate` (regression-guard)

## 2026-09-23 TASK-410 재현

사용자가 보고한 dry-run은 증거의 `./tools/planprogress/`, `./tools/doccheck/`,
`tasks/_archive/`, `./`를 ownership reconciliation 대상으로 읽고, TASK-411의
`depends-on: [TASK-410]` 구조 참조도 제거 차단으로 읽었다. 기존 두 유형의 새 사례다.
TASK-415에서 독립 리뷰와 canonical receipt로 보드 검증은 복구했으며,
finalize 성공은 TASK-411 완료의 선행 조건이 아니다. 증거·참조를 지워 우회하지 않는다.

## 후속 (2026-09-24)

우회 없이 남긴 상류 제한은
[TASK-430](../../done/430-done-finalize-ignores-evidence-paths.md)가 소유한다.

## 2026-09-25 후속 — canonical review receipt 보존 범위

TASK-430의 독립 리뷰에서 새 경계가 확인됐다. `quality-review-evidence`의 경로 산문과
`depends-on` 같은 카드 간 참조는 비소유로 다루는 것이 이 이슈의 해결 범위다.
하지만 TASK-410의 `quality-review-receipt`는 Git 추적 canonical receipt 경로를
명시적으로 가리키며, 현재 cleanup의 닫힌 소유 루트(`tasks/receipts/<ID>`,
`docs/task-reviews/<ID>`)에는 `tasks/done/evidence/<ID>`가 없다. 이 경로는 여전히
`ownership reconciliation`을 일으킬 수 있다.

결정: Git 추적 canonical receipt는 durable 감사 증거로 보존하고 cleanup ownership
검사는 실제 산출물 경로와 구분한다. 사용자가 이 방향을 승인했으며 TASK-430이
별도 worktree에서 계약·테스트를 구현 중이다. dry-run 증거와 독립 리뷰가 남아 있어
ISSUE-024는 열린 상태로 둔다.

clean CE branch `dev/codex/mbp/fix/cleanup-durable-review-receipts`의 commits
`c8e47681`/`0a35898e`가 semantic reference와 Git 추적 receipt 경계를 재구성한다.
Focused filesystem/CLI suites, lint, validate, gate는 통과했다. Full CI run
`6f7fc19a4287ad3da2c604440ae210a5`는 `go test -race ./...` 단계에서 10분 제한으로
끝났으며 test failure는 기록되지 않았다 (ISSUE-073). 독립 리뷰와 full CI 결과가
남아 있어 ISSUE-024는 todo 상태다.

## 2026-09-25 독립 리뷰 — receipt 경계 보완 필요

CE TASK-329의 독립 리뷰는 `0a35898e`를 FAIL 처리했다. 정규 receipt 경로 형태와
TASK ID만 맞으면 Git 추적, 파일 존재, receipt 유효성을 확인하지 않고 ownership
검사에서 건너뛴다. 따라서 missing, malformed, 또는 삭제 후보 경로를 본문에 싣지 않은
untracked canonical-shaped 파일도 허용될 수 있다. 또한 preview에는 durable receipt 예외가
있지만 apply 재검증에는 없어, 추적된 유효 receipt의 `reviewed-card-path`가 cleanup 대상이면
적용이 rollback될 수 있다. `{}`인 untracked lookalike 및 missing/malformed receipt의 음성
검사와 preview/apply 왕복 검사가 필요하다.

리뷰는 선행 `c8e47681`이 task semantic edge도 ownership 대상에서 제외하는 방향이 CE의
보존 이슈 ISSUE-051의 “카드에서는 구조화 간선만 본다” 결정 기록과 충돌한다고 추가로
지적했다. 기존 판정을 조용히 덮어쓰지 말고 CE 쪽에 새 계약을 채택하거나 해당 기록을
대체한다는 근거를 남겨야 한다. 수정, 독립 재검토, exact-HEAD full CI가 끝날 때까지
ISSUE-024와 TASK-430은 열린 상태다.

## 2026-09-25 후속 구현

CE `ed1f4574`는 receipt 예외를 실제 존재하는 Git 추적 JSON과 64-hex
`reviewed-card-sha256` 형식까지 검증한다. missing, `{}` malformed, untracked, 비정규
파일명 receipt가 계속 막는 회귀를 추가했고 preview/apply는 같은 reference 판정을 쓴다.
tracked receipt가 `reviewed-card-path`로 제거 대상을 언급해도 receipt를 남기고 cleanup하는
apply test가 있다. TASK-329에 ISSUE-051의 과거 범위와 새 계약의 supersession을 기록했고
archived 카드는 수정하지 않았다.

Exact implementation commit `ed1f4574019eb86e6c46fe96443bddbfd85c8a48`의 full
`dva ci commit` run `391b80213ee4fb464990c245ae5bf308`는 7m18s PASS, attestation 동일이다.
그 뒤 CI 증거만 기록한 `ec4112bd`가 추가됐다. 독립 최종 리뷰와 현재 branch HEAD의 CI
증거 연결 확인은 독립 리뷰에서 PASS했다. review evidence를 기록한 `1e408857`도
push했고 CE source master/origin에 통합·push, task worktree/branch 회수까지 완료했다.
통합 binary의 DVA TASK-410 preview는 ownership reconciliation 차단 없이 archived card
1건을 `WOULD REMOVE`로 판정했고 파일은 바꾸지 않았다. 원래 TASK-391/393/394 preview도
모두 `WOULD REMOVE`를 반환해 이 이슈의 cleanup 재현 조건은 해소됐다. 정확한 명령과 출력을
[TASK-430 tracked preview evidence](../../done/evidence/TASK-430/ce-task-cleanup-preview-20260925.md.txt)에
기록했다. TASK-430의 최종 독립 done-review PASS 후 ISSUE-024를 fixed로 resolve했다.
