---
id: TASK-426
title: "Write frontmatter status when a card changes zone or closes"
type: bug
priority: P2
effort: S
exec-tier: standard
status: done
created: 2026-09-24
quality-review: pass
quality-reviewed-at: 2026-09-25
quality-review-evidence: "독립 done-review PASS: move, resolve, archive가 본문 Status 셀 유무와 무관하게 frontmatter status를 기록하고, 회귀·superseded 테스트가 해당 동작을 고정한다."
---

## Summary

`ce-agent-kit#4`의 상류 수정으로 `move`, `resolve`, `archive`가 도착 상태를
frontmatter에 기록한다. 본문 Status 셀이 없는 카드도 갱신돼
[ISSUE-013](../_archive/issue/013-ce-task-move-does-not-sync-the-frontmatter-status-field.md)의
명령 소유 결함이 해결됐다.

방향: 존을 바꾸거나 이슈를 닫는 명령이 frontmatter `status:`를 도착 상태로
쓴다. 본문 셀이 없는 카드도 포함한다.

## Completion Criteria

- [x] `move`, `resolve`, `archive`가 도착 상태의 frontmatter `status:`를 쓴다 | verify: human — 본문 Status 셀이 없는 카드에 대한 상류 테스트가 이 카드에 링크된다

## Out of scope

- DVA doccheck의 zone 표를 이 명령의 대용으로 넓히는 일. 그 순서는 TASK-424가 소유한다.

## Sources

- [ISSUE-013](../_archive/issue/013-ce-task-move-does-not-sync-the-frontmatter-status-field.md)

## Current evidence

- `move`: `internal/adapter/filesystem/task_storage_zone_move.go:85-125`; no-body-status test: `internal/adapter/filesystem/task_storage_zone_test.go:32-53`.
- `resolve`: `internal/usecase/task/resolver.go:107`; status field test: `internal/usecase/task/resolver_test.go:19`.
- `archive`: `internal/adapter/filesystem/task_storage_mutate.go:82`; no-body-status test: `internal/adapter/filesystem/task_storage_archive_test.go:371`.
- Integrated implementation: `ce-agent-kit@8f2ed4514e010ec61cae35ec18d6fa8ea4ea85b8`.
- Full DVA commit CI run `6b88756eacac6c7e04f6ec57bdab50d2` passed with the implementation on master.
- Independent done-review: PASS; reviewer verified all three command paths, absent-body-status tests, and superseded archive behavior.
