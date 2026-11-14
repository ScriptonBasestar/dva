---
id: TASK-595
title: "실행 없이 CI 단계별 재사용 가능 여부를 조회하고 checkout 간 재사용을 증명 뒤 허용"
type: feature
priority: P2
effort: L
exec-tier: strong
status: todo
created: 2026-11-14
---

## Summary

현재 결과 재사용은 `dva ci <profile>` 실행 안에서만 일어나고 같은 canonical checkout으로
제한된다([docs/75-ci-result-reuse.md](../../docs/75-ci-result-reuse.md)). 리뷰어와 통합
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

- [ ] 조회 명령이 단계를 실행하지 않고 covered, stale, absent, not-reusable과 원 실행 ID·시각을 JSON으로 돌려주며 결과 캐시와 receipt를 쓰지 않는다 | verify: human — 구현 카드가 추가한 테스트 이름을 이 줄의 go test -run 바인딩으로 바꾸고 make test 통과를 확인한다
- [ ] 같은 저장소의 다른 worktree에서 같은 입력은 covered, 입력·도구·환경 변이와 파일 추가·삭제·산출물 유실·만료는 covered가 아님이 임시 저장소 대조로 고정된다 | verify: human — 대조 테스트 이름을 바인딩으로 바꾸고 통과를 확인한다
- [ ] fresh 프로필과 --fresh는 다른 checkout의 결과도 읽지 않는다 | verify: human — fresh 대조 테스트 이름을 바인딩으로 바꾸고 통과를 확인한다
