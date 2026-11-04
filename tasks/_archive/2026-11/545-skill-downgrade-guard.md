---
id: TASK-545
title: "Refuse skill downgrades and finish am-optional guidance"
type: bug
priority: P2
status: done
created: 2026-10-31
quality-review: pass
quality-reviewed-at: 2026-11-04
quality-review-evidence: tasks/done/evidence/TASK-545/independent-review.json
---

## Summary

스킬 설치 상태가 outdated·newer·differs로 구분되고 증명 가능한 다운그레이드는 기본 거부한다(--allow-downgrade로만 허용). Agent Mesh 생성기가 cross-skill 링크를 sibling 상대 경로로 재작성해 설치된 어느 위치에서도 해석되게 하고, am 선택형 안내로 마무리된다.

## Completion Criteria

- [x] status separates outdated, newer and differs | verify: `go test -count=1 -run TestStatusClassifiesBundleDrift ./internal/skillinstall`
- [x] install refuses a provable downgrade | verify: `go test -count=1 -run TestInstallRefusesDowngradeUnlessAllowed ./internal/skillinstall`
- [x] Agent Mesh keeps sibling links resolvable | verify: `go test -count=1 -run TestAgentMeshRewritesCrossSkillLinks ./internal/skillinstall`
