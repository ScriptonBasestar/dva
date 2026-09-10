---
id: TASK-321
title: "destructive interactions: schema flag and agent-deny projection"
type: feature
priority: P2
effort: M
exec-tier: strong
created-at: 2026-09-05T10:30:00+09:00
source: "docs/dogfood/scripton-nd-stack.md"
status: todo
needs-human: true
---

# Task 321: destructive interaction ↔ agent-deny 연동

## Summary

nd-stack의 `db reset`/`redis flush`는 description 문구로만 파괴성을 표시한다. 스키마에 `destructive:`/`confirm:` 필드가 없고
agent-deny 목록은 `config env seal|show` 2건뿐 (`dva agent-deny status`). 제안: interaction 필드 추가 + install 시
`Bash(dva <name> <sub> *)` deny 투영 + 대화형 confirm.

## Completion Criteria

- [ ] 설계 결정 (docs/agent-deny-rules.md 갱신) | verify: human — internal/agentdeny/rules.go에 destructive interaction(예: nd-stack db reset/redis flush)용 신규 Gated Command 규칙이 추가되고, make generate로 재생성된 docs/agent-deny-rules.md의 "Gated commands" 표에 반영되었는지 확인
- [ ] 구현 + 테스트 (설계 확정 전에는 기계 바인딩을 만들 수 없다; groom에서 테스트 함수명으로 교체) | verify: human — 설계 승인 뒤 구현 범위와 테스트 이름을 확정해 이 criterion의 기계 바인딩으로 교체
