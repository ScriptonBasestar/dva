---
id: ISSUE-036
title: "Archive fold turned 4 STALE path warnings into 293 and drowned the signal"
type: bug
status: done
priority: P2
effort: M
severity: medium
ownership: local
discovered-in: "TASK-411 (fold the legacy archive spelling into _archive)"
discovered-at: 2026-09-22
created: 2026-09-22
resolution: fixed
resolved-at: 2026-09-22T13:53:51Z
resolution-summary: "Resolved as fixed."
---

## Summary

`doccheck`의 STALE 경고는 문서·카드가 **적어 둔 태스크 경로가 디스크에 없을 때**
뜬다. TASK-411이 413장을 `tasks/archive/` → `tasks/_archive/`로 접자, 그 경로를
적어 둔 모든 인용이 한꺼번에 STALE이 됐다 — 접기 전 4건에서 **293건**으로.

exit code는 여전히 0이다(STALE은 경고). 망가진 것은 게이트가 아니라 **경고의
쓸모**다. 293줄은 아무도 읽지 않고, 그 안에 섞인 진짜 오타 한 건은 접기 전 4건
중에 있었을 때와 달리 발견되지 않는다.

## Evidence

```
$ make doc-check 2>&1 | grep -c STALE
293
$ make doc-check 2>&1 | grep STALE | sed 's#^  STALE *##;s#/.*##' | sort | uniq -c
  37 docs
 256 tasks
```

접기 직전 같은 명령은 4건이었다(전부 `tasks/archive/2026-09/` 카드가 서로를
`tasks/archive/done/` 옛 배치로 가리키던 것).

모든 인용은 **카드 id로 해소된다** — `doccheck`가 `resolved by card id to
"tasks/_archive/…"`까지 찍어 준다. 즉 깨진 링크는 하나도 없다. 경로 표기만 낡았다.

## Reproduction

1. TASK-410이 통합된 트리에서 시작한다 (`tools/`가 두 철자를 다 안다).
2. `make doc-check 2>&1 | grep -c STALE` → `4`.
3. `ce task archive --fold` 로 413장을 `tasks/_archive/`로 접는다.
4. `make doc-check 2>&1 | grep -c STALE` → `293`.

## Expected vs Actual

**Expected**: 접기는 저장 위치를 바꾸는 작업이다. 카드 id로 전부 해소되는 이상,
사람이 읽어야 할 경고 목록의 크기는 접기 전과 비슷해야 한다.

**Actual**: 목록이 4줄에서 293줄로 늘었다. exit code는 0이라 게이트는 통과하지만,
그 안에서 진짜 오타를 골라낼 수 있는 사람은 없다. 경고가 조용히 무용해졌다.

## Analysis — 두 모집단은 성격이 다르다

- **256건 (tasks/)**: 대부분 `tasks/_archive/` 안의 **닫힌 역사 카드**다. 쓰일
  당시엔 맞는 경로였다. 역사 기록을 일괄 치환하는 것은 기록을 고치는 일에
  가깝고, 봉인 정정 절차(docs/406)가 경계하는 방향이다.
- **37건 (docs/)**: 살아 있는 문서다. 지금 읽는 사람에게 없는 경로를 가리킨다.
  이쪽은 고치는 게 맞다.

## Recommended direction

1. `doccheck`가 **아카이브된 카드가 출처인 STALE은 세되 줄로 찍지 않는다** —
   `archive_stale: 256 (suppressed)` 한 줄 요약으로. 닫힌 카드의 낡은 경로 표기는
   고칠 대상이 아니라 역사이고, 경고 목록의 목적은 *고칠 것*을 보여 주는 것이다.
2. `docs/` 37건은 실제로 새 경로로 갱신한다.
3. 그러면 남는 목록이 다시 사람이 읽을 크기가 되고, STALE이 원래 하던 일
   — 진짜 오타 잡기 — 로 돌아온다.

대안(전부 치환)은 권장하지 않는다: 256장의 닫힌 카드를 건드리면 diff가 거대해지고,
그중 일부는 `blocks:`를 가진 done 카드의 digest에 영향을 준다.

## Resolution Criteria

- [x] 사람이 읽을 수 있는 크기로 돌아온다 — 개별 STALE 줄이 40건 이하 | verify: `test "$(make doc-check 2>&1 | /usr/bin/grep -c '^  STALE')" -le 40`
- [x] `docs/` 아래 STALE 인용이 0건이다 | verify: `! make doc-check 2>&1 | /usr/bin/grep 'STALE *docs/'`
- [x] 문서 게이트가 통과한다 | verify: `make doc-check`

## 2026-09-22 해결

권장안 그대로 구현했다.

1. `tools/doccheck/check.go`: `Result.StaleLinkPathsArchive` 신설. 인용한 파일(`e.Path`)이
   `isArchivePath`(아카이브 프리픽스 둘 다)에 걸리면 `StaleLinkPathDetail`에 줄을 찍지 않고
   이 카운터만 올린다. `docs/`발 STALE은 그대로 상세 목록에 남는다. `main.go`에
   `stale_link_paths_archive: N (suppressed, ISSUE-036)` 요약 줄을 추가했다.
   `TestLinks_suppressesStaleWrittenTaskPathFromArchivedCard` 신설, 음성 프로브로
   비어있지-않음을 확인(분기를 `false && ...`로 죽이면 FAIL 재현).
2. `docs/53,54,58,59,61` 5개 파일의 `tasks/archive/` 인용 42건을 `tasks/_archive/`로
   고쳤다(문자열 치환; STALE 37건 전부 포함, 나머지 5건은 같은 줄에 인용이 여럿이거나
   STALE로 잡히지 않은 표기).

결과: STALE 293 → 0줄(요약 `stale_link_paths_archive: 256 (suppressed)`). `make doc-check`,
`go test ./tools/...` 전부 통과.

## 소유권 — 이 저장소다

STALE 판정도, 무엇을 줄로 찍고 무엇을 요약으로 접을지도 전부 이 저장소의
`tools/doccheck/`가 정한다. 상류(`ce`)는 경로 인용을 읽지 않는다 — `--fold`는 파일을
옮겼을 뿐이고, 옮긴 결과를 어떻게 보고할지는 여기의 결정이다.

## Related

- [[TASK-411]] — 이 상태를 만든 접기
- [[TASK-410]] — 철자 확대
- ISSUE-035 — 도구가 `_archive/`를 못 보던 원래 결함
