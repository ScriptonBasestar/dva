---
id: TASK-411
title: "Fold the legacy archive spelling into _archive"
type: chore
priority: P2
effort: S
exec-tier: standard
status: review
created: 2026-09-22
depends-on: [TASK-410]
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

- ISSUE-035 — tasks/issue/035-repository-tooling-hardcodes-the-legacy-archive-spelling-and-cannot-see-archive.md
- [[TASK-410]] — 선행 조건
