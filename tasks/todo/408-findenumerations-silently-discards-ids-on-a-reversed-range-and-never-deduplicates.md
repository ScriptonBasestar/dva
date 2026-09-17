---
id: TASK-408
title: "findEnumerations silently discards ids on a reversed range and never deduplicates"
type: feature
priority: P2
status: todo
created: 2026-09-17
---

## Summary

`tools/planprogress/prose.go`의 `findEnumerations`에서 `[[TASK-NNN]]` 위키링크 형태로 쓰인 ID 나열이 인식되지 않아 검사가 조용히 건너뛰어지는 잔여 결함(ISSUE-017 F5b)을 해결한다. 위키링크 형태의 나열을 인식하거나, 인식하지 못하는 경우 스킵 사실을 명시적으로 보고하도록 개선한다.

## Completion Criteria

- [ ] `[[TASK-NNN]]` 위키링크 형태로 쓰인 id 나열이 `findEnumerations`에 인식되거나, 인식하지 못하는 경우 그 사실이 명시적으로 보고된다 | verify: human — probe P5 픽스처(`[[TASK-1]], [[TASK-9]] 두 장이 남았다`, TASK-9는 자식 아님)를 Goal로 줄 때 결함 보고 또는 스킵 알림 확인
- [ ] planprogress 패키지 테스트와 문서 게이트가 통과한다 | verify: `go test ./tools/planprogress/ && make doc-check`

## Sources

- ISSUE-017 — tasks/issue/017-findenumerations-silently-discards-ids-on-a-reversed-range-and-never-deduplicates.md
