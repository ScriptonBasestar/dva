---
id: TASK-447
title: "Close PLAN-011 and archive completed board cards"
type: chore
priority: P2
effort: S
status: done
quality-review: pass
quality-reviewed-at: 2026-09-28
quality-review-evidence: "Independent read-only review of 89e3e308 PASS: make doc-check OK, ce task validate --all 502 valid/0 invalid, ce task lint CLEAN; renames are status/link-only, evidence links resolve, PLAN-010/011 counts match planprogress, ISSUE-046 repro verified against the live repo, README condensation strands no fact not held by archived cards."
created: 2026-09-28
---

## Summary

완료 카드 6장(TASK-431, 442–446)과 역할을 다한 BACKLOG-009를 `tasks/_archive/2026-09/`로
`git mv`하고 `archived-at`을 붙였으며 증거 링크는 `../../done/evidence/`로 재지정했다.
PLAN-010(8/8)은 `ce task archive`로 `tasks/_archive/plan/`에 보관했다. 보드 인덱스의
현재 상태 절은 누적 로그 대신 현재 판정만 남기도록 줄였다. 소유자 확인이 필요한 grok
미푸시 브랜치는 ISSUE-046으로 기록했다. 통합 없는 회수 결함은 ce-agent-kit ISSUE-074
(`0605d3bc`)로 상류에 기록했다.

같은 시각 병행 세션(`fa0d518e`, codex/mbp)이 CE `run-discard`로 TASK-436을 실제 회수해
done 처리하고 ISSUE-039·PLAN-011을 보관했다. 이 카드의 초안은 TASK-436을 branch 보존·
superseded로 닫았으나 rebase에서 그 결과를 채택하고 겹치는 편집을 버렸다.

## Completion Criteria

- [x] Done cards, TASK-436, BACKLOG-009, PLAN-010 and PLAN-011 are archived and links resolve | verify: `make doc-check` (regression-guard)
- [x] The board passes strict validation and lint | verify: `ce task lint`
