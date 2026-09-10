---
id: TASK-370
title: "Keep the two errors frozen in the published v0.2.0 notes out of the next release"
type: docs
priority: P3
effort: S
exec-tier: standard
created-at: 2026-09-09
source: "v0.2.0 게시 후 도착한 독립 검증 D-1·D-5"
status: todo
needs-human: false
---

# Task 370: 게시된 v0.2.0 노트에 굳은 오류 두 건

## Summary

`v0.2.0` 태그를 자른 **뒤에** 두 번째 독립 검증이 도착했고, 그중 두 건이
`release-notes/v0.2.0.md`에만 있어 고칠 수 없는 자리에 굳었다. `docs/52`는 이미 게시된
tag를 이동하거나 다시 만들지 않는다고 정하고, 그 파일은 GitHub Release 본문이자
SHA-256이 TASK-369 결정 기록에 남은 파일이다. 그래서 **고치지 않고 적는다.**

| # | 자리 | 내용 |
|---|---|---|
| D-1 | `release-notes/v0.2.0.md` §"3. `--strict`에서만 exit 1이 됩니다" | "`examples/` 4개" — `9b74de9`의 `--stat`은 세 파일(`applications.yml`, `full-stack.yml`, `service-orchestration.yml`)이다. 그 커밋 **메시지 본문**이 "four examples/ files"라고 적었고, 노트가 메시지에서 쓰였다 |
| D-5 | `release-notes/v0.2.0.md` §"새 커맨드와 새 config 섹션"의 `dva ci` 항목 | `depends_on`이 `ci.profiles.<name>` 레벨에 `max_parallel`과 나란히 적혔다. 실제로는 step 필드다 — `internal/config/ci.go`의 `DependsOn`은 `CIStep`에 있고 `CIProfile`에는 없다 |

CHANGELOG 쪽 D-1은 이 카드와 같은 커밋에서 고쳤고, D-5는 CHANGELOG에서 처음부터 옳았다.
그래서 **두 문서가 서로 다르며, 옳은 쪽은 CHANGELOG다.** 이 카드가 존재하는 이유는
그 불일치를 나중에 발견한 사람이 "노트가 맞고 CHANGELOG가 틀렸나"를 다시 조사하지
않게 하려는 것이다.

실제 위험은 다음 릴리스 노트를 이번 노트에서 베낄 때다. `ci.profiles` 스키마 서술은
특히 그대로 옮겨 붙기 쉽고, 옮겨 붙는 순간 `depends_on`은 두 번째 릴리스에서도 잘못된
레벨에 서게 된다.

## Completion Criteria

- [ ] 게시된 노트가 게시 당시 바이트 그대로다 — 이 카드는 고치는 카드가 아니라 굳었다고 적는 카드다 | verify: `/usr/bin/shasum -a 256 release-notes/v0.2.0.md | /usr/bin/grep -q 7905843de27f4c3c7fcb43474165fa16bee89c0e381ff2577d4dc7e656d2a818`
- [ ] CHANGELOG의 예시 개수는 3이다 (게시된 노트는 4로 굳었고, 옳은 쪽은 CHANGELOG다) | verify: `! /usr/bin/grep -q '4개에서 실제로 죽은 config' CHANGELOG.md`
- [ ] 다음 릴리스 노트를 쓸 때 `ci.profiles` 서술을 이번 노트가 아니라 `internal/config/ci.go`에서 다시 유도했고, `depends_on`이 step 아래에 있다 | verify: human — 다음 `release-notes/v*.md` 작성 시 확인. 이 카드는 그때 닫힌다
- [ ] 문서 게이트 통과 | verify: `make doc-check` (regression-guard)

## Non-goals

- 게시된 태그·Release·노트 본문 수정. `docs/52` 위반이고, 이 카드의 존재 이유가 그것이다.
- 이 두 건 때문에 v0.2.1을 자르는 것. 둘 다 동작에 대한 진술이 아니라 개수와 필드 위치이며,
  옳은 서술이 CHANGELOG의 `dva ci` 항목과 struct 정의에 이미 있다.

## Notes

- 같은 검증이 올린 세 번째 항목(`dva init` 탐지 확대 누락)은 `84715c4`에서 이미
  CHANGELOG에 들어갔다. 검증자가 그 이전 커밋(`2e8f342`)을 봤기 때문에 누락으로 보고했다.
- `dva config migrate`의 `dva.yaml` 지원(TASK-304)은 이 카드와 같은 커밋에서 CHANGELOG에
  추가했다. 게시된 노트에는 없지만 노트는 breaking과 새 표면을 다루는 문서이고 이것은
  둘 다 아니라, 굳은 오류로 세지 않는다.
