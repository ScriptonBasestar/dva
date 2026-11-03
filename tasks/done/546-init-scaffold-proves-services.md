---
id: TASK-546
title: "Scaffold only interactions the project proves"
type: bug
priority: P2
status: done
created: 2026-10-31
quality-review: pass
quality-reviewed-at: 2026-11-04
quality-review-evidence: tasks/done/evidence/TASK-546/independent-review.json
---

## Summary

dva init 스큐폴드가 증명된 것만 남긴다: Compose app 서비스가 없으면 interactions 섹션을 만들지 않고, 선언되지 않은 npm 스크립트(dev/test/lint)를 drop하며 섹션 순서를 canonical으로 맞춘다.

## Completion Criteria

- [x] no interactions without an app service | verify: `go test -count=1 -run TestGenerateConfigOmitsInteractionsWithoutAppService ./internal/cli`
- [x] undeclared npm scripts are dropped | verify: `go test -count=1 -run TestGenerateConfigKeepsOnlyDeclaredNpmScripts ./internal/cli`
