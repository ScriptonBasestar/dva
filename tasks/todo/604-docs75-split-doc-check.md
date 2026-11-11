---
id: TASK-604
title: "docs/75 비교 설계 문서 분할로 doc-check 복구"
type: feature
priority: P2
status: todo
created: 2026-11-11
---

## Summary

docs/75-variant-comparison-design.md가 10240바이트 한도를 넘어 `make doc-check`가 실패했다. config 모드 격리 규칙(인스턴스·빌드 상태)과 결정 기록을 docs/75-variant-comparison-details.md로 옮기고 원문서에는 링크만 남긴다. 내용은 한 곳에만 둔다.

## Completion Criteria

- [ ] docs/75 비교 설계 문서가 크기 한도 안에 들고 doc-check가 통과한다 | verify: `make doc-check` (regression-guard)
- [ ] 분할로 옮긴 내용이 중복 없이 한 곳에만 있고 기존 링크가 깨지지 않는다 | verify: human — 리뷰어가 링크와 중복을 확인
