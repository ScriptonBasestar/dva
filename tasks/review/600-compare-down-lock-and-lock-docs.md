---
id: TASK-600
title: "compare down 락과 락 문서 보완"
type: bug
priority: P3
effort: S
exec-tier: standard
status: review
created: 2026-11-11
---

## Summary

TASK-595는 `compare up`만 이름 단위로 잠근다. `compare down`도 같은 기록을 읽고 쓰므로 up이 도는 동안 끼어들면 경합한다. 같은 락을 잡고, 잡혀 있으면 아무것도 멈추기 전에 실패하게 한다. 함께 `internal/config/constants.go:19`의 compare 디렉터리 주석에 `<이름>.lock`을 적는다. 남은 `.lock` 파일은 아무것도 잠그지 않으므로 지우는 일은 하지 않는다. TASK-595 리뷰 중 발견했다.

## Completion Criteria

- [ ] compare down이 같은 이름의 compare up이 도는 동안 아무것도 멈추기 전에 명확한 오류로 실패한다 | verify: `go test ./internal/cli/ -run TestCompareDownRefusesWhileAnotherRunHoldsTheLock -count=1 && go test ./internal/cli/ -run TestCompareDownReleasesTheLock -count=1 && go test ./internal/cli/ -run TestCompareDownDryRunIgnoresTheLock -count=1`
- [ ] internal/config/constants.go의 compare 디렉터리 주석이 <이름>.lock을 언급한다 | verify: `grep -q '<name>.lock' internal/config/constants.go`

## Scope Notes

- `runCompareDown`이 `compare up`과 같은 `compare.Lock(dir, name)`을 기록을 읽기 전에 비차단으로 잡고 `defer release()`로 모든 반환 경로에서 푼다. 잡혀 있으면 `compare_preflight_failed`로 "nothing was stopped: another `dva compare` of <name> is running"을 내고 아무것도 멈추지 않으며 기록도 건드리지 않는다. up의 "nothing was started: another `dva compare up`" 문구를 따르되, 잡고 있는 쪽이 up일 수도 down일 수도 있어 "`dva compare`"로 썼다.
- dry-run down은 아무것도 쓰거나 멈추지 않으므로 잠그지 않는다(up과 같음). 잠금 파일은 지우지 않는다: 남은 파일은 아무것도 잠그지 않는다.
- 잠금 구현상 up끼리·down끼리·up과 down 모두 같은 파일 flock으로 서로 배제한다. 잠금 순서는 선언 확인 뒤, 기록 읽기 전이다(선언이 없는 이름은 잠금 파일을 만들지 않는다).
- `constants.go` 주석에 `<name>.lock`을 적었다. USAGE.md의 `compare` 절에 up/down 락 문단을 추가했다.
- 바인딩: 기준 1은 세 테스트(락 보유 중 실패·멈춤 없음·기록 유지·해제 후 성공 / 실패·성공 경로 후 락 해제 / dry-run은 락 무시). 돌연변이: 락 획득을 no-op으로 바꾸면 `TestCompareDownRefusesWhileAnotherRunHoldsTheLock`가 실패했고, `defer release()`를 지우면 `TestCompareDownReleasesTheLock`가 실패했다. 각각 복원 후 통과.
