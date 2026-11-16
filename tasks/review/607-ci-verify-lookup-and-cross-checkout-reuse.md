---
id: TASK-607
title: "실행 없이 CI 단계별 재사용 가능 여부를 조회하고 checkout 간 재사용을 증명 뒤 허용"
type: feature
priority: P2
effort: L
exec-tier: strong
status: review
quality-review: pass
quality-review-evidence: tasks/done/evidence/TASK-607/review.md
created: 2026-11-14
---

## Summary

착수 시 결과 재사용은 `dva ci <profile>` 실행 안에서만 일어나고 같은 canonical checkout으로
제한됐다([docs/75-ci-result-reuse.md](../../docs/75-ci-result-reuse.md)). 리뷰어와 통합
세션은 "현재 트리에 유효한 성공 결과가 있는가"를 실행 없이 물을 수 없어 같은 검사를 다른
worktree에서 다시 돌린다. 소비자인 ce-agent-kit 결정 050
(`decisions/050-verification-receipt-as-handoff-unit.md`)의 21일 측정에서 리뷰어 검증
30.5h 중 선행 실행이 있는 것이 9.6h였고, 리뷰어 디렉터리에 그 사이 변경이 없던 6.2h 가운데
선행 실행이 같은 디렉터리였던 것은 0.14h였다. 사용자는 2026-11-14에 checkout 간 재사용을
변이 대조로 증명한 뒤 허용하는 방향을 승인했다.

## Scope

- 프로필의 단계별 재사용 키를 현재 트리에서 다시 계산해 상태만 보고하는 조회를 추가한다.
  명령 이름과 위치(`dva ci verify`는 제안)는 구현 카드가 정한다.
- 선언된 입력·도구·환경 지문이 같으면 같은 저장소의 다른 worktree 결과를 재사용한다.
  경로에 의존하는 검사를 가려내는 방법(선언 필드 또는 제외 규칙)을 설계에 포함한다.
- `docs/75-ci-result-reuse.md`의 "한계와 검증" 절을 바뀐 계약으로 고친다.

## Decision Boundaries

- 스스로 결정해도 된다: 명령 이름, JSON 필드 이름, 결과 저장 위치의 내부 구조.
- 멈추고 보고한다: 재사용 키 조건을 약하게 해야만 다른 worktree에서 적중할 때, fresh의
  의미가 바뀔 때, 선언 없는 단계나 OCI·child_ci 단계를 재사용 대상으로 넣어야 할 때.

## Out of Scope

- 다른 기기 사이의 결과 공유와 외부 게시, 산출물 복원, 프로젝트의 `reuse` 선언 작성.

## Completion Criteria

- [x] 조회 명령이 단계를 실행하지 않고 covered, stale, absent, not-reusable과 원 실행 ID·시각을 JSON으로 돌려주며 결과 캐시와 receipt를 쓰지 않는다 | verify: `go test ./internal/cli -run TestCILookupDoesNotExecuteOrCreateState` (observed: 2026-11-16)
- [x] 같은 저장소의 다른 worktree에서 같은 입력은 covered, 입력·도구·환경 변이와 파일 추가·삭제·산출물 유실·만료는 covered가 아님이 임시 저장소 대조로 고정된다 | verify: `go test ./internal/cirun -run 'TestRepositoryReuse|TestCheckoutScope|TestLookupExpired'` (observed: 2026-11-16)
- [x] fresh 프로필과 --fresh는 다른 checkout의 결과도 읽지 않는다 | verify: `go test ./internal/cirun -run 'TestRepositoryReuseAndLookupWorktrees|TestLookupFreshDoesNotReadMalformedCache'` (observed: 2026-11-16)

## Adopted interface

`dva ci <profile> --lookup`을 사용해 기존 verify 프로필 이름을 보존한다.
기본 checkout 격리를 유지하고 명시적 `scope: repository`에만 별도 v2 영역을 쓴다.
scope는 경로 독립에 대한 소유자 선언이며 임의 shell 동작의 자동 증명이 아니다.
정본 인터페이스와 한계는 [조회 계약](../../docs/ci-reuse-lookup.md)에 둔다.
새 API는 단계·probe를 실행하거나 DVA state를 쓰지 않고, 오류와 부재를 구분한다.
필수 전체 검사 범위는 유지하며 final full·독립 검토 후 통합한다.

## Verification and remaining delivery

독립 strong 소스 검토에서 전체 조회의 혼합 시점 문제를 찾아 final barrier를 추가했다.
실제 DVA 감독 아래 T2 실행은 일반 parent metadata를 composition으로 오인해 실패했고,
token을 지우지 않고 구분을 바로잡았다. Run의 중첩 실행 보호는 유지한다.
수정 후 scoped tests·make lint·make doc-check·check-generate와
T2 dev 9447314a5161cea73063bab478e2c473 (1m27.497s)가 통과했다.
빌드 바이너리의 실제 --lookup도 새 XDG state를 만들지 않고 9개 비선언 단계를 보고했다.
새 cache 선언의 실제 소비자 채택은 이 카드 범위 밖이다.
최신 source 반영과 최종 full·통합·설치는 아직 완료하지 않았다.
CE 상태 전환 확인을 기다리는 동안 코드와 증거를 작업 브랜치에 보존한다.
