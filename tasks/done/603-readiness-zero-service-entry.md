---
id: TASK-603
title: "compare readiness의 서비스 없는 항목 처리"
type: feature
priority: P2
status: done
quality-review: pass
quality-reviewed-at: 2026-11-11T09:30:00Z
quality-review-evidence: tasks/done/evidence/TASK-603/independent-review.json
quality-review-receipt: tasks/done/evidence/TASK-603/done-review-0d8ba647a66fcd3ce314dbfc6d6d064f1b4173c6a1bd3e4b99a3326e7566e26e.json
created: 2026-11-11
---

## Summary

`internal/cli/compare_result.go`의 `readinessText`는 서비스도 health 검사도 없는 항목(예: 멈춘 compose 프로젝트, `compose ps`가 `[]`를 돌려주는 경우)을 건너뛰므로, 다른 항목이 ready이면 그 항목이 ready를 막지 못한다. TASK-602 2차 리뷰에서 발견했고 master에 이미 있던 공백이다. 이런 항목을 `-`로 볼지 not ready로 볼지 정한다.

## Completion Criteria

- [x] 서비스·health 검사·오류가 모두 없는 항목은 판단 불가로 보아 다른 항목이 ready여도 readiness가 `-`이고, 다른 항목의 not ready·error는 그대로 우선한다 | verify: `go test ./internal/cli/ -run 'TestReadinessTextEmptyEntry|TestReadinessTextUnjudgeableServices' -count=1`

## Review Attempts

### 1. 2026-11-11 — reviewer: claude-code-subagent (opus 5.5, 독립), executor: claude-code-session (sonnet 5.5)

- 대상: 첫 리뷰는 head 2171e75c, 재기반 뒤 재리뷰는 head 0b6d4a84(코드 차이 없음, 597 마감 문서만 다름). 판정: pass. 기준 1 PASS: 서비스·health 검사·오류가 없는 항목은 `-`로 보며 다른 항목의 not ready·error가 우선한다. 새 분기를 지우면 "empty and ready" 케이스가 실패한다(첫 리뷰어가 변이로 확인; 재리뷰는 논리로만 추론).
- 비차단: `ps`에 `-a`가 없어 죽은 compose 프로젝트가 not ready가 아니라 `-`로 나온다(별도 카드 필요). "empty alone" 케이스는 새 분기 없이도 통과해 변별력이 없다. script 항목에 health_checks가 없거나 kubectl·vagrant 조회가 실패하면 이제 `-`로 나오며, 판정된 적이 없으므로 정직한 값으로 본다. 다음 전환: done/.
