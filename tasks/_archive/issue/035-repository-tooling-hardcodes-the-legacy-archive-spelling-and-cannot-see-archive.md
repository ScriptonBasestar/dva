---
id: ISSUE-035
title: "Repository tooling hardcodes the legacy archive spelling and cannot see _archive"
type: bug
status: done
priority: P2
effort: S
exec-tier: standard
severity: medium
ownership: local
discovered-in: "2026-09-22 board review after TASK-406·409 — `ce task gate` warns legacy-storage-dir and the fold that silences it would break make doc-check"
discovered-at: 2026-09-22
created: 2026-09-22
promoted-to: [TASK-410]
resolution: fixed
resolved-at: 2026-09-23T04:05:11Z
resolution-summary: "Resolved as fixed by TASK-410."
---

## Summary

`ce task archive`는 이제 `tasks/_archive/`에 쓴다. 이 저장소는 아직 옛 철자
`tasks/archive/`에 413장을 들고 있고, `ce task gate`가 그 이중 철자를
`legacy-storage-dir` 경고로 보고하며 `ce task archive --fold`를 처방한다.

**그런데 지금 접으면 `make doc-check`가 깨진다.** 이 저장소의 자체 도구 셋이 옛
철자를 하드코딩하기 때문이다. 경고를 지우라는 처방과 게이트를 통과시키라는 요구가
서로를 막는 상태이고, 결함은 경고 쪽이 아니라 **도구가 한 철자만 안다는 것**이다.

| 위치 | 하드코딩 |
|---|---|
| `tools/planprogress/check.go` | `zoneFromPath`가 `archive` / `archive/`만 `zoneClosed`로 본다 |
| `tools/doccheck/archive.go` | `archivePrefix = "tasks/archive/"` |
| `tools/doccheck/cardstatus.go` | 같은 접두사로 permitted status를 고른다 |

`tasks/_archive/`는 이미 존재한다 — 2026-09-22에 ISSUE-010·021을 포함해 7장이
그쪽으로 아카이브됐다. 즉 이 저장소는 **이미 두 철자를 동시에 들고 있고**, 새로
아카이브되는 카드는 전부 도구가 못 보는 쪽으로 간다.

## Reproduction

1. `ce task gate` — `WARNINGS`에 `legacy-storage-dir`가 있다.
2. `ce task archive --fold --dry-run` — 413장을 `tasks/_archive/2026-09/`로
   옮기겠다고 답한다.
3. `/usr/bin/grep -n 'archive' tools/planprogress/check.go tools/doccheck/archive.go` —
   `_archive`를 아는 분기가 없다.

## Expected vs Actual

- **Expected.** 저장 철자는 도구가 읽는 축이 아니라 저장소가 고르는 축이다.
  `archive/`든 `_archive/`든 같은 zone으로 분류돼야 한다.
- **Actual.** `_archive/` 아래 카드는 `zoneOther`로 떨어진다. 접는 순간
  PLAN-006·007·008·009의 아카이브된 자식이 전부 미완으로 재계산되고
  `planprogress`가, 따라서 `make doc-check`가 깨진다.

## 소유권 — 이 저장소다 (2026-09-22 명시)

결함은 `tools/` 아래 이 저장소의 Go 코드에 있다. 상류(`ce`)는 이미 새 철자를
쓰고 처방까지 내놓았다. 고칠 수 있고 고쳐야 하는 쪽은 여기다.

## Resolution Criteria

- [ ] `planprogress`가 두 철자를 같은 zone으로 분류한다 | verify: `go test ./tools/planprogress/ -run TestZoneFromPath -v 2>&1 | /usr/bin/grep -q '^--- PASS: TestZoneFromPath'`
- [ ] `doccheck`가 두 철자를 같은 아카이브로 분류한다 | verify: `go test ./tools/doccheck/ -run TestArchiveSpelling -v 2>&1 | /usr/bin/grep -q '^--- PASS: TestArchiveSpelling'`
- [ ] 접은 뒤 `legacy-storage-dir` 경고가 사라지고 보드 게이트가 READY다 | verify: `! ce task gate 2>&1 | /usr/bin/grep -q 'legacy-storage-dir'`
- [ ] 문서 게이트가 통과한다 | verify: `make doc-check`

## Related

- [[BACKLOG-009]] — 이 관측이 나온 트리아지
- `ce task archive --fold` — 처방. 도구 수정 **뒤에** 실행한다
