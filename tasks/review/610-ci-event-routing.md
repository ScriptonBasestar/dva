---
id: TASK-610
title: "CI 이벤트별 입력 검증과 merge queue 연결"
type: feature
priority: P2
effort: M
exec-tier: strong
status: review
quality-review: pass
quality-review-evidence: tasks/done/evidence/TASK-610/review.md
created: 2026-11-16
---

## Summary

GitHub의 branch head·PR merge·queue merge 입력을 구분하고 실제 checkout SHA를 검증한다.
기존 job과 복구 독립성 검사는 유지한다. 정기 검사는 미채택 이유와 재검토 조건을
[원격 이벤트 계약](../../docs/ci-event-routing.md)에 기록한다.

## Steps

1. 원격 이벤트별 입력과 보호 규칙 조회 가능 범위를 확인한다.
2. merge_group·동시성·공유 입력 검사와 실제 YAML/임시 Git 회귀 검증을 추가한다.
3. 독립 strong 리뷰, lint·문서·full 검사 후 source에 통합한다.

## Limits

branch protection 조회는 HTTP 403으로 미확인이다. 권한·보호 규칙을 변경하거나
PR·외부 이슈를 만들지 않는다. 원격 queue 활성화와 run 성공은 로컬 검사와 별도다.

## Completion Criteria

- [x] 기존 필수 CI 범위와 이벤트별 입력을 보존한다 | verify: `go test ./tools/cilabels` (observed: 2026-11-16)

## Verification and remaining delivery

독립 strong 소스 검토, sh -n, scoped tests, make doc-check·make lint와
T2 dev d1f49dea4f1eaf7faf5921d453faafc8 (2m47.878s)가 통과했다.
최신 source 반영과 최종 clean-head full·통합·설치본 전달은 아직 완료하지 않았다.
현재 CE epoch 전환 확인을 기다리며 작업 브랜치에 구현과 증거를 보존한다.
프로필 추가를 원격 queue 활성화나 실제 hosted CI 성공으로 보고하지 않는다.
