---
id: TASK-603
title: "compare readiness의 서비스 없는 항목 처리"
type: feature
priority: P2
status: todo
created: 2026-11-11
---

## Summary

`internal/cli/compare_result.go`의 `readinessText`는 서비스도 health 검사도 없는 항목(예: 멈춘 compose 프로젝트, `compose ps`가 `[]`를 돌려주는 경우)을 건너뛰므로, 다른 항목이 ready이면 그 항목이 ready를 막지 못한다. TASK-602 2차 리뷰에서 발견했고 master에 이미 있던 공백이다. 이런 항목을 `-`로 볼지 not ready로 볼지 정한다.

## Completion Criteria

- [ ] 서비스·health 검사·오류가 모두 없는 항목은 판단 불가로 보아 다른 항목이 ready여도 readiness가 `-`이고, 다른 항목의 not ready·error는 그대로 우선한다 | verify: `go test ./internal/cli/ -run 'TestReadinessTextEmptyEntry|TestReadinessTextUnjudgeableServices' -count=1`
