---
id: TASK-595
title: "compare up 동시 실행 락"
type: bug
priority: P2
effort: S
exec-tier: standard
status: done
quality-review: pass
quality-reviewed-at: 2026-11-11T02:00:00Z
quality-review-evidence: tasks/done/evidence/TASK-595/independent-review.json
quality-review-receipt: tasks/done/evidence/TASK-595/done-review-4606199f6757b5cd14e6f78aa5f5935a01db9789dec35170a57cdf035fa05543.json
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

## Review Attempts

### 1. 2026-11-11 — reviewer: claude-code-subagent (opus, 독립), executor: claude-sonnet-5-5

- 대상: head 23298ca5. 판정: pass. 기준 1·2 PASS. 락은 상태를 읽기 전에 잡히고 모든 반환 경로에서 풀리며, `up`이 띄운 자식은 락 fd를 물려받지 않는다(Go는 O_CLOEXEC로 연다; 별도 프로그램으로 재현). 락 블록을 지운 복사본 돌연변이를 테스트가 잡았다.
- 비차단(후속): `compare down`은 잠그지 않아 up 중의 down 경합이 남는다(카드 밖). 남은 `.lock` 파일은 지워지지 않으나 아무것도 잠그지 않는다. `internal/config/constants.go:19` 주석이 `<이름>.lock`을 빠뜨렸다.

### 2. 2026-11-11 — reviewer: claude-code-subagent (opus, 독립, 앞선 리뷰어와 다른 에이전트), executor: claude-sonnet-5-5

- 대상: master 갱신 후 자동 리베이스된 head 44db5f18. 판정: pass. range-diff가 두 커밋 모두 "="(같은 패치)로 보고했고 `git diff`도 9개 파일에서 바이트 단위로 같다. 카드 검증과 compare·cli·lifecycle 테스트가 통과했다. 다음 전환: done/.
