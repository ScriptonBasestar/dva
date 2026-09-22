---
id: TASK-408
title: "findEnumerations silently discards ids on a reversed range and never deduplicates"
type: feature
priority: P2
status: done
archived-at: 2026-09-22
verified-at: 2026-09-22
verification-summary: "Re-verified 2026-09-22. C1 the wikilink enumeration is recognized, not skipped — TestCheckPlanProse/a_wikilink_enumeration_naming_an_id_outside_children_is_reported passes and TestFindEnumerations now carries wikilink/middot/mixed cases. Title-scope B and C re-measured as already closed: reversed_range_does_not_swallow_the_run_after_it and TestIssue017Deduplication both pass (TASK-394, TASK-391). C2 `go test ./tools/planprogress/` ok, `make doc-check` exit 0. No follow-up."
created: 2026-09-17
quality-review: pass
quality-reviewed-at: 2026-09-22T02:19:30Z
quality-review-evidence: "Independent review (opencode, not the implementer). C1: TestCheckPlanProse/a_wikilink_enumeration_naming_an_id_outside_children_is_reported covers `[[TASK-1]], [[TASK-9]]` reporting TASK-9. C2: `go test ./tools/planprogress/` pass on 2026-09-22. Title-scope B/C already closed by TASK-394/391; not re-opened."
---

## Summary

`tools/planprogress/prose.go`의 `findEnumerations`에서 `[[TASK-NNN]]` 위키링크 형태로 쓰인 ID 나열이 인식되지 않아 검사가 조용히 건너뛰어지는 잔여 결함(ISSUE-017 F5b)을 해결한다. 위키링크 형태의 나열을 인식하거나, 인식하지 못하는 경우 스킵 사실을 명시적으로 보고하도록 개선한다.

## Completion Criteria

- [x] `[[TASK-NNN]]` 위키링크 형태로 쓰인 id 나열이 `findEnumerations`에 인식되거나, 인식하지 못하는 경우 그 사실이 명시적으로 보고된다 | verify: human — probe P5 픽스처(`[[TASK-1]], [[TASK-9]] 두 장이 남았다`, TASK-9는 자식 아님)를 Goal로 줄 때 결함 보고 또는 스킵 알림 확인
  - Evidence: 인식하는 쪽으로 해결. `runTailRE`가 구분자 앞뒤의 `]]`/`[[`와 이어지는 id 뒤의 `]]`를 허용하도록 확장하고, 앵커 앞의 `[[`를 열거 텍스트에 포함. `checkPlanProse`가 P5 probe에 `children: does not list (TASK-9)` 1건을 보고함을 `TestCheckPlanProse/a_wikilink_enumeration_naming_an_id_outside_children_is_reported`로 고정. `TestFindEnumerations`의 기존 `not recognized (ISSUE-021 C)` 기대값을 `wikilink enumeration is recognized` 등으로 교체하고 middot/혼합형 케이스 추가.
  - Title-scope disposition (재현 측정 후 무수정 기록): 역방향 범위는 TASK-394에서 이미 보존 방향으로 해결됨 — `findEnumerations("TASK-9..1, 2")` → `[TASK-9 TASK-2]`, `("TASK-365..358, 366, 367")` → `[TASK-365 TASK-366 TASK-367]`. 중복 제거는 TASK-391에서 이미 해결됨 — `("TASK-1, 1, 2")` → `[TASK-1 TASK-2]`. 모두 2026-09-21에 당 worktree에서 직접 측정, 기존 테스트(`reversed range does not swallow the run after it`, `TestIssue017Deduplication`)도 통과하므로 추가 수정 없음.
- [x] planprogress 패키지 테스트와 문서 게이트가 통과한다 | verify: `go test ./tools/planprogress/ && make doc-check`
  - Evidence: `go test ./tools/planprogress/` PASS (전 스위트), `go vet ./tools/planprogress/` clean, `gofmt -l tools/planprogress/` empty, `make doc-check` PASS (아래 커밋 전 최종 확인).

## Sources

- ISSUE-017 — tasks/issue/017-findenumerations-silently-discards-ids-on-a-reversed-range-and-never-deduplicates.md

## Review Attempts

- 2026-09-22T02:19:30Z | reviewer: opencode (independent of implementer) | executor-tier: standard | finding: pass | verification: TestCheckPlanProse wikilink case; `go test ./tools/planprogress/` pass | outcome: pass | next: done
