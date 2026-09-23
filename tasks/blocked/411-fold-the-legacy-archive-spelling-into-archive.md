---
id: TASK-411
title: "Fold the legacy archive spelling into _archive"
type: chore
priority: P2
effort: S
exec-tier: standard
status: blocked
created: 2026-09-22
depends-on: [TASK-410]
blocked-at: 2026-09-23T00:00:00Z
blocked-on: "AC3/AC4 currently fail live: (1) TASK-410's done card declares
  `blocks: [TASK-411]` but no `quality-review-receipt`, which `ce task validate`
  now treats as an error, so `ce task gate` reads NOT READY --
  task_validate_failed; (2) an untracked ISSUE-037 card in the working
  tree has an invalid priority and a missing Expected-vs-Actual section, which
  fails `make doc-check`. Neither cause is inside TASK-411's own fold commits
  (5d5c099b, 97291cdb) -- both are external to this card's diff. Retry once
  TASK-410's done card carries a quality-review-receipt and the ISSUE-037 card
  is either committed in valid form or removed from the tree, then re-run
  `ce task gate` and `make doc-check`."
---

## Summary

`ce task gate`의 `legacy-storage-dir` 경고를 `ce task archive --fold`로 닫는다.
413장이 `tasks/archive/2026-09/`에서 `tasks/_archive/2026-09/`로 옮겨간다.

**TASK-410이 먼저 끝나야 한다.** 그 전에 접으면 `planprogress`가 PLAN-006·007·
008·009의 아카이브된 자식을 미완으로 재계산해 `make doc-check`가 깨진다. 이
카드의 `depends-on`은 선호가 아니라 게이트 사실이다.

## Design

`--fold`는 경로의 뒷부분을 보존한다 — `archive/2026-09/x.md` → `_archive/2026-09/x.md`.
그래서 접은 뒤 `_archive/` 안에는 월별(`2026-09/`, 413장)과 zone별(`issue/`, 7장)이
나란히 서게 된다. **이것을 정규화하지 않는다**: `ce task archive`는 zone별로 쓰고
`--fold`는 월별을 보존하므로, 한쪽으로 강제하면 다음 아카이브가 도로 갈라놓는다.
게이트가 보는 축은 최상위 철자 하나다.

카드 본문의 `tasks/archive/...` 경로 표기는 고치지 않는다. `doccheck`는 이미 그런
표기를 카드 id로 재해석해 `STALE`로만 보고하고 exit 0이며, 413장의 역사 표기를
일괄 치환하는 것은 이 카드의 범위가 아니다.

## Completion Criteria

- [x] 저장소에 옛 철자 경로의 카드가 남아 있지 않다 | verify: `test -z "$(git ls-files tasks/archive)"`
- [x] 보드 게이트에 `legacy-storage-dir` 경고가 없다 | verify: `! ce task gate 2>&1 | /usr/bin/grep -q 'legacy-storage-dir'`
- [ ] 접은 뒤에도 plan 진행률이 맞고 문서 게이트가 통과한다 | verify: `make doc-check` (regression-guard) -- 현재 FAIL, 원인은 아래 Review Attempts 참조 (TASK-411 diff와 무관)
- [ ] 보드 게이트가 READY다 | verify: `ce task gate 2>&1 | /usr/bin/grep -q '^READY —'` -- 현재 FAIL, 원인은 아래 Review Attempts 참조 (TASK-411 diff와 무관)

## 2026-09-22 통합 후 관측 — 기준 한 줄을 고쳤다

처음 쓴 기준은 `! test -d tasks/archive`였다. 새 워크트리에서는 통과하고 기본
체크아웃에서는 실패한다 — git은 파일만 지우고 빈 디렉터리는 남기며, 그 안에
**gitignore된 CE 런타임 잔재**(`tasks/archive/.ce/`, 2026-08-20자 audit·heartbeat)가
남아 디렉터리를 살려 두기 때문이다.

접기가 보장하는 것은 저장소의 내용이지 체크아웃의 로컬 미추적 상태가 아니다.
그래서 기준을 `git ls-files`로 옮겼다 — 어느 체크아웃에서 물어도 같은 답이 나온다.

`rmdir`이 거부해서 알았다. `rm -rf`였다면 남의 런타임 상태를 조용히 지웠을 것이다.

**남은 잔재는 건드리지 않았다.** `tasks/.ce`, `tasks/todo/.ce`, `tasks/done/.ce`,
`tasks/archive/.ce` 넷이 같은 시기 잔재로 남아 있고, 내가 만든 것이 아니며 저장소
내용도 아니다. 지울지는 사용자 판단이다 — `tasks/archive/.ce`만 지우면 그 디렉터리가
사라진다.

## Out of scope

- `_archive/` 내부 배치 정규화 — 위 Design 참조
- 카드 본문의 역사적 `tasks/archive/...` 경로 표기 일괄 치환

## Sources

- ISSUE-035 — tasks/issue/035-repository-tooling-hardcodes-the-legacy-archive-spelling-and-cannot-see-archive.md
- [[TASK-410]] — 선행 조건

## Review Attempts

- 2026-09-23T00:00:00Z | reviewer: Claude Sonnet 5 (independent of implementer; fold
  commits 5d5c099b/97291cdb authored by celee, co-authored Claude Opus 5 -- this
  session did not implement TASK-411) | executor-tier: standard | finding: blocked |
  verification: all four `verify:` bindings re-executed live against current repo
  state (not the card's pre-checked boxes). AC1 `test -z "$(git ls-files
  tasks/archive)"` -> PASS (empty). AC2 `! ce task gate 2>&1 | /usr/bin/grep -q
  'legacy-storage-dir'` -> PASS (warning absent). AC3 `make doc-check` -> FAIL
  (exit 2, `ERROR 1 issue card(s) classified with no ## 소유권 section stating why`
  against the untracked `tasks/issue/037-branch-integrate-refuses-to-push-a-master-
  local-housekeeping-commit.md`, which also fails `ce task validate` for invalid
  priority P3 and a missing Expected vs Actual section). Isolated the file (moved
  aside to /tmp, reran, restored byte-identical) and confirmed doc-check passes
  clean (exit 0) with it absent -- the failure is caused entirely by that
  unrelated, uncommitted card, not by anything in tasks/_archive or tasks/archive.
  AC4 `ce task gate 2>&1 | /usr/bin/grep -q '^READY —'` -> FAIL, gate reports
  `NOT READY — task_validate_failed`; with ISSUE-037 aside the failure narrows to
  one invalid card: TASK-410's done card -- "Done card blocks TASK-411 but
  declares no quality-review-receipt: the successors were unblocked on an
  unrecorded review." TASK-410's done card (commit eb230bc7, a separate,
  already-passed independent review of TASK-410, not this session's work) carries
  `quality-review`/`quality-reviewed-at`/`quality-review-evidence` but no
  `quality-review-receipt`, and `ce task validate` requires one on any done card
  declaring `blocks:`. Confirmed via git log that TASK-411's own fold commits
  (5d5c099b, 97291cdb) predate eb230bc7 and were not gate-broken at landing time
  -- this is a regression introduced afterward by a sibling task's finalization
  gap, not by TASK-411's diff. Side-effect/scope check: `git show --stat
  5d5c099b` confirms 413 pure renames (`R100`, no content diffs) plus two
  disclosed non-fold changes bundled in the same commit --
  `tools/doccheck/archive.go` (archivePrefix repointed to `_archive/`) and a new
  ISSUE-036 card -- both named and justified in the commit message
  ("Two defects found and recorded rather than silently absorbed"), same
  disclosure pattern TASK-410's review accepted. Design-constraint spot-check:
  `tasks/_archive/` holds both `2026-09/` (417 files, month partition) and
  `issue/` (8 files, zone partition) side by side, as the card requires and
  declares intentional. | outcome: blocked | next: tasks/blocked/, retry after
  TASK-410's done card gets a quality-review-receipt and the ISSUE-037 card is
  fixed or removed from the tree; re-run AC3/AC4 then re-review.
