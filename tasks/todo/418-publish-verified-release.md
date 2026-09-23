---
id: TASK-418
title: "Publish the verified release through the manual runbook"
type: chore
priority: P2
effort: M
exec-tier: standard
status: todo
created: 2026-09-23
depends-on: [TASK-417, TASK-370]
needs-human: true
---

## Summary

docs/52의 승인된 immutable commit·notes SHA-256으로 공개한다. 이 카드는 후보 개발과 분리된 공개 단계다. 기존에 사람 전용으로 보류된 자원 폐기는 별도 결정이며 공개와 묶어 수행하지 않는다.

## Completion Criteria

- [ ] 릴리스 commit, 버전 및 노트 digest에 대한 공개 승인이 있다 | verify: human — 승인 대상 identity를 기록
- [ ] 승인된 detached worktree에서 preflight와 공개를 수행한다 | verify: human — docs/52 절차의 명령과 결과를 기록
- [ ] postflight에서 원격 identity, 7 assets 및 checksum을 확인한다 | verify: human — release-postflight exit 0과 cleanup 결과를 기록
