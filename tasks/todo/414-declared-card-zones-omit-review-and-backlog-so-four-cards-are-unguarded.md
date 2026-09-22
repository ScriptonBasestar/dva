---
id: TASK-414
title: "Declared card zones omit review/ and backlog/, so four cards are unguarded"
type: bug
priority: P1
effort: S
exec-tier: standard
status: todo
created: 2026-09-23
source: "2026-09-23 배치 7 통합 중 실측 — TASK-412 카드가 review/와 todo/에 동시에 존재했는데 doccheck가 `duplicate: 0`을 보고했다"
---

## Summary

`tools/doccheck/cardstatus.go:44-47`의 `cardZones`는 네 접두사만 선언한다 —
`tasks/done/`, `tasks/todo/`, `tasks/issue/`, `tasks/plan/`(skip) — 그리고
`archivePrefixes`. **`tasks/review/`와 `tasks/backlog/`는 없다.**

`resolveCardZone`이 `ok=false`를 내면 그 파일은 세 검사 **전부**에서 건너뛰어진다:

```
tools/doccheck/cardids.go:50     중복 카드 id
tools/doccheck/cardids.go:113    중복 파일명 번호
tools/doccheck/cardstatus.go:168 zone/status 일치
```

현재 사각지대에 있는 카드 4장:

```
tasks/review/410-…  TASK-410     status: review
tasks/review/411-…  TASK-411     status: review
tasks/review/412-…  TASK-412     status: review
tasks/backlog/009-… BACKLOG-009  status: (필드 없음)
```

## 실측 증거

2026-09-23, `tasks/review/412-…`(master)와 `tasks/todo/412-…`(태스크 브랜치)가 한
작업 트리에 동시에 존재하는 상태에서:

```
$ go run ./tools/doccheck
card_ids:            449 (duplicate: 0)
filename_numbers:    448 (duplicate: 0)
```

같은 id를 주장하는 카드가 둘인데 중복 0이다. 파일명 번호(`412-`)도 둘인데 0이다.
git은 경로가 달라 충돌로 보지 않고, 게이트는 zone 밖이라 보지 않는다.

이것이 [[TASK-407]]에 대한 직접적 반례다. 407은 할당 유일성이 깨져도 **DUP-ID 검사가
방어선**이라고 적었다. 그 방어선은 충돌한 카드 중 하나가 `review/`에 있으면 침묵한다.

## Design

접두사 두 개를 더하는 것으로는 부족하다 — 이 구멍이 생긴 방식이 재발 경로다.
`cardZones`는 TASK-287이 동결한 정적 목록이고, 보드에 디렉터리가 늘어도 아무것도
알려주지 않는다. `tasks/README.md`는 `review/`와 `backlog/`를 보드 구조로 문서화하는데
게이트는 모른다. 즉 **문서와 게이트가 갈렸고 그 갈림을 재는 것이 없다.**

따라서 지속되는 기준은 "선언되지 않은 `tasks/*/` 디렉터리가 존재하면 실패한다"이다.
그것이 없으면 다음 zone이 생길 때 같은 침묵이 반복된다.

`backlog/`의 permitted set은 별도 판단이 필요하다: `BACKLOG-009`는 `TASK-` 네임스페이스가
아니고 `status:` 필드가 아예 없다. `tasks/plan/`처럼 `skip: true`가 맞을 수 있다 —
다만 skip은 중복 검사에서도 빠진다는 뜻이므로, 그 대가를 카드에 적고 고른다.

## Completion Criteria

- [ ] `tasks/review/`가 선언된 zone이고 `status: review`를 허용한다 | verify: `/usr/bin/grep -rq 'TestCardZonesDeclareReviewAndBacklog' tools/doccheck/ && go test ./tools/doccheck/ 2>&1 | /usr/bin/grep -q '^ok'`
- [ ] `tasks/backlog/`가 선언된 zone이고, permitted든 skip이든 그 선택 이유가 카드에 적혀 있다 | verify: human — 완료 기록에서 backlog의 permitted set 결정과 근거를 읽는다
- [ ] `review/`와 `todo/`에 걸친 같은 id가 중복으로 잡힌다 | verify: `/usr/bin/grep -rq 'TestDuplicateCardIDAcrossReviewAndTodo' tools/doccheck/ && go test ./tools/doccheck/ 2>&1 | /usr/bin/grep -q '^ok'`
- [ ] 선언되지 않은 `tasks/*/` 디렉터리가 존재하면 게이트가 실패한다 | verify: `/usr/bin/grep -rq 'TestUndeclaredBoardDirectoryFailsTheGate' tools/doccheck/ && go test ./tools/doccheck/ 2>&1 | /usr/bin/grep -q '^ok'`
- [ ] 현재 보드가 통과한다 | verify: `make doc-check` (regression-guard)
- [ ] 보드 게이트가 READY다 | verify: `ce task gate 2>&1 | /usr/bin/grep -q '^READY —'`

## Out of scope

- 카드 id 할당 유일성 자체 — [[TASK-407]]과 [[ISSUE-033]]이 소유한다. 이 카드는
  할당이 깨졌을 때 **검출**이 되게 한다
- `tasks/archive/` ↔ `tasks/_archive/` 두 철자 처리 ([[TASK-410]]·[[TASK-411]]이 이미 소유)
- `ce lint`의 canonical zone 판정. 그것은 CE 런타임 소유이고 doccheck와 별개 축이다

## Sources

- `tools/doccheck/cardstatus.go:31-47` — `cardZones` 테이블과 TASK-287 동결 주석
- `tools/doccheck/cardids.go:41-70` — zone 밖 파일을 건너뛰는 지점
- `tasks/README.md` — `review/`·`backlog/`를 보드 구조로 문서화하는 곳
- [[TASK-407]] — DUP-ID를 방어선으로 지목한 카드
