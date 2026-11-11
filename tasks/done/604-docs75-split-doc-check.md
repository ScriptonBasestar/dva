---
id: TASK-604
title: "docs/75 비교 설계 문서 분할로 doc-check 복구"
type: feature
priority: P2
status: done
quality-review: pass
quality-reviewed-at: 2026-11-11T11:30:00Z
quality-review-evidence: tasks/done/evidence/TASK-604/independent-review.json
quality-review-receipt: tasks/done/evidence/TASK-604/done-review-4e54400c6c83c7f72c4083cb19f8a577899c12d31caf4ba8068e5dc83fd77068.json
created: 2026-11-11
---

## Summary

docs/75-variant-comparison-design.md가 10240바이트 한도를 넘어 `make doc-check`가 실패했다. config 모드 격리 규칙(인스턴스·빌드 상태)과 결정 기록을 docs/75-variant-comparison-details.md로 옮기고 원문서에는 링크만 남긴다. 내용은 한 곳에만 둔다.

## Completion Criteria

- [x] docs/75 비교 설계 문서가 크기 한도 안에 들고 doc-check가 통과한다 | verify: `make doc-check` (regression-guard)
- [x] 분할로 옮긴 내용이 중복 없이 한 곳에만 있고 기존 링크가 깨지지 않는다 | verify: human — 리뷰어가 링크와 중복을 확인

## Review Attempts

### 1. 2026-11-11 — reviewer: claude-code-subagent (opus 5.5, 독립), executor: claude-code-session/0479d29b-718c-480f-84a2-b12d94024f92

- 대상 head 92857574. 판정: pass. 기준 1: `make doc-check` 통과(문서 8269바이트). 기준 2: 옮긴 33줄이 details 문서와 글자 단위로 일치하고 중복이 없으며 앵커 `#config-모드-격리-규칙`·`#결정-기록`과 모든 링크가 유효하다. 기존 587 카드의 `#결정-기록` 링크는 스텁을 거쳐 한 번 더 이동한다.
- 기준 2는 `human` 바인딩이지만 사람이 확인한 것이 아니다. 독립 AI 리뷰어(서브에이전트)가 확인했고 사람의 확인은 없었다.
- 비차단: 설계 문서 모드 절의 "config 모드는 결정 기록을 따른다" 문장을 details 앵커로 직접 링크하면 이동 한 번을 줄일 수 있다. docs/README 항목에서 낡은 "(채택, 미구현)" 라벨이 빠졌다(이미 부분 구현 상태). 다음 전환: done/.
