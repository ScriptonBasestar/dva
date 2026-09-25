---
id: TASK-430
title: "Stop done-finalize from treating evidence paths as ownership"
type: bug
priority: P2
effort: M
exec-tier: strong
quality-review: pass
quality-reviewed-at: 2026-09-25
quality-review-evidence: "Independent final done-review PASS by /root/final_review_task430 on 2026-09-25T04:29:06Z: all four criteria, tracked preview receipt, CE implementation run, and TASK-410 boundary were verified; evidence SHA-256 89c6cccf8fce83247cff8f030036bdc9fbc22936baef0f07dfe5edff3308cc29."
status: done
created: 2026-09-24
---

## Summary

[ISSUE-024](../_archive/issue/024-done-finalize-blocks-cards-that-name-artifacts-in-review-evidence.md)는
과거 `done-finalize`가 증거의 저장소 경로와 `depends-on` 같은 구조 참조를 소유 위반으로
읽었던 문제를 기록한다. CE의 수정은 source master에 통합됐으며, 현재 카드는 회귀 테스트와
DVA cleanup preview 증거를 함께 검토한다. 증거와 참조를 지워 통과시키지 않는다.

방향: 정합성 검사는 실제 transient artifact 소유와 durable audit history를 구분한다.
Git 추적 canonical `quality-review-receipt`는 보존하고, 그 밖의 소유 바인딩은 계속 검사한다.

## Completion Criteria

- [x] 존재하고 유효하며 Git 추적된 canonical review receipt만 transient cleanup에서 보존된다; missing, malformed, untracked lookalikes는 거부된다 | verify: human — exact implementation CI run `391b80213ee4fb464990c245ae5bf308`
- [x] preview와 apply 재검증이 같은 receipt 예외를 적용하고 receipt 보존 cleanup이 완료된다 | verify: human — exact implementation CI run `391b80213ee4fb464990c245ae5bf308`
- [x] untracked receipt lookalike와 legacy explicit artifact binding은 계속 cleanup을 막는다 | verify: human — exact implementation CI run `391b80213ee4fb464990c245ae5bf308`
- [x] dry-run이 receipt 보존 범위에서는 ownership reconciliation으로 BLOCKED되지 않는다 | verify: human — [tracked cleanup preview evidence](evidence/TASK-430/ce-task-cleanup-preview-20260925.md.txt) records exact commands, binary revision, exit status, output, and unchanged worktree status

## Evidence

- [TASK-410 cleanup preview](evidence/TASK-430/ce-task-cleanup-preview-20260925.md.txt)

## Out of scope

- 막힌 카드의 증거 문장이나 `depends-on`을 지우는 일.

## Sources

- [ISSUE-024](../_archive/issue/024-done-finalize-blocks-cards-that-name-artifacts-in-review-evidence.md)

## Review Attempts

- 2026-09-25 independent review (`/root/review_433_434`): **FIX NEEDED**. The upstream implementation criteria passed, but criterion 4 was not verifiable because the DVA preview had only been summarized in prose. The reviewer requested tracked evidence with the command, CE binary revision, exit status, absence of `ownership reconciliation`, and unchanged worktree status. The repair is now recorded in `tasks/done/evidence/TASK-430/ce-task-cleanup-preview-20260925.md.txt`; fresh independent review is pending.

- 2026-09-25 independent final review (`/root/final_review_task430`): **PASS** at
  `2026-09-25T04:29:06Z`. The reviewer verified all four criteria, the tracked preview
  evidence and its SHA-256, the exact CE implementation run, and the TASK-410 evidence
  boundary. No further repair was requested.

## Blocker

ISSUE-024의 방향은 Git 추적 canonical `quality-review-receipt`를 durable audit evidence로
보존하는 것이다. CE clean task branch `dev/codex/mbp/fix/cleanup-durable-review-receipts`는
선행 semantics `c8e47681`과 receipt 보존 `0a35898e`를 포함하고 CE TASK-329를 review에
등록했다. focused filesystem/CLI tests, lint, validate 13/13, gate가 통과했다.

Full DVA CI run `6f7fc19a4287ad3da2c604440ae210a5`는 `go test -race ./...` 단계에서
10분 제한으로 끝났고 실패 출력은 없었다 (ISSUE-073). 2026-09-25 독립 리뷰는
`0a35898e`를 FAIL 처리했다. 후속 구현 `ed1f4574`는 예외를 정규 ID 경로, 존재, Git
추적, JSON의 64-hex digest 검증으로 제한하고 missing, malformed, untracked 및 비정규
이름 음성 회귀를 추가했다. preview와 apply가 공유 reference 판정을 쓰며, tracked
valid receipt가 candidate 경로를 언급해도 보존되는 apply test도 추가했다. TASK-329는
archived ISSUE-051의 역사 기록을 그대로 두고 새 cleanup 계약이 structured semantic
edge 규칙을 supersede한다고 명시했다.

Exact implementation commit `ed1f4574019eb86e6c46fe96443bddbfd85c8a48`의
`dva ci commit` run `391b80213ee4fb464990c245ae5bf308`는 7m18s PASS였고 attestation
before/after가 동일하다. CI 이후 evidence-only commit `ec4112bd`가 추가됐으므로
독립 최종 리뷰는 PASS했고 reviewer가 evidence SHA와 implementation run을 대조했다.
리뷰 기록을 추가한 `1e408857`까지 원격에 push했고 CE source master/origin 통합과 task
worktree/branch 회수를 완료했다. 통합된 CE binary로 DVA TASK-410 cleanup preview를 실행해
ownership reconciliation 차단 없이 아카이브 카드 1건이 제거 후보임을 확인했다. preview는
파일을 변경하지 않았다. 이전 중복 history checkpoint branch는 보존 중이며 사용하지 않는다.

## 2026-09-25 tracked DVA previews

깨끗한 DVA source checkout에서 통합 CE binary로 TASK-391/393/394와 TASK-410을 각각
preview했다. 네 카드 모두 `WOULD REMOVE`로 반환됐고 `ownership reconciliation` 차단은
없었다. 두 명령 모두 exit 0이며 before/after `git status --porcelain`은 clean으로 같다.
삭제는 적용하지 않았다. 재현 가능한 명령·stdout·binary revision은 위 tracked evidence에
기록했다. TASK-430의 최종 독립 done-review는 PASS로 기록됐다.
