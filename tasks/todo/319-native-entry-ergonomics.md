---
id: TASK-319
title: "native entries: optional dir, post-build steps, explicit primary compose entry"
type: feature
priority: P2
effort: M
exec-tier: strong
created-at: 2026-09-05T10:30:00+09:00
source: "docs/dogfood/{dripter,scripton-db-orchestrator,primeno1,funbricks-postkit}.md"
status: todo
needs-human: true
---

# Task 319: native 엔트리 표현력

## Summary

이 카드는 dripter, scripton-db-orchestrator, primeno1, funbricks-postkit 네 프로젝트를
dva로 마이그레이션하는 과정에서 native 엔트리 선언의 표현력이 부족해 겪은 문제들을 모은
것이다. optional dir 체크, post-build step, 다중 compose 엔트리 중 primary 지정, native
`env:` 필드 문서화 부재가 각각 다른 프로젝트에서 마이그레이션을 가로막거나 우회를
강요했다. 아래 항목은 각각 설계 결정이 필요한 별개 이슈이며 아직 채택 여부가 확정되지
않았다.

1. **optional/dir-exists 조건 없음** — 미체크아웃 subproject dir이면 plan 전체 실패 (dripter). `optional: true` 제안.
2. **post-build step 없음** — `make build-api`는 build/api 복사까지 하지만 `dva build`는 cargo만 실행 (db-orchestrator).
3. **PrimaryComposeEntry 암묵 선택** — order 제거 후 이름순 최소 엔트리(lifecycle_helpers.go:164)가 provision compose_up과
   service 지정 interaction의 compose 파일 세트를 결정. 다중 compose 엔트리에 `primary: true` 같은 명시 수단 필요 (primeno1).
4. native runner `env:` 필드는 존재하나 문서화 부족으로 Makefile 경유가 선택됨 (postkit) — TASK-323 문서 범위와 연계.

## Completion Criteria

- [ ] 각 항목 설계 결정 기록 | verify: human — 카드 하단에 "Design record" 절이 추가되어 1)optional/dir-exists, 2)post-build step, 3)PrimaryComposeEntry 명시 수단, 4)native env: 문서화 4개 항목 각각에 대해 채택/보류 결정과 근거가 기록되었는지 확인
- [ ] 구현 + 테스트 (바인딩 잠정 — 설계 확정 후 groom에서 테스트 함수명으로 교체) | verify: `make test`
