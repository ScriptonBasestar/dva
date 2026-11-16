---
id: TASK-609
title: "DVA staged commit 경량 검사와 core 게이트 분리"
type: feature
priority: P2
effort: L
exec-tier: strong
status: done
quality-review: pass
quality-review-evidence: tasks/done/evidence/TASK-609/review.md
created: 2026-11-16
---

## Summary

사용자가 채택한 CI 후속 작업의 첫 전달 단위다. 기존 core 검사를 보존하면서
입력 범위가 명확한 staged commit 파일럿을 적용한다. 실행·범위는
[파일럿 계약](../../docs/ci-commit-pilot.md)이 소유한다.

## Steps

1. staged verifier의 입력 경계·범위 선택을 fixture로 검증한다.
2. commit/core/full 프로필·Make·문서 비교 게이트를 일치시킨다.
3. 독립 strong 리뷰 후 실제 staged 실행의 예산과 입력 attestation을 확인한다.
4. 최종 lint·full·생성물 검사 후 source에 통합하고 작업 자원을 회수한다.

## Decision and followup

TASK-607은 별도 공개 실행 계약 단위로 이어서 구현한다. runtime의 공통 commit 상한을
모든 소비자에 바꾸거나 staged snapshot·자동 stash를 추가하지 않는다.

## Completion Criteria

- [x] staged 입력 차이와 검사 범위 선택을 회귀 검증한다 | verify: `go test ./tools/commitverify` (observed: 2026-11-16)
- [x] 기존 core와 full의 문서 게이트를 보존한다 | verify: `go test ./tools/ciparity` (observed: 2026-11-16)

## Verification

독립 strong 리뷰에서 숨김 index flag 우회를 발견해 거부·real Git 회귀 검증을 추가했다.
지원하는 staged Go fixture의 실제 프로필 4단계는 2.184초에 통과했다.
이는 fixture 관찰이며 DVA 전체 변경의 30초 달성 증거가 아니다.
공유 입력 전체 실행 실험 두 건은 test90초에서 실패했고 통과 근거로 쓰지 않는다.
그 결과 빠른 경로를 명시 제한했다. 현재 공유 입력은 0.772초에 요구대로 거부되고
후속 3단계는 미실행이다. 더 넓은 검사는 core/full 경계에서 유지한다.

make lint·make check-generate·문서 검사와 T2 dev run
83c7b85d5ee14b4fcd88e8023c972bd7 (2m39.690s)가 통과했다.
최종 커밋의 full·통합 리뷰·push·회수는 CE 실행 receipt와 별도 DVA receipt로 확인한다.
