---
id: TASK-410
title: "Teach planprogress and doccheck the _archive storage spelling"
type: bug
priority: P2
effort: S
exec-tier: standard
status: done
archived-at: 2026-09-23
verified-at: 2026-09-23
verification-summary: "Re-verified 2026-09-23. TestZoneFromPath, TestArchiveSpelling, and TestBuildTaskIndexSkipsPlanCardsInDatedArchivePartition/_archive passed. go test ./tools/planprogress/ ./tools/doccheck/ and make doc-check passed. Historical receipt left unchanged. No follow-up."
created: 2026-09-22
blocks: [TASK-411]
quality-review: pass
quality-review-receipt: tasks/done/evidence/TASK-410/done-review-c5b58f0ea33e1443707088627fd645461919bcfd3f08228494ffd42808c4b64f.json
quality-reviewed-at: 2026-09-23T00:00:00Z
quality-review-evidence: "Independent review (Claude Sonnet 5, not the implementer of 32e9fb09). C1: `go test ./tools/planprogress/ -run TestZoneFromPath -v` → PASS. C2: `go test ./tools/doccheck/ -run TestArchiveSpelling -v` → PASS (4 subtests). C3: `go test ./tools/planprogress/ -run TestBuildTaskIndexSkipsPlanCardsInDatedArchivePartition -v` → PASS, both archive/ and _archive/ subtests ran (not a zero-match false pass); asserts TASK-30 closed and TASK-5 absent from index. C4: `go test ./tools/planprogress/ ./tools/doccheck/ && make doc-check` → both ok, doc-check OK (archive_cards: 425 under tasks/_archive/, confirming visibility). Also verified `go build ./...` and `go test ./...` clean across the whole repo. Read the full diff (32e9fb09): zoneFromPath, buildTaskIndex, archivePrefixes/isArchivePath, buildCardZones all route through shared helpers so the two spellings cannot diverge; longest-prefix zone resolution is order-independent and the two prefixes never nest. Side effect noted: same commit corrected status: on 7 archived issue cards (newly visible defect, cross-referenced on ISSUE-013) — disclosed in the commit message, not hidden, and out of TASK-410's own criteria but not a regression."
---

## Summary

`tools/planprogress`와 `tools/doccheck`가 아카이브 저장 철자를 `tasks/archive/`
하나로 하드코딩한다. `ce task archive`는 이제 `tasks/_archive/`에 쓰고 이 저장소는
이미 두 철자를 동시에 들고 있어서(7장이 `_archive/`), 새로 아카이브되는 카드는
전부 도구가 못 보는 쪽으로 간다.

**저장 철자는 도구가 읽는 축이 아니다.** 두 철자를 같은 zone으로 분류하게 만든다.
어느 한쪽을 정본으로 고르는 일은 이 카드가 하지 않는다 — 그건 TASK-411이고, 이
카드가 먼저 끝나야 그쪽이 게이트를 깨지 않는다.

## Design

네 자리 전부 "접두사 하나"를 "접두사 둘"로 넓히되, 각 파일이 이미 가진 판정
구조는 바꾸지 않는다. (조사 시점에는 세 자리로 봤다 — 구현 중 `buildTaskIndex`의
네 번째 자리를 찾았다.)

- `tools/planprogress/check.go` — `zoneFromPath`의 `archive` 분기가 `_archive`도
  `zoneClosed`로 본다.
- `tools/planprogress/check.go` — `buildTaskIndex`의 아카이브된 **plan 카드** 제외
  필터. 이 자리를 놓치면 접은 뒤 `tasks/_archive/<YYYY-MM>/005-….md`(PLAN-005)가
  파일명 숫자로 TASK-5로 색인돼 실제 TASK-5 자식을 조용히 만족시킨다. 두 자리를
  `isArchiveRel` 하나로 모아 철자를 따로 배우지 못하게 한다.
- `tools/doccheck/archive.go` — `archivePrefix` 단일 상수를 접두사 목록으로.
- `tools/doccheck/cardstatus.go` — 같은 목록을 써서 permitted status를 고른다.

`_archive/` 아래 내부 배치(`2026-09/` 월별과 `issue/` zone별)는 **정규화하지
않는다.** `ce task archive`가 zone별로 쓰고 `--fold`가 월별 경로를 보존하므로,
한쪽으로 강제하면 도구와 싸우게 된다. 경고가 보는 축은 최상위 철자 하나다.

## Completion Criteria

- [x] `planprogress`가 두 철자를 같은 zone으로 분류하고 테스트가 그것을 고정한다 | verify: `go test ./tools/planprogress/ -run TestZoneFromPath -v 2>&1 | /usr/bin/grep -q '^--- PASS: TestZoneFromPath'`
- [x] `doccheck`가 두 철자를 같은 아카이브로 분류하고 테스트가 그것을 고정한다 | verify: `go test ./tools/doccheck/ -run TestArchiveSpelling -v 2>&1 | /usr/bin/grep -q '^--- PASS: TestArchiveSpelling'`
- [x] 접은 뒤에도 아카이브된 plan 카드가 TASK로 오색인되지 않는다 | verify: `go test ./tools/planprogress/ -run TestBuildTaskIndexSkipsPlanCardsInDatedArchivePartition -v 2>&1 | /usr/bin/grep -q '^    --- PASS: TestBuildTaskIndexSkipsPlanCardsInDatedArchivePartition/_archive'`
- [x] 두 패키지 테스트와 문서 게이트가 통과한다 | verify: `go test ./tools/planprogress/ ./tools/doccheck/ && make doc-check`

## Out of scope

- 실제 fold 실행 (TASK-411)
- `_archive/` 내부 배치 정규화 — 위 Design 참조

## Sources

- ISSUE-035 — tasks/_archive/issue/035-repository-tooling-hardcodes-the-legacy-archive-spelling-and-cannot-see-archive.md

## Archive

- [x] 아카이브
- 후속: 없음

## Review Attempts

- 2026-09-23T00:00:00Z | reviewer: Claude Sonnet 5 (independent of implementer, commit 32e9fb09 co-authored Claude Opus 5) | executor-tier: standard | finding: pass | verification: all 4 completion-criteria `verify:` bindings executed directly, all 4 green with real (non-vacuous) subtest evidence; `go test ./... ` and `go build ./...` also clean repo-wide; full diff of 32e9fb09 read for regressions/scope | outcome: pass | next: done
