---
id: TASK-599
title: "cli 테스트 helper의 EPERM 불안정"
type: bug
priority: P3
effort: S
exec-tier: standard
status: todo
created: 2026-11-11
---

## Summary

`internal/cli/process_group_kill_unix_test.go`의 `killLeftoverProcessGroup`은 `ESRCH`만 허용한다. TASK-598에서 macOS가 좀비만 남은 프로세스 그룹에 `ESRCH` 대신 `EPERM`을 돌려주는 것이 격리 재현으로 확인됐고(약 3.5%), `internal/lifecycle`의 같은 helper는 고쳤다. 이 helper는 pid 파일이 남아 있을 때만 도는 더 좁은 경로지만 같은 경합이 가능하다(추정, 재현 전). 재현해 같은 방식으로 고친다. TASK-598 리뷰 중 발견했다.

## Completion Criteria

- [ ] killLeftoverProcessGroup이 이미 죽은(좀비만 남은) 프로세스 그룹의 EPERM을 오류로 보고하지 않는다 | verify: human — 구현 카드가 go test -run 바인딩으로 바꾼다
