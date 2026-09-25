---
id: TASK-411
title: "Fold the legacy archive spelling into _archive"
type: chore
priority: P2
effort: S
exec-tier: standard
status: done
created: 2026-09-22
depends-on: [TASK-410]
quality-review: pass
quality-reviewed-at: 2026-09-25
quality-review-evidence: "Independent read-only review PASS by /root/review_437: gate READY, no tracked tasks/archive files, make doc-check and diff-check pass; primary checkout's ignored tasks/archive/.ce remains intact under .gitignore:45."
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
- [x] 보드 게이트가 READY이고 기존 체크아웃의 무시된 로컬 `.ce` 런타임 데이터를 삭제하지 않는다 | verify: human — `ce task gate --json`의 verdict와 `git ls-files tasks/archive`를 확인한다. 경고가 있으면 해당 경로가 Git 미추적·무시 상태임을 확인
- [x] 접은 뒤에도 plan 진행률이 맞고 문서 게이트가 통과한다 | verify: `make doc-check` (regression-guard)
- [x] 보드 게이트가 READY다 | verify: `ce task gate 2>&1 | /usr/bin/grep -q '^READY —'`

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

- ISSUE-035 — tasks/_archive/issue/035-repository-tooling-hardcodes-the-legacy-archive-spelling-and-cannot-see-archive.md
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

## 2026-09-23 재리뷰 완료

TASK-410 receipt와 TASK-415 zone 수정 뒤 네 기준을 독립 재실행해 모두 통과했다.
위 Review Attempts의 실패는 과거 상태다. primary checkout의 ignore된 잔재 삭제는
검증 범위가 아니며, 이 판정은 Git 추적 카드와 깨끗한 task worktree에 대한 것이다.

## 2026-09-23 아카이브 재검증 — 되돌림

기준 2는 이 체크아웃에서 미충족이다. `ce task gate`는 READY이면서도
`legacy-storage-dir` 경고를 낸다. 추적 파일은 없다(`git ls-files tasks/archive`가
비어 기준 1은 충족). 경고가 보는 것은 gitignore된 `tasks/archive/.ce`
(heartbeat·audit, 2026-08-20)가 살려 둔 디렉터리다. `make doc-check`는 통과해
기준 3도 충족하고, 게이트 출력의 `READY —`로 기준 4도 충족한다.

잔재는 지우지 않았다. 이 카드와 `tasks/README.md`가 이미 그 삭제를 사람 결정으로
남겨 두었다. 기준 문장을 깨끗한 워크트리로 좁히지 않으므로 아카이브하지 않고
`blocked/`로 되돌린다. 후속 카드는 만들지 않았다. 사람이 잔재를 지우거나 기준 2를
명시적으로 바꾸기 전에는 닫히지 않는다.

## 2026-09-25 범위 정합화

저장소가 보장하는 범위는 Git 추적 내용이다. `tasks/archive/.ce` 같은 무시된 로컬
런타임 데이터는 이 카드가 삭제하지 않는다. 완료 시 게이트는 READY여야 하며,
그 체크아웃에만 남은 `legacy-storage-dir` 경고는 삭제 근거가 아니다. 변경된 기준은
독립 재검증 후에만 닫는다.

## 2026-09-25 독립 재리뷰 PASS

`/root/review_437`가 read-only로 기준 2를 포함한 현재 기준을 재검증했다.
board worktree에서 `ce task gate --json`은 READY, `git ls-files tasks/archive`는
비어 있고, `make doc-check`와 `git diff --check`가 통과했다. 기본 체크아웃의
`tasks/archive/.ce/{heartbeat,sessions,audit/events.jsonl}`는 그대로 남아 있으며
`git check-ignore -v`가 `.gitignore:45:.ce/` 규칙을 확인했다. 제거하지 않았다.
