---
id: TASK-595
title: "compare up 동시 실행 락"
type: bug
priority: P2
effort: S
exec-tier: standard
status: review
created: 2026-11-09
---

## Summary

같은 이름의 `dva compare up`을 동시에 실행하면 둘 다 기록(`.sb/dva/compare/<name>.json`)과 프로세스를 쓰려 해 서로 덮어쓴다. `internal/cli`와 `internal/compare`에는 락이 없다. 비교 이름 단위의 락을 잡고, 이미 잡혀 있으면 아무것도 시작하기 전에 실패한다. TASK-592 리뷰 중 발견했다.

## Completion Criteria

- [x] 같은 비교 이름의 compare up 두 개가 겹쳐 실행되면 둘째는 아무것도 시작하기 전에 명확한 오류로 실패한다 | verify: `go test ./internal/compare/ -run TestLockExcludesSecondHolderOfSameComparison -count=1 && go test ./internal/cli/ -run TestCompareUpRefusesWhileAnotherUpHoldsTheLock -count=1`
- [x] 락은 프로세스가 죽어도 남지 않는다 | verify: `go test ./internal/compare/ -run TestLockDoesNotSurviveItsProcess -count=1`

## Scope Notes

- 락은 `.sb/dva/compare/<이름>.lock`에 건 비차단 flock이다. 파일이 아니라 열린 파일 기술자가 락을 쥐므로 프로세스가 죽으면 OS가 풀고, 남은 `.lock` 파일은 아무것도 잠그지 않는다.
- `compare up`만 잠근다(카드 범위). dry-run은 아무것도 쓰지 않아 잠그지 않는다. `compare down`은 잠그지 않으며, up이 도는 동안의 down과의 경합은 이 카드 밖이다.
- flock이 없는 운영체제(`lock_other.go`)에서는 jobrun과 같이 "지원하지 않음" 오류로 실패한다(조용히 보호를 빼지 않는다).
