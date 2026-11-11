---
id: TASK-600
title: "compare down 락과 락 문서 보완"
type: bug
priority: P3
effort: S
exec-tier: standard
status: todo
created: 2026-11-11
---

## Summary

TASK-595는 `compare up`만 이름 단위로 잠근다. `compare down`도 같은 기록을 읽고 쓰므로 up이 도는 동안 끼어들면 경합한다. 같은 락을 잡고, 잡혀 있으면 아무것도 멈추기 전에 실패하게 한다. 함께 `internal/config/constants.go:19`의 compare 디렉터리 주석에 `<이름>.lock`을 적는다. 남은 `.lock` 파일은 아무것도 잠그지 않으므로 지우는 일은 하지 않는다. TASK-595 리뷰 중 발견했다.

## Completion Criteria

- [ ] compare down이 같은 이름의 compare up이 도는 동안 아무것도 멈추기 전에 명확한 오류로 실패한다 | verify: human — 구현 카드가 go test -run 바인딩으로 바꾼다
- [ ] internal/config/constants.go의 compare 디렉터리 주석이 <이름>.lock을 언급한다 | verify: human — 구현 카드가 grep 바인딩으로 바꾼다
