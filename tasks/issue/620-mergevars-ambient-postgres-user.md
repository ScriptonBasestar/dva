---
id: ISSUE-620
title: "TestMergeVarsDefaultSyntax fails when the ambient environment exports POSTGRES_USER"
type: bug
priority: P3
effort: S
status: todo
severity: low
discovered-in: "TASK-621 delivery: make test failed in a shell exporting POSTGRES_USER; independent reviewer reproduced it at pristine base a58ddad7"
discovered-at: 2026-11-17
created: 2026-11-17
ownership: local
---

## Summary

`TestMergeVarsDefaultSyntax`(internal/config/environment_test.go)는 배치
`MergeVars` 해석기로 `${POSTGRES_USER:-app}`를 풀 때 `os.Environ()`이 값을 제공하는지
본다. 테스트는 `DVA_TEST_UNSET_303`만 정리하고 `POSTGRES_USER`는 그대로 두므로, 로컬
Postgres를 쓰는 개발 셸처럼 이 변수를 export한 환경에서는 기대값
`postgres://gorisa@localhost/db`가 `postgres://<ambient>@localhost/db`가 되어
스위트가 실패한다. 소스 결함이 아니라 테스트의 환경 위생 결함이다.

## Reproduction

- `env POSTGRES_USER=scripton go test ./internal/config -run TestMergeVarsDefaultSyntax -count=1`
  → FAIL: `DB_URL = "postgres://scripton@localhost/db"` (2026-11-17, TASK-621 작업 셸)
- pristine base `a58ddad7`에서 동일 실패를 독립 리뷰가 재현했다(2026-11-17).
- `env -u POSTGRES_USER go test ./internal/config -run TestMergeVarsDefaultSyntax -count=1`
  → ok

회피: 이 변수를 export한 셸에서는 `env -u POSTGRES_USER make test`로 전체 스위트를
실행한다.

## Cause

테스트가 `os.Unsetenv("DVA_TEST_UNSET_303")`로 참조하지 않을 변수만 격리하고,
기본값 연산자로 참조하는 `POSTGRES_USER`의 앰비언트 값을 격리하지 않는다. 배치
해석기는 형제 조회보다 프로세스 환경을 먼저 본다.

## Expected vs Actual

- Expected: 테스트는 앰비언트 환경과 무관하게 항상 동일하게 통과한다.
- Actual: `POSTGRES_USER`가 export된 모든 셸에서 실패한다.

## 소유권 — 이 저장소다

결함은 상류 의존이 아니라 이 저장소의 테스트 격리에 있다. `MergeVars` 해석기의
동작(프로세스 환경 우선)은 의도된 것이므로 수정 위치도 이 저장소의 테스트다.

## Scope note

기록 전용 — 이번 전달에서 수정하지 않는다. 뿌리 수정은 테스트 소유 영역이다:
`DVA_TEST_UNSET_303`처럼 `os.Unsetenv("POSTGRES_USER")`로 격리하거나 값을 명시적으로
주입한다.
